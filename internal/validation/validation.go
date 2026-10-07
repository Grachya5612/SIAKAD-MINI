package validation

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Errors menampung pesan validasi per field: {"nim": ["NIM sudah terdaftar"]}.
type Errors map[string][]string

func New() Errors { return Errors{} }

func (e Errors) Add(field, message string) { e[field] = append(e[field], message) }
func (e Errors) Any() bool                 { return len(e) > 0 }

// Required menambah error bila value kosong (setelah di-trim). Mengembalikan true jika terisi.
func (e Errors) Required(field, value string) bool {
	if strings.TrimSpace(value) == "" {
		e.Add(field, fmt.Sprintf("%s wajib diisi", field))
		return false
	}
	return true
}

// IsEmail memeriksa format email sederhana: alamat valid dan domain memuat titik.
func IsEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}

// IsDigits memeriksa s terdiri dari tepat n digit angka.
func IsDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// MinLen memeriksa panjang minimal dalam karakter (bukan byte).
func MinLen(s string, n int) bool { return utf8.RuneCountInString(s) >= n }

// MaxLen memeriksa panjang maksimal dalam karakter (bukan byte).
func MaxLen(s string, n int) bool { return utf8.RuneCountInString(s) <= n }
