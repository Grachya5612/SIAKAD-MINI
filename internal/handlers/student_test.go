package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestAngkatanError(t *testing.T) {
	year := time.Now().Year()
	tests := []struct {
		name     string
		angkatan int
		wantErr  bool
	}{
		{"angkatan lazim", 2022, false},
		{"tahun berjalan", year, false},
		{"tahun depan", year + 1, true},
		{"kurang dari 4 digit", 999, true},
		{"lebih dari 4 digit", 20222, true},
		{"negatif", -2022, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := angkatanError(tc.angkatan) != ""; got != tc.wantErr {
				t.Errorf("angkatanError(%d) error = %v, want %v", tc.angkatan, got, tc.wantErr)
			}
		})
	}
}

func doReq(t *testing.T, app *fiber.App, method, path, body string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
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

// Semua kasus di bawah ditolak sebelum database disentuh, jadi db boleh nil.
func TestStudentStoreValidation(t *testing.T) {
	h := NewStudentHandler(nil)
	app := fiber.New()
	app.Post("/students", h.Store)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantFields []string
	}{
		{"objek kosong", `{}`, 422, []string{"nim", "nama", "email", "prodi", "angkatan"}},
		{"nim bukan 12 digit & email salah", `{"nim":"123","nama":"A","email":"x","prodi":"SI","angkatan":2023}`, 422, []string{"nim", "email"}},
		{"nim mengandung huruf", `{"nim":"18722100000a","nama":"A","email":"x","prodi":"SI","angkatan":2023}`, 422, []string{"nim"}},
		{"angkatan masa depan & ipk di luar rentang", `{"nim":"1","nama":"A","email":"x","prodi":"SI","angkatan":9999,"ipk_terakhir":4.5}`, 422, []string{"angkatan", "ipk_terakhir"}},
		{"JSON rusak", `{rusak`, 400, nil},
		{"tipe data salah (nim angka)", `{"nim":187221000001}`, 400, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body, raw := doReq(t, app, "POST", "/students", tc.body)
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", status, tc.wantStatus, raw)
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

func TestStudentIndexRejectsBadQuery(t *testing.T) {
	app := fiber.New()
	app.Get("/students", NewStudentHandler(nil).Index)

	for _, path := range []string{"/students?sort=ngawur", "/students?angkatan=abc", "/students?sort=id&angkatan=x"} {
		t.Run(path, func(t *testing.T) {
			status, body, raw := doReq(t, app, "GET", path, "")
			if status != 422 || body["errors"] == nil {
				t.Errorf("status=%d body=%s, want 422 dengan errors", status, raw)
			}
		})
	}
}

func TestStudentNonNumericIDReturns400(t *testing.T) {
	h := NewStudentHandler(nil)
	app := fiber.New()
	app.Get("/students/:id", h.Show)
	app.Put("/students/:id", h.Update)
	app.Delete("/students/:id", h.Destroy)

	for _, id := range []string{"abc", "0", "-5", "1.5"} {
		for _, m := range []string{"GET", "PUT", "DELETE"} {
			t.Run(m+" "+id, func(t *testing.T) {
				status, _, raw := doReq(t, app, m, "/students/"+id, `{}`)
				if status != 400 {
					t.Errorf("status = %d, want 400 (%s)", status, raw)
				}
			})
		}
	}
}

func TestCourseIndexRejectsBadSemester(t *testing.T) {
	app := fiber.New()
	app.Get("/courses", NewCourseHandler(nil).Index)
	status, body, raw := doReq(t, app, "GET", "/courses?semester=abc", "")
	if status != 422 || body["errors"] == nil {
		t.Errorf("status=%d body=%s, want 422", status, raw)
	}
}
