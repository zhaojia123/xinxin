package mysql

import (
	"context"
	"database/sql"
	"errors"

	"friends-records/internal/apperror"
)

// LedgerAttachment 是台账详情页展示的转账凭证图片。
type LedgerAttachment struct {
	ID            uint64 `json:"id"`
	LedgerEntryID uint64 `json:"ledger_entry_id"`
	StorageKey    string `json:"-"`
	FileURL       string `json:"file_url"`
	OriginalName  string `json:"original_name"`
	MIMEType      string `json:"mime_type"`
	FileSize      int64  `json:"file_size"`
}

type LedgerAttachmentInput struct {
	LedgerEntryID uint64
	StorageKey    string
	FileURL       string
	OriginalName  string
	MIMEType      string
	FileSize      int64
	UploadedBy    uint64
}

func (s *Store) LedgerAttachments(ctx context.Context, ledgerID uint64) ([]LedgerAttachment, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,ledger_entry_id,storage_key,file_url,original_name,mime_type,file_size FROM ledger_attachments WHERE ledger_entry_id=? AND active=1 ORDER BY id`, ledgerID)
	if err != nil {
		return nil, apperror.Wrap(err, "查询台账附件失败")
	}
	defer rows.Close()
	items := make([]LedgerAttachment, 0)
	for rows.Next() {
		var item LedgerAttachment
		if err := rows.Scan(&item.ID, &item.LedgerEntryID, &item.StorageKey, &item.FileURL, &item.OriginalName, &item.MIMEType, &item.FileSize); err != nil {
			return nil, apperror.Wrap(err, "读取台账附件失败")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(err, "遍历台账附件失败")
	}
	return items, nil
}

func (s *Store) SaveLedgerAttachment(ctx context.Context, input LedgerAttachmentInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存台账附件事务失败")
	}
	defer tx.Rollback()
	var entryID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM ledger_entries WHERE id=? AND active=1 FOR UPDATE`, input.LedgerEntryID).Scan(&entryID); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	} else if err != nil {
		return 0, apperror.Wrap(err, "确认台账流水失败")
	}
	var uploadedBy any
	if input.UploadedBy > 0 {
		uploadedBy = input.UploadedBy
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO ledger_attachments(ledger_entry_id,storage_key,file_url,original_name,mime_type,file_size,uploaded_by) VALUES(?,?,?,?,?,?,?)`, input.LedgerEntryID, input.StorageKey, input.FileURL, input.OriginalName, input.MIMEType, input.FileSize, uploadedBy)
	if err != nil {
		return 0, apperror.Wrap(err, "写入台账附件记录失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取台账附件ID失败")
	}
	if err := miniAudit(ctx, tx, input.UploadedBy, "ledger_attachment", uint64(id), input.OriginalName); err != nil {
		return 0, apperror.Wrap(err, "记录台账附件上传日志失败")
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交台账附件事务失败")
	}
	return uint64(id), nil
}
