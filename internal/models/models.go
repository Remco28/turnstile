package models

import "time"

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type TokenRecord struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	User       string     `json:"user"`
	Token      string     `json:"token,omitempty"`
	Label      string     `json:"label,omitempty"`
	Projects   []string   `json:"projects,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type ValidationResult struct {
	Authorized bool       `json:"authorized"`
	Reason     string     `json:"reason,omitempty"`
	User       string     `json:"user,omitempty"`
	Project    string     `json:"project,omitempty"`
	TokenID    int64      `json:"token_id,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type ProjectAccess struct {
	Project    string     `json:"project"`
	TokenID    int64      `json:"token_id"`
	User       string     `json:"user"`
	Label      string     `json:"label,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type AccessLogEntry struct {
	ID                   int64     `json:"id"`
	TokenID              int64     `json:"token_id,omitempty"`
	User                 string    `json:"user,omitempty"`
	Project              string    `json:"project,omitempty"`
	PresentedTokenPrefix string    `json:"presented_token_prefix"`
	Authorized           bool      `json:"authorized"`
	Reason               string    `json:"reason,omitempty"`
	RemoteAddr           string    `json:"remote_addr,omitempty"`
	UserAgent            string    `json:"user_agent,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

type TokenReissueResult struct {
	Old TokenRecord `json:"old"`
	New TokenRecord `json:"new"`
}
