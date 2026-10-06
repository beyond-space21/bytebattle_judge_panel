-- Official solution used to validate custom-input runs (never exposed to students).
ALTER TABLE problems
    ADD COLUMN IF NOT EXISTS reference_code TEXT NOT NULL DEFAULT '';
