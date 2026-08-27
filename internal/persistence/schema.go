package persistence

const schema = `
CREATE TABLE IF NOT EXISTS batches (
 batch_id TEXT PRIMARY KEY, title TEXT NOT NULL, edition TEXT NOT NULL, owner TEXT NOT NULL,
 status TEXT NOT NULL, version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS pages (
 evidence_id TEXT PRIMARY KEY, page_id TEXT NOT NULL, batch_id TEXT NOT NULL, sequence_no INTEGER NOT NULL,
 image_digest TEXT NOT NULL, ocr_text TEXT NOT NULL, character_count INTEGER NOT NULL, confidence REAL NOT NULL,
 observed_at TEXT NOT NULL, revision INTEGER NOT NULL,
 FOREIGN KEY(batch_id) REFERENCES batches(batch_id), UNIQUE(batch_id, page_id, revision)
);
CREATE INDEX IF NOT EXISTS pages_batch_sequence ON pages(batch_id, sequence_no, revision);
CREATE TABLE IF NOT EXISTS issues (
 issue_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, page_id TEXT NOT NULL, category TEXT NOT NULL,
 severity TEXT NOT NULL, description TEXT NOT NULL, disposition TEXT NOT NULL,
 corrected_evidence_id TEXT NOT NULL DEFAULT '', resolved_at TEXT,
 FOREIGN KEY(batch_id) REFERENCES batches(batch_id)
);
CREATE TABLE IF NOT EXISTS quality_checks (
 check_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, passed INTEGER NOT NULL, coverage REAL NOT NULL,
 average_confidence REAL NOT NULL, checked_at TEXT NOT NULL, payload TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reviews (
 review_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, approved INTEGER NOT NULL,
 reviewer TEXT NOT NULL, comment TEXT NOT NULL, reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS snapshots (
 batch_id TEXT PRIMARY KEY, digest TEXT NOT NULL, frozen_at TEXT NOT NULL, payload TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS credentials (
 credential_id TEXT PRIMARY KEY, batch_id TEXT NOT NULL UNIQUE, snapshot_digest TEXT NOT NULL,
 issued_to TEXT NOT NULL, issued_at TEXT NOT NULL, signature TEXT NOT NULL, revoked_at TEXT
);
CREATE TABLE IF NOT EXISTS audit_events (
 sequence_no INTEGER PRIMARY KEY AUTOINCREMENT, event_id TEXT NOT NULL UNIQUE, batch_id TEXT NOT NULL,
 event_type TEXT NOT NULL, payload TEXT NOT NULL, previous_hash TEXT NOT NULL, event_hash TEXT NOT NULL,
 occurred_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS idempotency (
 operation TEXT NOT NULL, idempotency_key TEXT NOT NULL, request_hash TEXT NOT NULL,
 response TEXT NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(operation, idempotency_key)
);`
