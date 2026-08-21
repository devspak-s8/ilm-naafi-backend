CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================
-- QUR'AN TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS quran_surahs (
    id SERIAL PRIMARY KEY,
    number INTEGER UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    arabic_name VARCHAR(255) NOT NULL,
    english_name VARCHAR(255) NOT NULL,
    revelation_type VARCHAR(50) NOT NULL,
    ayah_count INTEGER NOT NULL,
    page_number INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quran_ayahs (
    id SERIAL PRIMARY KEY,
    surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    ayah_number INTEGER NOT NULL,
    text TEXT NOT NULL,
    page_number INTEGER,
    juz_number INTEGER,
    hizb_number INTEGER,
    rub_el_hizb INTEGER,
    sajdah BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(surah_id, ayah_number)
);

CREATE TABLE IF NOT EXISTS quran_reciters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    arabic_name VARCHAR(255) NOT NULL,
    identifier VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    audio_base_url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quran_audio (
    id SERIAL PRIMARY KEY,
    reciter_id INTEGER NOT NULL REFERENCES quran_reciters(id) ON DELETE CASCADE,
    surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    audio_url TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    start_ms INTEGER NOT NULL DEFAULT 0,
    end_ms INTEGER NOT NULL,
    format VARCHAR(20) DEFAULT 'mp3',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(reciter_id, ayah_id)
);

-- =============================================
-- USER QUR'AN DATA
-- =============================================

CREATE TABLE IF NOT EXISTS quran_bookmarks (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, ayah_id)
);

CREATE TABLE IF NOT EXISTS quran_reading_progress (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    last_ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    last_page_number INTEGER,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quran_reading_history (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- =============================================
-- ADHKAR & DU'A TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS dhikr_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    arabic_name VARCHAR(255),
    description TEXT,
    icon VARCHAR(100),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS adhkar (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES dhikr_categories(id) ON DELETE CASCADE,
    title VARCHAR(255),
    arabic_text TEXT NOT NULL,
    transliteration TEXT,
    translation TEXT,
    repeat_count INTEGER DEFAULT 1,
    sort_order INTEGER DEFAULT 0,
    audio_url TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS adhkar_sources (
    id SERIAL PRIMARY KEY,
    dhikr_id INTEGER NOT NULL REFERENCES adhkar(id) ON DELETE CASCADE,
    source_type VARCHAR(100) NOT NULL,
    collection VARCHAR(255),
    reference VARCHAR(255),
    source_text TEXT,
    verification_status VARCHAR(50) DEFAULT 'verified',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS duas (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES dhikr_categories(id) ON DELETE CASCADE,
    title VARCHAR(255),
    arabic_text TEXT NOT NULL,
    transliteration TEXT,
    translation TEXT,
    source_id INTEGER REFERENCES adhkar_sources(id) ON DELETE SET NULL,
    audio_url TEXT,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================
-- USER ADHKAR DATA
-- =============================================

CREATE TABLE IF NOT EXISTS adhkar_completion (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dhikr_id INTEGER NOT NULL REFERENCES adhkar(id) ON DELETE CASCADE,
    completion_date DATE NOT NULL DEFAULT CURRENT_DATE,
    completed_count INTEGER DEFAULT 0,
    required_count INTEGER NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, dhikr_id, completion_date)
);

CREATE TABLE IF NOT EXISTS adhkar_reminders (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES dhikr_categories(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT TRUE,
    time TIME,
    notification_type VARCHAR(50) DEFAULT 'normal',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category_id)
);

-- =============================================
-- INDEXES
-- =============================================

CREATE INDEX IF NOT EXISTS idx_quran_ayahs_surah_id ON quran_ayahs(surah_id);
CREATE INDEX IF NOT EXISTS idx_quran_ayahs_page_number ON quran_ayahs(page_number);
CREATE INDEX IF NOT EXISTS idx_quran_ayahs_juz_number ON quran_ayahs(juz_number);
CREATE INDEX IF NOT EXISTS idx_quran_ayahs_text ON quran_ayahs(text);

CREATE INDEX IF NOT EXISTS idx_quran_audio_reciter_id ON quran_audio(reciter_id);
CREATE INDEX IF NOT EXISTS idx_quran_audio_surah_id ON quran_audio(surah_id);
CREATE INDEX IF NOT EXISTS idx_quran_audio_ayah_id ON quran_audio(ayah_id);

CREATE INDEX IF NOT EXISTS idx_quran_bookmarks_user_id ON quran_bookmarks(user_id);
CREATE INDEX IF NOT EXISTS idx_quran_bookmarks_ayah_id ON quran_bookmarks(ayah_id);

CREATE INDEX IF NOT EXISTS idx_quran_reading_progress_user_id ON quran_reading_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_quran_reading_history_user_id ON quran_reading_history(user_id);

CREATE INDEX IF NOT EXISTS idx_adhkar_category_id ON adhkar(category_id);
CREATE INDEX IF NOT EXISTS idx_adhkar_sources_dhikr_id ON adhkar_sources(dhikr_id);

CREATE INDEX IF NOT EXISTS idx_adhkar_completion_user_id ON adhkar_completion(user_id);
CREATE INDEX IF NOT EXISTS idx_adhkar_completion_date ON adhkar_completion(completion_date);
CREATE INDEX IF NOT EXISTS idx_adhkar_reminders_user_id ON adhkar_reminders(user_id);
