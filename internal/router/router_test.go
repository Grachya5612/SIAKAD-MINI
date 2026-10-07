package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"siakad-mini/internal/config"
	"siakad-mini/internal/middleware"
	"siakad-mini/internal/response"

	"github.com/gofiber/fiber/v2"
)

var testCfg = &config.Config{AppEnv: "test", JWTSecret: "rahasia", JWTExpiresMinute: 60}

type result struct {
	Resp *http.Response
	Raw  string
	JSON map[string]any
}

func send(t *testing.T, app *fiber.App, method, path, contentType, body string) result {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return result{Resp: resp, Raw: string(raw), JSON: m}
}

func TestUnknownRouteReturns404JSON(t *testing.T) {
	r := send(t, New(testCfg, nil), "GET", "/api/v1/tidak-ada", "", "")
	if r.Resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", r.Resp.StatusCode)
	}
	if r.JSON["success"] != false || r.JSON["message"] != "Endpoint tidak ditemukan" {
		t.Errorf("body tidak sesuai: %s", r.Raw)
	}
}

func TestRequestIDHeaderPresent(t *testing.T) {
	r := send(t, New(testCfg, nil), "GET", "/apa-saja", "", "")
	if r.Resp.Header.Get("X-Request-Id") == "" {
		t.Error("header X-Request-Id harus ada")
	}
}

func TestRequireJSON(t *testing.T) {
	app := newApp(testCfg)
	app.Post("/x", middleware.RequireJSON, func(c *fiber.Ctx) error {
		return response.Created(c, "dibuat", fiber.Map{"id": 1}, "/x/1")
	})
	app.Get("/g", middleware.RequireJSON, func(c *fiber.Ctx) error {
		return response.OK(c, "ok", nil)
	})

	tests := []struct {
		name, method, path, ct string
		want                   int
	}{
		{"POST tanpa Content-Type", "POST", "/x", "", 415},
		{"POST text/plain", "POST", "/x", "text/plain", 415},
		{"POST application/json", "POST", "/x", "application/json", 201},
		{"POST application/json; charset", "POST", "/x", "application/json; charset=utf-8", 201},
		{"GET tidak diperiksa", "GET", "/g", "", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := send(t, app, tc.method, tc.path, tc.ct, `{"a":1}`)
			if r.Resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d (%s)", r.Resp.StatusCode, tc.want, r.Raw)
			}
		})
	}
}

func TestCreatedSetsLocationHeader(t *testing.T) {
	app := newApp(testCfg)
	app.Post("/x", func(c *fiber.Ctx) error {
		return response.Created(c, "dibuat", fiber.Map{"id": 7}, "/api/v1/students/7")
	})
	r := send(t, app, "POST", "/x", "application/json", "{}")
	if r.Resp.StatusCode != 201 || r.Resp.Header.Get("Location") != "/api/v1/students/7" {
		t.Errorf("status=%d Location=%q", r.Resp.StatusCode, r.Resp.Header.Get("Location"))
	}
}

func TestInvalidJSONBodyReturns400(t *testing.T) {
	app := newApp(testCfg)
	app.Post("/x", func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Fail(c, fiber.StatusBadRequest, "Body harus berupa JSON yang valid")
		}
		return response.OK(c, "ok", nil)
	})
	r := send(t, app, "POST", "/x", "application/json", "{bukan json")
	if r.Resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400 (%s)", r.Resp.StatusCode, r.Raw)
	}
}

func TestNoContentHasNoBody(t *testing.T) {
	app := newApp(testCfg)
	app.Delete("/d", func(c *fiber.Ctx) error { return response.NoContent(c) })
	r := send(t, app, "DELETE", "/d", "", "")
	if r.Resp.StatusCode != 204 || r.Raw != "" {
		t.Errorf("status=%d body=%q, want 204 tanpa body", r.Resp.StatusCode, r.Raw)
	}
}

func TestPanicReturnsGeneric500WithoutLeak(t *testing.T) {
	app := newApp(testCfg)
	app.Get("/panic", func(c *fiber.Ctx) error { panic("rahasia-internal-xyz") })
	r := send(t, app, "GET", "/panic", "", "")
	if r.Resp.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", r.Resp.StatusCode)
	}
	if r.JSON["message"] != "Terjadi kesalahan pada server" {
		t.Errorf("message = %v", r.JSON["message"])
	}
	if strings.Contains(r.Raw, "rahasia-internal-xyz") || strings.Contains(r.Raw, "goroutine") {
		t.Errorf("response membocorkan detail internal: %s", r.Raw)
	}
}

