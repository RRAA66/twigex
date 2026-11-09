CREATE TABLE IF NOT EXISTS workspace_forms (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    workspace_id   VARCHAR(36)  NOT NULL,
    table_id       VARCHAR(36)  NOT NULL,
    name           VARCHAR(255) NOT NULL,
    title          VARCHAR(255) NOT NULL DEFAULT '',
    description    TEXT,
    cover_image    VARCHAR(255),
    cover_position TINYINT UNSIGNED NOT NULL DEFAULT 50,
    logo           VARCHAR(255),
    accent_color   VARCHAR(7),
    public_token   VARCHAR(64)  NOT NULL UNIQUE,
    active         TINYINT(1)   NOT NULL DEFAULT 1,
    created_by     VARCHAR(36)  NOT NULL,
    created_at     BIGINT       NOT NULL,
    updated_at     BIGINT       NOT NULL,
    INDEX idx_workspace_forms_workspace_id (workspace_id),
    INDEX idx_workspace_forms_table_id     (table_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS workspace_form_fields (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    form_id         VARCHAR(36)  NOT NULL,
    header_id       VARCHAR(36)  NOT NULL,
    header_name     VARCHAR(255) NOT NULL DEFAULT '',
    label           VARCHAR(255) NOT NULL,
    placeholder     VARCHAR(255),
    description     TEXT,
    required        TINYINT(1)   NOT NULL DEFAULT 0,
    visible         TINYINT(1)   NOT NULL DEFAULT 1,
    sort_order      INT          NOT NULL DEFAULT 0,
    column_position VARCHAR(10)  NOT NULL DEFAULT 'full',
    FOREIGN KEY (form_id) REFERENCES workspace_forms(id) ON DELETE CASCADE,
    INDEX idx_workspace_form_fields_form_id (form_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
