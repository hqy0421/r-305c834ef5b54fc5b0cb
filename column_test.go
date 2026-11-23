package trdsql

import (
	"reflect"
	"sort"
	"testing"
)

func sortedEqual(t *testing.T, got, want []string) bool {
	t.Helper()
	if len(got) != len(want) {
		return false
	}
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	return reflect.DeepEqual(g, w)
}

func TestQueryColumns(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string // nil means fall back to importing every column
	}{
		{
			name:  "star",
			query: "SELECT * FROM device.jsonl",
			want:  nil,
		},
		{
			name:  "qualifiedStar",
			query: "SELECT d.* FROM device.jsonl AS d",
			want:  nil,
		},
		{
			name:  "countStar",
			query: "SELECT count(*) FROM device.jsonl",
			want:  nil,
		},
		{
			name:  "subquery",
			query: "SELECT device_id FROM (SELECT device_id, payload FROM device.jsonl)",
			want:  nil,
		},
		{
			name:  "explicitColumns",
			query: "SELECT device_id, event_date FROM device.jsonl",
			want:  []string{"device_id", "event_date"},
		},
		{
			name:  "whereAndGroupBy",
			query: "SELECT device_id, count(1) FROM device.jsonl WHERE event_date > '2026-01-01' GROUP BY device_id ORDER BY device_id",
			want:  []string{"device_id", "event_date"},
		},
		{
			name:  "qualifiedNames",
			query: "SELECT d.device_id, d.event_date FROM device.jsonl AS d WHERE d.device_id = 'x'",
			want:  []string{"device_id", "event_date"},
		},
		{
			name:  "join",
			query: "SELECT a.device_id, b.event_date FROM a.jsonl AS a LEFT JOIN b.jsonl AS b ON (a.device_id = b.device_id)",
			want:  []string{"device_id", "event_date"},
		},
		{
			name:  "implicitAliases",
			query: "SELECT t1.device_id, t2.event_date FROM a.jsonl t1 JOIN b.jsonl t2 ON t1.id = t2.id",
			want:  []string{"device_id", "event_date", "id"},
		},
		{
			name:  "commaJoinFileNameNotColumn",
			query: "SELECT a.device_id FROM a.jsonl a, b.jsonl b WHERE a.id=b.id",
			want:  []string{"device_id", "id"},
		},
		{
			name:  "whereGroupByWithPath",
			query: "SELECT device_id, event_date FROM `testdata/device.jsonl` WHERE device_id IS NOT NULL GROUP BY event_date",
			want:  []string{"device_id", "event_date"},
		},
		{
			name:  "quotedColumns",
			query: `SELECT "device id", "event date" FROM "device file.jsonl"`,
			want:  []string{"device id", "event date"},
		},
		{
			name:  "backQuotedColumns",
			query: "SELECT `device-id` FROM `device.jsonl`",
			want:  []string{"device-id"},
		},
		{
			name:  "functionOnColumn",
			query: "SELECT upper(device_id), substr(event_date, 1, 7) AS month FROM device.jsonl",
			want:  []string{"device_id", "event_date"},
		},
		{
			name:  "aliasNotImportedAsColumn",
			query: "SELECT device_id AS id FROM device.jsonl",
			want:  []string{"device_id"},
		},
		{
			name:  "distinct",
			query: "SELECT DISTINCT event_date FROM device.jsonl",
			want:  []string{"event_date"},
		},
		{
			name:  "update",
			query: "UPDATE device.jsonl SET status = 'x' WHERE device_id = '1'",
			want:  []string{"status", "device_id"},
		},
		{
			name:  "noFrom",
			query: "SELECT 1 + 1",
			want:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QueryColumns(tt.query)
			if tt.want == nil {
				if got != nil {
					t.Errorf("QueryColumns() = %v, want nil (fallback)", got)
				}
				return
			}
			if !sortedEqual(t, got, tt.want) {
				t.Errorf("QueryColumns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumnNameToken(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"device_id", "device_id"},
		{"d.device_id", "device_id"},
		{`"device id"`, "device id"},
		{"`device-id`", "device-id"},
		{"'literal'", ""},
		{"123", ""},
		{"1.5", ""},
		{"col?", ""},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			if got := columnNameToken(tt.token); got != tt.want {
				t.Errorf("columnNameToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}
