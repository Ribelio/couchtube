package middleware

import "net/http"

func EditorGuard(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}

func EditorGuardEnabled(next http.HandlerFunc) http.HandlerFunc {
	return next
}
