package router

import (
	"errors"

	"siakad-mini/internal/config"
	"siakad-mini/internal/handlers"
	"siakad-mini/internal/middleware"
	"siakad-mini/internal/models"
	"siakad-mini/internal/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"gorm.io/gorm"
)

// errorHandler mengubah semua error menjadi amplop JSON baku.
// Error yang bukan *fiber.Error (termasuk panic) dijawab 500 dengan pesan umum,
// sehingga stack trace dan detail internal tidak pernah bocor ke klien.
func errorHandler(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "Terjadi kesalahan pada server"

	var e *fiber.Error
	if errors.As(err, &e) && e.Code < fiber.StatusInternalServerError {
		status = e.Code
		message = e.Message
		switch status {
		case fiber.StatusNotFound:
			message = "Endpoint tidak ditemukan"
		case fiber.StatusMethodNotAllowed:
			message = "Metode tidak diizinkan untuk endpoint ini"
		}
	}
	return response.Fail(c, status, message)
}

func notFound(c *fiber.Ctx) error {
	return response.Fail(c, fiber.StatusNotFound, "Endpoint tidak ditemukan")
}

// newApp membuat aplikasi Fiber beserta middleware global.
// Urutan pemasangan menentukan urutan eksekusi.
func newApp(_ *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "SIAKAD Mini API",
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())   // panic -> error -> errorHandler (500 tanpa stack trace)
	app.Use(requestid.New()) // header X-Request-Id
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${method} ${path} ${status} ${latency}\n",
	}))
	app.Use(cors.New())
	return app
}

// registerRoutes mendaftarkan endpoint. Diisi bertahap:
//
//	Tahap 2: auth (1-2)
//	Tahap 3: students (3-7) dan courses (8)
//	Tahap 4: enrollments (9-10)
func registerRoutes(app *fiber.App, cfg *config.Config, db *gorm.DB) {
	api := app.Group("/api/v1")

	authHandler := handlers.NewAuthHandler(db, cfg)
	studentHandler := handlers.NewStudentHandler(db)
	courseHandler := handlers.NewCourseHandler(db)
	enrollmentHandler := handlers.NewEnrollmentHandler(db)
	authRequired := middleware.AuthRequired(db, cfg.JWTSecret)
	adminOnly := middleware.RequireRole(models.RoleAdmin)
	mahasiswaOnly := middleware.RequireRole(models.RoleMahasiswa)

	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/login", authHandler.Login)        // 1
	auth.Get("/me", authRequired, authHandler.Me) // 2

	// autentikasi dicek lebih dulu, baru Content-Type (token tanpa/salah -> 401 sebelum 415)
	students := api.Group("/students", authRequired, middleware.RequireJSON)
	students.Get("/", adminOnly, studentHandler.Index)         // 3
	students.Post("/", adminOnly, studentHandler.Store)        // 4
	students.Get("/:id", studentHandler.Show)                  // 5 (admin / mahasiswa milik sendiri)
	students.Put("/:id", adminOnly, studentHandler.Update)     // 6
	students.Delete("/:id", adminOnly, studentHandler.Destroy) // 7

	api.Get("/courses", authRequired, courseHandler.Index) // 8

	// token -> peran -> Content-Type
	enrollments := api.Group("/enrollments", authRequired, mahasiswaOnly, middleware.RequireJSON)
	enrollments.Post("/", enrollmentHandler.Store)        // 9
	enrollments.Delete("/:id", enrollmentHandler.Destroy) // 10
}

func New(cfg *config.Config, db *gorm.DB) *fiber.App {
	app := newApp(cfg)
	registerRoutes(app, cfg, db)
	app.Use(notFound) // harus paling akhir
	return app
}
