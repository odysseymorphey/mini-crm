package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/time/rate"
	"mini-crm/internal/config"
	"mini-crm/internal/middleware"
	"mini-crm/internal/service"
)

type RouterParams struct {
	Config         *config.Config
	AuthService    *service.AuthService
	AuthHandler    *AuthHandler
	TeamHandler    *TeamHandler
	TaskHandler    *TaskHandler
	CommentHandler *CommentHandler
	StatsHandler   *StatsHandler
}

func NewRouter(p RouterParams) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.Logging)

	ipLimiter := middleware.NewIPRateLimiter(
		rate.Limit(p.Config.Rate.RequestsPerSecond),
		p.Config.Rate.Burst,
	)
	r.Use(middleware.RateLimit(ipLimiter))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, map[string]string{
			"status": "healthy",
			"env":    p.Config.Server.Env,
		})
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	))
	r.Get("/docs/doc.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", p.AuthHandler.Register)
		r.Post("/login", p.AuthHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(p.AuthService))

			r.Route("/teams", func(r chi.Router) {
				r.Post("/", p.TeamHandler.CreateTeam)
				r.Get("/", p.TeamHandler.GetUserTeams)
				r.Post("/{id}/invite", p.TeamHandler.InviteMember)
				r.Get("/{team_id}/stats", p.StatsHandler.GetTeamStats)
			})

			r.Route("/tasks", func(r chi.Router) {
				r.Post("/", p.TaskHandler.CreateTask)
				r.Get("/", p.TaskHandler.ListTasks)
				r.Put("/{id}", p.TaskHandler.UpdateTask)
				r.Get("/{id}/history", p.TaskHandler.GetTaskHistory)
				r.Post("/{id}/comments", p.CommentHandler.AddComment)
				r.Get("/{id}/comments", p.CommentHandler.ListComments)
			})
		})
	})

	return r
}
