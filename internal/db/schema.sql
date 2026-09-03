CREATE TABLE IF NOT EXISTS people (
    id INTEGER PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL DEFAULT '',
    nickname TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    nudge_enabled INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS contact_info (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    value TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_contact_info_person ON contact_info(person_id);

CREATE TABLE IF NOT EXISTS circles (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS person_circles (
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    circle_id INTEGER NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    note TEXT,
    PRIMARY KEY (person_id, circle_id)
);
CREATE INDEX IF NOT EXISTS idx_person_circles_circle ON person_circles(circle_id);

CREATE TABLE IF NOT EXISTS relationship_types (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    name_reverse TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS relationships (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    related_person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    relationship_type_id INTEGER NOT NULL REFERENCES relationship_types(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_relationships_person ON relationships(person_id);
CREATE INDEX IF NOT EXISTS idx_relationships_related ON relationships(related_person_id);

CREATE TABLE IF NOT EXISTS important_dates (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'custom',
    label TEXT NOT NULL,
    month INTEGER NOT NULL,
    day INTEGER NOT NULL,
    year INTEGER
);
CREATE INDEX IF NOT EXISTS idx_important_dates_person ON important_dates(person_id);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY,
    kind TEXT NOT NULL DEFAULT 'visit',
    status TEXT NOT NULL DEFAULT 'idea',
    title TEXT NOT NULL DEFAULT '',
    start_date TEXT,
    end_date TEXT,
    timeframe_note TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS event_people (
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, person_id)
);
CREATE INDEX IF NOT EXISTS idx_event_people_person ON event_people(person_id);

CREATE TABLE IF NOT EXISTS reminders (
    id INTEGER PRIMARY KEY,
    person_id INTEGER REFERENCES people(id) ON DELETE CASCADE,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    due_date TEXT NOT NULL,
    recurrence_interval INTEGER,
    recurrence_unit TEXT,
    note TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending'
);
CREATE INDEX IF NOT EXISTS idx_reminders_person ON reminders(person_id);
CREATE INDEX IF NOT EXISTS idx_reminders_due ON reminders(due_date);

CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_notes_person ON notes(person_id);

-- Deliberately a separate table from notes rather than a generalized
-- person_id/event_id-nullable notes table (contrast with reminders, which
-- supports both from day one) — notes.person_id is NOT NULL and has real
-- production data, so relaxing that would need a full SQLite table-rebuild
-- migration for what started as a "keep it simple" ask. See PLANNING.md.
CREATE TABLE IF NOT EXISTS event_notes (
    id INTEGER PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_event_notes_event ON event_notes(event_id);

CREATE TABLE IF NOT EXISTS pets (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    species TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_pets_person ON pets(person_id);

CREATE TABLE IF NOT EXISTS facts (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    value TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_facts_person ON facts(person_id);

-- Managed suggestion lists for otherwise-free-text fields (Event kind/status,
-- ContactInfo/ImportantDate type). Deliberately not a foreign key from those
-- tables — deleting an option here must not affect rows that already used
-- it; see internal/model/optionvalue.go.
CREATE TABLE IF NOT EXISTS option_values (
    id INTEGER PRIMARY KEY,
    category TEXT NOT NULL,
    value TEXT NOT NULL,
    UNIQUE (category, value COLLATE NOCASE)
);
CREATE INDEX IF NOT EXISTS idx_option_values_category ON option_values(category);

-- Tracks one-time data migrations (as opposed to schema changes, which the
-- CREATE TABLE IF NOT EXISTS statements above handle) so each runs exactly
-- once ever, even across restarts — see internal/db/seed.go's
-- applyOneTimeMigrations. Unlike seedOptionValues/renameLegacyOptionValues,
-- some migrations aren't safely re-runnable by content alone (e.g. "disable
-- a flag for everyone" can't tell "never migrated" apart from "user turned
-- it back on since"), so they need an explicit applied-once marker instead.
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
