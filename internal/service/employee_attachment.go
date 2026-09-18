package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"friends-records/internal/apperror"
	modelmysql "friends-records/internal/models/mysql"
)

// EmployeeDocument 负责保存身份证、健康证等员工图片附件。
type EmployeeDocument struct {
	Store    *modelmysql.Store
	RootDir  string
	MaxBytes int64
}

type EmployeeDocumentResult struct {
	ID             uint64 `json:"id"`
	AttachmentType string `json:"attachment_type"`
	Title          string `json:"title"`
	FileURL        string `json:"file_url"`
	OriginalName   string `json:"original_name"`
}

func (s EmployeeDocument) Save(ctx context.Context, employeeID, actor uint64, kind string, file multipart.File, header *multipart.FileHeader) (EmployeeDocumentResult, error) {
	if employeeID == 0 {
		return EmployeeDocumentResult{}, InputError{"员工ID不能为空"}
	}
	if header == nil {
		return EmployeeDocumentResult{}, InputError{"请选择员工证件图片"}
	}
	prefix := make([]byte, 512)
	count, readErr := io.ReadFull(file, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return EmployeeDocumentResult{}, apperror.Wrap(readErr, "读取员工证件图片失败")
	}
	prefix = prefix[:count]
	mimeType := http.DetectContentType(prefix)
	extension := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}[mimeType]
	if extension == "" {
		return EmployeeDocumentResult{}, InputError{"员工证件图片只支持JPG、PNG或WebP格式"}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "重置员工证件图片读取位置失败")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "生成员工证件文件名失败")
	}
	filename := hex.EncodeToString(random) + extension
	directory := filepath.Join(s.RootDir, "employee-documents")
	if err := os.MkdirAll(directory, 0750); err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "创建员工证件目录失败")
	}
	temporary, err := os.CreateTemp(directory, ".upload-*")
	if err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "创建员工证件临时文件失败")
	}
	temporaryName := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(temporaryName)
		}
	}()
	written, err := io.Copy(temporary, io.LimitReader(file, s.MaxBytes+1))
	if err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "保存员工证件图片失败")
	}
	if written > s.MaxBytes {
		return EmployeeDocumentResult{}, InputError{"员工证件图片超过允许的大小"}
	}
	if err := temporary.Sync(); err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "同步员工证件图片失败")
	}
	if err := temporary.Close(); err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "关闭员工证件临时文件失败")
	}
	if err := os.Rename(temporaryName, filepath.Join(directory, filename)); err != nil {
		return EmployeeDocumentResult{}, apperror.Wrap(err, "完成员工证件图片保存失败")
	}
	keep = true
	storageKey := "employee-documents/" + filename
	fileURL := "/uploads/" + storageKey
	cleanName := filepath.Base(header.Filename)
	id, err := s.Store.SaveEmployeeAttachment(ctx, modelmysql.EmployeeAttachmentInput{EmployeeID: employeeID, AttachmentType: kind, StorageKey: storageKey, FileURL: fileURL, OriginalName: cleanName, MIMEType: mimeType, FileSize: written, UploadedBy: actor})
	if err != nil {
		_ = os.Remove(filepath.Join(directory, filename))
		return EmployeeDocumentResult{}, err
	}
	return EmployeeDocumentResult{ID: id, AttachmentType: kind, Title: cleanName, FileURL: fileURL, OriginalName: cleanName}, nil
}
