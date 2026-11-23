package cmd

import (
	"bytes"
	"log"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_parseInColumns(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "empty", value: "", want: nil},
		{name: "spaces", value: "   ", want: nil},
		{name: "single", value: "device_id", want: []string{"device_id"}},
		{name: "multiple", value: "device_id,event_date", want: []string{"device_id", "event_date"}},
		{name: "trimAndDedup", value: " device_id , event_date ,,device_id ", want: []string{"device_id", "event_date"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseInColumns(tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseInColumns(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestCli_Run_IColumns(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "manualPruning",
			args: []string{
				"trdsql",
				"-icolumns", "device_id,event_date",
				"SELECT * FROM " + filepath.Join("..", "testdata", "device.jsonl"),
			},
			want: "d001,2026-01-01\nd002,2026-01-02\nd003,2026-01-02\n",
		},
		{
			name: "autoPruning",
			args: []string{
				"trdsql",
				"SELECT device_id, event_date FROM " + filepath.Join("..", "testdata", "device.jsonl"),
			},
			want: "d001,2026-01-01\nd002,2026-01-02\nd003,2026-01-02\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
			cli := Cli{OutStream: outStream, ErrStream: errStream}
			var buf bytes.Buffer
			log.SetOutput(&buf)
			if got := cli.Run(tt.args); got != 0 {
				t.Fatalf("Run() = %d: %s", got, errStream.String())
			}
			if outStream.String() != tt.want {
				t.Errorf("Run() output = %q, want %q", outStream.String(), tt.want)
			}
		})
	}
}
