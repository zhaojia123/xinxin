package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"friends-records/internal/apperror"
)

type HealthCertificateInput struct {
	EmployeeID   uint64
	StorageKey   string
	FileURL      string
	OriginalName string
	MIMEType     string
	FileSize     int64
	IssuedOn     time.Time
	ExpiresOn    time.Time
	UploadedBy   uint64
}

func (s *Store) SaveHealthCertificate(ctx context.Context, input HealthCertificateInput) (uint64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Wrap(err, "开始保存健康证事务失败")
	}
	defer tx.Rollback()
	var employeeID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM employees WHERE id=? AND active=1 FOR UPDATE`, input.EmployeeID).Scan(&employeeID); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	} else if err != nil {
		return 0, apperror.Wrap(err, "确认健康证所属员工失败")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE employee_health_certificates SET is_current=0 WHERE employee_id=? AND is_current=1`, input.EmployeeID); err != nil {
		return 0, apperror.Wrap(err, "更新员工旧健康证状态失败")
	}
	var uploadedBy any
	if input.UploadedBy > 0 {
		uploadedBy = input.UploadedBy
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO employee_health_certificates(employee_id,storage_key,file_url,original_name,mime_type,file_size,issued_on,expires_on,is_current,uploaded_by) VALUES(?,?,?,?,?,?,?,?,1,?)`, input.EmployeeID, input.StorageKey, input.FileURL, input.OriginalName, input.MIMEType, input.FileSize, input.IssuedOn, input.ExpiresOn, uploadedBy)
	if err != nil {
		return 0, apperror.Wrap(err, "写入员工健康证记录失败")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, apperror.Wrap(err, "读取健康证ID失败")
	}
	if err := tx.Commit(); err != nil {
		return 0, apperror.Wrap(err, "提交保存健康证事务失败")
	}
	return uint64(id), nil
}
