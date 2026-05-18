package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCreateAndValidateToken(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "turnstile.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if _, err := store.CreateUser("james"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := store.CreateProject("notesmith", ""); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	record, err := store.CreateToken("james", []string{"notesmith"}, "james phone", nil)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}
	if len(record.Projects) != 1 || record.Projects[0] != "notesmith" {
		t.Fatalf("unexpected projects: %#v", record.Projects)
	}

	result, err := store.ValidateToken(record.Token, "notesmith", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if !result.Authorized {
		t.Fatalf("expected authorized result, got %#v", result)
	}

	denied, err := store.ValidateToken(record.Token, "bag-app", "127.0.0.1", "test")
	if err == nil && denied.Authorized {
		t.Fatalf("expected denial for unknown project")
	}
}

func TestExpiredTokenDenied(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "turnstile.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if _, err := store.CreateUser("lisa"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := store.CreateProject("notesmith", ""); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	expiresAt := time.Now().UTC().Add(-time.Hour)
	record, err := store.CreateToken("lisa", []string{"notesmith"}, "expired", &expiresAt)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	result, err := store.ValidateToken(record.Token, "notesmith", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if result.Authorized || result.Reason != "token expired" {
		t.Fatalf("expected expired denial, got %#v", result)
	}
}
