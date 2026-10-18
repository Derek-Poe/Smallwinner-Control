package web

import (
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/Derek-Poe/Smallwinner-Control/internal/control"
)

//go:embed assets/*
var assets embed.FS

type Server struct {
	controller *control.Controller
	token      string
	host       string
	origin     string
	slots      chan struct{}
}

func New(controller *control.Controller, token string, port int) (*Server, error) {
	b, err := hex.DecodeString(token)
	if err != nil || len(b) != 32 {
		return nil, errors.New("token must be 32 bytes encoded as hexadecimal")
	}
	if port < 1 || port > 65535 {
		return nil, errors.New("invalid port")
	}
	host := "127.0.0.1:" + strconv.Itoa(port)
	return &Server{controller: controller, token: token, host: host, origin: "http://" + host, slots: make(chan struct{}, 4)}, nil
}
func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	if r.Host != s.host {
		fail(w, 403, "unrecognized host")
		return
	}
	origins := r.Header.Values("Origin")
	if len(origins) > 1 || len(origins) == 1 && origins[0] != s.origin {
		fail(w, 403, "unrecognized origin")
		return
	}
	if r.URL.RawQuery != "" {
		fail(w, 400, "query parameters are not supported")
		return
	}
	if r.Method == http.MethodGet {
		switch r.URL.Path {
		case "/api/v1/state":
			reply(w, 200, s.controller.Snapshot())
		case "/api/v1/config":
			reply(w, 200, s.controller.Config())
		case "/", "/app.js", "/style.css":
			file := strings.TrimPrefix(r.URL.Path, "/")
			if file == "" {
				file = "index.html"
			}
			b, err := assets.ReadFile("assets/" + file)
			if err != nil {
				fail(w, 500, "asset unavailable")
				return
			}
			kind := "text/html; charset=utf-8"
			if file == "app.js" {
				kind = "text/javascript; charset=utf-8"
			}
			if file == "style.css" {
				kind = "text/css; charset=utf-8"
			}
			w.Header().Set("Content-Type", kind)
			_, _ = w.Write(b)
		default:
			fail(w, 404, "route not found")
		}
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		fail(w, 405, "method not allowed")
		return
	}
	if r.URL.Path != "/api/v1/config" && r.URL.Path != "/api/v1/simulation" && r.URL.Path != "/api/v1/actions/restore" {
		fail(w, 404, "route not found")
		return
	}
	if len(origins) != 1 {
		fail(w, 403, "mutation requires same origin")
		return
	}
	auth := r.Header.Values("Authorization")
	if len(auth) != 1 || subtle.ConstantTimeCompare([]byte(auth[0]), []byte("Bearer "+s.token)) != 1 {
		fail(w, 401, "mutation requires session token")
		return
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		fail(w, 415, "JSON required")
		return
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		fail(w, 429, "mutation capacity reached")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var state control.State
	switch r.URL.Path {
	case "/api/v1/config":
		var cfg control.Config
		if err = control.DecodeJSON(r.Body, &cfg); err != nil {
			fail(w, 400, "invalid or oversized JSON")
			return
		}
		if err = cfg.Validate(); err != nil {
			fail(w, 400, err.Error())
			return
		}
		state, err = s.controller.Update(r.Context(), cfg)
	case "/api/v1/simulation":
		var o control.Observation
		if err = control.DecodeJSON(r.Body, &o); err != nil {
			fail(w, 400, "invalid or oversized JSON")
			return
		}
		if err = o.Validate(); err != nil {
			fail(w, 400, err.Error())
			return
		}
		state, err = s.controller.Observe(r.Context(), o)
	case "/api/v1/actions/restore":
		var empty map[string]json.RawMessage
		if err = control.DecodeJSON(r.Body, &empty); err != nil || empty == nil || len(empty) != 0 {
			fail(w, 400, "expected empty object")
			return
		}
		state, err = s.controller.Restore(r.Context())
	}
	if errors.Is(err, control.ErrConflict) {
		fail(w, 409, "revision conflict; reload configuration")
		return
	}
	if err != nil {
		reply(w, 503, map[string]any{"error": "operation failed; read current state and configuration before retrying", "state": state})
		return
	}
	reply(w, 200, state)
}
func (s *Server) URL() string { return fmt.Sprintf("http://%s/", s.host) }
