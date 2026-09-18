-- 已有项目执行一次即可；如果从头执行 sql/schema.sql，则不需要再次执行本文件。
CREATE TABLE IF NOT EXISTS ledger_attachments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '台账附件ID',
    ledger_entry_id BIGINT UNSIGNED NOT NULL COMMENT '所属台账流水ID',
    storage_key VARCHAR(255) NOT NULL COMMENT '服务器内的相对存储路径',
    file_url VARCHAR(500) NOT NULL COMMENT '页面访问地址',
    original_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '上传时原文件名',
    mime_type VARCHAR(64) NOT NULL COMMENT '图片类型',
    file_size BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '文件字节数',
    uploaded_by BIGINT UNSIGNED NULL COMMENT '上传用户ID',
    active TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1正常，0已删除',
    deleted_at DATETIME NULL COMMENT '删除时间',
    deleted_by BIGINT UNSIGNED NULL COMMENT '执行删除的用户ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ledger_attachments_entry (ledger_entry_id, active, id),
    KEY idx_ledger_attachments_deleted_by (deleted_by),
    CONSTRAINT fk_ledger_attachments_entry FOREIGN KEY (ledger_entry_id) REFERENCES ledger_entries (id) ON DELETE CASCADE,
    CONSTRAINT fk_ledger_attachments_uploader FOREIGN KEY (uploaded_by) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='台账转账凭证附件';
