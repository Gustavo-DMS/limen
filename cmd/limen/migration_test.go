package main

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thecodearcher/limen"
)

func TestMigrationVersion(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		seq        int
		schemaName string
		want       string
	}{
		{name: "first migration", seq: 0, schemaName: "users", want: "20240101120000_users"},
		{name: "second advances one second", seq: 1, schemaName: "sessions", want: "20240101120001_sessions"},
		{name: "third advances two seconds", seq: 2, schemaName: "accounts", want: "20240101120002_accounts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := migrationVersion(base, tt.seq, tt.schemaName); got != tt.want {
				t.Errorf("migrationVersion(seq=%d, %q) = %q, want %q", tt.seq, tt.schemaName, got, tt.want)
			}
		})
	}
}

func TestMigrationVersionUniqueAndIncreasing(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	names := []string{"a", "b", "c", "d", "e"}

	seen := map[string]bool{}
	prevPrefix := ""

	for seq, name := range names {
		version := migrationVersion(base, seq, name)

		if seen[version] {
			t.Fatalf("duplicate version %q at seq %d", version, seq)
		}
		seen[version] = true

		prefix, _, _ := strings.Cut(version, "_")
		if prevPrefix != "" && prefix <= prevPrefix {
			t.Fatalf("version prefix not strictly increasing: %q after %q", prefix, prevPrefix)
		}
		prevPrefix = prefix
	}
}

func TestOrderSchemasByDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schemas limen.SchemaDefinitionMap
		want    []limen.SchemaName
	}{
		{
			name: "referenced table precedes referencing table",
			schemas: limen.SchemaDefinitionMap{
				"accounts": {
					TableName:   "accounts",
					ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "users"}},
				},
				"users": {TableName: "users"},
			},
			want: []limen.SchemaName{"users", "accounts"},
		},
		{
			name: "chain orders referenced tables first",
			schemas: limen.SchemaDefinitionMap{
				"a": {TableName: "a", ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "b"}}},
				"b": {TableName: "b", ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "c"}}},
				"c": {TableName: "c"},
			},
			want: []limen.SchemaName{"c", "b", "a"},
		},
		{
			name: "independent schemas are alphabetical",
			schemas: limen.SchemaDefinitionMap{
				"charlie": {TableName: "charlie"},
				"alpha":   {TableName: "alpha"},
				"bravo":   {TableName: "bravo"},
			},
			want: []limen.SchemaName{"alpha", "bravo", "charlie"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := orderSchemasByDependency(tt.schemas); !slices.Equal(got, tt.want) {
				t.Errorf("orderSchemasByDependency() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOrderSchemasByDependencyEmitsEverySchema covers cases where the spec only
// guarantees that every schema is emitted exactly once, not a specific order:
// non-ordering references are dropped, and cycles must still terminate.
func TestOrderSchemasByDependencyEmitsEverySchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schemas limen.SchemaDefinitionMap
	}{
		{
			name: "foreign key to table outside the config is ignored",
			schemas: limen.SchemaDefinitionMap{
				"accounts": {
					TableName:   "accounts",
					ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "users"}},
				},
			},
		},
		{
			name: "self-referencing foreign key is ignored",
			schemas: limen.SchemaDefinitionMap{
				"nodes": {
					TableName:   "nodes",
					ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "nodes"}},
				},
			},
		},
		{
			name: "cycle still terminates",
			schemas: limen.SchemaDefinitionMap{
				"a": {TableName: "a", ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "b"}}},
				"b": {TableName: "b", ForeignKeys: []limen.ForeignKeyDefinition{{ReferencedSchema: "a"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := orderSchemasByDependency(tt.schemas)

			if len(got) != len(tt.schemas) {
				t.Fatalf("got %d schemas %v, want %d", len(got), got, len(tt.schemas))
			}

			seen := map[limen.SchemaName]bool{}
			for _, name := range got {
				if seen[name] {
					t.Fatalf("schema %q emitted more than once: %v", name, got)
				}
				seen[name] = true

				if _, ok := tt.schemas[name]; !ok {
					t.Fatalf("unexpected schema %q in output: %v", name, got)
				}
			}
		})
	}
}
