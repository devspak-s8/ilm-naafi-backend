package model

type DhikrCategory struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	ArabicName  string `json:"arabic_name" db:"arabic_name"`
	Description string `json:"description,omitempty" db:"description"`
	Icon        string `json:"icon,omitempty" db:"icon"`
	SortOrder   int    `json:"sort_order" db:"sort_order"`
	IsActive    bool   `json:"is_active" db:"is_active"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

type Dhikr struct {
	ID            int    `json:"id" db:"id"`
	CategoryID    int    `json:"category_id" db:"category_id"`
	Title         string `json:"title,omitempty" db:"title"`
	ArabicText    string `json:"arabic_text" db:"arabic_text"`
	Transliteration string `json:"transliteration,omitempty" db:"transliteration"`
	Translation   string `json:"translation,omitempty" db:"translation"`
	RepeatCount   int    `json:"repeat_count" db:"repeat_count"`
	SortOrder     int    `json:"sort_order" db:"sort_order"`
	AudioURL      string `json:"audio_url,omitempty" db:"audio_url"`
	IsActive      bool   `json:"is_active" db:"is_active"`
	CreatedAt     string `json:"created_at" db:"created_at"`
	UpdatedAt     string `json:"updated_at" db:"updated_at"`
}

type DhikrSource struct {
	ID               int    `json:"id" db:"id"`
	DhikrID          int    `json:"dhikr_id" db:"dhikr_id"`
	SourceType       string `json:"source_type" db:"source_type"`
	Collection       string `json:"collection,omitempty" db:"collection"`
	Reference        string `json:"reference,omitempty" db:"reference"`
	SourceText       string `json:"source_text,omitempty" db:"source_text"`
	VerificationStatus string `json:"verification_status" db:"verification_status"`
	CreatedAt        string `json:"created_at" db:"created_at"`
}

type DhikrWithSource struct {
	Dhikr  `db:"dhikr"`
	Sources []DhikrSource `json:"sources,omitempty"`
}

type AdhkarCompletion struct {
	ID             int    `json:"id" db:"id"`
	UserID         string `json:"user_id" db:"user_id"`
	DhikrID        int    `json:"dhikr_id" db:"dhikr_id"`
	CompletionDate string `json:"completion_date" db:"completion_date"`
	CompletedCount int    `json:"completed_count" db:"completed_count"`
	RequiredCount  int    `json:"required_count" db:"required_count"`
	CompletedAt    *string `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt      string `json:"created_at" db:"created_at"`
}

type AdhkarReminder struct {
	ID               int    `json:"id" db:"id"`
	UserID           string `json:"user_id" db:"user_id"`
	CategoryID       int    `json:"category_id" db:"category_id"`
	Enabled          bool   `json:"enabled" db:"enabled"`
	Time             string `json:"time,omitempty" db:"time"`
	NotificationType string `json:"notification_type" db:"notification_type"`
	CreatedAt        string `json:"created_at" db:"created_at"`
	UpdatedAt        string `json:"updated_at" db:"updated_at"`
}

type DailyProgress struct {
	Date              string `json:"date"`
	MorningCompleted  bool   `json:"morning_completed"`
	EveningCompleted  bool   `json:"evening_completed"`
	PostSalahCompleted bool  `json:"post_salah_completed"`
	OverallPercent    int    `json:"overall_percent"`
}
