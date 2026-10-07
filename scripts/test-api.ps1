# SIAKAD Mini - pembuktian business rule lewat terminal
#
# Jalankan dari CMD (server harus sedang berjalan di terminal lain):
#   powershell -ExecutionPolicy Bypass -File scripts\test-api.ps1
#
# Skrip memakai tahun akademik acak di setiap run, sehingga aman diulang.
# Skenario rate limit ada di paling akhir dan memblokir login selama 1 menit.

param([string]$Base = "http://localhost:8081")

$ErrorActionPreference = "Stop"
$script:pass = 0
$script:fail = 0

# ---------------------------------------------------------------- helper
function Api {
    param([string]$Method, [string]$Path, [string]$Token = "", $Body = $null)

    $p = @{ Uri = "$Base$Path"; Method = $Method; UseBasicParsing = $true }
    if ($Token) { $p.Headers = @{ Authorization = "Bearer $Token" } }
    if ($null -ne $Body) {
        $p.Body = ($Body | ConvertTo-Json -Compress)
        $p.ContentType = "application/json"
    }

    $code = 0
    $content = ""
    try {
        $r = Invoke-WebRequest @p
        $code = [int]$r.StatusCode
        $content = $r.Content
    } catch {
        $resp = $_.Exception.Response
        if ($null -eq $resp) {
            throw "Server tidak dapat dihubungi di $Base. Pastikan 'go run ./cmd/server serve' sedang berjalan."
        }
        $code = [int]$resp.StatusCode
        if ($resp -is [System.Net.HttpWebResponse]) {
            $sr = New-Object System.IO.StreamReader($resp.GetResponseStream())
            $content = $sr.ReadToEnd()
        } elseif ($_.ErrorDetails) {
            $content = $_.ErrorDetails.Message
        }
    }

    $json = $null
    if ($content) { try { $json = $content | ConvertFrom-Json } catch { } }
    return [pscustomobject]@{ Code = $code; Json = $json; Raw = $content }
}

function Check {
    param([string]$Name, [bool]$Ok, [string]$Detail = "")
    if ($Ok) {
        $script:pass++
        Write-Host ("  [PASS] " + $Name) -ForegroundColor Green
    } else {
        $script:fail++
        Write-Host ("  [FAIL] " + $Name + "   " + $Detail) -ForegroundColor Red
    }
}

function Expect {
    param([string]$Name, $Res, [int]$Status)
    Check $Name ($Res.Code -eq $Status) ("(diharapkan $Status, dapat " + $Res.Code + ": " + $Res.Raw + ")")
}

function Section([string]$Title) {
    Write-Host ""
    Write-Host ("== " + $Title + " ==") -ForegroundColor Cyan
}

function Login([string]$Email, [string]$Password) {
    return Api "POST" "/api/v1/auth/login" "" @{ email = $Email; password = $Password }
}

function LoginMhs([int]$N) {
    $email = "mhs{0:D2}@siakad.test" -f $N
    $pw = "1872210{0:D5}" -f $N
    return Login $email $pw
}

# ---------------------------------------------------------------- persiapan
$rnd = Get-Random -Minimum 3000 -Maximum 9000
$ta = "$rnd/$($rnd + 1)-Ganjil"
$taQ = [uri]::EscapeDataString($ta)
Write-Host "SIAKAD Mini - pembuktian business rule"
Write-Host "Server: $Base | Tahun akademik uji (acak): $ta"

Section "Persiapan: login akun seeder"
$r = Login "admin@siakad.test" "admin12345";  Expect "login admin" $r 200;  $adminTok = $r.Json.data.access_token
$r = LoginMhs 1;  Expect "login mhs01" $r 200;  $tok1 = $r.Json.data.access_token
$r = LoginMhs 2;  Expect "login mhs02" $r 200;  $tok2 = $r.Json.data.access_token
$r = LoginMhs 3;  Expect "login mhs03" $r 200;  $tok3 = $r.Json.data.access_token
$r = LoginMhs 7;  Expect "login mhs07" $r 200;  $tok7 = $r.Json.data.access_token

if (-not ($adminTok -and $tok1 -and $tok2 -and $tok3 -and $tok7)) {
    Write-Host "Login akun seeder gagal. Pastikan 'go run ./cmd/server seed' sudah dijalankan." -ForegroundColor Red
    exit 1
}

