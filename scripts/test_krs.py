"""Uji end-to-end endpoint 9-10 (KRS) termasuk uji konkurensi row locking.

Opsional (butuh Python 3, tanpa library tambahan). Server harus sedang berjalan.
Jalankan pada database yang baru di-migrate dan di-seed:
    python scripts/test_krs.py
Ubah alamat server dengan variabel lingkungan BASE_URL.
"""
import json, urllib.request, urllib.error, sys, random, threading
from concurrent.futures import ThreadPoolExecutor
import os
B=os.environ.get("BASE_URL","http://localhost:8081")+"/api/v1"
ok=bad=0
def call(method, path, token=None, body=None, ct=True, raw=None):
    data=None; h={}
    if token: h["Authorization"]="Bearer "+token
    if body is not None or raw is not None:
        data=(raw if raw is not None else json.dumps(body)).encode()
        if ct: h["Content-Type"]="application/json"
    req=urllib.request.Request(B+path,data=data,headers=h,method=method)
    try:
        r=urllib.request.urlopen(req); code,hdr,txt=r.status,dict(r.headers),r.read().decode()
    except urllib.error.HTTPError as e:
        code,hdr,txt=e.code,dict(e.headers),e.read().decode()
    try: js=json.loads(txt) if txt else None
    except Exception: js=None
    return code,js,hdr,txt
def check(name,cond,detail=""):
    global ok,bad
    if cond: ok+=1; print("  [PASS]",name)
    else: bad+=1; print("  [FAIL]",name,detail)
def tok(email,pw):
    c,j,_,_=call("POST","/auth/login",body={"email":email,"password":pw}); return j["data"]["access_token"] if c==200 else None
def mhs(n): return tok("mhs%02d@siakad.test"%n,"1872210%05d"%n)

adm=tok("admin@siakad.test","admin12345"); T={n:mhs(n) for n in (1,2,3,4,5,6,7)}
c,j,_,_=call("GET","/courses",adm); CID={x["kode_mk"]:x["id"] for x in j["data"]}
def enroll(token,kode,ta,cid=None): return call("POST","/enrollments",token,{"course_id":cid or CID[kode],"tahun_akademik":ta})
ta="%d/%d-Ganjil"%((n:=random.randint(3000,8999)),n+1)
print("tahun akademik uji:",ta)

print("== Hak akses & validasi ==")
c,_,_,_=enroll(adm,"SI102",ta); check("admin ambil MK -> 403",c==403)
c,_,_,_=call("POST","/enrollments",None,{"course_id":1,"tahun_akademik":ta}); check("tanpa token -> 401",c==401)
c,_,_,_=call("POST","/enrollments",T[1],{"course_id":1,"tahun_akademik":ta},ct=False); check("tanpa Content-Type -> 415",c==415)
c,_,_,_=call("POST","/enrollments",T[1],raw="{rusak"); check("JSON rusak -> 400",c==400)
c,j,_,_=call("POST","/enrollments",T[1],{}); check("objek kosong -> 422 (2 field)",c==422 and set(j["errors"])=={"course_id","tahun_akademik"},j)
c,j,_,_=call("POST","/enrollments",T[1],{"course_id":1,"tahun_akademik":"2026-Ganjil"}); check("format tahun salah -> 422",c==422 and "tahun_akademik" in j["errors"])
c,j,_,_=enroll(T[1],None,ta,cid=99999); check("course_id tidak ada -> 422",c==422 and j["errors"]["course_id"]==["Mata kuliah tidak ditemukan"],j)
c,_,_,_=call("DELETE","/enrollments/1",adm); check("admin batalkan KRS -> 403",c==403)

print("== B. Duplikasi (409) ==")
c,j,h,_=enroll(T[1],"SI101",ta); eid=j["data"]["id"] if c==201 else None
check("mhs01 ambil SI101 -> 201 + Location",c==201 and h.get("Location")==f"/api/v1/enrollments/{eid}" and j["data"]["total_sks"]==3 and j["data"]["batas_sks"]==24 and j["data"]["sisa_sks"]==21,(c,j))
c,j,_,_=enroll(T[1],"SI101",ta); check("ambil lagi -> 409",c==409 and "sudah diambil" in j["message"],j)
c,_,_,_=enroll(T[1],"SI101","%d/%d-Genap"%(n,n+1)); check("tahun akademik lain -> boleh (201)",c==201)

