package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	authsvc "github.com/novudesk/novudesk/internal/application/auth"
	"github.com/novudesk/novudesk/internal/interfaces/http/handlers"
	"github.com/novudesk/novudesk/internal/interfaces/http/middleware"
	"github.com/novudesk/novudesk/internal/interfaces/sse"
)

type Router struct {
	auth       *handlers.AuthHandler
	tickets    *handlers.TicketHandler
	sseManager *sse.Manager
	authSvc    *authsvc.Service
	corsOrigins []string
}

func NewRouter(
	auth *handlers.AuthHandler,
	tickets *handlers.TicketHandler,
	sseManager *sse.Manager,
	authSvc *authsvc.Service,
	corsOrigins []string,
) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health checks — no auth required.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", auth.Login)
			r.Post("/register", auth.Register)
			r.Post("/logout", auth.Logout)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(authSvc))

			// SSE realtime stream
			r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
				claims := middleware.ClaimsFromContext(r.Context())
				sseManager.ServeHTTP(w, r, claims.OrgID, claims.UserID)
			})

			// Tickets
			r.Route("/tickets", func(r chi.Router) {
				r.With(middleware.RequirePermission("tickets:view")).Get("/", tickets.List)
				r.With(middleware.RequirePermission("tickets:create")).Post("/", tickets.Create)
				r.With(middleware.RequirePermission("tickets:view")).Get("/{id}", tickets.Get)
				r.With(middleware.RequirePermission("tickets:update_any")).Patch("/{id}", tickets.Update)
				r.With(middleware.RequirePermission("tickets:delete")).Delete("/{id}", tickets.Delete)
			})
		})
	})

	return r
}
