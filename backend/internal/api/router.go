package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pebt/backend/internal/httpx"
)

// NewRouter wires every route of the API contract (design §3) plus, when a
// built frontend exists, the SPA served from the same origin (D37 — no CORS).
// Two muxes: the API mux owns everything under /api/ (unknown API routes get
// the 404 envelope and can never fall through to the SPA); the root mux
// serves the SPA (or the not-found envelope when no build is present).
func (s *Server) NewRouter() http.Handler {
	apiMux := http.NewServeMux()

	// Authentication (all POSTs carry the CSRF requirement, D36)
	apiMux.HandleFunc("POST /api/v1/auth/register", s.requireCSRF(s.handleRegister))
	apiMux.HandleFunc("POST /api/v1/auth/login", s.requireCSRF(s.handleLogin))
	apiMux.HandleFunc("POST /api/v1/auth/logout", s.requireCSRF(s.requireAuth(s.handleLogout)))

	// Categories (FR2)
	apiMux.HandleFunc("GET /api/v1/categories", s.requireAuth(s.handleCategoryList))
	apiMux.HandleFunc("POST /api/v1/categories", s.requireCSRF(s.requireAuth(s.handleCategoryCreate)))
	apiMux.HandleFunc("PUT /api/v1/categories/{id}", s.requireCSRF(s.requireAuth(s.handleCategoryUpdate)))
	apiMux.HandleFunc("DELETE /api/v1/categories/{id}", s.requireCSRF(s.requireAuth(s.handleCategoryDelete)))

	// Transactions (FR3)
	apiMux.HandleFunc("GET /api/v1/transactions", s.requireAuth(s.handleTransactionList))
	apiMux.HandleFunc("POST /api/v1/transactions", s.requireCSRF(s.requireAuth(s.handleTransactionCreate)))
	apiMux.HandleFunc("GET /api/v1/transactions/{id}", s.requireAuth(s.handleTransactionGet))
	apiMux.HandleFunc("PUT /api/v1/transactions/{id}", s.requireCSRF(s.requireAuth(s.handleTransactionUpdate)))
	apiMux.HandleFunc("DELETE /api/v1/transactions/{id}", s.requireCSRF(s.requireAuth(s.handleTransactionDelete)))

	// Budgets (FR4)
	apiMux.HandleFunc("GET /api/v1/budgets", s.requireAuth(s.handleBudgetList))
	apiMux.HandleFunc("POST /api/v1/budgets", s.requireCSRF(s.requireAuth(s.handleBudgetCreate)))
	apiMux.HandleFunc("PUT /api/v1/budgets/{id}", s.requireCSRF(s.requireAuth(s.handleBudgetUpdate)))
	apiMux.HandleFunc("DELETE /api/v1/budgets/{id}", s.requireCSRF(s.requireAuth(s.handleBudgetDelete)))

	// Dashboard (FR5)
	apiMux.HandleFunc("GET /api/v1/dashboard", s.requireAuth(s.handleDashboard))

	// Unknown API routes get the standard 404 envelope (D28); as the API
	// mux's fallback it also covers method mismatches under /api/.
	apiMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Unknown API route")
	})

	root := http.NewServeMux()
	root.Handle("/api/", apiMux)

	// Everything else: the built SPA when available, else a 404 envelope.
	if s.Cfg.StaticDir != "" {
		spa := spaHandler(s.Cfg.StaticDir)
		root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Unknown route")
				return
			}
			spa.ServeHTTP(w, r)
		})
	} else {
		root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Unknown route")
		})
	}

	s.router = s.withLogging(root)
	return s.router
}

// spaHandler serves files from dir with an index.html fallback (client-side
// routing), refusing paths that escape the directory.
func spaHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleaned := filepath.Clean(r.URL.Path)
		abs := filepath.Join(dir, cleaned)
		root, err := filepath.Abs(dir)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong. Please try again.")
			return
		}
		absFull, err := filepath.Abs(abs)
		if err != nil || absFull != root && !strings.HasPrefix(absFull, root+string(os.PathSeparator)) {
			// Path traversal attempt: serve the app shell instead.
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	})
}
