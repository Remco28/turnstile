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
