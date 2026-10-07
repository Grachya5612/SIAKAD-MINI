package handlers

import (
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestTahunAkademikValid(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"2026/2027-Ganjil", true},
		{"2026/2027-Genap", true},
		{"2025/2026-Ganjil", true},
		{"2026/2028-Ganjil", false}, // selisih tahun salah
		{"2027/2026-Ganjil", false},
		{"2026-2027-Ganjil", false}, // pemisah salah
		{"2026/2027-ganjil", false}, // huruf kecil
		{"2026/2027-Semester", false},
		{"2026/2027", false},
		{"26/27-Ganjil", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := tahunAkademikValid(tc.input); got != tc.want {
				t.Errorf("tahunAkademikValid(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// Validasi berjalan sebelum database disentuh, jadi db boleh nil.
func TestEnrollmentStoreValidation(t *testing.T) {
	app := fiber.New()
	app.Post("/enrollments", NewEnrollmentHandler(nil).Store)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantFields []string
	}{
		{"objek kosong", `{}`, 422, []string{"course_id", "tahun_akademik"}},
		{"format tahun salah", `{"course_id":1,"tahun_akademik":"2026-Ganjil"}`, 422, []string{"tahun_akademik"}},
		{"selisih tahun salah", `{"course_id":1,"tahun_akademik":"2026/2028-Ganjil"}`, 422, []string{"tahun_akademik"}},
		{"course_id hilang", `{"tahun_akademik":"2026/2027-Ganjil"}`, 422, []string{"course_id"}},
		{"JSON rusak", `{rusak`, 400, nil},
		{"course_id negatif (tipe salah)", `{"course_id":-1,"tahun_akademik":"2026/2027-Ganjil"}`, 400, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body, raw := doReq(t, app, "POST", "/enrollments", tc.body)
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

func TestEnrollmentDestroyNonNumericIDReturns400(t *testing.T) {
	app := fiber.New()
	app.Delete("/enrollments/:id", NewEnrollmentHandler(nil).Destroy)
	for _, id := range []string{"abc", "0", "-1", "1.5"} {
		t.Run(id, func(t *testing.T) {
			if status, _, raw := doReq(t, app, "DELETE", "/enrollments/"+id, ""); status != 400 {
				t.Errorf("status = %d, want 400 (%s)", status, raw)
			}
		})
	}
}
