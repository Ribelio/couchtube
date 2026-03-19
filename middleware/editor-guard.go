package middleware

import "net/http"

func EditorGuardOff(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}

func EditorGuardReadOnly(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func EditorGuardEnabled(next http.HandlerFunc) http.HandlerFunc {
	return next
}
