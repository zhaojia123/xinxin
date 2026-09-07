package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"friends-records/internal/httpx"
	"friends-records/internal/service"
)

func (h *Handler) HealthCertificateUpload(upload service.HealthCertificate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		file, header, err := r.FormFile("certificate")
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "请选择健康证照片")
			return
		}
		defer file.Close()
		result, err := upload.Save(r.Context(), employeeID, r.FormValue("issued_on"), r.FormValue("expires_on"), file, header)
		if err != nil {
			var inputErr service.InputError
			if errors.As(err, &inputErr) {
				log.Printf("健康证上传参数错误：%v", err)
				httpx.Error(w, http.StatusBadRequest, inputErr.Message)
				return
			}
			fail(w, err, "健康证上传失败")
			return
		}
		httpx.JSON(w, http.StatusCreated, result)
	}
}
