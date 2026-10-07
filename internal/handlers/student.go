package handlers

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"siakad-mini/internal/middleware"
	"siakad-mini/internal/models"
	"siakad-mini/internal/response"
	"siakad-mini/internal/validation"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type StudentHandler struct {
	db *gorm.DB
}

func NewStudentHandler(db *gorm.DB) *StudentHandler {
	return &StudentHandler{db: db}
}

// ---------- DTO ----------

type studentResponse struct {
	ID          uint    `json:"id"`
	NIM         string  `json:"nim"`
	Nama        string  `json:"nama"`
	Prodi       string  `json:"prodi"`
	Angkatan    int     `json:"angkatan"`
	IPKTerakhir float64 `json:"ipk_terakhir"`
	Email       string  `json:"email,omitempty"`
}

func toStudentResponse(s models.Student) studentResponse {
	return studentResponse{
		ID: s.ID, NIM: s.NIM, Nama: s.Nama, Prodi: s.Prodi,
		Angkatan: s.Angkatan, IPKTerakhir: s.IPKTerakhir,
	}
}

type courseTakenResponse struct {
	EnrollmentID  uint   `json:"enrollment_id"`
	CourseID      uint   `json:"course_id"`
	KodeMK        string `json:"kode_mk"`
	NamaMK        string `json:"nama_mk"`
	SKS           int    `json:"sks"`
	Semester      int    `json:"semester"`
	TahunAkademik string `json:"tahun_akademik"`
}

type studentDetailResponse struct {
	studentResponse
	MataKuliah []courseTakenResponse `json:"mata_kuliah"`
	TotalSKS   int                   `json:"total_sks"`
	BatasSKS   int                   `json:"batas_sks"`
}

// ---------- request ----------

