package domain

import "time"

type ItemComment struct {
	ID        string    `json:"id" db:"id"`
	ItemID    string    `json:"item_id" db:"item_id"`
	Content   string    `json:"content" db:"content"`
	UserID    string    `json:"user_id" db:"user_id"`
	ParentID  *string   `json:"parent_id" db:"parent_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
