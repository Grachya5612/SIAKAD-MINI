package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"siakad-mini/internal/auth"
	"siakad-mini/internal/models"

	"github.com/gofiber/fiber/v2"
)

func roleApp(user *models.User, roles ...string) *fiber.App {
	app := fiber.New()
	app.Get("/x",
		func(c *fiber.Ctx) error {
			if user != nil {
				c.Locals(LocalUser, user)
			}
			return c.Next()
		},
		RequireRole(roles...),
		func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
	)
	return app
}

func TestRequireRole(t *testing.T) {
	admin := &models.User{Role: models.RoleAdmin}
	mhs := &models.User{Role: models.RoleMahasiswa}

	tests := []struct {
		name  string
		user  *models.User
		roles []string
		want  int
	}{
		{"admin ke endpoint admin", admin, []string{models.RoleAdmin}, 200},
		{"mahasiswa ke endpoint admin", mhs, []string{models.RoleAdmin}, 403},
		{"admin ke endpoint mahasiswa", admin, []string{models.RoleMahasiswa}, 403},
		{"mahasiswa ke endpoint mahasiswa", mhs, []string{models.RoleMahasiswa}, 200},
		{"tanpa pengguna di context", nil, []string{models.RoleAdmin}, 401},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := roleApp(tc.user, tc.roles...).Test(httptest.NewRequest("GET", "/x", nil), -1)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

// Kasus berikut ditolak sebelum database disentuh, jadi db boleh nil.
func TestAuthRequiredRejectsBadTokens(t *testing.T) {
	expired, _, _ := auth.GenerateToken("rahasia", 1, models.RoleAdmin, -time.Minute)
	wrongSecret, _, _ := auth.GenerateToken("lain", 1, models.RoleAdmin, time.Minute)

	tests := []struct{ name, header string }{
		{"tanpa header", ""},
		{"skema bukan Bearer", "Basic abc"},
		{"token kosong", "Bearer "},
		{"token acak", "Bearer abc.def.ghi"},
		{"token kedaluwarsa", "Bearer " + expired},
		{"secret salah", "Bearer " + wrongSecret},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/x", AuthRequired(nil, "rahasia"), func(c *fiber.Ctx) error { return c.SendStatus(200) })

			req := httptest.NewRequest("GET", "/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 401 {
				t.Errorf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
}
