package middleware

import "net/http"

// JWTMiddleware is a stub that always passes for the skeleton.
func JWTMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // TODO: verify JWT
        next.ServeHTTP(w, r)
    })
}
