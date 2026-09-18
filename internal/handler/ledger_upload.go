package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/service"
	"friends-records/internal/token"
)

func (h *Handler) AdminLedgerProofUpload(upload service.LedgerProof, tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		h.uploadLedgerProof(w, r, upload, actor)
	}
}

func (h *Handler) MiniLedgerProofUpload(upload service.LedgerProof) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		h.uploadLedgerProof(w, r, upload, miniActor(r))
	}
}

func (h *Handler) uploadLedgerProof(w http.ResponseWriter, r *http.Request, upload service.LedgerProof, actor uint64) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes+(1<<20))
	if err := r.ParseMultipartForm(upload.MaxBytes); err != nil {
		httpx.Error(w, http.StatusBadRequest, "上传内容过大或格式不正确")
		return
	}
	ledgerID, err := strconv.ParseUint(r.FormValue("ledger_id"), 10, 64)
	if err != nil || ledgerID == 0 {
		httpx.Error(w, http.StatusBadRequest, "台账流水ID不正确")
		return
	}
	file, header, err := r.FormFile("proof")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "请选择转账凭证图片")
		return
	}
	defer file.Close()
	result, err := upload.Save(r.Context(), ledgerID, actor, file, header)
	if err != nil {
		var inputErr service.InputError
		if errors.As(err, &inputErr) {
			log.Printf("台账凭证上传参数错误：%v", err)
			httpx.Error(w, http.StatusBadRequest, inputErr.Message)
			return
		}
		if errors.Is(err, mysql.ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "台账流水不存在或已删除")
			return
		}
		fail(w, err, "台账凭证上传失败")
		return
	}
	httpx.JSON(w, http.StatusCreated, result)
}
