# Mencetak access_token hasil login. Dipakai oleh tokens.bat.
# Contoh: powershell -NoProfile -ExecutionPolicy Bypass -File scripts\get-token.ps1 admin@siakad.test admin12345
param(
    [Parameter(Mandatory = $true)][string]$Email,
    [Parameter(Mandatory = $true)][string]$Password,
    [string]$Base = "http://localhost:8080"
)
$ErrorActionPreference = "Stop"
$body = @{ email = $Email; password = $Password } | ConvertTo-Json -Compress
$r = Invoke-RestMethod -Method Post -Uri "$Base/api/v1/auth/login" -ContentType "application/json" -Body $body
Write-Output $r.data.access_token