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
	auth        *handlers.AuthHandler
	tickets     *handlers.TicketHandler
	members     *handlers.MemberHandler
	roles       *handlers.RoleHandler
	teams       *handlers.TeamHandler
	categories  *handlers.CategoryHandler
	comments    *handlers.CommentHandler
	attachments *handlers.AttachmentHandler
	sla         *handlers.SLAHandler
	sseManager  *sse.Manager
	authSvc     *authsvc.Service
	corsOrigins []string
}

func NewRouter(
	auth *handlers.AuthHandler,
	tickets *handlers.TicketHandler,
	members *handlers.MemberHandler,
	roles *handlers.RoleHandler,
	teams *handlers.TeamHandler,
	categories *handlers.CategoryHandler,
	comments *handlers.CommentHandler,
	attachments *handlers.AttachmentHandler,
	sla *handlers.SLAHandler,
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

	// Health check — no auth required.
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

				// Comments / timeline (nested under ticket)
				r.With(middleware.RequirePermission("tickets:view")).Get("/{id}/comments", comments.List)
				r.Post("/{id}/comments", comments.Create)

				// Attachments (nested under ticket)
				r.With(middleware.RequirePermission("tickets:view")).Get("/{id}/attachments", attachments.List)
				r.Post("/{id}/attachments", attachments.Upload)
			})

			// Members
			r.Route("/members", func(r chi.Router) {
				r.With(middleware.RequirePermission("users:view")).Get("/", members.List)
				r.With(middleware.RequirePermission("users:invite")).Post("/", members.Create)
				r.With(middleware.RequirePermission("users:manage_roles")).Patch("/{id}", members.UpdateRole)
				r.With(middleware.RequirePermission("users:deactivate")).Delete("/{id}", members.Deactivate)
				r.With(middleware.RequirePermission("users:deactivate")).Post("/{id}/activate", members.Activate)
				r.With(middleware.RequirePermission("users:manage_roles")).Patch("/{id}/profile", members.UpdateProfile)
				r.With(middleware.RequirePermission("users:manage_roles")).Patch("/{id}/password", members.UpdatePassword)
				r.With(middleware.RequirePermission("users:view")).Get("/{id}/permissions", members.GetPermissions)
				r.With(middleware.RequirePermission("users:manage_roles")).Put("/{id}/permissions", members.SetPermissions)
			})

			// Roles
			r.Route("/roles", func(r chi.Router) {
				r.With(middleware.RequirePermission("users:view")).Get("/", roles.List)
				r.With(middleware.RequirePermission("organization:manage_settings")).Post("/", roles.Create)
				r.With(middleware.RequirePermission("users:view")).Get("/{id}", roles.Get)
				r.With(middleware.RequirePermission("organization:manage_settings")).Patch("/{id}", roles.Update)
				r.With(middleware.RequirePermission("organization:manage_settings")).Delete("/{id}", roles.Delete)
			})

			// Permissions — list all available system permissions for role/member editing forms
			r.With(middleware.RequirePermission("organization:view_settings")).Get("/permissions", roles.ListPermissions)

			// Teams
			r.Route("/teams", func(r chi.Router) {
				r.With(middleware.RequirePermission("teams:view")).Get("/", teams.List)
				r.With(middleware.RequirePermission("teams:manage")).Post("/", teams.Create)
				r.With(middleware.RequirePermission("teams:view")).Get("/{id}", teams.Get)
				r.With(middleware.RequirePermission("teams:manage")).Patch("/{id}", teams.Update)
				r.With(middleware.RequirePermission("teams:manage")).Delete("/{id}", teams.Delete)

				r.With(middleware.RequirePermission("teams:view")).Get("/{id}/members", teams.ListMembers)
				r.With(middleware.RequirePermission("teams:manage")).Post("/{id}/members", teams.AddMember)
				r.With(middleware.RequirePermission("teams:manage")).Delete("/{id}/members/{userId}", teams.RemoveMember)

				r.With(middleware.RequirePermission("teams:view")).Get("/{id}/categories", teams.ListCategories)
				r.With(middleware.RequirePermission("teams:manage")).Post("/{id}/categories", teams.AddCategory)
				r.With(middleware.RequirePermission("teams:manage")).Delete("/{id}/categories/{catId}", teams.RemoveCategory)
			})

			// Categories
			r.Route("/categories", func(r chi.Router) {
				r.With(middleware.RequirePermission("teams:view")).Get("/", categories.List)
				r.With(middleware.RequirePermission("teams:manage")).Post("/", categories.Create)
				r.With(middleware.RequirePermission("teams:manage")).Patch("/{id}", categories.Update)
				r.With(middleware.RequirePermission("teams:manage")).Delete("/{id}", categories.Delete)
			})

			// SLA policies
			r.Route("/sla-policies", func(r chi.Router) {
				r.With(middleware.RequirePermission("sla:view")).Get("/", sla.List)
				r.With(middleware.RequirePermission("sla:manage")).Put("/category/{categoryId}", sla.UpsertForCategory)
				r.With(middleware.RequirePermission("sla:manage")).Delete("/{id}", sla.Delete)
			})

		})
	})

	return r
}