type createStudentRequest struct {
	NIM         string   `json:"nim"`
	Nama        string   `json:"nama"`
	Email       string   `json:"email"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
}

type updateStudentRequest struct {
	NIM         *string  `json:"nim"` // tidak boleh diubah
	Nama        string   `json:"nama"`
	Prodi       string   `json:"prodi"`
	Angkatan    int      `json:"angkatan"`
	IPKTerakhir *float64 `json:"ipk_terakhir"`
}

func angkatanError(a int) string {
	if a < 1000 || a > 9999 || a > time.Now().Year() {
		return "Angkatan harus 4 digit dan tidak boleh lebih dari tahun berjalan"
	}
	return ""
}

// validateProfile memeriksa field yang sama pada POST dan PUT.
func validateProfile(errs validation.Errors, nama, prodi string, angkatan int, ipk *float64) {
	if errs.Required("nama", nama) && !validation.MaxLen(nama, 150) {
		errs.Add("nama", "nama maksimal 150 karakter")
	}
	if errs.Required("prodi", prodi) && !validation.MaxLen(prodi, 100) {
		errs.Add("prodi", "prodi maksimal 100 karakter")
	}
	if angkatan == 0 {
		errs.Add("angkatan", "angkatan wajib diisi")
	} else if msg := angkatanError(angkatan); msg != "" {
		errs.Add("angkatan", msg)
	}
	if ipk != nil && (*ipk < 0 || *ipk > 4) {
		errs.Add("ipk_terakhir", "ipk_terakhir harus antara 0 dan 4")
	}
}

// parseID membaca parameter :id. Nilai non-angka atau <= 0 adalah kesalahan klien (400).
func parseID(c *fiber.Ctx) (uint, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

func badID(c *fiber.Ctx) error {
	return response.Fail(c, fiber.StatusBadRequest, "ID harus berupa angka positif")
}

func notFoundStudent(c *fiber.Ctx) error {
	return response.Fail(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan")
}

// ---------- 3. GET /students ----------

func (h *StudentHandler) Index(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(c.Query("per_page", "10"))
	if err != nil || perPage < 1 {
		perPage = 10
	}
	if perPage > 50 {
		perPage = 50
	}

	// validasi parameter query dilakukan sebelum menyentuh database
	errs := validation.New()
	var angkatan int
	hasAngkatan := false
	if raw := c.Query("angkatan"); raw != "" {
		a, convErr := strconv.Atoi(raw)
		if convErr != nil {
			errs.Add("angkatan", "Angkatan harus berupa angka")
		} else {
			angkatan, hasAngkatan = a, true
		}
	}
	order := "id ASC"
	switch c.Query("sort") {
	case "":
	case "nama":
		order = "nama ASC, id ASC"
	case "-ipk_terakhir":
		order = "ipk_terakhir DESC, id ASC"
	default:
		errs.Add("sort", "Sort hanya boleh 'nama' atau '-ipk_terakhir'")
	}
	if errs.Any() {
		return response.FailValidation(c, errs)
	}

	q := h.db.Model(&models.Student{}) // soft delete otomatis tersaring
	if prodi := strings.TrimSpace(c.Query("prodi")); prodi != "" {
		q = q.Where("prodi ILIKE ?", prodi)
	}
	if hasAngkatan {
		q = q.Where("angkatan = ?", angkatan)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		q = q.Where("(nim ILIKE ? OR nama ILIKE ?)", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return err
	}

	var students []models.Student
	if err := q.Order(order).Offset((page - 1) * perPage).Limit(perPage).Find(&students).Error; err != nil {
		return err
	}

	items := make([]studentResponse, 0, len(students))
	for _, s := range students {
		items = append(items, toStudentResponse(s))
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage < 1 {
		lastPage = 1
	}

	return response.OKList(c, "Data mahasiswa berhasil diambil", items, &response.Meta{
		CurrentPage: page, PerPage: perPage, Total: int(total), LastPage: lastPage,
	})
}

// ---------- 4. POST /students ----------

func (h *StudentHandler) Store(c *fiber.Ctx) error {
	var req createStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Fail(c, fiber.StatusBadRequest, "Body harus berupa JSON yang valid")
	}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Nama = strings.TrimSpace(req.Nama)
	req.Prodi = strings.TrimSpace(req.Prodi)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	errs := validation.New()
	nimOK := errs.Required("nim", req.NIM)
	if nimOK && !validation.IsDigits(req.NIM, 12) {
		errs.Add("nim", "nim harus 12 digit angka")
		nimOK = false
	}
	emailOK := errs.Required("email", req.Email)
	if emailOK && (!validation.IsEmail(req.Email) || !validation.MaxLen(req.Email, 150)) {
		errs.Add("email", "Format email tidak valid")
		emailOK = false
	}
	validateProfile(errs, req.Nama, req.Prodi, req.Angkatan, req.IPKTerakhir)

	// keunikan hanya diperiksa bila format field sudah benar
	var n int64
	if nimOK {
		// Unscoped: NIM milik mahasiswa yang di-soft delete tetap terpakai (unique index DB)
		if err := h.db.Unscoped().Model(&models.Student{}).Where("nim = ?", req.NIM).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			errs.Add("nim", "NIM sudah terdaftar")
		}
	}
	if emailOK {
		if err := h.db.Model(&models.User{}).Where("email = ?", req.Email).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			errs.Add("email", "Email sudah terdaftar")
		}
	}
	if errs.Any() {
		return response.FailValidation(c, errs)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NIM), bcrypt.DefaultCost) // password awal = NIM
	if err != nil {
		return err
	}

	student := models.Student{NIM: req.NIM, Nama: req.Nama, Prodi: req.Prodi, Angkatan: req.Angkatan}
	if req.IPKTerakhir != nil {
		student.IPKTerakhir = *req.IPKTerakhir
	}

	// users + students dalam satu transaction
	err = h.db.Transaction(func(tx *gorm.DB) error {
		user := models.User{Email: req.Email, Password: string(hashed), Role: models.RoleMahasiswa}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		student.UserID = user.ID
		return tx.Create(&student).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) { // balapan antar request
			return response.FailValidation(c, validation.Errors{"nim": {"NIM atau email sudah terdaftar"}})
		}
		return err
	}

	data := toStudentResponse(student)
	data.Email = req.Email
	return response.Created(c, "Mahasiswa berhasil ditambahkan", data, fmt.Sprintf("/api/v1/students/%d", student.ID))
}

// ---------- 5. GET /students/:id ----------

func (h *StudentHandler) Show(c *fiber.Ctx) error {
	id, ok := parseID(c)
	if !ok {
		return badID(c)
	}

	user := middleware.CurrentUser(c)
	if user == nil {
		return response.Fail(c, fiber.StatusUnauthorized, "Tidak terautentikasi")
	}
	// mahasiswa hanya boleh melihat data dirinya sendiri; dicek sebelum 404
	// agar mahasiswa tidak dapat mengetahui ada/tidaknya data orang lain
	if user.Role == models.RoleMahasiswa && (user.Student == nil || user.Student.ID != id) {
		return response.Fail(c, fiber.StatusForbidden, "Anda hanya dapat mengakses data diri sendiri")
	}

	tahun := c.Query("tahun_akademik") // filter opsional
	q := h.db.Preload("Enrollments", func(db *gorm.DB) *gorm.DB {
		if tahun != "" {
			db = db.Where("tahun_akademik = ?", tahun)
		}
		return db.Order("id ASC")
	}).Preload("Enrollments.Course")

	var student models.Student
	if err := q.First(&student, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFoundStudent(c)
		}
		return err
	}

	courses := make([]courseTakenResponse, 0, len(student.Enrollments))
	total := 0
	for _, e := range student.Enrollments {
		if e.Course == nil {
			continue
		}
		total += e.Course.SKS
		courses = append(courses, courseTakenResponse{
			EnrollmentID: e.ID, CourseID: e.CourseID, KodeMK: e.Course.KodeMK,
			NamaMK: e.Course.NamaMK, SKS: e.Course.SKS, Semester: e.Course.Semester,
			TahunAkademik: e.TahunAkademik,
		})
	}

	return response.OK(c, "Detail mahasiswa berhasil diambil", studentDetailResponse{
		studentResponse: toStudentResponse(student),
		MataKuliah:      courses,
		TotalSKS:        total,
		BatasSKS:        models.BatasSKS(student.IPKTerakhir),
	})
}

// ---------- 6. PUT /students/:id ----------

// Update menerapkan semantik PUT: representasi diganti seluruhnya.
// nama, prodi, angkatan wajib; ipk_terakhir yang tidak dikirim di-reset ke 0
// (sama dengan nilai bawaan saat POST). nim tidak boleh diubah.
func (h *StudentHandler) Update(c *fiber.Ctx) error {
	id, ok := parseID(c)
	if !ok {
		return badID(c)
	}

	var student models.Student
	if err := h.db.First(&student, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFoundStudent(c)
		}
		return err
	}

	var req updateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Fail(c, fiber.StatusBadRequest, "Body harus berupa JSON yang valid")
	}
	req.Nama = strings.TrimSpace(req.Nama)
	req.Prodi = strings.TrimSpace(req.Prodi)

	errs := validation.New()
	validateProfile(errs, req.Nama, req.Prodi, req.Angkatan, req.IPKTerakhir)
	if req.NIM != nil && *req.NIM != student.NIM {
		errs.Add("nim", "NIM tidak dapat diubah")
	}
	if errs.Any() {
		return response.FailValidation(c, errs)
	}

	student.Nama = req.Nama
	student.Prodi = req.Prodi
	student.Angkatan = req.Angkatan
	student.IPKTerakhir = 0
	if req.IPKTerakhir != nil {
		student.IPKTerakhir = *req.IPKTerakhir
	}
	if err := h.db.Save(&student).Error; err != nil {
		return err
	}

	return response.OK(c, "Data mahasiswa berhasil diperbarui", toStudentResponse(student))
}

// ---------- 7. DELETE /students/:id ----------

func (h *StudentHandler) Destroy(c *fiber.Ctx) error {
	id, ok := parseID(c)
	if !ok {
		return badID(c)
	}

	var student models.Student
	if err := h.db.First(&student, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFoundStudent(c)
		}
		return err
	}

	// soft delete: mengisi deleted_at (model Student memakai gorm.DeletedAt)
	if err := h.db.Delete(&student).Error; err != nil {
		return err
	}
	return response.NoContent(c)
}