func TestEnvelopeShapes(t *testing.T) {
	app := newApp(testCfg)
	app.Get("/list", func(c *fiber.Ctx) error {
		return response.OKList(c, "daftar", []int{}, &response.Meta{CurrentPage: 1, PerPage: 10, Total: 20, LastPage: 2})
	})
	app.Post("/val", func(c *fiber.Ctx) error {
		return response.FailValidation(c, map[string][]string{"nim": {"NIM sudah terdaftar"}})
	})

	r := send(t, app, "GET", "/list", "", "")
	meta, _ := r.JSON["meta"].(map[string]any)
	data, isArr := r.JSON["data"].([]any)
	if r.JSON["success"] != true || !isArr || len(data) != 0 {
		t.Errorf("daftar kosong harus berupa array []: %s", r.Raw)
	}
	if meta["current_page"] != float64(1) || meta["per_page"] != float64(10) ||
		meta["total"] != float64(20) || meta["last_page"] != float64(2) {
		t.Errorf("meta tidak sesuai: %v", meta)
	}

	r = send(t, app, "POST", "/val", "application/json", "{}")
	errs, _ := r.JSON["errors"].(map[string]any)
	nim, _ := errs["nim"].([]any)
	if r.Resp.StatusCode != 422 || r.JSON["message"] != "Validasi gagal" || len(nim) != 1 {
		t.Errorf("response validasi tidak sesuai: %d %s", r.Resp.StatusCode, r.Raw)
	}
}

func TestAuthRoutes(t *testing.T) {
	app := New(testCfg, nil)

	t.Run("me tanpa token -> 401", func(t *testing.T) {
		r := send(t, app, "GET", "/api/v1/auth/me", "", "")
		if r.Resp.StatusCode != 401 || r.JSON["success"] != false {
			t.Errorf("status=%d body=%s", r.Resp.StatusCode, r.Raw)
		}
	})
	t.Run("login tanpa Content-Type -> 415", func(t *testing.T) {
		r := send(t, app, "POST", "/api/v1/auth/login", "", `{"email":"a@b.com","password":"12345678"}`)
		if r.Resp.StatusCode != 415 {
			t.Errorf("status=%d body=%s", r.Resp.StatusCode, r.Raw)
		}
	})
	t.Run("login input kosong -> 422", func(t *testing.T) {
		r := send(t, app, "POST", "/api/v1/auth/login", "application/json", `{}`)
		errs, _ := r.JSON["errors"].(map[string]any)
		if r.Resp.StatusCode != 422 || errs["email"] == nil || errs["password"] == nil {
			t.Errorf("status=%d body=%s", r.Resp.StatusCode, r.Raw)
		}
	})
	t.Run("login JSON rusak -> 400", func(t *testing.T) {
		r := send(t, app, "POST", "/api/v1/auth/login", "application/json", `{rusak`)
		if r.Resp.StatusCode != 400 {
			t.Errorf("status=%d body=%s", r.Resp.StatusCode, r.Raw)
		}
	})
}

func TestAllProtectedRoutesRequireToken(t *testing.T) {
	app := New(testCfg, nil)
	routes := []struct{ method, path string }{
		{"GET", "/api/v1/auth/me"},
		{"GET", "/api/v1/students"},
		{"POST", "/api/v1/students"},
		{"GET", "/api/v1/students/1"},
		{"PUT", "/api/v1/students/1"},
		{"DELETE", "/api/v1/students/1"},
		{"GET", "/api/v1/courses"},
		{"POST", "/api/v1/enrollments"},
		{"DELETE", "/api/v1/enrollments/1"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			// tanpa Content-Type sekalipun: 401 harus lebih dulu daripada 415
			r := send(t, app, rt.method, rt.path, "", "")
			if r.Resp.StatusCode != 401 || r.JSON["success"] != false {
				t.Errorf("status=%d body=%s, want 401", r.Resp.StatusCode, r.Raw)
			}
		})
	}
}
