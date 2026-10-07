"""Uji end-to-end endpoint 1-8 (auth, students, courses, rate limit).

Opsional (butuh Python 3, tanpa library tambahan). Server harus sedang berjalan.
Jalankan pada database yang baru di-migrate dan di-seed:
    python scripts/test_api.py
Ubah alamat server dengan variabel lingkungan BASE_URL.
"""
import json, urllib.request, urllib.error, sys
import os
B=os.environ.get("BASE_URL","http://localhost:8081")+"/api/v1"
ok=bad=0
def call(method, path, token=None, body=None, ct=True, raw=None):
    data = None
    h = {}
    if token: h["Authorization"]="Bearer "+token
    if body is not None or raw is not None:
        data = (raw if raw is not None else json.dumps(body)).encode()
        if ct: h["Content-Type"]="application/json"
    req = urllib.request.Request(B+path, data=data, headers=h, method=method)
    try:
        r = urllib.request.urlopen(req)
        code, hdr, txt = r.status, dict(r.headers), r.read().decode()
    except urllib.error.HTTPError as e:
        code, hdr, txt = e.code, dict(e.headers), e.read().decode()
    try: js = json.loads(txt) if txt else None
    except Exception: js = None
    return code, js, hdr, txt
def check(name, cond, detail=""):
    global ok, bad
    if cond: ok+=1; print("  [PASS]", name)
    else: bad+=1; print("  [FAIL]", name, detail)
def tok(email, pw):
    c,j,_,_ = call("POST","/auth/login",body={"email":email,"password":pw})
    return j["data"]["access_token"] if c==200 else None

print("== Auth ==")
adm = tok("admin@siakad.test","admin12345"); m1 = tok("mhs01@siakad.test","187221000001"); m2 = tok("mhs02@siakad.test","187221000002")
check("login admin/mhs01/mhs02", adm and m1 and m2)
c,j,_,_ = call("GET","/auth/me",m1); check("me mahasiswa memuat student.nim", c==200 and j["data"]["student"]["nim"]=="187221000001", j)
c,j,_,_ = call("GET","/auth/me",adm); check("me admin tanpa student", c==200 and "student" not in j["data"])

print("== 3. GET /students ==")
c,j,_,_ = call("GET","/students",adm)
check("daftar 200, 10 item, meta benar", c==200 and len(j["data"])==10 and j["meta"]=={"current_page":1,"per_page":10,"total":20,"last_page":2}, j and j.get("meta"))
c,j,_,_ = call("GET","/students?page=2&per_page=5",adm); check("page=2 per_page=5 -> id 6..10", c==200 and [s["id"] for s in j["data"]]==[6,7,8,9,10] and j["meta"]["last_page"]==4, j)
c,j,_,_ = call("GET","/students?per_page=1000",adm); check("per_page dibatasi maks 50", c==200 and j["meta"]["per_page"]==50)
c,j,_,_ = call("GET","/students?sort=nama&per_page=50",adm); n=[s["nama"] for s in j["data"]]; check("sort=nama menaik", n==sorted(n))
c,j,_,_ = call("GET","/students?sort=-ipk_terakhir&per_page=50",adm); i=[s["ipk_terakhir"] for s in j["data"]]; check("sort=-ipk_terakhir menurun", i==sorted(i,reverse=True) and i[0]==3.95, i[:3])
c,j,_,_ = call("GET","/students?search=rina",adm); check("search=rina menemukan Rina Putri", c==200 and any(s["nama"]=="Rina Putri" for s in j["data"]))
c,j,_,_ = call("GET","/students?search=187221000007",adm); check("search nim", c==200 and j["meta"]["total"]==1)
c,j,_,_ = call("GET","/students?prodi=Sistem%20Informasi&angkatan=2022&per_page=50",adm); check("filter prodi+angkatan", c==200 and all(s["prodi"]=="Sistem Informasi" and s["angkatan"]==2022 for s in j["data"]) and j["meta"]["total"]>0)
c,j,_,_ = call("GET","/students?search=tidakadanama",adm); check("hasil kosong -> data [] bukan null", c==200 and j["data"]==[] and j["meta"]["total"]==0 and j["meta"]["last_page"]==1, j)
c,j,_,_ = call("GET","/students?sort=ngawur",adm); check("sort tidak valid -> 422", c==422 and "sort" in j["errors"])
c,j,_,_ = call("GET","/students?angkatan=abc",adm); check("angkatan non-angka -> 422", c==422 and "angkatan" in j["errors"])
c,_,_,_ = call("GET","/students",m1); check("mahasiswa -> 403", c==403)
c,_,_,_ = call("GET","/students"); check("tanpa token -> 401", c==401)

