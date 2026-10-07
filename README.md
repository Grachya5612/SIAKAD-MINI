# SIAKAD Mini API

RESTful API back end SIAKAD Mini (mahasiswa, mata kuliah, KRS) untuk UTS Pemrograman Berbasis Enterprise.

**Stack:** Go 1.22 · Fiber v2 · GORM · PostgreSQL · JWT (HS256) · bcrypt

## Menjalankan

```bash
copy .env.example .env        # lalu isi DB_PASSWORD dan JWT_SECRET
createdb -U postgres siakad_mini
go mod tidy
go run ./cmd/server migrate   # membuat tabel
go run ./cmd/server seed      # 1 admin, 20 mahasiswa, 10 mata kuliah
go run ./cmd/server serve     # http://localhost:8081
```

Mode produksi: `APP_ENV=production` di `.env` (log SQL hanya error).

### Akun hasil seeder
| Peran | Email | Password |
|---|---|---|
| admin | `admin@siakad.test` | `admin12345` |
| mahasiswa | `mhs01@siakad.test` ... `mhs20@siakad.test` | NIM masing-masing (`187221000001` ... `187221000020`) |

Mahasiswa yang dibuat lewat `POST /students` juga memakai NIM sebagai password awal.

## Struktur proyek

```
cmd/server/main.go            entry point: migrate | seed | serve
internal/config               pembacaan .env
internal/database             koneksi, migration (AutoMigrate), seeder
internal/models               users, students, courses, enrollments + BatasSKS()
internal/auth                 pembuatan & validasi JWT
internal/middleware           AuthRequired, RequireRole, RequireJSON, rate limiter login
internal/validation           validasi manual (Errors, IsEmail, IsDigits, ...)
internal/response             amplop JSON baku (OK, OKList, Created, NoContent, Fail, FailValidation)
internal/handlers             auth, student, course, enrollment
internal/router               aplikasi Fiber, middleware global, ErrorHandler, 10 endpoint
scripts/                      skrip pengujian (PowerShell, Python)
postman/                      koleksi Postman
```

## Endpoint

| No | Method | Endpoint | Akses | Sukses |
|---|---|---|---|---|
| 1 | POST | `/api/v1/auth/login` | Publik | 200 |
| 2 | GET | `/api/v1/auth/me` | Semua role | 200 |
| 3 | GET | `/api/v1/students` | Admin | 200 |
| 4 | POST | `/api/v1/students` | Admin | 201 |
| 5 | GET | `/api/v1/students/{id}` | Admin, mahasiswa (data sendiri) | 200 |
| 6 | PUT | `/api/v1/students/{id}` | Admin | 200 |
| 7 | DELETE | `/api/v1/students/{id}` | Admin | 204 |
| 8 | GET | `/api/v1/courses` | Semua role | 200 |
| 9 | POST | `/api/v1/enrollments` | Mahasiswa | 201 |
| 10 | DELETE | `/api/v1/enrollments/{id}` | Mahasiswa (milik sendiri) | 204 |

Semua endpoint selain login memerlukan `Authorization: Bearer <access_token>`.

### Query parameter
- `GET /students`: `page` (default 1), `per_page` (default 10, maks. 50), `prodi`, `angkatan`, `search` (nim/nama), `sort` (`nama` atau `-ipk_terakhir`)
- `GET /students/{id}`: `tahun_akademik` (opsional, membatasi KRS yang dihitung)
- `GET /courses`: `semester`, `search` (kode_mk/nama_mk), `available=true`, `tahun_akademik` (opsional)

### Format response
Sukses:
```json
{ "success": true, "message": "Data mahasiswa berhasil diambil",
  "data": [ { "id": 1, "nim": "187221000001", "nama": "Rina Putri", "prodi": "Sistem Informasi", "angkatan": 2022, "ipk_terakhir": 3.45 } ],
  "meta": { "current_page": 1, "per_page": 10, "total": 20, "last_page": 2 } }
```
Error validasi (422):
```json
{ "success": false, "message": "Validasi gagal",
  "errors": { "nim": ["NIM sudah terdaftar"], "email": ["Format email tidak valid"] } }
```

