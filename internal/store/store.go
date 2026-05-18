package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Remco28/turnstile/internal/models"
	turnstileToken "github.com/Remco28/turnstile/internal/token"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func initSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tokens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  expires_at TEXT,
  revoked_at TEXT,
  last_used_at TEXT
);
CREATE TABLE IF NOT EXISTS token_projects (
  token_id INTEGER NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
  project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY (token_id, project_id)
);
CREATE TABLE IF NOT EXISTS access_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token_id INTEGER REFERENCES tokens(id) ON DELETE SET NULL,
  project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
  presented_token_prefix TEXT NOT NULL,
  authorized INTEGER NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  remote_addr TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
`
	_, err := db.Exec(schema)
	return err
}

func (s *Store) CreateUser(name string) (models.User, error) {
	name = normalizeName(name)
	if name == "" {
		return models.User{}, errors.New("user name is required")
	}
	now := time.Now().UTC()
	result, err := s.db.Exec(`INSERT INTO users(name, created_at) VALUES(?, ?)`, name, now.Format(time.RFC3339))
	if err != nil {
		return models.User{}, err
	}
	id, _ := result.LastInsertId()
	return models.User{ID: id, Name: name, CreatedAt: now}, nil
}

func (s *Store) CreateProject(slug, description string) (models.Project, error) {
	slug = normalizeSlug(slug)
	if slug == "" {
		return models.Project{}, errors.New("project slug is required")
	}
	now := time.Now().UTC()
	result, err := s.db.Exec(`INSERT INTO projects(slug, description, created_at) VALUES(?, ?, ?)`, slug, strings.TrimSpace(description), now.Format(time.RFC3339))
	if err != nil {
		return models.Project{}, err
	}
	id, _ := result.LastInsertId()
	return models.Project{ID: id, Slug: slug, Description: strings.TrimSpace(description), CreatedAt: now}, nil
}

func (s *Store) CreateToken(userName string, projectSlugs []string, label string, expiresAt *time.Time) (models.TokenRecord, error) {
	userName = normalizeName(userName)
	if userName == "" {
		return models.TokenRecord{}, errors.New("user is required")
	}
	if len(projectSlugs) == 0 {
		return models.TokenRecord{}, errors.New("at least one project is required")
	}
	projects := uniqueNormalized(projectSlugs)
	tx, err := s.db.Begin()
	if err != nil {
		return models.TokenRecord{}, err
	}
	defer tx.Rollback()

	userID, err := lookupUserID(tx, userName)
	if err != nil {
		return models.TokenRecord{}, err
	}
	projectIDs := make([]int64, 0, len(projects))
	for _, project := range projects {
		projectID, err := lookupProjectID(tx, project)
		if err != nil {
			return models.TokenRecord{}, err
		}
		projectIDs = append(projectIDs, projectID)
	}

	raw, err := turnstileToken.New()
	if err != nil {
		return models.TokenRecord{}, err
	}
	now := time.Now().UTC()
	var expires any
	if expiresAt != nil {
		expires = expiresAt.UTC().Format(time.RFC3339)
	}
	result, err := tx.Exec(`INSERT INTO tokens(user_id, token, label, created_at, expires_at) VALUES(?, ?, ?, ?, ?)`, userID, raw, strings.TrimSpace(label), now.Format(time.RFC3339), expires)
	if err != nil {
		return models.TokenRecord{}, err
	}
	tokenID, _ := result.LastInsertId()
	for _, projectID := range projectIDs {
		if _, err := tx.Exec(`INSERT INTO token_projects(token_id, project_id, created_at) VALUES(?, ?, ?)`, tokenID, projectID, now.Format(time.RFC3339)); err != nil {
			return models.TokenRecord{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.TokenRecord{}, err
	}
	return s.GetToken(tokenID)
}

func (s *Store) ListUsers() ([]models.User, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var user models.User
		var created string
		if err := rows.Scan(&user.ID, &user.Name, &created); err != nil {
			return nil, err
		}
		user.CreatedAt, _ = time.Parse(time.RFC3339, created)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) ListProjects() ([]models.Project, error) {
	rows, err := s.db.Query(`SELECT id, slug, description, created_at FROM projects ORDER BY slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []models.Project
	for rows.Next() {
		var project models.Project
		var created string
		if err := rows.Scan(&project.ID, &project.Slug, &project.Description, &created); err != nil {
			return nil, err
		}
		project.CreatedAt, _ = time.Parse(time.RFC3339, created)
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (s *Store) ListTokens(userName string) ([]models.TokenRecord, error) {
	base := `
SELECT t.id, u.name, t.token, t.label, t.created_at, t.expires_at, t.revoked_at, t.last_used_at,
       COALESCE(GROUP_CONCAT(p.slug, ','), '')
FROM tokens t
JOIN users u ON u.id = t.user_id
LEFT JOIN token_projects tp ON tp.token_id = t.id
LEFT JOIN projects p ON p.id = tp.project_id`
	args := []any{}
	where := ""
	if normalized := normalizeName(userName); normalized != "" {
		where = ` WHERE u.name = ?`
		args = append(args, normalized)
	}
	query := base + where + ` GROUP BY t.id, u.name, t.token, t.label, t.created_at, t.expires_at, t.revoked_at, t.last_used_at ORDER BY t.id`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.TokenRecord
	for rows.Next() {
		item, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetToken(id int64) (models.TokenRecord, error) {
	row := s.db.QueryRow(`
SELECT t.id, u.name, t.token, t.label, t.created_at, t.expires_at, t.revoked_at, t.last_used_at,
       COALESCE(GROUP_CONCAT(p.slug, ','), '')
FROM tokens t
JOIN users u ON u.id = t.user_id
LEFT JOIN token_projects tp ON tp.token_id = t.id
LEFT JOIN projects p ON p.id = tp.project_id
WHERE t.id = ?
GROUP BY t.id, u.name, t.token, t.label, t.created_at, t.expires_at, t.revoked_at, t.last_used_at`, id)
	return scanToken(row)
}

func (s *Store) RevokeToken(id int64) (models.TokenRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`UPDATE tokens SET revoked_at = COALESCE(revoked_at, ?) WHERE id = ?`, now, id)
	if err != nil {
		return models.TokenRecord{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return models.TokenRecord{}, fmt.Errorf("token %d not found", id)
	}
	return s.GetToken(id)
}

func (s *Store) ValidateToken(rawToken, projectSlug, remoteAddr, userAgent string) (models.ValidationResult, error) {
	projectSlug = normalizeSlug(projectSlug)
	presentedPrefix := turnstileToken.Prefix(strings.TrimSpace(rawToken))
	if strings.TrimSpace(rawToken) == "" || projectSlug == "" {
		result := models.ValidationResult{Authorized: false, Reason: "token and project are required", Project: projectSlug}
		return result, s.logAccess(nil, nil, presentedPrefix, false, result.Reason, remoteAddr, userAgent)
	}

	var tokenID int64
	var user string
	var expiresRaw, revokedRaw sql.NullString
	err := s.db.QueryRow(`
SELECT t.id, u.name, t.expires_at, t.revoked_at
FROM tokens t
JOIN users u ON u.id = t.user_id
WHERE t.token = ?`, rawToken).Scan(&tokenID, &user, &expiresRaw, &revokedRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			result := models.ValidationResult{Authorized: false, Reason: "token not found", Project: projectSlug}
			return result, s.logAccess(nil, nil, presentedPrefix, false, result.Reason, remoteAddr, userAgent)
		}
		return models.ValidationResult{}, err
	}

	projectID, err := lookupProjectID(s.db, projectSlug)
	if err != nil {
		result := models.ValidationResult{Authorized: false, Reason: err.Error(), User: user, Project: projectSlug, TokenID: tokenID}
		projectIDCopy := int64(0)
		return result, s.logAccess(&tokenID, &projectIDCopy, presentedPrefix, false, result.Reason, remoteAddr, userAgent)
	}
	if revokedRaw.Valid {
		result := models.ValidationResult{Authorized: false, Reason: "token revoked", User: user, Project: projectSlug, TokenID: tokenID}
		return result, s.logAccess(&tokenID, &projectID, presentedPrefix, false, result.Reason, remoteAddr, userAgent)
	}
	if expiresRaw.Valid {
		expiresAt, err := time.Parse(time.RFC3339, expiresRaw.String)
		if err == nil && time.Now().UTC().After(expiresAt) {
			result := models.ValidationResult{Authorized: false, Reason: "token expired", User: user, Project: projectSlug, TokenID: tokenID, ExpiresAt: &expiresAt}
			return result, s.logAccess(&tokenID, &projectID, presentedPrefix, false, result.Reason, remoteAddr, userAgent)
		}
	}

	var allowed int
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM token_projects WHERE token_id = ? AND project_id = ?)`, tokenID, projectID).Scan(&allowed); err != nil {
		return models.ValidationResult{}, err
	}
	if allowed == 0 {
		result := models.ValidationResult{Authorized: false, Reason: "token does not grant access to project", User: user, Project: projectSlug, TokenID: tokenID}
		return result, s.logAccess(&tokenID, &projectID, presentedPrefix, false, result.Reason, remoteAddr, userAgent)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE tokens SET last_used_at = ? WHERE id = ?`, now, tokenID); err != nil {
		return models.ValidationResult{}, err
	}
	var expiresAt *time.Time
	if expiresRaw.Valid {
		if parsed, err := time.Parse(time.RFC3339, expiresRaw.String); err == nil {
			expiresAt = &parsed
		}
	}
	result := models.ValidationResult{Authorized: true, User: user, Project: projectSlug, TokenID: tokenID, ExpiresAt: expiresAt}
	return result, s.logAccess(&tokenID, &projectID, presentedPrefix, true, "", remoteAddr, userAgent)
}

