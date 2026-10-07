package middleware

import (
	"errors"
	"strings"

	"siakad-mini/internal/auth"
	"siakad-mini/internal/models"
	"siakad-mini/internal/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// LocalUser adalah kunci c.Locals untuk pengguna yang sedang login.
const LocalUser = "current_user"

// AuthRequired memvalidasi header "Authorization: Bearer <token>", memuat pengguna
// (beserta data mahasiswa) dari database, lalu menyimpannya di c.Locals.
func AuthRequired(db *gorm.DB, secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		parts := strings.SplitN(c.Get(fiber.HeaderAuthorization), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return response.Fail(c, fiber.StatusUnauthorized, "Token tidak ditemukan")
		}

		claims, err := auth.ParseToken(secret, strings.TrimSpace(parts[1]))
		if err != nil {
			return response.Fail(c, fiber.StatusUnauthorized, "Token tidak valid atau sudah kedaluwarsa")
		}

		var user models.User
		if err := db.Preload("Student").First(&user, claims.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.Fail(c, fiber.StatusUnauthorized, "Pengguna tidak ditemukan")
			}
			return err // kesalahan database -> 500 lewat errorHandler
		}
		// mahasiswa yang sudah di-soft delete tidak boleh mengakses API
		if user.Role == models.RoleMahasiswa && user.Student == nil {
			return response.Fail(c, fiber.StatusUnauthorized, "Akun tidak aktif")
		}

		c.Locals(LocalUser, &user)
		return c.Next()
	}
}

// RequireRole membatasi akses endpoint ke peran tertentu (403 bila tidak cocok).
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := CurrentUser(c)
		if user == nil {
			return response.Fail(c, fiber.StatusUnauthorized, "Tidak terautentikasi")
		}
		for _, r := range roles {
			if user.Role == r {
				return c.Next()
			}
		}
		return response.Fail(c, fiber.StatusForbidden, "Anda tidak memiliki akses ke resource ini")
	}
}

// CurrentUser mengambil pengguna yang sudah diautentikasi dari c.Locals.
func CurrentUser(c *fiber.Ctx) *models.User {
	u, _ := c.Locals(LocalUser).(*models.User)
	return u
}
