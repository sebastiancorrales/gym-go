package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
)

// TestBackupProducesRestorableCopy es la prueba de que las copias sirven.
//
// La version anterior copiaba el .db con os.ReadFile/os.WriteFile, lo que no es un
// backup valido de SQLite: la lectura abarca varias syscalls, asi que un COMMIT en
// medio mezcla paginas de dos estados, y nunca se copiaba el sidecar -wal donde vive
// parte del estado comprometido. Nadie lo habria notado hasta necesitar restaurar.
func TestBackupProducesRestorableCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gym-go.db")

	db, err := config.NewDatabase(&config.DatabaseConfig{DatabasePath: path, MaxOpenConns: 4, MaxIdleConns: 4})
	if err != nil { t.Fatal(err) }
	if err := migrations.Migrate(db.DB); err != nil { t.Fatal(err) }
	if err := db.DB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil { t.Fatal(err) }
	for i := 0; i < 500; i++ {
		if err := db.DB.Exec("INSERT INTO t (v) VALUES (?)", "fila").Error; err != nil { t.Fatal(err) }
	}

	if err := backupDatabase(db.DB, path, 7); err != nil { t.Fatalf("backup: %v", err) }

	entries, _ := os.ReadDir(filepath.Join(dir, "backups"))
	if len(entries) != 1 { t.Fatalf("se esperaba 1 backup, hay %d", len(entries)) }
	dest := filepath.Join(dir, "backups", entries[0].Name())
	t.Logf("backup: %s (%d bytes; original %d bytes)", entries[0].Name(), dbSize(t, dest), dbSize(t, path))

	// Lo que importa: la copia se puede abrir y esta intacta.
	restored, err := config.NewDatabase(&config.DatabaseConfig{DatabasePath: dest, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil { t.Fatalf("abriendo el backup: %v", err) }
	defer restored.Close()

	var check string
	restored.DB.Raw("PRAGMA integrity_check").Scan(&check)
	if check != "ok" { t.Errorf("integrity_check del backup = %q, want ok", check) }

	var n int64
	restored.DB.Raw("SELECT COUNT(*) FROM t").Scan(&n)
	if n != 500 { t.Errorf("filas en el backup = %d, want 500", n) }
	t.Logf("backup restaurable: integrity_check=%s, filas=%d", check, n)
	db.Close()
}

func dbSize(t *testing.T, p string) int64 {
	fi, err := os.Stat(p)
	if err != nil { t.Fatal(err) }
	return fi.Size()
}
