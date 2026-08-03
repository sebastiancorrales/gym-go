package config

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestPragmasAppliedToEveryPooledConnection is the regression test for the root
// cause of the SQLITE_BUSY reports: the PRAGMAs live in the DSN so the driver
// applies them on every Open, whereas the previous code ran
// db.Exec("PRAGMA foreign_keys = ON") once, which reaches exactly one arbitrary
// connection out of the pool and leaves the rest with the defaults.
func TestPragmasAppliedToEveryPooledConnection(t *testing.T) {
	const poolSize = 8

	db, err := NewDatabase(&DatabaseConfig{
		DatabasePath: filepath.Join(t.TempDir(), "pragmas.db"),
		MaxIdleConns: poolSize,
		MaxOpenConns: poolSize,
	})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	defer db.Close()

	sqlDB, err := db.DB.DB()
	if err != nil {
		t.Fatalf("DB(): %v", err)
	}

	// Grab every connection in the pool at once and hold them, so each check
	// provably runs on a distinct connection instead of reusing a lucky one.
	ctx := context.Background()
	for i := 0; i < poolSize; i++ {
		conn, err := sqlDB.Conn(ctx)
		if err != nil {
			t.Fatalf("conn %d: %v", i, err)
		}
		defer conn.Close()

		var busyTimeout, foreignKeys int
		var journalMode string
		if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
			t.Fatalf("conn %d busy_timeout: %v", i, err)
		}
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatalf("conn %d foreign_keys: %v", i, err)
		}
		if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
			t.Fatalf("conn %d journal_mode: %v", i, err)
		}

		if busyTimeout != 5000 {
			t.Errorf("conn %d: busy_timeout = %d, want 5000", i, busyTimeout)
		}
		if foreignKeys != 1 {
			t.Errorf("conn %d: foreign_keys = %d, want 1", i, foreignKeys)
		}
		if !strings.EqualFold(journalMode, "wal") {
			t.Errorf("conn %d: journal_mode = %q, want wal", i, journalMode)
		}
	}
}

// TestWALPersistsInHeader checks that WAL is recorded in the database file
// itself (bytes 18 and 19 go from 1 to 2), which is what makes the setting
// survive restarts and is how it can be verified from outside the process.
func TestWALPersistsInHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "header.db")

	db, err := NewDatabase(&DatabaseConfig{DatabasePath: path, MaxOpenConns: 2, MaxIdleConns: 2})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	// Force at least one page to be written so the header is flushed.
	if err := db.DB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	header := readHeader(t, path)
	if header[18] != 2 || header[19] != 2 {
		t.Errorf("header[18,19] = %d,%d — want 2,2 (WAL); 1,1 means rollback journal",
			header[18], header[19])
	}
}

func TestBuildSQLiteDSN(t *testing.T) {
	t.Run("windows path is passed through verbatim", func(t *testing.T) {
		// modernc.org/sqlite strips everything from the first '?' before opening
		// the file (as long as there is no "file:" prefix), so the path must not
		// be rewritten: no forward slashes, no percent-encoding.
		const path = `C:\ProgramData\Gym-Go\gym-go.db`
		dsn := buildSQLiteDSN(path)

		gotPath, params, found := strings.Cut(dsn, "?")
		if !found {
			t.Fatalf("DSN has no parameters: %q", dsn)
		}
		if gotPath != path {
			t.Errorf("path = %q, want %q", gotPath, path)
		}
		for _, want := range []string{
			"_pragma=busy_timeout(5000)",
			"_pragma=journal_mode(WAL)",
			"_pragma=synchronous(NORMAL)",
			"_pragma=foreign_keys(1)",
			"_txlock=immediate",
		} {
			if !strings.Contains(params, want) {
				t.Errorf("DSN is missing %s: %q", want, params)
			}
		}
	})

	t.Run("WAL is skipped on UNC paths", func(t *testing.T) {
		// WAL relies on shared memory and is unsafe over SMB/NFS.
		dsn := buildSQLiteDSN(`\\server\share\gym-go.db`)
		if strings.Contains(dsn, "journal_mode(WAL)") {
			t.Errorf("WAL must not be enabled for a UNC path: %q", dsn)
		}
		if !strings.Contains(dsn, "busy_timeout(5000)") {
			t.Errorf("busy_timeout must still be set for a UNC path: %q", dsn)
		}
	})
}

// TestConcurrentReadsAndWritesDoNotReturnBusy exercises the combination that
// used to fail: concurrent readers and writers on the same file. Under a
// rollback journal with busy_timeout=0 this produced "database is locked";
// under WAL with a busy timeout it must complete cleanly.
func TestConcurrentReadsAndWritesDoNotReturnBusy(t *testing.T) {
	db, err := NewDatabase(&DatabaseConfig{
		DatabasePath: filepath.Join(t.TempDir(), "concurrent.db"),
		MaxIdleConns: 8,
		MaxOpenConns: 8,
	})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	defer db.Close()

	if err := db.DB.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, n INTEGER)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	const workers, iterations = 20, 25
	var wg sync.WaitGroup
	errs := make(chan error, workers*iterations)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if w%2 == 0 {
					if err := db.DB.Exec("INSERT INTO items (n) VALUES (?)", i).Error; err != nil {
						errs <- err
					}
					continue
				}
				var count int64
				if err := db.DB.Raw("SELECT COUNT(*) FROM items").Scan(&count).Error; err != nil {
					errs <- err
				}
			}
		}(w)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent access failed: %v", err)
	}
}

// readHeader returns the 100-byte SQLite file header.
func readHeader(t *testing.T, path string) []byte {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer f.Close()

	header := make([]byte, 100)
	if _, err := io.ReadFull(f, header); err != nil {
		t.Fatalf("reading header of %s: %v", path, err)
	}
	return header
}
