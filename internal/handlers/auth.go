package handlers

import (
	"errors"
	"strings"
	"time"

	"siakad-mini/internal/auth"
	"siakad-mini/internal/config"
	"siakad-mini/internal/middleware"
	"siakad-mini/internal/models"
	"siakad-mini/internal/response"
	"siakad-mini/internal/validation"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db      *gorm.DB
	cfg     *config.Config
	limiter *middleware.LoginLimiter
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db:      db,
		cfg:     cfg,
		limiter: middleware.NewLoginLimiter(5, time.Minute), // maks. 5 gagal / menit
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login  POST /api/v1/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	key := c.IP()

	if h.limiter.Blocked(key) {
		c.Set(fiber.HeaderRetryAfter, "60")
		return response.Fail(c, fiber.StatusTooManyRequests, "Terlalu banyak percobaan login, coba lagi dalam 1 menit")
	}

	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Fail(c, fiber.StatusBadRequest, "Body harus berupa JSON yang valid")
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	errs := validation.New()
	if errs.Required("email", req.Email) && !validation.IsEmail(req.Email) {
		errs.Add("email", "Format email tidak valid")
	}
	if errs.Required("password", req.Password) && !validation.MinLen(req.Password, 8) {
		errs.Add("password", "password minimal 8 karakter")
	}
	if errs.Any() {
		return response.FailValidation(c, errs)
	}

	var user models.User
	err := h.db.Preload("Student").Where("email = ?", req.Email).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // kesalahan database -> 500
	}

	invalid := err != nil ||
		bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil ||
		(user.Role == models.RoleMahasiswa && user.Student == nil) // mahasiswa soft delete tidak bisa login
	if invalid {
		h.limiter.Fail(key)
		return response.Fail(c, fiber.StatusUnauthorized, "Email atau password salah")
	}
	h.limiter.Reset(key)

	ttl := time.Duration(h.cfg.JWTExpiresMinute) * time.Minute
	token, expiresIn, err := auth.GenerateToken(h.cfg.JWTSecret, user.ID, user.Role, ttl)
	if err != nil {
		return err
	}

	return response.OK(c, "Login berhasil", fiber.Map{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// Me  GET /api/v1/auth/me
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	user := middleware.CurrentUser(c)

	data := fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}
	if user.Role == models.RoleMahasiswa && user.Student != nil {
		data["student"] = fiber.Map{
			"nim":      user.Student.NIM,
			"nama":     user.Student.Nama,
			"prodi":    user.Student.Prodi,
			"angkatan": user.Student.Angkatan,
		}
	}
	return response.OK(c, "Profil pengguna berhasil diambil", data)
}
