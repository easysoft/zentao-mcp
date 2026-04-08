package middleware

import (
	"log/slog"
	"net/http"
)

// AuthorizationHandler extracts auth information from headers and stores it in the context.
type AuthorizationHandler struct {
	handler http.Handler
}

// NewAuthorizationHandler wraps an http.Handler with authorization extraction.
func NewAuthorizationHandler(handler http.Handler) http.Handler {
	return &AuthorizationHandler{handler: handler}
}

func (h *AuthorizationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 优先使用自定义的 token 头，用于类似 Zentao 这种使用 token 头的服务。
	if token := r.Header.Get("token"); token != "" {
		ctx := WithAuthorization(r.Context(), token)
		ctx = WithAuthorizationSource(ctx, true)
		r = r.WithContext(ctx)

		h.handler.ServeHTTP(w, r)

		return
	}

	// 如果没有 token 头，则退回到 Authorization 头，并要求必须提供。
	auth := r.Header.Get("Authorization")
	if auth == "" {
		slog.WarnContext(r.Context(), "missing authorization header")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	ctx := WithAuthorization(r.Context(), auth)
	ctx = WithAuthorizationSource(ctx, false)
	r = r.WithContext(ctx)

	h.handler.ServeHTTP(w, r)
}
