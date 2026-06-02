package v1

import (
    "log/slog"
    "net/http"
    "time"

    "github.com/coreos/go-oidc/v3/oidc"
    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"
    "github.com/rabobank/papi/internal/api/v1/handlers"
    "github.com/rabobank/papi/internal/api/v1/middleware"
)

type RouterConfig struct {
    Verifier         *oidc.IDTokenVerifier
    Logger           *slog.Logger
    SystemHandler    *handlers.SystemHandler
    NamespaceHandler *handlers.NamespaceHandler
    GroupHandler     *handlers.GroupHandler
    RateLimiter      *middleware.RateLimiter
}

func NewRouter(cfg RouterConfig) chi.Router {
    r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(requestLogger(cfg.Logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	if cfg.RateLimiter != nil {
		r.Use(cfg.RateLimiter.Handler)
	}

    // API v1 routes
    r.Route("/api/v1", func(r chi.Router) {
        // Public endpoints (no auth)
        r.Group(func(r chi.Router) {
            r.Get("/health", cfg.SystemHandler.Health)
            r.Get("/info", cfg.SystemHandler.Info)
        })

        // Authenticated endpoints
        r.Group(func(r chi.Router) {
            r.Use(middleware.OIDCAuth(cfg.Verifier))
            r.Get("/namespaces", cfg.NamespaceHandler.List)
            r.Post("/namespaces", cfg.NamespaceHandler.Create)
            r.Get("/namespaces/{namespaceId}/status", cfg.NamespaceHandler.GetStatus)
            r.Get("/groups", cfg.GroupHandler.List)
            r.Post("/groups", cfg.GroupHandler.Create)
            r.Put("/groups/{groupId}", cfg.GroupHandler.Update)
            r.Delete("/groups/{groupId}", cfg.GroupHandler.Delete)
        })
    })

    return r
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
            next.ServeHTTP(ww, r)
            logger.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.Status(),
                "duration_ms", time.Since(start).Milliseconds(),
                "request_id", middleware.RequestIDFromContext(r.Context()),
            )
        })
    }
}
