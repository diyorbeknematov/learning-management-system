package models

import "time"

type RefreshToken struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Token     string    `db:"token" json:"-"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
}


