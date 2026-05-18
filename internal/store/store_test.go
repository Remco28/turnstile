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

func TestWhoHasAccessAndGrantUpdate(t *testing.T) {
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
	if _, err := store.CreateProject("bag-app", ""); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := store.CreateProject("tinyfish", ""); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	record, err := store.CreateToken("james", []string{"notesmith"}, "james phone", nil)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	access, err := store.ListProjectAccess("notesmith")
	if err != nil {
		t.Fatalf("ListProjectAccess() error = %v", err)
	}
	if len(access) != 1 || access[0].TokenID != record.ID || access[0].User != "james" {
		t.Fatalf("unexpected access rows: %#v", access)
	}

	updated, err := store.ReplaceTokenProjects(record.ID, []string{"notesmith", "bag-app"})
	if err != nil {
		t.Fatalf("ReplaceTokenProjects() error = %v", err)
	}
	if len(updated.Projects) != 2 {
		t.Fatalf("expected 2 projects after grant update, got %#v", updated.Projects)
	}

	bagResult, err := store.ValidateToken(record.Token, "bag-app", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if !bagResult.Authorized {
		t.Fatalf("expected bag-app access after grant update, got %#v", bagResult)
	}
}

func TestAccessLogListing(t *testing.T) {
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
	if _, err := store.CreateProject("bag-app", ""); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	record, err := store.CreateToken("james", []string{"notesmith"}, "james phone", nil)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	if _, err := store.ValidateToken(record.Token, "notesmith", "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("ValidateToken(notesmith) error = %v", err)
	}
	if _, err := store.ValidateToken(record.Token, "bag-app", "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("ValidateToken(bag-app) error = %v", err)
	}

	logs, err := store.ListAccessLog("", 10)
	if err != nil {
		t.Fatalf("ListAccessLog() error = %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 log rows, got %#v", logs)
	}
	if logs[0].Project == "" {
		t.Fatalf("expected project on access log row, got %#v", logs[0])
	}
}

func TestReissueToken(t *testing.T) {
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
	if _, err := store.CreateProject("bag-app", ""); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	oldRecord, err := store.CreateToken("lisa", []string{"notesmith", "bag-app"}, "lisa laptop", &expiresAt)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	reissued, err := store.ReissueToken(oldRecord.ID, "", nil)
	if err != nil {
		t.Fatalf("ReissueToken() error = %v", err)
	}
	if reissued.Old.ID != oldRecord.ID || reissued.New.ID == oldRecord.ID {
		t.Fatalf("unexpected token ids after reissue: %#v", reissued)
	}
	if reissued.Old.RevokedAt == nil {
		t.Fatalf("expected old token revoked after reissue: %#v", reissued.Old)
	}
	if len(reissued.New.Projects) != 2 {
		t.Fatalf("expected new token to keep grants, got %#v", reissued.New)
	}
	if reissued.New.Token == oldRecord.Token {
		t.Fatalf("expected new raw token value")
	}

	oldResult, err := store.ValidateToken(oldRecord.Token, "notesmith", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("ValidateToken(old) error = %v", err)
	}
	if oldResult.Authorized || oldResult.Reason != "token revoked" {
		t.Fatalf("expected revoked old token, got %#v", oldResult)
	}

	newResult, err := store.ValidateToken(reissued.New.Token, "bag-app", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("ValidateToken(new) error = %v", err)
	}
	if !newResult.Authorized {
		t.Fatalf("expected new token authorized, got %#v", newResult)
	}
}
