package validation

import "testing"

func TestIsEmail(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"admin@siakad.test", true},
		{"a.b+c@contoh.co.id", true},
		{"bukan-email", false},
		{"@siakad.test", false},
		{"admin@", false},
		{"admin@localhost", false},
		{"Nama <admin@siakad.test>", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := IsEmail(tc.in); got != tc.want {
				t.Errorf("IsEmail(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsDigits(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want bool
	}{
		{"187221000001", 12, true},
		{"18722100000", 12, false},
		{"1872210000011", 12, false},
		{"18722100000a", 12, false},
		{"", 12, false},
	}
	for _, tc := range tests {
		if got := IsDigits(tc.in, tc.n); got != tc.want {
			t.Errorf("IsDigits(%q,%d) = %v, want %v", tc.in, tc.n, got, tc.want)
		}
	}
}

func TestMinLen(t *testing.T) {
	if !MinLen("12345678", 8) || MinLen("1234567", 8) {
		t.Error("MinLen salah pada batas 8 karakter")
	}
	if !MinLen("äöüäöüäö", 8) { // 8 karakter, 16 byte
		t.Error("MinLen harus menghitung karakter, bukan byte")
	}
}

func TestMaxLen(t *testing.T) {
	if !MaxLen("abc", 3) || MaxLen("abcd", 3) {
		t.Error("MaxLen salah pada batas 3 karakter")
	}
	if !MaxLen("äöü", 3) { // 3 karakter, 6 byte
		t.Error("MaxLen harus menghitung karakter, bukan byte")
	}
}

func TestErrors(t *testing.T) {
	e := New()
	if e.Any() {
		t.Fatal("Errors baru tidak boleh berisi apa pun")
	}
	if e.Required("nama", "   ") {
		t.Error("string spasi harus dianggap kosong")
	}
	e.Add("nim", "NIM sudah terdaftar")
	if !e.Any() || len(e["nama"]) != 1 || e["nim"][0] != "NIM sudah terdaftar" {
		t.Errorf("isi Errors tidak sesuai: %v", e)
	}
}
