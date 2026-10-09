package testkit

import (
	"bytes"
	"testing"

	luca "git.bytestone.uk/hum3/go-luca"
)

func exportGoluca(t *testing.T, ledger luca.Ledger) string {
	t.Helper()
	var buf bytes.Buffer
	if err := ledger.Export(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
