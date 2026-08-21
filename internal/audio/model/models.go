package model

type ReciterPreference struct {
	UserID      string `json:"user_id" db:"user_id"`
	ReciterID   int    `json:"reciter_id" db:"reciter_id"`
	AutoPlay    bool   `json:"auto_play" db:"auto_play"`
	Translation bool   `json:"translation" db:"translation"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

type AudioTrack struct {
	ID         int    `json:"id" db:"id"`
	ReciterID  int    `json:"reciter_id" db:"reciter_id"`
	SurahID    int    `json:"surah_id" db:"surah_id"`
	AyahID     int    `json:"ayah_id" db:"ayah_id"`
	AudioURL   string `json:"audio_url" db:"audio_url"`
	DurationMs int    `json:"duration_ms" db:"duration_ms"`
	StartMs    int    `json:"start_ms" db:"start_ms"`
	EndMs      int    `json:"end_ms" db:"end_ms"`
	Format     string `json:"format" db:"format"`
	CreatedAt  string `json:"created_at" db:"created_at"`
}
