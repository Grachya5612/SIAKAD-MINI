package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"siakad-mini/internal/middleware"
	"siakad-mini/internal/models"
	"siakad-mini/internal/response"
	"siakad-mini/internal/validation"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EnrollmentHandler struct {
	db *gorm.DB
}

func NewEnrollmentHandler(db *gorm.DB) *EnrollmentHandler {
	return &EnrollmentHandler{db: db}
}

type createEnrollmentRequest struct {
	CourseID      uint   `json:"course_id"`
	TahunAkademik string `json:"tahun_akademik"`
}

var tahunAkademikRe = regexp.MustCompile(`^(\d{4})/(\d{4})-(Ganjil|Genap)$`)

// tahunAkademikValid memeriksa format 2026/2027-Ganjil (tahun kedua = tahun pertama + 1).
func tahunAkademikValid(s string) bool {
	m := tahunAkademikRe.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	return b == a+1
}

// enrollFail membatalkan transaction sekaligus membawa info error HTTP.
type enrollFail struct {
	Status  int
	Message string
	Errors  map[string][]string
}

func (e *enrollFail) Error() string { return e.Message }

// Store  POST /api/v1/enrollments  (mahasiswa)
//
// Seluruh pengecekan berjalan dalam satu transaction dengan row locking
// (SELECT ... FOR UPDATE) pada baris mahasiswa lalu baris mata kuliah. Urutan kunci
// selalu sama sehingga tidak terjadi deadlock; dua request bersamaan tidak dapat
// melampaui kuota maupun batas SKS.
func (h *EnrollmentHandler) Store(c *fiber.Ctx) error {
	var req createEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Fail(c, fiber.StatusBadRequest, "Body harus berupa JSON yang valid")
	}

	errs := validation.New()
	if req.CourseID == 0 {
		errs.Add("course_id", "course_id wajib diisi")
	}
	if errs.Required("tahun_akademik", req.TahunAkademik) && !tahunAkademikValid(req.TahunAkademik) {
		errs.Add("tahun_akademik", "Format tahun_akademik harus seperti 2026/2027-Ganjil")
	}
	if errs.Any() {
		return response.FailValidation(c, errs)
	}

	user := middleware.CurrentUser(c)
	if user == nil || user.Student == nil {
		return response.Fail(c, fiber.StatusForbidden, "Hanya mahasiswa yang dapat mengambil mata kuliah")
	}

	var (
		enrollment models.Enrollment
		course     models.Course
		totalSKS   int
		batasSKS   int
	)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var student models.Student
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&student, user.Student.ID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&course, req.CourseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &enrollFail{fiber.StatusUnprocessableEntity, "Validasi gagal",
					map[string][]string{"course_id": {"Mata kuliah tidak ditemukan"}}}
			}
			return err
		}

		// 1) duplikasi -> 409
		var n int64
		if err := tx.Model(&models.Enrollment{}).
			Where("student_id = ? AND course_id = ? AND tahun_akademik = ?", student.ID, course.ID, req.TahunAkademik).
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return &enrollFail{Status: fiber.StatusConflict,
				Message: "Mata kuliah sudah diambil pada tahun akademik yang sama"}
		}

		// 2) kuota -> 422
		if err := tx.Model(&models.Enrollment{}).
			Where("course_id = ? AND tahun_akademik = ?", course.ID, req.TahunAkademik).
			Count(&n).Error; err != nil {
			return err
		}
		if int(n) >= course.Kuota {
			return &enrollFail{Status: fiber.StatusUnprocessableEntity,
				Message: "Kuota mata kuliah sudah penuh"}
		}

		// 3) batas SKS per tahun akademik -> 422
		if err := tx.Model(&models.Enrollment{}).
			Joins("JOIN courses ON courses.id = enrollments.course_id").
			Where("enrollments.student_id = ? AND enrollments.tahun_akademik = ?", student.ID, req.TahunAkademik).
			Select("COALESCE(SUM(courses.sks), 0)").Scan(&totalSKS).Error; err != nil {
			return err
		}
		batasSKS = models.BatasSKS(student.IPKTerakhir)
		if totalSKS+course.SKS > batasSKS {
			sisa := batasSKS - totalSKS
			if sisa < 0 {
				sisa = 0
			}
			return &enrollFail{Status: fiber.StatusUnprocessableEntity,
				Message: fmt.Sprintf("Total SKS melebihi batas. Batas %d SKS, sudah diambil %d SKS, sisa %d SKS (mata kuliah ini %d SKS)",
					batasSKS, totalSKS, sisa, course.SKS)}
		}

		enrollment = models.Enrollment{
			StudentID: student.ID, CourseID: course.ID, TahunAkademik: req.TahunAkademik,
		}
		return tx.Create(&enrollment).Error
	})

	if err != nil {
		var ef *enrollFail
		switch {
		case errors.As(err, &ef):
			return response.FailWithErrors(c, ef.Status, ef.Message, ef.Errors)
		case errors.Is(err, gorm.ErrDuplicatedKey):
			return response.Fail(c, fiber.StatusConflict, "Mata kuliah sudah diambil pada tahun akademik yang sama")
		default:
			return err
		}
	}

	totalSKS += course.SKS
	return response.Created(c, "Mata kuliah berhasil ditambahkan ke KRS", fiber.Map{
		"id":             enrollment.ID,
		"student_id":     enrollment.StudentID,
		"course_id":      enrollment.CourseID,
		"tahun_akademik": enrollment.TahunAkademik,
		"created_at":     enrollment.CreatedAt,
		"course": fiber.Map{
			"kode_mk":  course.KodeMK,
			"nama_mk":  course.NamaMK,
			"sks":      course.SKS,
			"semester": course.Semester,
		},
		"total_sks": totalSKS,
		"batas_sks": batasSKS,
		"sisa_sks":  batasSKS - totalSKS,
	}, fmt.Sprintf("/api/v1/enrollments/%d", enrollment.ID))
}

// Destroy  DELETE /api/v1/enrollments/:id  (mahasiswa, milik sendiri)
func (h *EnrollmentHandler) Destroy(c *fiber.Ctx) error {
	id, ok := parseID(c)
	if !ok {
		return badID(c)
	}

	user := middleware.CurrentUser(c)
	if user == nil || user.Student == nil {
		return response.Fail(c, fiber.StatusForbidden, "Hanya mahasiswa yang dapat membatalkan KRS")
	}

	var enrollment models.Enrollment
	if err := h.db.First(&enrollment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, fiber.StatusNotFound, "Data KRS tidak ditemukan")
		}
		return err
	}

	if enrollment.StudentID != user.Student.ID {
		return response.Fail(c, fiber.StatusForbidden, "Anda tidak dapat membatalkan KRS milik mahasiswa lain")
	}

	// kuota otomatis bertambah karena terisi/sisa_kuota dihitung dari tabel enrollments
	if err := h.db.Delete(&enrollment).Error; err != nil {
		return err
	}
	return response.NoContent(c)
}
