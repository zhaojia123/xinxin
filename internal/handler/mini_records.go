package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"friends-records/api/request"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	"friends-records/internal/token"
	driver "github.com/go-sql-driver/mysql"
)

type miniActorKey struct{}

// RequireMini 在每次请求时复查用户状态，停用账号后旧令牌立即失效。
func RequireMini(tokens *token.Manager, enabled func(context.Context, uint64) (bool, error), next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpx.Error(w, 401, "请先登录小程序")
			return
		}
		claims, err := tokens.Parse(parts[1], "mini")
		if err != nil {
			httpx.Error(w, 401, "登录已过期，请重新登录")
			return
		}
		ok, err := enabled(r.Context(), claims.SubjectID)
		if err != nil {
			miniFail(w, err)
			return
		}
		if !ok {
			httpx.Error(w, 403, "账号尚未启用或已停用，请联系管理员")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), miniActorKey{}, claims.SubjectID)))
	}
}

func miniActor(r *http.Request) uint64 {
	id, _ := r.Context().Value(miniActorKey{}).(uint64)
	return id
}

func miniFail(w http.ResponseWriter, err error) {
	log.Printf("小程序请求失败：%v", err)
	var input mysql.MiniInputError
	var dbErr *driver.MySQLError
	switch {
	case errors.As(err, &input):
		httpx.Error(w, 400, input.Message)
	case errors.Is(err, mysql.ErrNotFound):
		httpx.Error(w, 404, "记录不存在，请刷新后重试")
	case errors.Is(err, mysql.ErrNotConfigured):
		httpx.Error(w, 503, "数据库尚未配置，请先配置服务端 MySQL")
	case errors.As(err, &dbErr) && dbErr.Number == 1062:
		httpx.Error(w, 409, "编号或名称已存在，请修改后保存")
	default:
		httpx.Error(w, 500, "服务暂时不可用，请稍后重试或联系管理员")
	}
}

func miniID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	value := r.URL.Query().Get("id")
	if value == "" && r.Method != http.MethodPut {
		return 0, true
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		httpx.Error(w, 400, "记录ID不正确")
		return 0, false
	}
	return id, true
}

func miniPage(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := r.URL.Query().Get("page")
	if value == "" {
		return 0, true
	}
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 || page > 10000 {
		httpx.Error(w, 400, "页码不正确")
		return 0, false
	}
	return (page - 1) * 30, true
}

func (h *Handler) MiniLedgerAPI(w http.ResponseWriter, r *http.Request) {
	id, ok := miniID(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if id != 0 {
			v, err := h.Store.MiniLedger(r.Context(), id)
			if err != nil {
				miniFail(w, err)
				return
			}
			httpx.JSON(w, 200, v)
			return
		}
		offset, ok := miniPage(w, r)
		if !ok {
			return
		}
		month := r.URL.Query().Get("month")
		if month == "" {
			month = time.Now().Format("2006-01")
		}
		date, err := time.Parse("2006-01", month)
		if err != nil || date.Year() < 1000 || date.Year() > 9998 {
			httpx.Error(w, 400, "月份格式应为YYYY-MM")
			return
		}
		direction := r.URL.Query().Get("direction")
		if direction != "" && direction != "income" && direction != "expense" {
			httpx.Error(w, 400, "收支方向不正确")
			return
		}
		records, more, err := h.Store.MiniLedgerList(r.Context(), month, direction, r.URL.Query().Get("q"), offset)
		if err != nil {
			miniFail(w, err)
			return
		}
		summary, err := h.Store.LedgerSummary(r.Context(), month)
		if err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, 200, map[string]any{"records": records, "has_more": more, "summary": summary})
	case http.MethodPost, http.MethodPut:
		if r.Method == http.MethodPost && id != 0 {
			httpx.Error(w, 400, "新增记录不能携带ID")
			return
		}
		var input request.LedgerInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.SaveMiniLedger(r.Context(), id, miniActor(r), input)
		if err != nil {
			miniFail(w, err)
			return
		}
		status := 200
		if r.Method == http.MethodPost {
			status = 201
		}
		httpx.JSON(w, status, map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST", "PUT")
	}
}

func (h *Handler) MiniEmployeesAPI(w http.ResponseWriter, r *http.Request) {
	id, ok := miniID(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if id != 0 {
			v, err := h.Store.MiniEmployee(r.Context(), id)
			if err != nil {
				miniFail(w, err)
				return
			}
			httpx.JSON(w, 200, v)
			return
		}
		offset, ok := miniPage(w, r)
		if !ok {
			return
		}
		status := r.URL.Query().Get("status")
		if status != "" && status != "active" && status != "probation" && status != "left" {
			httpx.Error(w, 400, "员工状态不正确")
			return
		}
		records, more, err := h.Store.MiniEmployeeList(r.Context(), r.URL.Query().Get("q"), status, offset)
		if err != nil {
			miniFail(w, err)
			return
		}
		summary, err := h.Store.EmployeeSummary(r.Context())
		if err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, 200, map[string]any{"records": records, "has_more": more, "summary": summary})
	case http.MethodPost, http.MethodPut:
		if r.Method == http.MethodPost && id != 0 {
			httpx.Error(w, 400, "新增记录不能携带ID")
			return
		}
		var input request.EmployeeInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if err := input.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.SaveEmployee(r.Context(), id, miniActor(r), input)
		if err != nil {
			miniFail(w, err)
			return
		}
		status := 200
		if r.Method == http.MethodPost {
			status = 201
		}
		httpx.JSON(w, status, map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST", "PUT")
	}
}

func (h *Handler) MiniOptionsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		v, err := h.Store.MiniOptions(r.Context())
		if err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, 200, v)
	case http.MethodPost:
		var input struct {
			Kind         string `json:"kind"`
			Name         string `json:"name"`
			Direction    string `json:"direction"`
			DepartmentID uint64 `json:"department_id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		if input.Name == "" || utf8.RuneCountInString(input.Name) > 64 {
			httpx.Error(w, 400, "名称必填且不能超过64字")
			return
		}
		if input.Kind == "categories" && input.Direction != "income" && input.Direction != "expense" {
			httpx.Error(w, 400, "请选择分类的收支方向")
			return
		}
		id, err := h.Store.CreateMiniOption(r.Context(), input.Kind, input.Name, input.Direction, input.DepartmentID, miniActor(r))
		if err != nil {
			miniFail(w, err)
			return
		}
		httpx.JSON(w, 201, map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST")
	}
}
