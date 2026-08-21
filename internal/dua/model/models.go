package model

type Dua struct {
	ID            int    `json:"id" db:"id"`
	CategoryID    int    `json:"category_id" db:"category_id"`
	Title         string `json:"title,omitempty" db:"title"`
	ArabicText    string `json:"arabic_text" db:"arabic_text"`
	Transliteration string `json:"transliteration,omitempty" db:"transliteration"`
	Translation   string `json:"translation,omitempty" db:"translation"`
	SourceID      *int   `json:"source_id,omitempty" db:"source_id"`
	AudioURL      string `json:"audio_url,omitempty" db:"audio_url"`
	SortOrder     int    `json:"sort_order" db:"sort_order"`
	IsActive      bool   `json:"is_active" db:"is_active"`
	CreatedAt     string `json:"created_at" db:"created_at"`
	UpdatedAt     string `json:"updated_at" db:"updated_at"`
}

type DuaWithSource struct {
	Dua     `db:"dua"`
	Sources []DuaSource `json:"sources,omitempty"`
}

type DuaSource struct {
	ID               int    `json:"id" db:"id"`
	Collection       string `json:"collection,omitempty" db:"collection"`
	Reference        string `json:"reference,omitempty" db:"reference"`
	SourceText       string `json:"source_text,omitempty" db:"source_text"`
	VerificationStatus string `json:"verification_status" db:"verification_status"`
}
