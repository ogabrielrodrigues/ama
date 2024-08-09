package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ogabrielrodrigues/ama/api/internal/store/pg"
)

type apiHandler struct {
	queries *pg.Queries
	router  *chi.Mux
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func NewHandler(queries *pg.Queries) http.Handler {
	return apiHandler{
		queries: queries,
		router:  chi.NewRouter(),
	}
}
