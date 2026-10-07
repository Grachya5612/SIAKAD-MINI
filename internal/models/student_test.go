package models

import "testing"

func TestBatasSKS(t *testing.T) {
	tests := []struct {
		name string
		ipk  float64
		want int
	}{
		{"IPK maksimum", 4.00, 24},
		{"batas bawah 24 SKS", 3.00, 24},
		{"tepat di bawah 3,00", 2.99, 21},
		{"batas bawah 21 SKS", 2.50, 21},
		{"tepat di bawah 2,50", 2.49, 18},
		{"IPK rendah", 1.00, 18},
		{"IPK nol (mahasiswa baru)", 0, 18},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := BatasSKS(tc.ipk); got != tc.want {
				t.Errorf("BatasSKS(%.2f) = %d, want %d", tc.ipk, got, tc.want)
			}
		})
	}
}
