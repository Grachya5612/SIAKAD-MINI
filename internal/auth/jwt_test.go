package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, expiresIn, err := GenerateToken("rahasia", 7, "admin", time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if expiresIn != 60 {
		t.Errorf("expiresIn = %d, want 60", expiresIn)
	}

	claims, err := ParseToken("rahasia", token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != 7 || claims.Role != "admin" {
		t.Errorf("claims = %+v, want user 7 role admin", claims)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	valid, _, _ := GenerateToken("rahasia", 1, "mahasiswa", time.Minute)
	expired, _, _ := GenerateToken("rahasia", 1, "mahasiswa", -time.Minute)

	tests := []struct {
		name   string
		secret string
		token  string
	}{
		{"secret berbeda", "salah", valid},
		{"token kedaluwarsa", "rahasia", expired},
		{"token acak", "rahasia", "bukan-token"},
		{"token kosong", "rahasia", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseToken(tc.secret, tc.token); err == nil {
				t.Error("seharusnya error, tapi berhasil")
			}
		})
	}
}
