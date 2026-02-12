# Script para construir el instalador de Gym-Go
# Requiere: Go, Node.js, NSIS instalados

Write-Host "================================" -ForegroundColor Cyan
Write-Host "  Gym-Go - Build Installer" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Verificar que NSIS está instalado
$nsisPath = "C:\Program Files (x86)\NSIS\makensis.exe"
if (-not (Test-Path $nsisPath)) {
    Write-Host "ERROR: NSIS no esta instalado" -ForegroundColor Red
    Write-Host "Descarga NSIS desde: https://nsis.sourceforge.io/Download" -ForegroundColor Yellow
    exit 1
}

# 1. Build del Backend (Go)
Write-Host "[1/3] Construyendo backend (Go)..." -ForegroundColor Green
go build -o gym-go.exe -ldflags="-s -w" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la compilacion del backend" -ForegroundColor Red
    exit 1
}
Write-Host "OK - Backend compilado exitosamente" -ForegroundColor Green
Write-Host ""

# 2. Build del Frontend (React + Vite)
Write-Host "[2/3] Construyendo frontend (React)..." -ForegroundColor Green
Set-Location frontend
npm install
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la compilacion del frontend" -ForegroundColor Red
    Set-Location ..
    exit 1
}
Set-Location ..
Write-Host "OK - Frontend compilado exitosamente" -ForegroundColor Green
Write-Host ""

# 3. Crear el instalador con NSIS
Write-Host "[3/3] Creando instalador con NSIS..." -ForegroundColor Green
& $nsisPath installer.nsi
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la creacion del instalador" -ForegroundColor Red
    exit 1
}
Write-Host "OK - Instalador creado exitosamente" -ForegroundColor Green
Write-Host ""

# Mostrar resultado
if (Test-Path "Gym-Go-Installer.exe") {
    $size = (Get-Item "Gym-Go-Installer.exe").Length / 1MB
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host "  BUILD COMPLETADO" -ForegroundColor Green
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Instalador creado: Gym-Go-Installer.exe" -ForegroundColor Yellow
    Write-Host "Tamaño: " -NoNewline -ForegroundColor Yellow
    Write-Host "$([math]::Round($size, 2)) MB" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Para instalar, ejecuta: .\Gym-Go-Installer.exe" -ForegroundColor Cyan
}
else {
    Write-Host "ERROR: No se pudo crear el instalador" -ForegroundColor Red
    exit 1
}

