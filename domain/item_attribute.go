package domain

import "time"

type ItemAttribute struct {
	ID        string    `json:"id" db:"id"`
	ItemID    string    `json:"item_id" db:"item_id"`
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
