package api

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"github.com/ogabrielrodrigues/ama/api/internal/store/pg"
)

type apiHandler struct {
	queries     *pg.Queries
	router      *chi.Mux
	upgrader    websocket.Upgrader
	subscribers map[string]map[*websocket.Conn]context.CancelFunc
	mut         *sync.Mutex
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func NewHandler(queries *pg.Queries) http.Handler {
	handler := apiHandler{
		queries: queries,
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
			return true
		}},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mut:         &sync.Mutex{},
	}

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.Recoverer,
		middleware.Logger,
	)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{os.Getenv("AMA_API_CORS_ORIGIN")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/subscribe/{room_id}", handler.handleSubscribe)

	router.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", handler.handleCreateRoom)
			r.Get("/", handler.handleFindRooms)

			r.Route("/{room_id}/messages", func(r chi.Router) {
				r.Post("/", handler.handleCreateRoomMessage)
				r.Get("/", handler.handleGetRoomMessages)

				r.Route("/{message_id}", func(r chi.Router) {
					r.Get("/", handler.handleFindRoomMessage)
					r.Patch("/react", handler.handleReactToMessage)
					r.Delete("/react", handler.handleRemoveReactFromMessage)
					r.Patch("/answer", handler.handleMarkMessageAsAnswered)
				})
			})
		})
	})

	handler.router = router
	return handler
}