print("== 4. POST /students ==")
body={"nim":"187221009901","nama":"Tes Satu","email":"Tes.Satu@Siakad.test","prodi":"Sistem Informasi","angkatan":2023,"ipk_terakhir":3.2}
c,j,h,_ = call("POST","/students",adm,body)
new_id = j["data"]["id"] if c==201 else None
check("201 + Location + email di-lowercase", c==201 and h.get("Location")==f"/api/v1/students/{new_id}" and j["data"]["email"]=="tes.satu@siakad.test", (c,h.get("Location"),j))
c,j,_,_ = call("POST","/students",adm,body); check("duplikat -> 422 nim & email", c==422 and j["errors"]["nim"]==["NIM sudah terdaftar"] and j["errors"]["email"]==["Email sudah terdaftar"], j)
c,j,_,_ = call("POST","/students",adm,{}); check("objek kosong -> 422 (5 field)", c==422 and set(j["errors"])=={"nim","nama","email","prodi","angkatan"}, j)
c,j,_,_ = call("POST","/students",adm,{"nim":"123","nama":"A","email":"x","prodi":"SI","angkatan":2023}); check("nim pendek & email salah -> 422", c==422 and "nim" in j["errors"] and j["errors"]["email"]==["Format email tidak valid"], j)
c,j,_,_ = call("POST","/students",adm,{"nim":"187221009902","nama":"A","email":"a@b.co","prodi":"SI","angkatan":2999}); check("angkatan masa depan -> 422", c==422 and "angkatan" in j["errors"])
c,j,_,_ = call("POST","/students",adm,{"nim":"187221009902","nama":"A","email":"a@b.co","prodi":"SI","angkatan":2023,"ipk_terakhir":4.5}); check("ipk > 4 -> 422", c==422 and "ipk_terakhir" in j["errors"])
c,_,_,_ = call("POST","/students",adm,raw="{rusak"); check("JSON rusak -> 400", c==400)
c,_,_,_ = call("POST","/students",adm,body,ct=False); check("tanpa Content-Type -> 415", c==415)
c,_,_,_ = call("POST","/students",m1,{"nim":"187221009903"}); check("mahasiswa -> 403", c==403)
t_new = tok("tes.satu@siakad.test","187221009901"); check("mahasiswa baru login dengan password = NIM", t_new is not None)

print("== 5. GET /students/{id} ==")
c,j,_,_ = call("GET","/students/1",adm); check("admin 200, batas_sks 24, mata_kuliah []", c==200 and j["data"]["batas_sks"]==24 and j["data"]["mata_kuliah"]==[] and j["data"]["total_sks"]==0, j)
c,j,_,_ = call("GET","/students/7",adm); check("mhs07 (IPK 1,95) batas_sks 18", j["data"]["batas_sks"]==18)
c,j,_,_ = call("GET","/students/3",adm); check("mhs03 (IPK 2,75) batas_sks 21", j["data"]["batas_sks"]==21)
c,_,_,_ = call("GET","/students/1",m1); check("mhs01 data sendiri -> 200", c==200)
c,_,_,_ = call("GET","/students/2",m1); check("mhs01 data mhs02 -> 403", c==403)
c,_,_,_ = call("GET","/students/9999",m1); check("mhs01 id tak ada -> 403 (tidak bocorkan keberadaan)", c==403)
c,j,_,_ = call("GET","/students/9999",adm); check("admin id tak ada -> 404", c==404 and j["message"]=="Mahasiswa tidak ditemukan")
c,_,_,_ = call("GET","/students/abc",adm); check("id non-angka -> 400", c==400)

