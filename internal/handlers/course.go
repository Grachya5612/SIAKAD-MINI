package handlers

import (
	"strconv"
	"strings"

	"siakad-mini/internal/response"
	"siakad-mini/internal/validation"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CourseHandler struct {
	db *gorm.DB
}

func NewCourseHandler(db *gorm.DB) *CourseHandler {
	return &CourseHandler{db: db}
}

type courseItem struct {
	ID        uint   `json:"id" gorm:"column:id"`
	KodeMK    string `json:"kode_mk" gorm:"column:kode_mk"`
	NamaMK    string `json:"nama_mk" gorm:"column:nama_mk"`
	SKS       int    `json:"sks" gorm:"column:sks"`
	Semester  int    `json:"semester" gorm:"column:semester"`
	Kuota     int    `json:"kuota" gorm:"column:kuota"`
	Terisi    int    `json:"terisi" gorm:"column:terisi"`
	SisaKuota int    `json:"sisa_kuota" gorm:"-"`
}

// Index  GET /api/v1/courses
// Query: semester, search (kode_mk / nama_mk), available=true,
// tahun_akademik (opsional: menghitung terisi hanya pada tahun akademik tsb).
func (h *CourseHandler) Index(c *fiber.Ctx) error {
	// validasi parameter query sebelum menyentuh database
	errs := validation.New()
	var semester int
	hasSemester := false
	if raw := c.Query("semester"); raw != "" {
		s, err := strconv.Atoi(raw)
		if err != nil {
			errs.Add("semester", "Semester harus berupa angka")
		} else {
			semester, hasSemester = s, true
		}
	}
	if errs.Any() {
		return response.FailValidation(c, errs)
	}

	join := "LEFT JOIN enrollments ON enrollments.course_id = courses.id"
	var joinArgs []any
	if ta := strings.TrimSpace(c.Query("tahun_akademik")); ta != "" {
		join += " AND enrollments.tahun_akademik = ?"
		joinArgs = append(joinArgs, ta)
	}

	q := h.db.Table("courses").
		Select("courses.id, courses.kode_mk, courses.nama_mk, courses.sks, courses.semester, courses.kuota, COUNT(enrollments.id) AS terisi").
		Joins(join, joinArgs...).
		Group("courses.id")

	if hasSemester {
		q = q.Where("courses.semester = ?", semester)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		q = q.Where("(courses.kode_mk ILIKE ? OR courses.nama_mk ILIKE ?)", like, like)
	}
	if c.Query("available") == "true" {
		q = q.Having("COUNT(enrollments.id) < courses.kuota")
	}

	items := []courseItem{}
	if err := q.Order("courses.kode_mk ASC").Scan(&items).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].SisaKuota = items[i].Kuota - items[i].Terisi
		if items[i].SisaKuota < 0 {
			items[i].SisaKuota = 0
		}
	}

	return response.OK(c, "Data mata kuliah berhasil diambil", items)
}