func (s *Store) logAccess(tokenID, projectID *int64, presentedPrefix string, authorized bool, reason, remoteAddr, userAgent string) error {
	var tokenValue any
	if tokenID != nil && *tokenID != 0 {
		tokenValue = *tokenID
	}
	var projectValue any
	if projectID != nil && *projectID != 0 {
		projectValue = *projectID
	}
	_, err := s.db.Exec(`
INSERT INTO access_log(token_id, project_id, presented_token_prefix, authorized, reason, remote_addr, user_agent, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, tokenValue, projectValue, presentedPrefix, boolToInt(authorized), reason, remoteAddr, userAgent, time.Now().UTC().Format(time.RFC3339))
	return err
}

func lookupUserID(queryer interface{ QueryRow(string, ...any) *sql.Row }, name string) (int64, error) {
	var id int64
	if err := queryer.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("user %q not found", name)
		}
		return 0, err
	}
	return id, nil
}

func lookupProjectID(queryer interface{ QueryRow(string, ...any) *sql.Row }, slug string) (int64, error) {
	var id int64
	if err := queryer.QueryRow(`SELECT id FROM projects WHERE slug = ?`, slug).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("project %q not found", slug)
		}
		return 0, err
	}
	return id, nil
}

type tokenScanner interface {
	Scan(dest ...any) error
}

func scanToken(scanner tokenScanner) (models.TokenRecord, error) {
	var record models.TokenRecord
	var created string
	var expiresRaw, revokedRaw, lastUsedRaw sql.NullString
	var projectsCSV string
	if err := scanner.Scan(&record.ID, &record.User, &record.Token, &record.Label, &created, &expiresRaw, &revokedRaw, &lastUsedRaw, &projectsCSV); err != nil {
		return models.TokenRecord{}, err
	}
	record.CreatedAt, _ = time.Parse(time.RFC3339, created)
	record.Projects = splitCSV(projectsCSV)
	if expiresRaw.Valid {
		if parsed, err := time.Parse(time.RFC3339, expiresRaw.String); err == nil {
			record.ExpiresAt = &parsed
		}
	}
	if revokedRaw.Valid {
		if parsed, err := time.Parse(time.RFC3339, revokedRaw.String); err == nil {
			record.RevokedAt = &parsed
		}
	}
	if lastUsedRaw.Valid {
		if parsed, err := time.Parse(time.RFC3339, lastUsedRaw.String); err == nil {
			record.LastUsedAt = &parsed
		}
	}
	return record, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func uniqueNormalized(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		normalized := normalizeSlug(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeName(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func normalizeSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
