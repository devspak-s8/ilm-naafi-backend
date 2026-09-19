package provider

import "context"

type QuranProvider interface {
	ListChapters(ctx context.Context, language string) ([]Chapter, error)
	GetChapter(ctx context.Context, chapterID int, language string) (*Chapter, error)
	GetVerses(ctx context.Context, chapterID int, query VerseQuery) (VersePage, error)
	GetVerse(ctx context.Context, verseKey string, query VerseQuery) (*Verse, error)
	GetJuz(ctx context.Context, juzID int, mushafID int) (*Juz, error)
	GetPage(ctx context.Context, pageID int, query VerseQuery) (VersePage, error)
	Search(ctx context.Context, query SearchQuery) (SearchResponse, error)
	ListResources(ctx context.Context, resourceType string) ([]Resource, error)
}

type VerseQuery struct {
	Language     string
	Words        bool
	Translations string
	Tafsirs      string
	Audio        int
	Page         int
	PerPage      int
}

type SearchQuery struct {
	Query          string
	Page           int
	Size           int
	TranslationIDs string
}

type Chapter struct {
	ID              int            `json:"id"`
	RevelationPlace string         `json:"revelation_place"`
	RevelationOrder int            `json:"revelation_order"`
	BismillahPre    bool           `json:"bismillah_pre"`
	NameComplex     string         `json:"name_complex"`
	NameArabic      string         `json:"name_arabic"`
	NameSimple      string         `json:"name_simple"`
	VersesCount     int            `json:"verses_count"`
	Pages           []int          `json:"pages,omitempty"`
	TranslatedName  TranslatedName `json:"translated_name"`
	Provider        string         `json:"provider"`
}

type TranslatedName struct {
	LanguageName string `json:"language_name"`
	Name         string `json:"name"`
}

type Verse struct {
	ID              int           `json:"id"`
	ChapterID       int           `json:"chapter_id"`
	VerseNumber     int           `json:"verse_number"`
	VerseKey        string        `json:"verse_key"`
	TextUthmani     string        `json:"text_uthmani,omitempty"`
	TextImlaei      string        `json:"text_imlaei,omitempty"`
	PageNumber      int           `json:"page_number"`
	JuzNumber       int           `json:"juz_number"`
	HizbNumber      int           `json:"hizb_number"`
	RubElHizbNumber int           `json:"rub_el_hizb_number"`
	RukuNumber      int           `json:"ruku_number,omitempty"`
	ManzilNumber    int           `json:"manzil_number,omitempty"`
	SajdahType      string        `json:"sajdah_type,omitempty"`
	Words           []Word        `json:"words,omitempty"`
	Translations    []Translation `json:"translations,omitempty"`
	Tafsirs         []Tafsir      `json:"tafsirs,omitempty"`
	Audio           *Audio        `json:"audio,omitempty"`
	Provider        string        `json:"provider"`
}

type Word struct {
	ID              int    `json:"id"`
	Position        int    `json:"position"`
	TextUthmani     string `json:"text_uthmani,omitempty"`
	PageNumber      int    `json:"page_number,omitempty"`
	LineNumber      int    `json:"line_number,omitempty"`
	AudioURL        string `json:"audio_url,omitempty"`
	Translation     string `json:"translation,omitempty"`
	Transliteration string `json:"transliteration,omitempty"`
}

type Translation struct {
	ResourceID   int    `json:"resource_id"`
	ResourceName string `json:"resource_name,omitempty"`
	LanguageName string `json:"language_name,omitempty"`
	Text         string `json:"text"`
	Provider     string `json:"provider"`
}

type Tafsir struct {
	ID           int    `json:"id"`
	ResourceID   int    `json:"resource_id"`
	Name         string `json:"name,omitempty"`
	LanguageName string `json:"language_name,omitempty"`
	Text         string `json:"text"`
	Provider     string `json:"provider"`
}

type Audio struct {
	VerseKey string `json:"verse_key"`
	URL      string `json:"url"`
}

type Pagination struct {
	PerPage      int  `json:"per_page"`
	CurrentPage  int  `json:"current_page"`
	NextPage     *int `json:"next_page"`
	TotalPages   int  `json:"total_pages"`
	TotalRecords int  `json:"total_records"`
}

type VersePage struct {
	Verses     []Verse    `json:"verses"`
	Pagination Pagination `json:"pagination"`
}

type Juz struct {
	ID           int               `json:"id"`
	JuzNumber    int               `json:"juz_number"`
	VerseMapping map[string]string `json:"verse_mapping"`
	FirstVerseID int               `json:"first_verse_id"`
	LastVerseID  int               `json:"last_verse_id"`
	VersesCount  int               `json:"verses_count"`
	Provider     string            `json:"provider"`
}

type Resource struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	AuthorName   string `json:"author_name,omitempty"`
	LanguageName string `json:"language_name,omitempty"`
	LanguageID   int    `json:"language_id,omitempty"`
	ResourceType string `json:"resource_type"`
	Provider     string `json:"provider"`
}

type SearchResult struct {
	ResultType string      `json:"result_type"`
	Key        interface{} `json:"key"`
	Name       string      `json:"name"`
	Arabic     string      `json:"arabic,omitempty"`
}

type SearchResponse struct {
	Navigation []SearchResult `json:"navigation"`
	Verses     []SearchResult `json:"verses"`
	Pagination Pagination     `json:"pagination"`
}
