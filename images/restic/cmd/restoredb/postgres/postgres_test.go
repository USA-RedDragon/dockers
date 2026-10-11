package postgres_test

import (
	"testing"

	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/postgres"
)

func TestIsCustomFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		head string
		want bool
	}{
		{name: "empty", head: "", want: false},
		{name: "magic only", head: "PGDMP", want: true},
		{name: "magic and header", head: "PGDMP\x01\x10\x00", want: true},
		{name: "short prefix", head: "PGDM", want: false},
		{name: "lowercase", head: "pgdmp", want: false},
		{name: "plain sql comment", head: "--\n-- PostgreSQL database dump", want: false},
		{name: "psql restrict meta-command", head: "\\restrict abc", want: false},
		{name: "gzip", head: "\x1f\x8b\x08\x00", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := postgres.IsCustomFormat([]byte(tt.head)); got != tt.want {
				t.Errorf("IsCustomFormat(%q) = %v, want %v", tt.head, got, tt.want)
			}
		})
	}
}
