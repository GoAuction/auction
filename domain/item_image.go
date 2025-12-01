package domain

import "time"

type ItemImage struct {
	ID        string    `json:"id" db:"id"`
	ItemID    string    `json:"item_id" db:"item_id"`
	ImageURL  string    `json:"url" db:"url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
