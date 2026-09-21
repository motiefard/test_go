package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"card2sheba/internal/apperr"
	"card2sheba/internal/inquiry"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

type Handler struct {
	svc *inquiry.Service
	log *slog.Logger
}

func NewHandler(svc *inquiry.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

type cardRequest struct {
	Card string `json:"card"`
}

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func (h *Handler) CardToSheba(w http.ResponseWriter, r *http.Request) {
	requestID := RequestIDFrom(r.Context())
	var req cardRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	if err := dec.Decode(&req); err != nil {
		writeError(w, requestID, apperr.New(http.StatusBadRequest, apperr.CodeValidation, "request body must be JSON with a card field"))
		return
	}

	res := h.svc.Convert(r.Context(), requestID, clientIP(r), req.Card)
	if res.Error != nil {
		writeError(w, requestID, res.Error)
		return
	}
	writeJSON(w, http.StatusOK, res.Payload)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func NewRouter(h *Handler, corsOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /api/v1/card-to-sheba", h.CardToSheba)

	return chain(mux, recoverer(h.log), requestID, cors(corsOrigin))
}

func chain(next http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		next = mws[i](next)
	}
	return next
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func cors(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", "request_id", RequestIDFrom(r.Context()), "error", "panic")
					writeError(w, RequestIDFrom(r.Context()), apperr.New(http.StatusInternalServerError, apperr.CodeInternal, "internal error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func writeError(w http.ResponseWriter, requestID string, err *apperr.Error) {
	writeJSON(w, err.HTTPStatus, errorBody{Error: errorPayload{
		Code:      err.Code,
		Message:   err.Message,
		RequestID: requestID,
	}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}
