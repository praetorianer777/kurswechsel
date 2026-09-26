// Package api serves the Kurswechsel JSON API and the website.
package api

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/timeline"
	"github.com/praetorianer777/kurswechsel/internal/topic"
	"github.com/praetorianer777/kurswechsel/internal/web"
)

// Limits for error reports.
const (
	MinMessage   = 10
	MaxMessage   = 2000
	MaxContact   = 200
	ReportsPerIP = 5
	ReportWindow = 10 * time.Minute
	maxBodyBytes = 16 << 10
	searchLimit  = 50
)

// Server holds the API's dependencies.
type Server struct {
	Store *store.Store
	// Frontend is the built website; nil serves the API only.
	Frontend fs.FS
	Now      func() time.Time
	limiter  *limiter
}

// NewHandler returns the root handler with all routes registered.
func NewHandler(s *Server) http.Handler {
	if s.Now == nil {
		s.Now = time.Now
	}
	s.limiter = newLimiter(ReportsPerIP, ReportWindow, s.Now)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /api/topics", s.topics)
	mux.HandleFunc("GET /api/politicians", s.politicians)
	mux.HandleFunc("GET /api/politicians/{id}", s.politician)
	mux.HandleFunc("GET /api/timeline", s.timeline)
	mux.HandleFunc("POST /api/reports", s.report)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found")
	})
	if s.Frontend != nil {
		mux.Handle("/", web.Handler(s.Frontend))
	}
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) topics(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Store.Topics(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}
	if ts == nil {
		ts = []store.TopicSummary{}
	}
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) politicians(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	slug := r.URL.Query().Get("topic")
	if slug != "" {
		if _, ok := topic.Get(slug); !ok {
			writeError(w, http.StatusNotFound, "unknown_topic")
			return
		}
	}
	all, err := s.Store.Politicians(r.Context(), slug)
	if err != nil {
		internalError(w, r, err)
		return
	}
	out := []store.PoliticianSummary{}
	needle := fold(q)
	for _, p := range all {
		if needle == "" || strings.Contains(fold(p.Name), needle) {
			out = append(out, p)
			if len(out) == searchLimit {
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// fold makes name search forgiving: case, umlauts written out, and ß.
func fold(s string) string {
	return strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "é", "e", "è", "e", "á", "a", "ç", "c", "ć", "c", "š", "s").
		Replace(strings.ToLower(s))
}

func (s *Server) politician(w http.ResponseWriter, r *http.Request) {
	d, err := s.Store.Politician(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "unknown_politician")
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// TimelineResponse is the body of GET /api/timeline.
type TimelineResponse struct {
	Politician store.PoliticianDetail `json:"politician"`
	Topic      TopicRef               `json:"topic"`
	Changes    int                    `json:"changes"`
	Entries    []timeline.Entry       `json:"entries"`
}

// TopicRef names a topic.
type TopicRef struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Question string `json:"question"`
}

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	id, slug := r.URL.Query().Get("politician"), r.URL.Query().Get("topic")
	t, ok := topic.Get(slug)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown_topic")
		return
	}
	d, err := s.Store.Politician(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "unknown_politician")
		return
	}
	if err != nil {
		internalError(w, r, err)
		return
	}
	es, err := s.Store.Timeline(r.Context(), id, slug)
	if err != nil {
		internalError(w, r, err)
		return
	}
	n := timeline.MarkChanges(es)
	writeJSON(w, http.StatusOK, TimelineResponse{
		Politician: d,
		Topic:      TopicRef{Slug: t.Slug, Name: t.Name, Question: t.Question},
		Changes:    n,
		Entries:    es,
	})
}

// ReportRequest is the body of POST /api/reports.
type ReportRequest struct {
	ParagraphID int64  `json:"paragraph_id"`
	Topic       string `json:"topic"`
	Message     string `json:"message"`
	Contact     string `json:"contact"`
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow(clientIP(r)) {
		w.Header().Set("Retry-After", strconv.Itoa(int(ReportWindow.Seconds())))
		writeError(w, http.StatusTooManyRequests, "too_many_reports")
		return
	}
	var req ReportRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	req.Message, req.Contact = strings.TrimSpace(req.Message), strings.TrimSpace(req.Contact)
	switch n := utf8.RuneCountInString(req.Message); {
	case n < MinMessage:
		writeError(w, http.StatusUnprocessableEntity, "message_too_short")
		return
	case n > MaxMessage:
		writeError(w, http.StatusUnprocessableEntity, "message_too_long")
		return
	}
	if utf8.RuneCountInString(req.Contact) > MaxContact {
		writeError(w, http.StatusUnprocessableEntity, "contact_too_long")
		return
	}
	if _, ok := topic.Get(req.Topic); !ok || req.ParagraphID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "unknown_entry")
		return
	}
	err := s.Store.SaveReport(r.Context(), store.Report{
		ParagraphID: req.ParagraphID, Topic: req.Topic, Message: req.Message, Contact: req.Contact,
	}, s.Now())
	if err != nil {
		// A foreign-key failure means the paragraph does not exist.
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			writeError(w, http.StatusUnprocessableEntity, "unknown_entry")
			return
		}
		internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError answers with a stable code; the website turns codes into
// German messages.
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal")
}
