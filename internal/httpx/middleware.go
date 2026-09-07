package httpx

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func Common(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("程序发生未处理异常：%v\n%s", recovered, debug.Stack())
				Error(w, http.StatusInternalServerError, "服务器内部错误")
			}
			log.Printf("请求完成：%s %s，耗时 %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
		}()
		next.ServeHTTP(w, r)
	})
}
