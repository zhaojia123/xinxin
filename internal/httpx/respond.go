package httpx

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"

	"friends-records/internal/apperror"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("返回 JSON 数据失败：%v", apperror.At(file, line, "响应内容编码失败"))
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	_, file, line, _ := runtime.Caller(1)
	located := apperror.At(file, line, message)
	log.Printf("请求处理失败：%v", located)
	writeJSON(w, status, ErrorBody{Error: located.Error()})
}

// ErrorWithCause 同时返回处理位置和底层错误位置，数据库查询失败时可直接定位到 SQL 所在的 Go 文件与行号。
func ErrorWithCause(w http.ResponseWriter, status int, message string, cause error) {
	_, file, line, _ := runtime.Caller(1)
	located := apperror.At(file, line, message)
	fullMessage := located.Error()
	if cause != nil {
		fullMessage += "；详细原因：" + cause.Error()
	}
	log.Printf("请求处理失败：%s", fullMessage)
	writeJSON(w, status, ErrorBody{Error: fullMessage})
}

func MethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	for _, method := range allowed {
		w.Header().Add("Allow", method)
	}
	_, file, line, _ := runtime.Caller(1)
	message := "请求方法不允许"
	located := apperror.At(file, line, message)
	log.Printf("请求处理失败：%v", located)
	writeJSON(w, http.StatusMethodNotAllowed, ErrorBody{Error: located.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("返回错误信息失败：%v", apperror.Wrap(err, "错误响应编码失败"))
	}
}
