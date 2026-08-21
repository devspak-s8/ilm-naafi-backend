package model

type Surah struct {
	ID            int    `json:"id" db:"id"`
	Number        int    `json:"number" db:"number"`
	Name          string `json:"name" db:"name"`
	ArabicName    string `json:"arabic_name" db:"arabic_name"`
	EnglishName   string `json:"english_name" db:"english_name"`
	RevelationType string `json:"revelation_type" db:"revelation_type"`
	AyahCount     int    `json:"ayah_count" db:"ayah_count"`
	PageNumber    *int   `json:"page_number,omitempty" db:"page_number"`
	CreatedAt     string `json:"created_at" db:"created_at"`
	UpdatedAt     string `json:"updated_at" db:"updated_at"`
}

type Ayah struct {
	ID          int    `json:"id" db:"id"`
	SurahID     int    `json:"surah_id" db:"surah_id"`
	AyahNumber  int    `json:"ayah_number" db:"ayah_number"`
	Text        string `json:"text" db:"text"`
	PageNumber  *int   `json:"page_number,omitempty" db:"page_number"`
	JuzNumber   *int   `json:"juz_number,omitempty" db:"juz_number"`
	HizbNumber  *int   `json:"hizb_number,omitempty" db:"hizb_number"`
	RubElHizb   *int   `json:"rub_el_hizb,omitempty" db:"rub_el_hizb"`
	Sajdah      bool   `json:"sajdah" db:"sajdah"`
	CreatedAt   string `json:"created_at" db:"created_at"`
}

type Reciter struct {
	ID           int    `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	ArabicName   string `json:"arabic_name" db:"arabic_name"`
	Identifier   string `json:"identifier" db:"identifier"`
	Description  string `json:"description,omitempty" db:"description"`
	AudioBaseURL string `json:"audio_base_url" db:"audio_base_url"`
	IsActive     bool   `json:"is_active" db:"is_active"`
	CreatedAt    string `json:"created_at" db:"created_at"`
	UpdatedAt    string `json:"updated_at" db:"updated_at"`
}

type AudioMetadata struct {
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

type Bookmark struct {
	ID        int    `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	AyahID    int    `json:"ayah_id" db:"ayah_id"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

type ReadingProgress struct {
	ID            int    `json:"id" db:"id"`
	UserID        string `json:"user_id" db:"user_id"`
	LastSurahID   int    `json:"last_surah_id" db:"last_surah_id"`
	LastAyahID    int    `json:"last_ayah_id" db:"last_ayah_id"`
	LastPageNumber *int  `json:"last_page_number,omitempty" db:"last_page_number"`
	UpdatedAt     string `json:"updated_at" db:"updated_at"`
}

type ReadingHistory struct {
	ID          int    `json:"id" db:"id"`
	UserID      string `json:"user_id" db:"user_id"`
	SurahID     int    `json:"surah_id" db:"surah_id"`
	AyahID      int    `json:"ayah_id" db:"ayah_id"`
	StartedAt   string `json:"started_at" db:"started_at"`
	CompletedAt *string `json:"completed_at,omitempty" db:"completed_at"`
}

type Juz struct {
	ID          int    `json:"id" db:"id"`
	Number      int    `json:"number" db:"number"`
	SurahID     int    `json:"surah_id" db:"surah_id"`
	AyahNumber  int    `json:"ayah_number" db:"ayah_number"`
}

type SearchResult struct {
	SurahID      int    `json:"surah_id"`
	SurahName    string `json:"surah_name"`
	AyahID       int    `json:"ayah_id"`
	AyahNumber   int    `json:"ayah_number"`
	Text         string `json:"text"`
}
