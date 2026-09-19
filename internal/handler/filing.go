package handler

import "net/http"

// FilingLanding 展示公安备案审核期间使用的公开说明页。
func (h *Handler) FilingLanding(w http.ResponseWriter, r *http.Request, filingNumber string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	h.render(w, "filing.html", struct{ FilingNumber string }{FilingNumber: filingNumber})
}
