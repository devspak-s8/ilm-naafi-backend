PRAGMA foreign_keys = ON;

-- Track when a reminder last fired so the cron doesn't notify twice.
ALTER TABLE adhkar_reminders ADD COLUMN last_reminded_at TEXT;