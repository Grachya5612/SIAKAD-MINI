package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"siakad-mini/internal/config"

	"github.com/gofiber/fiber/v2"
)

func loginApp(h *AuthHandler) *fiber.App {
	app := fiber.New()
	app.Post("/login", h.Login)
	return app
}

func postLogin(t *testing.T, app *fiber.App, body string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp.StatusCode, m, string(raw)
}

// Validasi dijalankan sebelum database disentuh, jadi db boleh nil.
func TestLoginValidation(t *testing.T) {
	app := loginApp(NewAuthHandler(nil, &config.Config{JWTSecret: "x", JWTExpiresMinute: 60}))

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantFields []string
	}{
		{"objek kosong", `{}`, 422, []string{"email", "password"}},
		{"email tidak valid", `{"email":"bukan-email","password":"12345678"}`, 422, []string{"email"}},
		{"password terlalu pendek", `{"email":"a@b.com","password":"123"}`, 422, []string{"password"}},
		{"JSON rusak", `{bukan json`, 400, nil},
		{"tipe data salah", `{"email":123,"password":"12345678"}`, 400, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body, raw := postLogin(t, app, tc.body)
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", status, tc.wantStatus, raw)
			}
			if len(tc.wantFields) == 0 {
				return
			}
			errs, _ := body["errors"].(map[string]any)
			for _, f := range tc.wantFields {
				if _, ok := errs[f]; !ok {
					t.Errorf("errors tidak memuat %q: %s", f, raw)
				}
			}
		})
	}
}

func TestLoginRateLimited(t *testing.T) {
	h := NewAuthHandler(nil, &config.Config{JWTSecret: "x", JWTExpiresMinute: 60})
	app := loginApp(h)

	// Cari kunci IP yang dipakai Fiber pada app.Test, lalu isi 5 kegagalan untuk kunci itu.
	var key string
	probe := fiber.New()
	probe.Get("/ip", func(c *fiber.Ctx) error { key = c.IP(); return c.SendStatus(200) })
	if _, err := probe.Test(httptest.NewRequest("GET", "/ip", nil), -1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		h.limiter.Fail(key)
	}

	req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"a@b.com","password":"12345678"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 429 {
		t.Fatalf("status = %d, want 429", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want 60", resp.Header.Get("Retry-After"))
	}
}
