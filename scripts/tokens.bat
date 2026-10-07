@echo off
rem Mengisi variabel untuk sesi CMD ini: B (alamat server), J (header JSON),
rem A (token admin), M1, M2, M3, M7 (token mahasiswa).
rem Server harus sedang berjalan. Pakai:
rem   call scripts\tokens.bat          (port dibaca dari APP_PORT di file .env)
rem   call scripts\tokens.bat 8081     (port diisi manual)
set PORT=8080
if exist .env for /f "tokens=2 delims==" %%p in ('findstr /b "APP_PORT=" .env') do set PORT=%%p
if not "%~1"=="" set PORT=%~1
set B=http://localhost:%PORT%
set J=-H "Content-Type: application/json"
call :login admin@siakad.test admin12345 A
call :login mhs01@siakad.test 187221000001 M1
call :login mhs02@siakad.test 187221000002 M2
call :login mhs03@siakad.test 187221000003 M3
call :login mhs07@siakad.test 187221000007 M7
for %%v in (A M1 M2 M3 M7) do if not defined %%v echo GAGAL: variabel %%v kosong. Server sudah jalan di %B% dan seed sudah dijalankan?
echo Variabel siap: B J A M1 M2 M3 M7  (server: %B%)
goto :eof

:login
for /f "usebackq delims=" %%t in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0get-token.ps1" %~1 %~2 %B%`) do set %~3=%%t
goto :eof