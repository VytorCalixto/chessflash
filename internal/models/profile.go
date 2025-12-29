package models

import "time"

type Profile struct {
	ID         int64      `json:"id"`
	Username   string     `json:"username"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSyncAt *time.Time `json:"last_sync_at"`
}
