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

// LedgerProof 负责保存台账转账凭证图片，并把文件元数据写入数据库。
type LedgerProof struct {
	Store    *modelmysql.Store
	RootDir  string
	MaxBytes int64
}

type LedgerProofResult struct {
	ID           uint64 `json:"id"`
	FileURL      string `json:"file_url"`
	OriginalName string `json:"original_name"`
}

func (s LedgerProof) Save(ctx context.Context, ledgerID, actor uint64, file multipart.File, header *multipart.FileHeader) (LedgerProofResult, error) {
	if ledgerID == 0 {
		return LedgerProofResult{}, InputError{"台账流水ID不能为空"}
	}
	if header == nil {
		return LedgerProofResult{}, InputError{"请选择转账凭证图片"}
	}
	prefix := make([]byte, 512)
	count, readErr := io.ReadFull(file, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return LedgerProofResult{}, apperror.Wrap(readErr, "读取台账凭证图片失败")
	}
	prefix = prefix[:count]
	mimeType := http.DetectContentType(prefix)
	extension := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}[mimeType]
	if extension == "" {
		return LedgerProofResult{}, InputError{"转账凭证只支持JPG、PNG或WebP格式"}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "重置台账凭证读取位置失败")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "生成台账凭证文件名失败")
	}
	filename := hex.EncodeToString(random) + extension
	directory := filepath.Join(s.RootDir, "ledger-proofs")
	if err := os.MkdirAll(directory, 0750); err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "创建台账凭证目录失败")
	}
	temporary, err := os.CreateTemp(directory, ".upload-*")
	if err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "创建台账凭证临时文件失败")
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
		return LedgerProofResult{}, apperror.Wrap(err, "保存台账凭证图片失败")
	}
	if written > s.MaxBytes {
		return LedgerProofResult{}, InputError{"转账凭证图片超过允许的大小"}
	}
	if err := temporary.Sync(); err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "同步台账凭证图片失败")
	}
	if err := temporary.Close(); err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "关闭台账凭证临时文件失败")
	}
	finalPath := filepath.Join(directory, filename)
	if err := os.Rename(temporaryName, finalPath); err != nil {
		return LedgerProofResult{}, apperror.Wrap(err, "完成台账凭证图片保存失败")
	}
	keep = true
	storageKey := "ledger-proofs/" + filename
	fileURL := "/uploads/" + storageKey
	id, err := s.Store.SaveLedgerAttachment(ctx, modelmysql.LedgerAttachmentInput{LedgerEntryID: ledgerID, StorageKey: storageKey, FileURL: fileURL, OriginalName: filepath.Base(header.Filename), MIMEType: mimeType, FileSize: written, UploadedBy: actor})
	if err != nil {
		_ = os.Remove(finalPath)
		return LedgerProofResult{}, err
	}
	return LedgerProofResult{ID: id, FileURL: fileURL, OriginalName: filepath.Base(header.Filename)}, nil
}
