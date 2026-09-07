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
	"time"

	"friends-records/internal/apperror"
	modelmysql "friends-records/internal/models/mysql"
)

type HealthCertificate struct {
	Store    *modelmysql.Store
	RootDir  string
	MaxBytes int64
}

type InputError struct{ Message string }

func (e InputError) Error() string { return e.Message }

type HealthCertificateResult struct {
	ID        uint64 `json:"id"`
	FileURL   string `json:"file_url"`
	ExpiresOn string `json:"expires_on"`
}

func (s HealthCertificate) Save(ctx context.Context, employeeID uint64, issuedOn, expiresOn string, file multipart.File, header *multipart.FileHeader) (HealthCertificateResult, error) {
	if employeeID == 0 {
		return HealthCertificateResult{}, InputError{"员工ID不能为空"}
	}
	issued, err := time.Parse("2006-01-02", issuedOn)
	if err != nil {
		return HealthCertificateResult{}, InputError{"发证日期格式必须是YYYY-MM-DD"}
	}
	expires, err := time.Parse("2006-01-02", expiresOn)
	if err != nil {
		return HealthCertificateResult{}, InputError{"到期日期格式必须是YYYY-MM-DD"}
	}
	if expires.Before(issued) {
		return HealthCertificateResult{}, InputError{"健康证到期日期不能早于发证日期"}
	}
	prefix := make([]byte, 512)
	count, readErr := io.ReadFull(file, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return HealthCertificateResult{}, apperror.Wrap(readErr, "读取健康证图片失败")
	}
	prefix = prefix[:count]
	mimeType := http.DetectContentType(prefix)
	extension := ""
	switch mimeType {
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	case "image/webp":
		extension = ".webp"
	default:
		return HealthCertificateResult{}, InputError{"健康证照片只支持JPG、PNG或WebP格式"}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "重置健康证图片读取位置失败")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "生成健康证文件名失败")
	}
	filename := hex.EncodeToString(random) + extension
	directory := filepath.Join(s.RootDir, "health-certificates")
	if err := os.MkdirAll(directory, 0750); err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "创建健康证上传目录失败")
	}
	temporary, err := os.CreateTemp(directory, ".upload-*")
	if err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "创建健康证临时文件失败")
	}
	temporaryName := temporary.Name()
	keep := false
	defer func() {
		temporary.Close()
		if !keep {
			_ = os.Remove(temporaryName)
		}
	}()
	written, err := io.Copy(temporary, io.LimitReader(file, s.MaxBytes+1))
	if err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "保存健康证图片失败")
	}
	if written > s.MaxBytes {
		return HealthCertificateResult{}, InputError{"健康证照片超过允许的大小"}
	}
	if err := temporary.Sync(); err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "同步健康证图片失败")
	}
	if err := temporary.Close(); err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "关闭健康证临时文件失败")
	}
	finalPath := filepath.Join(directory, filename)
	if err := os.Rename(temporaryName, finalPath); err != nil {
		return HealthCertificateResult{}, apperror.Wrap(err, "完成健康证图片保存失败")
	}
	keep = true
	storageKey := "health-certificates/" + filename
	fileURL := "/uploads/" + storageKey
	id, err := s.Store.SaveHealthCertificate(ctx, modelmysql.HealthCertificateInput{EmployeeID: employeeID, StorageKey: storageKey, FileURL: fileURL, OriginalName: filepath.Base(header.Filename), MIMEType: mimeType, FileSize: written, IssuedOn: issued, ExpiresOn: expires})
	if err != nil {
		_ = os.Remove(finalPath)
		return HealthCertificateResult{}, err
	}
	return HealthCertificateResult{ID: id, FileURL: fileURL, ExpiresOn: expiresOn}, nil
}