### Status code
| Kode | Arti pada API ini |
|---|---|
| 200 / 201 / 204 | Berhasil / dibuat (disertai header `Location`) / berhasil tanpa body |
| 400 | Body bukan JSON yang valid, atau `id` bukan angka positif |
| 401 | Token tidak ada, salah, atau kedaluwarsa; kredensial login salah; akun tidak aktif |
| 403 | Peran tidak sesuai, atau mengakses data/KRS milik mahasiswa lain |
| 404 | Data tidak ditemukan (termasuk mahasiswa yang sudah di-soft delete) atau route tidak ada |
| 409 | Mata kuliah sudah diambil pada tahun akademik yang sama |
| 415 | `Content-Type` bukan `application/json` pada POST/PUT |
| 422 | Validasi gagal; kuota penuh; total SKS melebihi batas |
| 429 | Login gagal lebih dari 5 kali per menit (header `Retry-After: 60`) |
| 500 | Kesalahan server, selalu berupa pesan umum tanpa stack trace |

Urutan pengecekan: token (401) -> peran (403) -> Content-Type (415) -> isi request (400/422) -> aturan bisnis.

## Business rule

| Aturan | Implementasi |
|---|---|
| Batas SKS: IPK >= 3,00 -> 24; 2,50-2,99 -> 21; < 2,50 -> 18 | `models.BatasSKS()`, dihitung per tahun akademik pada `POST /enrollments` |
| Mata kuliah yang sama tidak boleh diambil dua kali pada tahun akademik yang sama | cek di transaction (409) + unique index `(student_id, course_id, tahun_akademik)` |
| Mata kuliah berkuota penuh tidak dapat diambil | hitung enrollments per mata kuliah dan tahun akademik di transaction (422) |
| Mahasiswa hanya mengakses/mengubah KRS miliknya | 403 pada endpoint 5 dan 10 |
| Soft delete mahasiswa | `deleted_at`; hilang dari daftar, tidak bisa login, token lama ditolak |

**Concurrency (endpoint 9):** transaction mengunci baris mahasiswa lalu baris mata kuliah dengan `SELECT ... FOR UPDATE` (urutan selalu sama, sehingga tidak deadlock). Dua request bersamaan tidak dapat melampaui kuota maupun batas SKS (terbukti pada `scripts/test_krs.py`).

**Rate limiting login:** setelah 5 kali gagal dalam 1 menit dari IP yang sama, percobaan berikutnya mendapat 429. Login berhasil mereset hitungan. Penghitung disimpan di memori proses (hilang saat server restart). Fiber hanya memakai alamat koneksi langsung; header `X-Forwarded-For` tidak dipercaya.

## Keputusan desain (hal yang tidak dirinci soal)
- **Framework:** Fiber v2, mengikuti modul praktikum (REST API dan HTTP Deep Dive).
- **PUT /students/{id}** mengikuti semantik PUT di modul: representasi diganti seluruhnya. `nama`, `prodi`, `angkatan` wajib; `ipk_terakhir` yang tidak dikirim di-reset ke 0. `nim` yang dikirim dan berbeda dari data lama ditolak 422.
- **Kuota** dihitung per tahun akademik. `GET /courses` tanpa `tahun_akademik` menghitung semua enrollment.
- **`total_sks` di endpoint 5** menjumlah seluruh KRS mahasiswa, atau satu tahun akademik jika `?tahun_akademik=` diberikan.
- **NIM** milik mahasiswa yang sudah di-soft delete tetap dianggap terpakai.
- **Migration** memakai GORM AutoMigrate.

## Pengujian

Tes otomatis (tanpa database):
```bash
go test ./...
```
Mencakup batas SKS (3,00 / 2,99 / 2,50 / 2,49), validasi (tahun akademik, angkatan, email, NIM), JWT, rate limiter, pembatasan role, ErrorHandler (404, 415, 400, 500 tanpa kebocoran), format response, dan 401 pada seluruh endpoint tanpa token.

Pengujian end-to-end (server dan database harus berjalan):
```
powershell -ExecutionPolicy Bypass -File scripts\test-api.ps1
```
Skrip menjalankan seluruh skenario dan menampilkan PASS/FAIL per pengecekan. Bagian rate limit ada di akhir dan memblokir login selama 1 menit.

Opsional (Python 3, tanpa library tambahan): `python scripts/test_api.py` dan `python scripts/test_krs.py` (termasuk uji konkurensi row locking). Jalankan pada database yang baru di-seed.

Koleksi Postman: impor `postman/SIAKAD-Mini.postman_collection.json`, jalankan **Login admin** dan **Login mahasiswa** lebih dulu (token tersimpan otomatis).