print("== C. Batas SKS (mhs07, IPK 1,95 -> 18) ==")
codes=[enroll(T[7],k,ta)[0] for k in ("SI101","SI102","SI103","SI104","SI105","SI107")]
check("6 MK (17 SKS) -> 201 semua",codes==[201]*6,codes)
c,j,_,_=enroll(T[7],"SI108",ta); check("SI108 (3 SKS) -> 422 sisa 1 SKS",c==422 and "sisa 1 SKS" in j["message"] and "Batas 18" in j["message"],j)
c,j,_,_=call("GET",f"/students/7?tahun_akademik={ta.replace('/','%2F')}",adm); check("detail: total_sks 17, 6 MK, batas 18",c==200 and j["data"]["total_sks"]==17 and len(j["data"]["mata_kuliah"])==6 and j["data"]["batas_sks"]==18,j)
c,j,_,_=enroll(T[7],"SI109",ta); check("SI109 (2 SKS) -> 422 (17+2=19>18)",c==422)
ta_q=ta.replace("/","%2F")
def sisa(k): c,j,_,_=call("GET",f"/courses?tahun_akademik={ta_q}&search={k}",adm); return j["data"][0]["sisa_kuota"]

print("== D. Kuota (SI106 kuota 2) & pembatalan ==")
c,j,_,_=enroll(T[1],"SI106",ta); e1=j["data"]["id"]; check("mhs01 -> 201 (total 3+4=7)",c==201 and j["data"]["total_sks"]==7)
c,j,_,_=enroll(T[2],"SI106",ta); e2=j["data"]["id"]; check("mhs02 -> 201",c==201)
c,j,_,_=enroll(T[3],"SI106",ta); check("mhs03 -> 422 kuota penuh",c==422 and j["message"]=="Kuota mata kuliah sudah penuh",j)
check("sisa_kuota SI106 = 0",sisa("SI106")==0)
c,j,_,_=call("GET",f"/courses?available=true&tahun_akademik={ta_q}",adm); check("available=true tidak memuat SI106",c==200 and "SI106" not in [x["kode_mk"] for x in j["data"]])
c,_,_,_=call("DELETE",f"/enrollments/{e1}",T[3]); check("mhs03 batalkan KRS mhs01 -> 403",c==403)
c,_,_,_=call("DELETE","/enrollments/9999999",T[2]); check("id tidak ada -> 404",c==404)
c,_,_,_=call("DELETE","/enrollments/abc",T[2]); check("id non-angka -> 400",c==400)
c,_,_,txt=call("DELETE",f"/enrollments/{e2}",T[2]); check("mhs02 batalkan sendiri -> 204 tanpa body",c==204 and txt=="")
check("sisa_kuota kembali = 1",sisa("SI106")==1)
c,_,_,_=enroll(T[3],"SI106",ta); check("mhs03 ambil SI106 -> 201",c==201)
c,_,_,_=call("DELETE",f"/enrollments/{e2}",T[2]); check("batalkan lagi -> 404",c==404)

print("== F. Konkurensi (row locking) ==")
ta2="%d/%d-Genap"%(n+10,n+11)
with ThreadPoolExecutor(6) as ex: res=list(ex.map(lambda i: enroll(T[i],"SI106",ta2)[0],(1,2,3,4,5,6)))
check("6 mahasiswa serentak ke MK kuota 2 -> tepat 2 sukses",res.count(201)==2 and res.count(422)==4,sorted(res))
check("sisa_kuota SI106 tahun itu = 0 (tidak minus)",call("GET",f"/courses?tahun_akademik={ta2.replace('/','%2F')}&search=SI106",adm)[1]["data"][0]["sisa_kuota"]==0)

ta3="%d/%d-Ganjil"%(n+20,n+21)
kodes=["SI101","SI102","SI103","SI104","SI105","SI107","SI108","SI109","SI110","SI106"]
with ThreadPoolExecutor(10) as ex: res=list(ex.map(lambda k: enroll(T[7],k,ta3)[0],kodes))
c,j,_,_=call("GET",f"/students/7?tahun_akademik={ta3.replace('/','%2F')}",adm)
check("10 MK serentak untuk mhs07 -> total SKS tidak melebihi 18",j["data"]["total_sks"]<=18,j["data"]["total_sks"])
check("jumlah 201 = jumlah MK tersimpan",res.count(201)==len(j["data"]["mata_kuliah"]) and res.count(201)>=4 and res.count(422)>=1,(sorted(res),j["data"]["total_sks"]))
print("     detail: 201 x%d, 422 x%d, total_sks=%d"%(res.count(201),res.count(422),j["data"]["total_sks"]))

ta4="%d/%d-Ganjil"%(n+30,n+31)
with ThreadPoolExecutor(8) as ex: res=list(ex.map(lambda i: enroll(T[4],"SI110",ta4)[0],range(8)))
check("8 request duplikat serentak (mhs04, SI110) -> tepat 1x 201, sisanya 409",res.count(201)==1 and res.count(409)==7,sorted(res))

print(f"\nHASIL: {ok} lulus, {bad} gagal"); sys.exit(1 if bad else 0)
