package mysql

import (
	"context"
	"database/sql"
	"errors"

	"friends-records/api/response"
	"friends-records/internal/apperror"
)

func employeeAttachmentTitle(kind string) string {
	return map[string]string{
		"id_card_front":            "身份证人像面",
		"id_card_back":             "身份证国徽面",
		"health_certificate_front": "健康证正面",
		"health_certificate_back":  "健康证反面",
		"other":                    "其他证件",
	}[kind]
}

func validEmployeeAttachmentType(kind string) bool {
	return employeeAttachmentTitle(kind) != ""
}

func (s *Store) EmployeeAttachments(ctx context.Context, employeeID uint64) ([]response.EmployeeAttachment, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,employee_id,attachment_type,title,file_url,original_name,mime_type,file_size FROM employee_attachments WHERE employee_id=? AND active=1 ORDER BY attachment_type,id`, employeeID)
	if err != nil {
		return nil, apperror.Wrap(err, "查询员工附件失败")
	}
	defer rows.Close()
	items := make([]response.EmployeeAttachment, 0)
	for rows.Next() {
		var item response.EmployeeAttachment
		if err := rows.Scan(&item.ID, &item.EmployeeID, &item.AttachmentType, &item.Title, &item.FileURL, &item.OriginalName, &item.MIMEType, &item.FileSize); err != nil {
			return nil, apperror.Wrap(err, "读取员工附件失败")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type EmployeeAttachmentInput struct {
	EmployeeID     uint64
	AttachmentType string
	Title          string
	StorageKey     string
	FileURL        string
	OriginalName   string
	MIMEType       string
	FileSize       int64
	UploadedBy     uint64
}

func (s *Store) SaveEmployeeAttachment(ctx context.Context, input EmployeeAttachmentInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if !validEmployeeAttachmentType(input.AttachmentType) {
		return 0, MiniInputError{"不支持的员工附件类型"}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存员工附件事务失败")
	}
	defer tx.Rollback()
	var employeeID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM employees WHERE id=? AND active=1 FOR UPDATE`, input.EmployeeID).Scan(&employeeID); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	} else if err != nil {
		return 0, apperror.Wrap(err, "确认员工附件所属员工失败")
	}
	var uploadedBy any
	if input.UploadedBy > 0 {
		uploadedBy = input.UploadedBy
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO employee_attachments(employee_id,attachment_type,title,storage_key,file_url,original_name,mime_type,file_size,uploaded_by) VALUES(?,?,?,?,?,?,?,?,?)`, input.EmployeeID, input.AttachmentType, employeeAttachmentTitle(input.AttachmentType), input.StorageKey, input.FileURL, input.OriginalName, input.MIMEType, input.FileSize, uploadedBy)
	if err != nil {
		return 0, apperror.Wrap(err, "写入员工附件记录失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取员工附件ID失败")
	}
	if err := miniAudit(ctx, tx, input.UploadedBy, "employee_attachment", uint64(id), employeeAttachmentTitle(input.AttachmentType)); err != nil {
		return 0, apperror.Wrap(err, "记录员工附件上传日志失败")
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交员工附件事务失败")
	}
	return uint64(id), nil
}
