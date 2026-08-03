package migrations

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebastiancorrales/gym-go/internal/config"
	"gorm.io/gorm"
)

// TestHotQueriesUseIndexes asserts, via EXPLAIN QUERY PLAN, that the queries on
// the hot paths resolve through an index instead of scanning the table.
//
// This matters beyond raw speed: a scan holds its read lock for longer, and the
// hourly MarkExpiredSubscriptions scan held a WRITE lock over the whole
// subscriptions table. It also guards against someone changing a WHERE clause
// and silently losing the index.
func TestHotQueriesUseIndexes(t *testing.T) {
	db := freshMigratedDB(t)

	cases := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "login por email",
			query: "SELECT * FROM users WHERE email = ?",
			args:  []interface{}{"a@b.com"},
		},
		{
			name:  "usuarios del gym ordenados",
			query: "SELECT * FROM users WHERE gym_id = ? ORDER BY created_at DESC",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000"},
		},
		{
			name:  "busqueda por documento en el gym",
			query: "SELECT * FROM users WHERE document_number = ? AND gym_id = ?",
			args:  []interface{}{"123", "00000000-0000-0000-0000-000000000000"},
		},
		{
			name:  "suscripcion activa del usuario (cada check-in)",
			query: "SELECT * FROM subscriptions WHERE user_id = ? AND status = ? AND end_date > ?",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000", "ACTIVE", "2026-01-01"},
		},
		{
			name:  "listado de suscripciones del gym",
			query: "SELECT * FROM subscriptions WHERE gym_id = ? AND status = ? ORDER BY created_at DESC",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000", "ACTIVE"},
		},
		{
			name:  "auto-expirar suscripciones (cada hora, bajo lock de escritura)",
			query: "UPDATE subscriptions SET status = 'EXPIRED' WHERE status = ? AND end_date < ?",
			args:  []interface{}{"ACTIVE", "2026-01-01"},
		},
		{
			name:  "suscripciones por rango de fecha (reportes)",
			query: "SELECT * FROM subscriptions WHERE gym_id = ? AND date >= ? AND date <= ?",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000", "2026-01-01", "2026-01-31"},
		},
		{
			name:  "detalles de una venta (1473 veces en el cierre diario)",
			query: "SELECT * FROM sale_details WHERE sale_id = ?",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000"},
		},
		{
			name:  "ventas por rango de fecha",
			query: "SELECT * FROM sales WHERE date >= ? AND date <= ?",
			args:  []interface{}{"2026-01-01", "2026-01-31"},
		},
		{
			name:  "miembros de una suscripcion grupal",
			query: "SELECT * FROM subscription_members WHERE subscription_id = ?",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000"},
		},
		{
			name:  "accesos del gym por fecha",
			query: "SELECT * FROM access_logs WHERE gym_id = ? AND access_time >= ? ORDER BY access_time DESC",
			args:  []interface{}{"00000000-0000-0000-0000-000000000000", "2026-01-01"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := explain(t, db, tc.query, tc.args...)

			if !strings.Contains(plan, "USING INDEX") && !strings.Contains(plan, "USING COVERING INDEX") {
				t.Errorf("la consulta no usa índice.\nquery: %s\nplan:\n%s", tc.query, plan)
			}
			// SEARCH significa búsqueda indexada; SCAN es recorrido completo.
			if strings.Contains(plan, "SCAN") {
				t.Errorf("la consulta hace SCAN de la tabla.\nquery: %s\nplan:\n%s", tc.query, plan)
			}
		})
	}
}

// TestEmailIsUnique documents that the email uniqueness rule is enforced by the
// database now, not only by a check-then-insert in the handler (which two
// concurrent sign-ups could slip through).
func TestEmailIsUnique(t *testing.T) {
	db := freshMigratedDB(t)

	insert := func() error {
		return db.Exec(`INSERT INTO users (id, gym_id, email, first_name, last_name, role, status)
		                VALUES (?, ?, 'dup@example.com', 'A', 'B', 'MEMBER', 'ACTIVE')`,
			randomUUID(), "00000000-0000-0000-0000-000000000000").Error
	}

	if err := insert(); err != nil {
		t.Fatalf("primer insert debería funcionar: %v", err)
	}
	if err := insert(); err == nil {
		t.Fatal("el segundo insert con el mismo email debería fallar por el índice único")
	}
}

func freshMigratedDB(t *testing.T) *gorm.DB {
	t.Helper()

	database, err := config.NewDatabase(&config.DatabaseConfig{
		DatabasePath: filepath.Join(t.TempDir(), "indexes.db"),
		ForMigration: true,
	})
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := Migrate(database.DB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return database.DB
}

func explain(t *testing.T, db *gorm.DB, query string, args ...interface{}) string {
	t.Helper()

	var rows []struct {
		ID     int
		Parent int
		NotUse int
		Detail string
	}
	if err := db.Raw("EXPLAIN QUERY PLAN "+query, args...).Scan(&rows).Error; err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}

	var b strings.Builder
	for _, r := range rows {
		b.WriteString("  " + r.Detail + "\n")
	}
	return b.String()
}

func randomUUID() string {
	// Suficiente para el test: solo hace falta que sea distinto entre llamadas.
	uuidCounter++
	return "00000000-0000-0000-0000-" + pad(uuidCounter)
}

var uuidCounter int

func pad(n int) string {
	s := ""
	for i := 0; i < 12; i++ {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
