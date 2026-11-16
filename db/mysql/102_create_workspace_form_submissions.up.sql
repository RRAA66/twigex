CREATE TABLE IF NOT EXISTS workspace_form_submissions (
    id           VARCHAR(36) NOT NULL PRIMARY KEY,
    form_id      VARCHAR(36) NOT NULL,
    task_id      VARCHAR(36) NOT NULL,
    submitted_by VARCHAR(36),
    submitted_at BIGINT      NOT NULL,
    FOREIGN KEY (form_id) REFERENCES workspace_forms(id) ON DELETE CASCADE,
    INDEX idx_workspace_form_submissions_form_id (form_id, submitted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
