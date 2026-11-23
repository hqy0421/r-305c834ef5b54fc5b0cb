package trdsql

import (
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func arraySortEqual(t *testing.T, a []string, b []string) bool {
	t.Helper()
	if len(a) != len(b) {
		return false
	}

	copyA := make([]string, len(a))
	copyB := make([]string, len(b))

	copy(copyA, a)
	copy(copyB, b)

	sort.Strings(copyA)
	sort.Strings(copyB)

	return reflect.DeepEqual(copyA, copyB)
}

func TestNewJSONReader(t *testing.T) {
	type args struct {
		reader io.Reader
		opts   *ReadOpts
	}
	tests := []struct {
		name    string
		args    args
		want    *JSONReader
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				reader: strings.NewReader(""),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   nil,
				preRead: nil,
			},
			wantErr: false,
		},
		{
			name: "invalidJSON",
			args: args{
				reader: strings.NewReader("t"),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   nil,
				preRead: nil,
			},
			wantErr: true,
		},
		{
			name: "emptyJSON",
			args: args{
				reader: strings.NewReader("{}"),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   nil,
				preRead: []map[string]any{{}},
			},
			wantErr: false,
		},
		{
			name: "test1",
			args: args{
				reader: strings.NewReader(`[{"c1":"1","c2":"Orange"},{"c1":"2","c2":"Melon"},{"c1":"3","c2":"Apple"}]`),
				opts:   NewReadOpts(InPreRead(3)),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]any{{"c1": "1", "c2": "Orange"}, {"c1": "2", "c2": "Melon"}, {"c1": "3", "c2": "Apple"}},
			},
			wantErr: false,
		},
		{
			name: "test2",
			args: args{
				reader: strings.NewReader(`
{"c1":"1","c2":"Orange"}
{"c1":"2","c2":"Melon"}
{"c1":"3","c2":"Apple"}`),
				opts: NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]any{{"c1": "1", "c2": "Orange"}},
			},
			wantErr: false,
		},
		{
			name: "testArray",
			args: args{
				reader: strings.NewReader(`[["a"],["b"]]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1"},
				preRead: []map[string]any{{"c1": "[\"a\"]"}, {"c1": "[\"b\"]"}},
			},
			wantErr: false,
		},
		{
			name: "testArray2",
			args: args{
				reader: strings.NewReader(`[["a","b"],["c","d"]]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1"},
				preRead: []map[string]any{{"c1": "[\"a\",\"b\"]"}, {"c1": "[\"c\",\"d\"]"}},
			},
			wantErr: false,
		},
		{
			name: "testArray3",
			args: args{
				reader: strings.NewReader(`["a","b"]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1"},
				preRead: []map[string]any{{"c1": "a"}, {"c1": "b"}},
			},
			wantErr: false,
		},
		{
			name: "testObject",
			args: args{
				reader: strings.NewReader(`{"a":"b"}`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"a"},
				preRead: []map[string]any{{"a": "b"}},
			},
			wantErr: false,
		},
		{
			name: "diffColumn",
			args: args{
				reader: strings.NewReader(`
{"id":"1","name":"Orange"}
{"id":"2","name":"Melon"}
{"id":"3","name":"Apple"}
{"id":"4","name":"Banana","color":"Yellow"}`),
				opts: NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"id", "name"},
				preRead: []map[string]any{{"id": "1", "name": "Orange"}},
			},
			wantErr: false,
		},
		{
			name: "diffColumn2",
			args: args{
				reader: strings.NewReader(`
{"id":"1","name":"Orange"}
{"id":"2","name":"Melon"}
{"id":"3","name":"Apple"}
{"id":"4","name":"Banana","color":"Yellow"}`),
				opts: NewReadOpts(InPreRead(5)),
			},
			want: &JSONReader{
				names: []string{"id", "name", "color"},
				preRead: []map[string]any{
					{"id": "1", "name": "Orange"},
					{"id": "2", "name": "Melon"},
					{"id": "3", "name": "Apple"},
					{"id": "4", "name": "Banana", "color": "Yellow"},
				},
			},
			wantErr: false,
		},
		{
			name: "testJQ",
			args: args{
				reader: strings.NewReader(`[{"c1":"1","c2":"Orange"},{"c1":"2","c2":"Melon"},{"c1":"3","c2":"Apple"}]`),
				opts:   NewReadOpts(InJQ(".[0]")),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]any{{"c1": "1", "c2": "Orange"}},
			},
			wantErr: false,
		},
		{
			name: "testJQ2",
			args: args{
				reader: strings.NewReader(`{"employees":[
						{"name":"Shyam", "email":"shyamjaiswal@gmail.com"},
						{"name":"Bob", "email":"bob32@gmail.com"},
						{"name":"Jai", "email":"jai87@gmail.com"}
					]}`),
				opts: NewReadOpts(InJQ(".employees")),
			},
			want: &JSONReader{
				names: []string{"name", "email"},
				preRead: []map[string]any{
					{"name": "Shyam", "email": "shyamjaiswal@gmail.com"},
					{"name": "Bob", "email": "bob32@gmail.com"},
					{"name": "Jai", "email": "jai87@gmail.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "testJQ3",
			args: args{
				reader: strings.NewReader(`{"menu": {
						"id": "file",
						"value": "File",
						"popup": {
						  "menuitem": [
							{"value": "New", "onclick": "CreateDoc()"},
							{"value": "Open", "onclick": "OpenDoc()"},
							{"value": "Save", "onclick": "SaveDoc()"}
						  ]
						}
					  }}`),
				opts: NewReadOpts(InJQ(`".menu.popup.menuitem"`)),
			},
			want: &JSONReader{
				names: []string{"value", "onclick"},
				preRead: []map[string]any{
					{"value": "New", "onclick": "CreateDoc()"},
					{"value": "Open", "onclick": "OpenDoc()"},
					{"value": "Save", "onclick": "SaveDoc()"},
				},
			},
			wantErr: false,
		},
		{
			name: "testJQ4",
			args: args{
				reader: strings.NewReader(`[{"id":1},{"id":2},{"id":3}]`),
				opts:   NewReadOpts(InJQ(".")),
				// opts:   NewReadOpts(InJQ(`. as {$a} ?// [$a] ?// $a | $a`)),
			},
			want: &JSONReader{
				names: []string{"id"},
				preRead: []map[string]any{
					{"id": "1"},
					{"id": "2"},
					{"id": "3"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewJSONReader(tt.args.reader, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJSONReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !arraySortEqual(t, got.names, tt.want.names) {
				t.Errorf("NewJSONReader() = %v, want %v", got.names, tt.want.names)
			}
			if !reflect.DeepEqual(got.preRead, tt.want.preRead) {
				t.Errorf("NewJSONReader() = %v, want %v", got.preRead, tt.want.preRead)
			}
		})
	}
}

func TestNewJSONReaderWithColumnPruning(t *testing.T) {
	jsonl := `
{"device_id":"d1","event_date":"2026-01-01","payload":{"temp":1},"attachment":"AAAA"}
{"device_id":"d2","event_date":"2026-01-02","payload":{"temp":2},"attachment":"BBBB"}
{"device_id":"d3","event_date":"2026-01-03","payload":{"temp":3},"attachment":"CCCC"}`
	// rowMap converts a positional row into a name-keyed map because JSON
	// object key order (and therefore trdsql column order) is not stable.
	rowMap := func(names []string, row []any) map[string]any {
		m := make(map[string]any, len(names))
		for i, name := range names {
			m[name] = row[i]
		}
		return m
	}
	t.Run("prunePreRead", func(t *testing.T) {
		reader, err := NewJSONReader(
			strings.NewReader(jsonl),
			NewReadOpts(InColumns([]string{"device_id", "event_date"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		names, _ := reader.Names()
		if !arraySortEqual(t, names, []string{"device_id", "event_date"}) {
			t.Errorf("names = %v", names)
		}
		pre := reader.PreReadRow()
		want := map[string]any{"device_id": "d1", "event_date": "2026-01-01"}
		if !reflect.DeepEqual(rowMap(names, pre[0]), want) {
			t.Errorf("PreReadRow() = %v, want %v", pre, want)
		}
	})
	t.Run("pruneStreamRows", func(t *testing.T) {
		reader, err := NewJSONReader(
			strings.NewReader(jsonl),
			NewReadOpts(InColumns([]string{"device_id", "event_date"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		names, _ := reader.Names()
		row := make([]any, 2)
		var got []map[string]any
		for {
			r, err := reader.ReadRow(row)
			if err != nil {
				break
			}
			got = append(got, rowMap(names, append([]any(nil), r...)))
		}
		want := []map[string]any{
			{"device_id": "d2", "event_date": "2026-01-02"},
			{"device_id": "d3", "event_date": "2026-01-03"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ReadRow() = %v, want %v", got, want)
		}
	})
	t.Run("nestedFieldPreserved", func(t *testing.T) {
		reader, err := NewJSONReader(
			strings.NewReader(jsonl),
			NewReadOpts(InColumns([]string{"device_id", "payload"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		names, _ := reader.Names()
		if !arraySortEqual(t, names, []string{"device_id", "payload"}) {
			t.Errorf("names = %v", names)
		}
		pre := reader.PreReadRow()
		if len(pre) != 1 || len(pre[0]) != 2 {
			t.Fatalf("PreReadRow() = %v", pre)
		}
		want := map[string]any{"device_id": "d1", "payload": `{"temp":1}`}
		if got := rowMap(names, pre[0]); !reflect.DeepEqual(got, want) {
			t.Errorf("PreReadRow() = %v, want %v", got, want)
		}
	})
	t.Run("missingColumnIsNULL", func(t *testing.T) {
		reader, err := NewJSONReader(
			strings.NewReader(jsonl),
			NewReadOpts(InColumns([]string{"device_id", "absent"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		names, _ := reader.Names()
		row := make([]any, 2)
		r, err := reader.ReadRow(row)
		if err != nil {
			t.Fatal(err)
		}
		got := rowMap(names, r)
		if got["device_id"] != "d2" || got["absent"] != nil {
			t.Errorf("ReadRow() = %v, want device_id=d2 absent=nil", got)
		}
	})
	t.Run("numericValueMatchesFullPath", func(t *testing.T) {
		numeric := `{"id":1,"big":100000000000000000000,"drop":"x"}` + "\n" +
			`{"id":2,"big":2.5,"drop":"y"}`
		pruned, err := NewJSONReader(
			strings.NewReader(numeric),
			NewReadOpts(InColumns([]string{"id", "big"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		full, err := NewJSONReader(
			strings.NewReader(numeric),
			NewReadOpts(),
		)
		if err != nil {
			t.Fatal(err)
		}
		// Consume the preread row and compare the streamed rows.
		_ = pruned.PreReadRow()
		_ = full.PreReadRow()
		prunedRow := make([]any, 2)
		fullRow := make([]any, 3)
		pr, _ := pruned.ReadRow(prunedRow)
		fr, _ := full.ReadRow(fullRow)
		// Full path columns are id, big, drop; compare the first two by name.
		fullNames, _ := full.Names()
		fullMap := map[string]any{}
		for i, n := range fullNames {
			if n == "id" || n == "big" {
				fullMap[n] = fr[i]
			}
		}
		prunedNames, _ := pruned.Names()
		prunedMap := map[string]any{}
		for i, n := range prunedNames {
			prunedMap[n] = pr[i]
		}
		if !reflect.DeepEqual(prunedMap, fullMap) {
			t.Errorf("numeric mismatch: pruned=%v full=%v", prunedMap, fullMap)
		}
	})
	t.Run("jsonArrayPruned", func(t *testing.T) {
		array := `[{"device_id":"d1","attachment":"X"},{"device_id":"d2","attachment":"Y"}]`
		reader, err := NewJSONReader(
			strings.NewReader(array),
			NewReadOpts(InPreRead(2), InColumns([]string{"device_id"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		names, _ := reader.Names()
		if !arraySortEqual(t, names, []string{"device_id"}) {
			t.Errorf("names = %v", names)
		}
		pre := reader.PreReadRow()
		want := [][]any{{"d1"}, {"d2"}}
		if !reflect.DeepEqual(pre, want) {
			t.Errorf("PreReadRow() = %v, want %v", pre, want)
		}
	})
	t.Run("scalarArrayFallbackColumn", func(t *testing.T) {
		reader, err := NewJSONReader(
			strings.NewReader(`["a","b"]`),
			NewReadOpts(InColumns([]string{"c1"})),
		)
		if err != nil {
			t.Fatal(err)
		}
		names, _ := reader.Names()
		if !arraySortEqual(t, names, []string{"c1"}) {
			t.Errorf("names = %v", names)
		}
	})
}

func TestNewJSONReaderWithNULL(t *testing.T) {
	type args struct {
		reader io.Reader
		opts   *ReadOpts
	}
	tests := []struct {
		name    string
		args    args
		want    *JSONReader
		wantErr bool
	}{
		{
			name: "testNULL1",
			args: args{
				reader: strings.NewReader(`[{"id":1},{"id":null},{"id":3}]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names: []string{"id"},
				preRead: []map[string]any{
					{"id": "1"},
					{"id": nil},
					{"id": "3"},
				},
			},
			wantErr: false,
		},
		{
			name: "testNULL2",
			args: args{
				reader: strings.NewReader(`[{"id":1},{"id":"N"},{"id":3}]`),
				opts: NewReadOpts(
					InNeedNULL(true),
					InNULL("N"),
				),
			},
			want: &JSONReader{
				names: []string{"id"},
				preRead: []map[string]any{
					{"id": "1"},
					{"id": nil},
					{"id": "3"},
				},
			},
			wantErr: false,
		},
		{
			name: "testNULL3",
			args: args{
				reader: strings.NewReader(`[{"c1":"1","c2":null},{"c1":"2","c2":"\\N"},{"c1":"3","c2":"Apple"}]`),
				opts: NewReadOpts(
					InNeedNULL(true),
					InNULL("\\N"),
				),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]any{{"c1": "1", "c2": nil}, {"c1": "2", "c2": nil}, {"c1": "3", "c2": "Apple"}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewJSONReader(tt.args.reader, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJSONReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !arraySortEqual(t, got.names, tt.want.names) {
				t.Errorf("NewJSONReader() = %v, want %v", got.names, tt.want.names)
			}
			if !reflect.DeepEqual(got.preRead, tt.want.preRead) {
				t.Errorf("NewJSONReader() = %v, want %v", got.preRead, tt.want.preRead)
			}
		})
	}
}
