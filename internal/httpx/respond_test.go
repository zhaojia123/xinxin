package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"friends-records/internal/apperror"
)

func TestErrorIncludesFileAndLine(t *testing.T) {
	recorder := httptest.NewRecorder()
	Error(recorder, http.StatusBadRequest, "测试错误")
	if body := recorder.Body.String(); !strings.Contains(body, "respond_test.go:") || !strings.Contains(body, "测试错误") {
		t.Fatalf("错误响应没有文件行号：%s", body)
	}
}

func TestErrorWithCauseIncludesSQLLocation(t *testing.T) {
	recorder := httptest.NewRecorder()
	cause := apperror.Wrap(errors.New("模拟MySQL错误"), "查询员工失败")
	ErrorWithCause(recorder, http.StatusInternalServerError, "数据读取失败", cause)
	body := recorder.Body.String()
	for _, expected := range []string{"respond_test.go:", "数据读取失败", "查询员工失败", "模拟MySQL错误"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("错误响应缺少%q：%s", expected, body)
		}
	}
}
