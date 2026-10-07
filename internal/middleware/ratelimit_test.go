package middleware

import (
	"testing"
	"time"
)

func TestLoginLimiterBlocksAfterMaxFailures(t *testing.T) {
	l := NewLoginLimiter(5, time.Minute)

	for i := 0; i < 4; i++ {
		l.Fail("1.1.1.1")
	}
	if l.Blocked("1.1.1.1") {
		t.Fatal("belum boleh diblokir setelah 4 kegagalan")
	}

	l.Fail("1.1.1.1") // kegagalan ke-5
	if !l.Blocked("1.1.1.1") {
		t.Fatal("harus diblokir setelah 5 kegagalan")
	}
	if l.Blocked("2.2.2.2") {
		t.Fatal("IP lain tidak boleh ikut diblokir")
	}
}

func TestLoginLimiterReset(t *testing.T) {
	l := NewLoginLimiter(2, time.Minute)
	l.Fail("a")
	l.Fail("a")
	if !l.Blocked("a") {
		t.Fatal("harus diblokir")
	}
	l.Reset("a")
	if l.Blocked("a") {
		t.Fatal("setelah reset tidak boleh diblokir")
	}
}

func TestLoginLimiterWindowExpires(t *testing.T) {
	l := NewLoginLimiter(2, 50*time.Millisecond)
	l.Fail("a")
	l.Fail("a")
	if !l.Blocked("a") {
		t.Fatal("harus diblokir")
	}
	time.Sleep(80 * time.Millisecond)
	if l.Blocked("a") {
		t.Fatal("setelah jendela waktu lewat tidak boleh diblokir")
	}
}