# peta kode_mk -> id mata kuliah
$cr = Api "GET" "/api/v1/courses" $tok1
$CID = @{}
foreach ($course in $cr.Json.data) { $CID[$course.kode_mk] = $course.id }

function Enroll([string]$Token, [string]$Kode) {
    return Api "POST" "/api/v1/enrollments" $Token @{ course_id = $CID[$Kode]; tahun_akademik = $ta }
}

function StudentId([int]$N) {
    $nim = "1872210{0:D5}" -f $N
    $x = Api "GET" "/api/v1/students?search=$nim" $adminTok
    return $x.Json.data[0].id
}

function SisaKuota([string]$Token, [string]$Kode) {
    $x = Api "GET" ("/api/v1/courses?tahun_akademik=" + $taQ + "&search=" + $Kode) $Token
    return [int]$x.Json.data[0].sisa_kuota
}

$id1 = StudentId 1
$id2 = StudentId 2
$id7 = StudentId 7

# ---------------------------------------------------------------- A
Section "A. Autentikasi dan hak akses"
$r = Login "admin@siakad.test" "salahsalah";                       Expect "login password salah -> 401" $r 401
$r = Login "bukan-email" "123";                                    Expect "login validasi gagal -> 422" $r 422
$r = Api "GET" "/api/v1/auth/me";                                  Expect "me tanpa token -> 401" $r 401
$r = Api "GET" "/api/v1/auth/me" $tok1;                            Expect "me mahasiswa -> 200" $r 200
Check "me mahasiswa memuat data student (nim)" ($null -ne $r.Json.data.student.nim)
$r = Api "GET" "/api/v1/students" $tok1;                           Expect "mahasiswa buka daftar mahasiswa -> 403" $r 403
$r = Api "GET" "/api/v1/students/$id2" $tok1;                      Expect "mhs01 buka data mhs02 -> 403" $r 403
$r = Api "GET" "/api/v1/students/$id1" $tok1;                      Expect "mhs01 buka data sendiri -> 200" $r 200
$r = Api "GET" "/api/v1/students/9999999" $adminTok;               Expect "admin buka id tidak ada -> 404" $r 404
$r = Api "GET" "/api/v1/students/abc" $adminTok;                   Expect "id non-angka -> 400" $r 400
$r = Enroll $adminTok "SI102";                                     Expect "admin ambil mata kuliah -> 403" $r 403

# ---------------------------------------------------------------- B
Section "B. Duplikasi mata kuliah (409)"
$r = Enroll $tok1 "SI101";  Expect "mhs01 ambil SI101 -> 201" $r 201;  $enr1 = $r.Json.data.id
$r = Enroll $tok1 "SI101";  Expect "mhs01 ambil SI101 lagi -> 409" $r 409

# ---------------------------------------------------------------- C
Section "C. Batas SKS (mhs07, IPK 1,95 -> batas 18 SKS)"
$d = Api "GET" "/api/v1/students/$id7" $adminTok
Check "batas_sks mhs07 = 18" ($d.Json.data.batas_sks -eq 18) ("(dapat " + $d.Json.data.batas_sks + ")")
foreach ($kode in @("SI101", "SI102", "SI103", "SI104", "SI105", "SI107")) {
    $r = Enroll $tok7 $kode
    Expect ("mhs07 ambil " + $kode + " -> 201") $r 201
}
Write-Host "  (total SKS mhs07 sekarang 17)"
$r = Enroll $tok7 "SI108"
Expect "mhs07 ambil SI108 (3 SKS, total jadi 20) -> 422" $r 422
Check "pesan error menyebut sisa SKS" ([string]$r.Json.message -match "sisa 1 SKS") ("(pesan: " + $r.Json.message + ")")

# ---------------------------------------------------------------- D
Section "D. Kuota penuh (SI106 berkuota 2) dan pembatalan KRS"
$r = Enroll $tok1 "SI106";  Expect "mhs01 ambil SI106 -> 201" $r 201;  $enr1b = $r.Json.data.id
$r = Enroll $tok2 "SI106";  Expect "mhs02 ambil SI106 -> 201" $r 201;  $enr2 = $r.Json.data.id
$r = Enroll $tok3 "SI106";  Expect "mhs03 ambil SI106 (kuota penuh) -> 422" $r 422
Check "pesan error menyebut kuota" ([string]$r.Json.message -match "Kuota") ("(pesan: " + $r.Json.message + ")")