print("== 6. PUT /students/{id} ==")
c,j,_,_ = call("PUT",f"/students/{new_id}",adm,{"nama":"Tes Satu Ubah","prodi":"Teknik Informatika","angkatan":2022,"ipk_terakhir":2.8}); check("update 200", c==200 and j["data"]["nama"]=="Tes Satu Ubah" and j["data"]["ipk_terakhir"]==2.8, j)
c,j,_,_ = call("PUT",f"/students/{new_id}",adm,{"nama":"Tes Satu Ubah","prodi":"Teknik Informatika","angkatan":2022}); check("PUT tanpa ipk -> di-reset 0 (semantik PUT)", c==200 and j["data"]["ipk_terakhir"]==0, j)
c,j,_,_ = call("PUT",f"/students/{new_id}",adm,{"nim":"999999999999","nama":"X","prodi":"SI","angkatan":2022}); check("ubah nim -> 422", c==422 and j["errors"]["nim"]==["NIM tidak dapat diubah"], j)
c,j,_,_ = call("PUT",f"/students/{new_id}",adm,{"nim":"187221009901","nama":"Sama Nim","prodi":"SI","angkatan":2022}); check("nim sama dikirim -> 200", c==200)
c,j,_,_ = call("PUT",f"/students/{new_id}",adm,{"prodi":"SI"}); check("field wajib kosong -> 422", c==422 and {"nama","angkatan"}<=set(j["errors"]), j)
c,_,_,_ = call("PUT","/students/9999",adm,{"nama":"X","prodi":"SI","angkatan":2022}); check("id tak ada -> 404", c==404)
c,_,_,_ = call("PUT",f"/students/{new_id}",m1,{"nama":"X","prodi":"SI","angkatan":2022}); check("mahasiswa -> 403", c==403)

print("== 7. DELETE /students/{id} (soft delete) ==")
c,_,_,txt = call("DELETE",f"/students/{new_id}",adm); check("204 tanpa body", c==204 and txt=="")
c,_,_,_ = call("GET",f"/students/{new_id}",adm); check("detail setelah dihapus -> 404", c==404)
c,j,_,_ = call("GET","/students?per_page=50",adm); check("hilang dari daftar (total 20)", j["meta"]["total"]==20)
c,_,_,_ = call("POST","/auth/login",body={"email":"tes.satu@siakad.test","password":"187221009901"}); check("login setelah dihapus -> 401", c==401)
c,_,_,_ = call("GET","/auth/me",t_new); check("token lama setelah dihapus -> 401", c==401)
c,_,_,_ = call("DELETE",f"/students/{new_id}",adm); check("hapus lagi -> 404", c==404)
c,j,_,_ = call("POST","/students",adm,{**body,"email":"lain@siakad.test"}); check("NIM bekas soft delete tetap terpakai -> 422", c==422 and "nim" in j["errors"], j)
c,_,_,_ = call("DELETE","/students/1",m1); check("mahasiswa -> 403", c==403)
c,_,_,_ = call("DELETE","/students/abc",adm); check("id non-angka -> 400", c==400)

print("== 8. GET /courses ==")
c,j,_,_ = call("GET","/courses",m1); check("200, 10 mata kuliah, sisa_kuota = kuota", c==200 and len(j["data"])==10 and all(x["terisi"]==0 and x["sisa_kuota"]==x["kuota"] for x in j["data"]), j)
c,j,_,_ = call("GET","/courses?semester=5",adm); check("filter semester=5", c==200 and len(j["data"])==2 and all(x["semester"]==5 for x in j["data"]), j)
c,j,_,_ = call("GET","/courses?search=basis%20data",adm); check("search nama", c==200 and len(j["data"])==1 and j["data"][0]["kode_mk"]=="SI102", j)
c,j,_,_ = call("GET","/courses?search=basis",adm); check("search nama (substring, 2 hasil)", c==200 and len(j["data"])==2)
c,j,_,_ = call("GET","/courses?search=SI106",adm); check("search kode + kuota 2", j["data"][0]["kuota"]==2)
c,j,_,_ = call("GET","/courses?available=true",adm); check("available=true (semua masih tersedia)", c==200 and len(j["data"])==10)
c,j,_,_ = call("GET","/courses?semester=abc",adm); check("semester non-angka -> 422", c==422)
c,_,_,_ = call("GET","/courses"); check("tanpa token -> 401", c==401)

print("== Rate limit ==")
call("POST","/auth/login",body={"email":"admin@siakad.test","password":"admin12345"})  # login sukses mereset hitungan
codes=[call("POST","/auth/login",body={"email":"admin@siakad.test","password":"salahsalah"})[0] for _ in range(6)]
check("5x 401 lalu 429", codes==[401,401,401,401,401,429], codes)
c,_,h,_ = call("POST","/auth/login",body={"email":"admin@siakad.test","password":"admin12345"}); check("password benar tetap 429 + Retry-After", c==429 and h.get("Retry-After")=="60")

print(f"\nHASIL: {ok} lulus, {bad} gagal"); sys.exit(1 if bad else 0)
