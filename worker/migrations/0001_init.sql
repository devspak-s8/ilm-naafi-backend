PRAGMA foreign_keys = ON;

-- =============================================
-- AUTH TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    email_verified INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    last_login_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS user_profiles (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    bio TEXT,
    avatar_url TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name TEXT NOT NULL,
    device_type TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    refresh_token_hash TEXT,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    revoked_at TEXT
);

CREATE TABLE IF NOT EXISTS email_verifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS password_resets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL
);

-- =============================================
-- QUR'AN TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS quran_surahs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    number INTEGER UNIQUE NOT NULL,
    name TEXT NOT NULL,
    arabic_name TEXT NOT NULL,
    english_name TEXT NOT NULL,
    revelation_type TEXT NOT NULL,
    ayah_count INTEGER NOT NULL,
    page_number INTEGER,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quran_ayahs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    ayah_number INTEGER NOT NULL,
    text TEXT NOT NULL,
    page_number INTEGER,
    juz_number INTEGER,
    hizb_number INTEGER,
    rub_el_hizb INTEGER,
    sajdah INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE(surah_id, ayah_number)
);

CREATE TABLE IF NOT EXISTS quran_reciters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    arabic_name TEXT NOT NULL,
    identifier TEXT UNIQUE NOT NULL,
    description TEXT,
    audio_base_url TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quran_audio (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reciter_id INTEGER NOT NULL REFERENCES quran_reciters(id) ON DELETE CASCADE,
    surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    audio_url TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    start_ms INTEGER NOT NULL DEFAULT 0,
    end_ms INTEGER NOT NULL,
    format TEXT DEFAULT 'mp3',
    created_at TEXT NOT NULL,
    UNIQUE(reciter_id, ayah_id)
);

-- =============================================
-- USER QUR'AN DATA
-- =============================================

CREATE TABLE IF NOT EXISTS quran_bookmarks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    UNIQUE(user_id, ayah_id)
);

CREATE TABLE IF NOT EXISTS quran_reading_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    last_surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    last_ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    last_page_number INTEGER,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quran_reading_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    surah_id INTEGER NOT NULL REFERENCES quran_surahs(id) ON DELETE CASCADE,
    ayah_id INTEGER NOT NULL REFERENCES quran_ayahs(id) ON DELETE CASCADE,
    started_at TEXT NOT NULL,
    completed_at TEXT
);

-- =============================================
-- ADHKAR & DU'A TABLES
-- =============================================

CREATE TABLE IF NOT EXISTS dhikr_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    arabic_name TEXT,
    description TEXT,
    icon TEXT,
    sort_order INTEGER DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS adhkar (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES dhikr_categories(id) ON DELETE CASCADE,
    title TEXT,
    arabic_text TEXT NOT NULL,
    transliteration TEXT,
    translation TEXT,
    repeat_count INTEGER DEFAULT 1,
    sort_order INTEGER DEFAULT 0,
    audio_url TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS adhkar_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dhikr_id INTEGER NOT NULL REFERENCES adhkar(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL,
    collection TEXT,
    reference TEXT,
    source_text TEXT,
    verification_status TEXT DEFAULT 'verified',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS duas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES dhikr_categories(id) ON DELETE CASCADE,
    title TEXT,
    arabic_text TEXT NOT NULL,
    transliteration TEXT,
    translation TEXT,
    source_id INTEGER REFERENCES adhkar_sources(id) ON DELETE SET NULL,
    audio_url TEXT,
    sort_order INTEGER DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- =============================================
-- USER ADHKAR DATA
-- =============================================

CREATE TABLE IF NOT EXISTS adhkar_completion (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dhikr_id INTEGER NOT NULL REFERENCES adhkar(id) ON DELETE CASCADE,
    completion_date TEXT NOT NULL,
    completed_count INTEGER DEFAULT 0,
    required_count INTEGER NOT NULL,
    completed_at TEXT,
    created_at TEXT NOT NULL,
    UNIQUE(user_id, dhikr_id, completion_date)
);

CREATE TABLE IF NOT EXISTS adhkar_reminders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES dhikr_categories(id) ON DELETE CASCADE,
    enabled INTEGER NOT NULL DEFAULT 1,
    time TEXT,
    notification_type TEXT DEFAULT 'normal',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, category_id)
);

-- =============================================
-- INDEXES
-- =============================================

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_email_verifications_user_id ON email_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_email_verifications_token_hash ON email_verifications(token_hash);
CREATE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets(user_id);
CREATE INDEX IF NOT EXISTS idx_password_resets_token_hash ON password_resets(token_hash);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at);

CREATE INDEX IF NOT EXISTS idx_quran_ayahs_surah_id ON quran_ayahs(surah_id);
CREATE INDEX IF NOT EXISTS idx_quran_ayahs_page_number ON quran_ayahs(page_number);
CREATE INDEX IF NOT EXISTS idx_quran_ayahs_juz_number ON quran_ayahs(juz_number);

CREATE INDEX IF NOT EXISTS idx_quran_audio_reciter_id ON quran_audio(reciter_id);
CREATE INDEX IF NOT EXISTS idx_quran_audio_surah_id ON quran_audio(surah_id);
CREATE INDEX IF NOT EXISTS idx_quran_audio_ayah_id ON quran_audio(ayah_id);

CREATE INDEX IF NOT EXISTS idx_quran_bookmarks_user_id ON quran_bookmarks(user_id);
CREATE INDEX IF NOT EXISTS idx_quran_reading_progress_user_id ON quran_reading_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_quran_reading_history_user_id ON quran_reading_history(user_id);

CREATE INDEX IF NOT EXISTS idx_adhkar_category_id ON adhkar(category_id);
CREATE INDEX IF NOT EXISTS idx_adhkar_sources_dhikr_id ON adhkar_sources(dhikr_id);
CREATE INDEX IF NOT EXISTS idx_adhkar_completion_user_id ON adhkar_completion(user_id);
CREATE INDEX IF NOT EXISTS idx_adhkar_completion_date ON adhkar_completion(completion_date);
CREATE INDEX IF NOT EXISTS idx_adhkar_reminders_user_id ON adhkar_reminders(user_id);