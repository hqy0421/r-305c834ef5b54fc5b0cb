package trdsql

import (
	"bytes"
	"path/filepath"
	"testing"
)

// deviceFile is a JSONL file whose attachment column dominates its size.
const deviceFile = "device.jsonl"

func runPruningQuery(t *testing.T, query string, opts ...ReadOpt) string {
	t.Helper()
	return runPruningQueryWrite(t, query, nil, opts...)
}

// runPruningRows returns the output lines as a set, for assertions that must
// not depend on JSON object key order (which determines column order).
func runPruningRows(t *testing.T, query string, opts ...ReadOpt) map[string]bool {
	out := runPruningQuery(t, query, opts...)
	set := make(map[string]bool)
	for _, line := range splitLines(out) {
		if line != "" {
			set[line] = true
		}
	}
	return set
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func runPruningQueryWrite(t *testing.T, query string, wopts []WriteOpt, opts ...ReadOpt) string {
	t.Helper()
	outStream := new(bytes.Buffer)
	trd := NewTRDSQL(NewImporter(opts...), NewExporter(NewWriter(append(wopts, OutStream(outStream))...)))
	if err := trd.Exec(query); err != nil {
		t.Fatalf("Exec(%q): %v", query, err)
	}
	return outStream.String()
}

func TestFieldPruning_AutoDetectedColumns(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	got := runPruningQuery(t, "SELECT device_id, event_date FROM "+file)
	want := "d001,2026-01-01\nd002,2026-01-02\nd003,2026-01-02\n"
	if got != want {
		t.Errorf("auto pruning result = %q, want %q", got, want)
	}
}

func TestFieldPruning_WhereAndGroupBy(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	query := "SELECT event_date, count(device_id) FROM " + file +
		" WHERE device_id IS NOT NULL GROUP BY event_date ORDER BY event_date"
	got := runPruningQuery(t, query)
	want := "2026-01-01,1\n2026-01-02,2\n"
	if got != want {
		t.Errorf("group by result = %q, want %q", got, want)
	}
}

func TestFieldPruning_JoinUnchanged(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	// The report-style join: only device code and date are referenced.
	query := "SELECT d.device_id, d.event_date FROM " + file +
		" AS d LEFT JOIN " + filepath.Join(dataDir, "test.csv") + " AS c ON (d.device_id = c.c1)"
	got := runPruningQuery(t, query)
	want := "d001,2026-01-01\nd002,2026-01-02\nd003,2026-01-02\n"
	if got != want {
		t.Errorf("join result = %q, want %q", got, want)
	}
}

func TestFieldPruning_ManualColumns(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	// SELECT * emits columns in reader (JSON key) order, which is not stable,
	// so compare the per-row field sets instead of positional strings.
	got := runPruningRows(t, "SELECT * FROM "+file, InColumns([]string{"device_id", "event_date"}))
	want := map[string]bool{
		"d001,2026-01-01": true,
		"d002,2026-01-02": true,
		"d003,2026-01-02": true,
		"2026-01-01,d001": true,
		"2026-01-02,d002": true,
		"2026-01-02,d003": true,
	}
	for row := range got {
		if !want[row] {
			t.Errorf("unexpected pruned row %q", row)
		}
	}
	if len(got) != 3 {
		t.Errorf("manual pruning rows = %v, want 3 rows", got)
	}
}

func TestFieldPruning_StarFallbackKeepsEverything(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	// A star query must fall back to importing every column, including the
	// large attachment and the nested meta object.
	star := runPruningQuery(t, "SELECT * FROM "+file)
	pruned := runPruningQuery(t, "SELECT device_id, event_date FROM "+file)
	if len(star) <= len(pruned) {
		t.Errorf("star fallback did not retain the attachment column")
	}
	// The full attachment value must still be queryable after the fallback.
	raw := runPruningQueryWrite(t, "SELECT attachment FROM "+file+" WHERE device_id = 'd001'", []WriteOpt{OutFormat(RAW)})
	want := repeatA(20000) + "\n"
	if raw != want {
		t.Errorf("attachment content mismatch (got %d bytes, want %d)", len(raw), len(want))
	}
}

func TestFieldPruning_NestedFieldPreserved(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	got := runPruningQuery(t, "SELECT device_id, meta FROM "+file+" ORDER BY device_id")
	want := "d001,\"{\"\"firmware\"\":\"\"1.0\"\"}\"\n" +
		"d002,\"{\"\"firmware\"\":\"\"1.1\"\"}\"\n" +
		"d003,\"{\"\"firmware\"\":\"\"1.1\"\"}\"\n"
	if got != want {
		t.Errorf("nested field result = %q, want %q", got, want)
	}
}

func TestFieldPruning_ResultsMatchFullImport(t *testing.T) {
	file := filepath.Join(dataDir, deviceFile)
	query := "SELECT event_date, device_id FROM " + file + " ORDER BY device_id"
	full := runPruningQuery(t, query)
	// An unrelated extra column in the manual list must not change the
	// connection/grouping result; missing columns read as NULL.
	pruned := runPruningQuery(t, query, InColumns([]string{"device_id", "event_date", "unused"}))
	if full != pruned {
		t.Errorf("pruned result %q != full import %q", pruned, full)
	}
}

func repeatA(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'A'
	}
	return string(b)
}
