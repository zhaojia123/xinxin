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

func (h *Handler) AdminEmployeeDocumentUpload(upload service.EmployeeDocument, tokens *token.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := adminActor(w, r, tokens)
		if !ok {
			return
		}
		h.uploadEmployeeDocument(w, r, upload, actor)
	}
}

func (h *Handler) MiniEmployeeDocumentUpload(upload service.EmployeeDocument) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.MethodNotAllowed(w, http.MethodPost)
			return
		}
		h.uploadEmployeeDocument(w, r, upload, miniActor(r))
	}
}

func (h *Handler) uploadEmployeeDocument(w http.ResponseWriter, r *http.Request, upload service.EmployeeDocument, actor uint64) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w, http.MethodPost)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes+(1<<20))
	if err := r.ParseMultipartForm(upload.MaxBytes); err != nil {
		httpx.Error(w, http.StatusBadRequest, "上传内容过大或格式不正确")
		return
	}
	employeeID, err := strconv.ParseUint(r.FormValue("employee_id"), 10, 64)
	if err != nil || employeeID == 0 {
		httpx.Error(w, http.StatusBadRequest, "员工ID不正确")
		return
	}
	file, header, err := r.FormFile("document")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "请选择员工证件图片")
		return
	}
	defer file.Close()
	result, err := upload.Save(r.Context(), employeeID, actor, r.FormValue("attachment_type"), file, header)
	if err != nil {
		var inputErr service.InputError
		if errors.As(err, &inputErr) {
			log.Printf("员工证件上传参数错误：%v", err)
			httpx.Error(w, http.StatusBadRequest, inputErr.Message)
			return
		}
		if errors.Is(err, mysql.ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "员工不存在或已删除")
			return
		}
		fail(w, err, "员工证件上传失败")
		return
	}
	httpx.JSON(w, http.StatusCreated, result)
}
