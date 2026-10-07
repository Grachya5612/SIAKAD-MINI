package database

import (
	"log"

	"siakad-mini/internal/config"
	"siakad-mini/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) *gorm.DB {
	logLevel := logger.Info
	if cfg.AppEnv == "production" {
		logLevel = logger.Error
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:         logger.Default.LogMode(logLevel),
		TranslateError: true,
	})
	if err != nil {
		log.Fatalf("gagal koneksi database: %v", err)
	}
	return db
}

// Migrate membuat/menyesuaikan tabel: users, students, courses, enrollments.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Student{},
		&models.Course{},
		&models.Enrollment{},
	)
}
