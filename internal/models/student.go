package models

import (
	"time"

	"gorm.io/gorm"
)

type Student struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	NIM         string         `gorm:"column:nim;size:12;uniqueIndex;not null" json:"nim"`
	Nama        string         `gorm:"size:150;not null" json:"nama"`
	Prodi       string         `gorm:"size:100;not null" json:"prodi"`
	Angkatan    int            `gorm:"not null" json:"angkatan"`
	IPKTerakhir float64        `gorm:"column:ipk_terakhir;type:numeric(3,2);default:0" json:"ipk_terakhir"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // soft delete

	User        *User        `gorm:"foreignKey:UserID" json:"-"`
	Enrollments []Enrollment `gorm:"foreignKey:StudentID" json:"-"` // 1-N
}

// BatasSKS menerapkan business rule batas SKS berdasarkan IPK terakhir.
func BatasSKS(ipk float64) int {
	switch {
	case ipk >= 3.00:
		return 24
	case ipk >= 2.50:
		return 21
	default:
		return 18
	}
}
