package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Database holds database connection
type Database struct {
	DB *gorm.DB
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DatabasePath string
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
	LogLevel     logger.LogLevel

	// ForMigration opens a short-lived handle meant only for schema migration:
	// a single connection and no prepared-statement cache. The sqlite migrator
	// implements AlterColumn/DropColumn by recreating the table, and GORM's
	// statement cache does not invalidate itself on DDL — running migrations on
	// the application handle can leave statements pointing at a dropped table.
	ForMigration bool
}

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DatabasePath: "gym-go.db",
		MaxIdleConns: 8,
		MaxOpenConns: 8,
		MaxLifetime:  0,
		LogLevel:     logger.Warn,
	}
}

// buildSQLiteDSN builds the DSN for modernc.org/sqlite from a plain filesystem path.
//
// Do NOT add a "file:" prefix: modernc.org/sqlite (sqlite.go, newConn) strips
// everything from the first '?' before calling sqlite3_open_v2 whenever the DSN
// does not start with "file:". That is why the path reaches the engine verbatim
// and is never URL-decoded — backslashes, spaces, '&', '=' and '%' in a Windows
// path are safe as-is. With a "file:" prefix we would have to switch to forward
// slashes and percent-encode the whole path.
//
// Every parameter here is per-connection except journal_mode, which is persisted
// in the database header. Setting them in the DSN is the only way to have them
// applied to *every* pooled connection — a `db.Exec("PRAGMA ...")` reaches
// exactly one arbitrary connection out of the pool.
func buildSQLiteDSN(path string) string {
	params := []string{
		// The driver does not set a busy timeout by default (unlike mattn/go-sqlite3,
		// which uses 5000ms). Without it, any lock collision fails instantly with
		// SQLITE_BUSY instead of waiting.
		"_pragma=busy_timeout(5000)",
	}

	if walSupported(path) {
		params = append(params, "_pragma=journal_mode(WAL)")
	} else {
		log.Printf("⚠️  WAL desactivado: %q parece estar en red (UNC). "+
			"WAL usa memoria compartida y no es seguro sobre SMB/NFS.", path)
	}

	params = append(params,
		"_pragma=synchronous(NORMAL)", // safe against crashes under WAL
		"_pragma=foreign_keys(1)",
		"_pragma=cache_size(-16000)", // 16 MB page cache per connection (negative = KiB)
		"_txlock=immediate",          // BEGIN IMMEDIATE: avoids BUSY on deferred→write upgrade
	)

	return path + "?" + strings.Join(params, "&")
}

// walSupported reports whether WAL is safe for this path. WAL relies on shared
// memory and must not be used on network filesystems. UNC paths are the case we
// can detect portably; anything that slips through is caught at startup by
// verifyPragmas, which reports the journal mode the engine actually settled on.
func walSupported(path string) bool {
	return !strings.HasPrefix(path, `\\`) && !strings.HasPrefix(path, "//")
}

// NewDatabase creates a new database connection
func NewDatabase(config *DatabaseConfig) (*Database, error) {
	if config == nil {
		config = DefaultDatabaseConfig()
	}

	level := config.LogLevel
	if level == 0 {
		// The zero value is not a valid gorm log level and silences everything,
		// including SQL errors and slow queries. Warn is the sane floor.
		level = logger.Warn
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "[gorm] ", log.LstdFlags),
		logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      level,
			// Several repositories use First() as an existence check, so
			// ErrRecordNotFound is expected control flow, not a problem.
			IgnoreRecordNotFoundError: true,
			Colorful:                  false, // Windows console
		},
	)

	// Use pure Go SQLite driver for cross-platform compatibility
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        buildSQLiteDSN(config.DatabasePath),
	}, &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC().Round(0)
		},
		PrepareStmt: !config.ForMigration,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Connection pool. A SQLite file has a single effective writer, so a large
	// pool buys no throughput — it only multiplies lock collisions, memory
	// (cache_size per connection) and prepared statements.
	if config.ForMigration {
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetConnMaxLifetime(0)
	} else {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(config.MaxLifetime)
	}

	if err := verifyPragmas(db, config); err != nil {
		return nil, err
	}

	return &Database{DB: db}, nil
}

// verifyPragmas reports the settings the engine actually applied. journal_mode
// is the one that can silently fall back (a filesystem without shared-memory
// support keeps the rollback journal), so it is checked explicitly instead of
// assumed — the previous version of this file assumed WAL was active and it
// never was.
func verifyPragmas(db *gorm.DB, config *DatabaseConfig) error {
	var journalMode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		return fmt.Errorf("checking journal_mode: %w", err)
	}

	var busyTimeout, foreignKeys int
	db.Raw("PRAGMA busy_timeout").Scan(&busyTimeout)
	db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys)

	maxOpen := config.MaxOpenConns
	if config.ForMigration {
		maxOpen = 1
	}

	log.Printf("✅ SQLite: journal_mode=%s busy_timeout=%dms foreign_keys=%d max_open=%d",
		journalMode, busyTimeout, foreignKeys, maxOpen)

	if !strings.EqualFold(journalMode, "wal") && walSupported(config.DatabasePath) {
		log.Printf("⚠️  WAL NO quedó activo (journal_mode=%s). Se continúa con rollback "+
			"journal; lectores y escritores se excluirán entre sí. Revisar permisos de "+
			"escritura en el directorio de la base de datos.", journalMode)
	}
	if busyTimeout == 0 {
		log.Printf("⚠️  busy_timeout=0: cualquier colisión de lock fallará al instante " +
			"con SQLITE_BUSY. El DSN no se aplicó como se esperaba.")
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck checks database connection health
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
