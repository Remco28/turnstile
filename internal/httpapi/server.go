package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Remco28/turnstile/internal/models"
	"github.com/Remco28/turnstile/internal/store"
)

type Server struct {
	store *store.Store
}

type validateRequest struct {
	Token   string `json:"token"`
	Project string `json:"project"`
}

func New(s *store.Store) *Server {
	return &Server{store: s}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/validate", s.handleValidate)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req validateRequest
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}
	}

	tokenValue := strings.TrimSpace(firstNonEmpty(
		r.Header.Get("X-API-Key"),
		strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
		req.Token,
	))
	project := strings.TrimSpace(req.Project)
	if tokenValue == "" || project == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "token and project are required"})
		return
	}

	result, err := s.store.ValidateToken(tokenValue, project, r.RemoteAddr, r.UserAgent())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	status := http.StatusOK
	if !result.Authorized {
		status = http.StatusForbidden
	}
	writeJSON(w, status, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var _ = models.ValidationResult{}