Check "sisa_kuota SI106 = 0" ((SisaKuota $tok1 "SI106") -eq 0)
$av = Api "GET" ("/api/v1/courses?available=true&tahun_akademik=" + $taQ) $tok1
$kodeTersedia = @($av.Json.data | ForEach-Object { $_.kode_mk })
Check "SI106 tidak muncul pada available=true" (-not ($kodeTersedia -contains "SI106"))

$r = Api "DELETE" "/api/v1/enrollments/$enr1b" $tok3;              Expect "mhs03 batalkan KRS milik mhs01 -> 403" $r 403
$r = Api "DELETE" "/api/v1/enrollments/9999999" $tok2;             Expect "batalkan KRS tidak ada -> 404" $r 404
$r = Api "DELETE" "/api/v1/enrollments/abc" $tok2;                 Expect "id KRS non-angka -> 400" $r 400
$r = Api "DELETE" "/api/v1/enrollments/$enr2" $tok2;               Expect "mhs02 batalkan KRS sendiri -> 204" $r 204
Check "sisa_kuota SI106 kembali = 1" ((SisaKuota $tok1 "SI106") -eq 1)
$r = Enroll $tok3 "SI106";  Expect "mhs03 ambil SI106 setelah kuota kembali -> 201" $r 201

# ---------------------------------------------------------------- E
Section "E. Soft delete mahasiswa"
$n = Get-Random -Minimum 1000000000 -Maximum 2147483646
$nim = "99$n"
$email = "uji$n@siakad.test"
$body = @{ nim = $nim; nama = "Mahasiswa Uji"; email = $email; prodi = "Sistem Informasi"; angkatan = 2023; ipk_terakhir = 3.2 }

$r = Api "POST" "/api/v1/students" $adminTok $body;  Expect "admin tambah mahasiswa -> 201" $r 201;  $newId = $r.Json.data.id
$r = Api "POST" "/api/v1/students" $adminTok $body
Expect "tambah dengan nim/email sama -> 422" $r 422
Check "errors memuat nim" ($null -ne $r.Json.errors.nim)

$r = Login $email $nim;  Expect "mahasiswa baru login (password = NIM) -> 200" $r 200;  $newTok = $r.Json.data.access_token
$r = Api "DELETE" "/api/v1/students/$newId" $adminTok;             Expect "admin soft delete -> 204" $r 204
$r = Login $email $nim;                                            Expect "login setelah dihapus -> 401" $r 401
$r = Api "GET" "/api/v1/auth/me" $newTok;                          Expect "token lama setelah dihapus -> 401" $r 401
$r = Api "GET" "/api/v1/students/$newId" $adminTok;                Expect "detail setelah dihapus -> 404" $r 404
$r = Api "GET" "/api/v1/students?search=$nim" $adminTok
Check "hilang dari daftar mahasiswa (total = 0)" ($r.Json.meta.total -eq 0) ("(total: " + $r.Json.meta.total + ")")

# ---------------------------------------------------------------- F
Section "F. Rate limiting login (5 kali gagal per menit)"
$r = Login "admin@siakad.test" "admin12345";  Expect "login sukses mereset hitungan" $r 200
for ($i = 1; $i -le 5; $i++) {
    $r = Login "admin@siakad.test" "salahsalah"
    Expect ("gagal login ke-" + $i + " -> 401") $r 401
}
$r = Login "admin@siakad.test" "salahsalah"
Expect "gagal login ke-6 -> 429" $r 429
Write-Host "  (login diblokir selama 1 menit; tunggu sebelum menjalankan ulang skrip ini)"

# ---------------------------------------------------------------- hasil
Write-Host ""
$color = "Green"
if ($script:fail -gt 0) { $color = "Red" }
Write-Host ("HASIL: " + $script:pass + " lulus, " + $script:fail + " gagal") -ForegroundColor $color
exit $script:fail
