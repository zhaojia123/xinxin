package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"friends-records/api/request"
	"friends-records/internal/httpx"
	"friends-records/internal/models/mysql"
	driver "github.com/go-sql-driver/mysql"
)

func adminID(w http.ResponseWriter, r *http.Request, required bool) (uint64, bool) {
	value := r.URL.Query().Get("id")
	if value == "" && !required {
		return 0, true
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		httpx.Error(w, 400, "记录ID不正确")
		return 0, false
	}
	return id, true
}
func adminWriteError(w http.ResponseWriter, err error) {
	var input mysql.MiniInputError
	var dbErr *driver.MySQLError
	switch {
	case errors.As(err, &input):
		httpx.Error(w, 400, input.Message)
	case errors.Is(err, mysql.ErrNotFound):
		httpx.Error(w, 404, "记录不存在，请刷新列表")
	case errors.As(err, &dbErr) && dbErr.Number == 1062:
		httpx.Error(w, 409, "编号或名称已存在，请修改后保存")
	default:
		fail(w, err, "数据保存失败")
	}
}
func createdStatus(method string) int {
	if method == http.MethodPost {
		return http.StatusCreated
	}
	return http.StatusOK
}

func (h *Handler) AdminOptionsAPI(w http.ResponseWriter, r *http.Request) {
	if !getOnly(w, r) {
		return
	}
	v, err := h.Store.AdminOptions(r.Context())
	if err != nil {
		adminWriteError(w, err)
		return
	}
	httpx.JSON(w, 200, v)
}

func (h *Handler) AdminDepartmentsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id, ok := adminID(w, r, true)
		if !ok {
			return
		}
		v, err := h.Store.DepartmentRecord(r.Context(), id)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, 200, v)
	case http.MethodPost, http.MethodPut:
		id, ok := adminID(w, r, r.Method == http.MethodPut)
		if !ok {
			return
		}
		var v request.DepartmentInput
		if !decodeJSON(w, r, &v) {
			return
		}
		if err := v.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.SaveDepartment(r.Context(), id, v)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST", "PUT")
	}
}
func (h *Handler) AdminPositionsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id, ok := adminID(w, r, true)
		if !ok {
			return
		}
		v, err := h.Store.PositionRecord(r.Context(), id)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, 200, v)
	case http.MethodPost, http.MethodPut:
		id, ok := adminID(w, r, r.Method == http.MethodPut)
		if !ok {
			return
		}
		var v request.PositionInput
		if !decodeJSON(w, r, &v) {
			return
		}
		if err := v.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.SavePosition(r.Context(), id, v)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST", "PUT")
	}
}

func (h *Handler) AdminAttendanceAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if id, ok := adminID(w, r, false); !ok {
			return
		} else if id != 0 {
			v, err := h.Store.AttendanceRecordInput(r.Context(), id)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, 200, v)
			return
		}
		h.AttendanceAPI(w, r)
	case http.MethodPost, http.MethodPut:
		id, ok := adminID(w, r, r.Method == http.MethodPut)
		if !ok {
			return
		}
		var v request.AttendanceInput
		if !decodeJSON(w, r, &v) {
			return
		}
		if err := v.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.SaveAttendance(r.Context(), id, v)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST", "PUT")
	}
}
func (h *Handler) AdminAttendanceStatusAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httpx.MethodNotAllowed(w, "PUT")
		return
	}
	id, ok := adminID(w, r, true)
	if !ok {
		return
	}
	var v struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &v) {
		return
	}
	if err := h.Store.SetAttendanceStatus(r.Context(), id, v.Status); err != nil {
		adminWriteError(w, err)
		return
	}
	httpx.JSON(w, 200, map[string]any{"id": id})
}

func (h *Handler) AdminChangesAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if id, ok := adminID(w, r, false); !ok {
			return
		} else if id != 0 {
			v, err := h.Store.ChangeRecordInput(r.Context(), id)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, 200, v)
			return
		}
		h.ChangesAPI(w, r)
	case http.MethodPost:
		var v request.ChangeInput
		if !decodeJSON(w, r, &v) {
			return
		}
		if err := v.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.CreateEmploymentChange(r.Context(), v)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, 201, map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST")
	}
}
func (h *Handler) AdminSalaryAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if id, ok := adminID(w, r, false); !ok {
			return
		} else if id != 0 {
			v, err := h.Store.SalaryRecordInput(r.Context(), id)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, 200, v)
			return
		}
		h.SalaryAPI(w, r)
	case http.MethodPost:
		var v request.SalaryAdjustmentInput
		if !decodeJSON(w, r, &v) {
			return
		}
		if err := v.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.CreateSalaryAdjustment(r.Context(), v)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, 201, map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST")
	}
}
func (h *Handler) AdminLedgerAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if id, ok := adminID(w, r, false); !ok {
			return
		} else if id != 0 {
			v, err := h.Store.MiniLedger(r.Context(), id)
			if err != nil {
				adminWriteError(w, err)
				return
			}
			httpx.JSON(w, 200, v)
			return
		}
		h.LedgerAPI(w, r)
	case http.MethodPost, http.MethodPut:
		id, ok := adminID(w, r, r.Method == http.MethodPut)
		if !ok {
			return
		}
		var v request.LedgerInput
		if !decodeJSON(w, r, &v) {
			return
		}
		if err := v.Validate(); err != nil {
			httpx.Error(w, 400, err.Error())
			return
		}
		id, err := h.Store.SaveMiniLedger(r.Context(), id, 0, v)
		if err != nil {
			adminWriteError(w, err)
			return
		}
		httpx.JSON(w, createdStatus(r.Method), map[string]any{"id": id})
	default:
		httpx.MethodNotAllowed(w, "GET", "POST", "PUT")
	}
}
func (h *Handler) AdminEmployeeLeaveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httpx.MethodNotAllowed(w, "PUT")
		return
	}
	id, ok := adminID(w, r, true)
	if !ok {
		return
	}
	var v struct {
		LeftOn string `json:"left_on"`
	}
	if !decodeJSON(w, r, &v) {
		return
	}
	if v.LeftOn == "" {
		v.LeftOn = time.Now().Format("2006-01-02")
	}
	if err := h.Store.LeaveEmployee(r.Context(), id, v.LeftOn); err != nil {
		adminWriteError(w, err)
		return
	}
	httpx.JSON(w, 200, map[string]any{"id": id})
}
