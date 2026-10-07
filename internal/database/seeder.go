package database

import (
	"fmt"
	"log"

	"siakad-mini/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func hash(plain string) string {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("gagal hash password: %v", err)
	}
	return string(b)
}

// Seed mengisi 1 admin, 20 mahasiswa, dan 10 mata kuliah. Aman dijalankan berulang.
func Seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var n int64

		// --- 1 admin ---
		tx.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&n)
		if n == 0 {
			admin := models.User{
				Email:    "admin@siakad.test",
				Password: hash("admin12345"),
				Role:     models.RoleAdmin,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
		}

		// --- 20 mahasiswa (password awal = NIM) ---
		tx.Model(&models.Student{}).Unscoped().Count(&n)
		if n == 0 {
			nama := []string{
				"Rina Putri", "Budi Santoso", "Siti Aminah", "Andi Pratama", "Dewi Lestari",
				"Rizky Ramadhan", "Nadia Safitri", "Fajar Nugroho", "Intan Permata", "Hendra Wijaya",
				"Maya Anggraini", "Agus Setiawan", "Putri Maharani", "Dimas Prakoso", "Laras Ayu",
				"Yoga Firmansyah", "Citra Kirana", "Bayu Saputra", "Wulan Sari", "Eko Prasetyo",
			}
			prodi := []string{"Sistem Informasi", "Teknik Informatika"}
			// variasi IPK agar ketiga batas SKS (24/21/18) bisa diuji
			ipk := []float64{3.45, 3.80, 2.75, 2.30, 3.10, 2.55, 1.95, 3.60, 2.90, 2.40,
				3.25, 2.70, 3.95, 2.10, 3.00, 2.99, 2.50, 3.50, 2.85, 2.20}

			for i := 0; i < 20; i++ {
				nim := fmt.Sprintf("1872210%05d", i+1) // 12 digit
				u := models.User{
					Email:    fmt.Sprintf("mhs%02d@siakad.test", i+1),
					Password: hash(nim),
					Role:     models.RoleMahasiswa,
				}
				if err := tx.Create(&u).Error; err != nil {
					return err
				}
				s := models.Student{
					UserID:      u.ID,
					NIM:         nim,
					Nama:        nama[i],
					Prodi:       prodi[i%2],
					Angkatan:    2022 + i%3,
					IPKTerakhir: ipk[i],
				}
				if err := tx.Create(&s).Error; err != nil {
					return err
				}
			}
		}

		// --- 10 mata kuliah ---
		tx.Model(&models.Course{}).Count(&n)
		if n == 0 {
			courses := []models.Course{
				{KodeMK: "SI101", NamaMK: "Pemrograman Berbasis Enterprise", SKS: 3, Semester: 5, Kuota: 30},
				{KodeMK: "SI102", NamaMK: "Basis Data Lanjut", SKS: 3, Semester: 3, Kuota: 25},
				{KodeMK: "SI103", NamaMK: "Rekayasa Perangkat Lunak", SKS: 3, Semester: 4, Kuota: 25},
				{KodeMK: "SI104", NamaMK: "Jaringan Komputer", SKS: 3, Semester: 3, Kuota: 20},
				{KodeMK: "SI105", NamaMK: "Sistem Operasi", SKS: 3, Semester: 2, Kuota: 30},
				{KodeMK: "SI106", NamaMK: "Analisis dan Desain Sistem", SKS: 4, Semester: 4, Kuota: 2}, // kuota kecil untuk uji kuota penuh
				{KodeMK: "SI107", NamaMK: "Keamanan Informasi", SKS: 2, Semester: 6, Kuota: 20},
				{KodeMK: "SI108", NamaMK: "Kecerdasan Buatan", SKS: 3, Semester: 6, Kuota: 25},
				{KodeMK: "SI109", NamaMK: "Manajemen Proyek TI", SKS: 2, Semester: 5, Kuota: 20},
				{KodeMK: "SI110", NamaMK: "Pemrograman Web", SKS: 3, Semester: 2, Kuota: 30},
			}
			if err := tx.Create(&courses).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
