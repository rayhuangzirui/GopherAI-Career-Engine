ALTER TABLE tasks
ADD COLUMN artifact_key VARCHAR(255) NULL AFTER result_payload,
ADD COLUMN artifact_storage VARCHAR(20) NULL AFTER artifact_key;