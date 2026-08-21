package model

type Notification struct {
	ID        int    `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	Type      string `json:"type" db:"type"`
	Title     string `json:"title" db:"title"`
	Body      string `json:"body" db:"body"`
	Data      map[string]interface{} `json:"data,omitempty" db:"data"`
	ReadAt    *string `json:"read_at,omitempty" db:"read_at"`
	CreatedAt string `json:"created_at" db:"created_at"`
}
