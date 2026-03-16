package testkit

import (
	"testing"

	luca "github.com/drummonds/go-luca"
	_ "github.com/drummonds/go-postgres"
)

// NewTestLedger creates an in-memory SQLite ledger for testing.
func NewTestLedger(t *testing.T) luca.Ledger {
	t.Helper()
	ledger, err := luca.NewLedger(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ledger.Close() })
	return ledger
}
