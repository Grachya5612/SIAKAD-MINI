package models

type Course struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	KodeMK   string `gorm:"column:kode_mk;size:20;uniqueIndex;not null" json:"kode_mk"`
	NamaMK   string `gorm:"column:nama_mk;size:150;not null" json:"nama_mk"`
	SKS      int    `gorm:"column:sks;not null" json:"sks"`
	Semester int    `gorm:"not null" json:"semester"`
	Kuota    int    `gorm:"not null" json:"kuota"`

	Enrollments []Enrollment `gorm:"foreignKey:CourseID" json:"-"` // 1-N
}
