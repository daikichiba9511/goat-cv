-- Why: 旧escalated statusには復帰先のlifecycleがなく、安全な自動変換を定義できない。
CREATE TEMP TABLE image_workflow_migration_guard (
    image_id TEXT NOT NULL
);

CREATE TEMP TRIGGER reject_legacy_escalated_image
BEFORE INSERT ON image_workflow_migration_guard
BEGIN
    SELECT RAISE(ABORT, 'cannot migrate legacy escalated Image: ' || NEW.image_id);
END;

INSERT INTO image_workflow_migration_guard (image_id)
SELECT id FROM images WHERE status = 'escalated';

DROP TRIGGER reject_legacy_escalated_image;
DROP TABLE image_workflow_migration_guard;

ALTER TABLE images
ADD COLUMN escalated BOOLEAN NOT NULL DEFAULT 0 CHECK (escalated IN (0, 1));

-- Why: SQLiteでは既存CHECKを変更できないため、旧statusの禁止と直交状態の制約をtriggerで補う。
CREATE TRIGGER images_validate_workflow_on_insert
BEFORE INSERT ON images
WHEN NEW.status = 'escalated' OR (NEW.status = 'approved' AND NEW.escalated = 1)
BEGIN
    SELECT RAISE(ABORT, 'invalid image workflow state');
END;

CREATE TRIGGER images_validate_workflow_on_update
BEFORE UPDATE OF status, escalated ON images
WHEN NEW.status = 'escalated' OR (NEW.status = 'approved' AND NEW.escalated = 1)
BEGIN
    SELECT RAISE(ABORT, 'invalid image workflow state');
END;
