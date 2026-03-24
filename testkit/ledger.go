package testkit

import (
	"testing"

	luca "codeberg.org/hum3/go-luca"
	_ "codeberg.org/hum3/go-postgres"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// NewTestLedger creates an in-memory SQLite ledger for testing.
func NewTestLedger(t *testing.T) luca.Ledger {
	t.Helper()
	ledger, err := luca.NewLedger(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ledger.Close(); err != nil {
			t.Errorf("closing ledger: %v", err)
		}
	})
	return ledger
}
