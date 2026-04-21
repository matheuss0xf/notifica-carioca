package postgres

import "testing"

func TestNewMigrator(t *testing.T) {
	m := NewMigrator("file://migrations")
	if m.sourceURL != "file://migrations" {
		t.Fatalf("unexpected source url: %q", m.sourceURL)
	}
}

func TestMigratorUpReturnsErrorForInvalidSource(t *testing.T) {
	m := NewMigrator("://bad-source")
	if err := m.Up("postgres://notifica:notifica@localhost:5432/notifica_carioca?sslmode=disable"); err == nil {
		t.Fatalf("expected error for invalid migration source")
	}
}
