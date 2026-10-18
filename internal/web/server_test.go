package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Derek-Poe/Smallwinner-Control/internal/control"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func serverForTest(t *testing.T) *Server {
	t.Helper()
	store, err := control.OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	s, err := New(control.NewController(store, control.NewSimulator()), testToken, 1717)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func call(s *Server, method, path, body string, setup func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:1717"+path, strings.NewReader(body))
	r.Header.Set("Origin", "http://127.0.0.1:1717")
	r.Header.Set("Authorization", "Bearer "+testToken)
	r.Header.Set("Content-Type", "application/json")
	if setup != nil {
		setup(r)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestRequestBoundary(t *testing.T) {
	s := serverForTest(t)
	valid := `{"foreground":"demo.game","temperature_c":35}`
	cases := []struct {
		name, method, path, body string
		setup                    func(*http.Request)
		status                   int
	}{
		{"valid", "POST", "/api/v1/simulation", valid, nil, 200},
		{"foreign host", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Host = "attacker.example:1717" }, 403},
		{"hostname alias", "GET", "/api/v1/state", "", func(r *http.Request) { r.Host = "localhost:1717" }, 403},
		{"foreign origin", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Set("Origin", "http://attacker.example") }, 403},
		{"null origin", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Set("Origin", "null") }, 403},
		{"missing origin", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Del("Origin") }, 403},
		{"duplicate origin", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Add("Origin", "http://127.0.0.1:1717") }, 403},
		{"no token", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Del("Authorization") }, 401},
		{"wrong token", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Set("Authorization", "Bearer wrong") }, 401},
		{"duplicate token", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Add("Authorization", "Bearer "+testToken) }, 401},
		{"wrong media", "POST", "/api/v1/simulation", valid, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, 415},
		{"preflight", "OPTIONS", "/api/v1/simulation", "", nil, 405},
		{"shell absent", "POST", "/api/v1/exec", `{"command":"anything"}`, nil, 404},
		{"malformed", "POST", "/api/v1/simulation", `{`, nil, 400},
		{"unknown field", "POST", "/api/v1/simulation", `{"foreground":"demo.game","temperature_c":35,"shell":"x"}`, nil, 400},
		{"trailing JSON", "POST", "/api/v1/simulation", valid + `{}`, nil, 400},
		{"null", "POST", "/api/v1/simulation", `null`, nil, 400},
		{"oversized", "POST", "/api/v1/simulation", `{"foreground":"` + strings.Repeat("x", 17000) + `"}`, nil, 400},
		{"unknown package", "POST", "/api/v1/simulation", `{"foreground":"other","temperature_c":35}`, nil, 400},
		{"temperature bounds", "POST", "/api/v1/simulation", `{"foreground":"demo.game","temperature_c":121}`, nil, 400},
		{"query rejected", "GET", "/api/v1/state?token=bad", "", nil, 400},
		{"restore null", "POST", "/api/v1/actions/restore", `null`, nil, 400},
		{"restore arguments", "POST", "/api/v1/actions/restore", `{"command":"anything"}`, nil, 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := call(s, tc.method, tc.path, tc.body, tc.setup)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("CORS enabled")
			}
			if strings.Contains(w.Body.String(), testToken) {
				t.Fatal("token leaked")
			}
		})
	}
}
func TestConfigurationConflictAndRestoreHTTP(t *testing.T) {
	s := serverForTest(t)
	cfg := s.controller.Config()
	cfg.ThermalLimit = 80
	b, _ := json.Marshal(cfg)
	if w := call(s, "POST", "/api/v1/config", string(b), nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := call(s, "POST", "/api/v1/config", string(b), nil); w.Code != 409 {
		t.Fatal(w.Code)
	}
	call(s, "POST", "/api/v1/simulation", `{"foreground":"demo.game","temperature_c":35}`, nil)
	w := call(s, "POST", "/api/v1/actions/restore", `{}`, nil)
	var st control.State
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if w.Code != 200 || st.Policy != "restored" || st.Device.Resources != st.Baseline {
		t.Fatal(w.Body.String())
	}
}
func TestMutationAdmissionBound(t *testing.T) {
	s := serverForTest(t)
	for i := 0; i < cap(s.slots); i++ {
		s.slots <- struct{}{}
	}
	w := call(s, "POST", "/api/v1/actions/restore", `{}`, nil)
	if w.Code != 429 {
		t.Fatal(w.Code)
	}
}
func TestEmbeddedUIAndReadOnlyNoOrigin(t *testing.T) {
	s := serverForTest(t)
	for _, path := range []string{"/", "/app.js", "/style.css", "/api/v1/state", "/api/v1/config"} {
		w := call(s, "GET", path, "", func(r *http.Request) { r.Header.Del("Origin"); r.Header.Del("Authorization") })
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatal(path, w.Code)
		}
		if w.Header().Get("Content-Security-Policy") == "" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing headers")
		}
		if bytes.Contains(w.Body.Bytes(), []byte(testToken)) {
			t.Fatal("token embedded")
		}
	}
}
