package domain

import "time"

type ItemCategory struct {
	ItemID     string    `json:"item_id" db:"item_id"`
	CategoryID string    `json:"category_id" db:"category_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
