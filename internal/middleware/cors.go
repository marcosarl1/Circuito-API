package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	origins := parseOrigins(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")

			if origin != "" && isAllowedOrigin(origin, origins) {
				writer.Header().Set("Access-Control-Allow-Origin", origin)
				writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Request-Id")
				writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			}

			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

func parseOrigins(allowedOrigins string) map[string]struct{} {
	origins := make(map[string]struct{})

	for _, origin := range strings.Split(allowedOrigins, ",") {
		trimmedOrigin := strings.TrimSpace(origin)

		if trimmedOrigin != "" {
			origins[trimmedOrigin] = struct{}{}
		}
	}
	return origins
}

func isAllowedOrigin(origin string, allowedOrigins map[string]struct{}) bool {
	if _, exists := allowedOrigins[origin]; exists {
		return true
	}

	_, allowAnyOrigin := allowedOrigins["*"]

	return allowAnyOrigin
}
