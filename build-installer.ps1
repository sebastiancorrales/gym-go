# Script para construir el instalador de Gym-Go
# Requiere: Go, Node.js, NSIS instalados

Write-Host "================================" -ForegroundColor Cyan
Write-Host "  Gym-Go - Build Installer" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Verificar que NSIS esta instalado
$nsisPath = "C:\Program Files (x86)\NSIS\makensis.exe"
if (-not (Test-Path $nsisPath)) {
    Write-Host "ERROR: NSIS no esta instalado" -ForegroundColor Red
    Write-Host "Descarga NSIS desde: https://nsis.sourceforge.io/Download" -ForegroundColor Yellow
    exit 1
}

# Verificar herramientas imprescindibles
$tools = @(
    @{ Name = "Go"; Cmd = "go" },
    @{ Name = "Node.js"; Cmd = "node" }
)
foreach ($tool in $tools) {
    if (-not (Get-Command $tool.Cmd -ErrorAction SilentlyContinue)) {
        Write-Host "ERROR: $($tool.Name) no esta instalado" -ForegroundColor Red
        exit 1
    }
}

# El servicio biometrico de DigitalPersona (biometric-service, .NET 8 +
# DPUruNet.dll) ya no se construye ni se instala: la puerta la maneja el terminal
# Hikvision por la red. Exigia el SDK de .NET y una DLL propietaria que no esta en
# el repo, y era lo que hacia fallar este build.

# 1. Build del Frontend (React + Vite)
# VA PRIMERO: gym-go.exe incrusta frontend/dist con //go:embed en el momento de
# compilar. Con el backend primero, el instalador salia con la interfaz del
# build ANTERIOR aunque el codigo nuevo estuviera en el repo.
Write-Host "[1/3] Construyendo frontend (React)..." -ForegroundColor Green
Push-Location frontend
npm install --silent 2>$null
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la compilacion del frontend" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location
Write-Host "  OK - frontend/dist/" -ForegroundColor Green
Write-Host ""

# 2. Build del Backend (Go) - sin consola visible, con el frontend recien compilado
Write-Host "[2/3] Construyendo backend (Go)..." -ForegroundColor Green
go build -o gym-go.exe -ldflags="-s -w -H windowsgui" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la compilacion del backend" -ForegroundColor Red
    exit 1
}
Write-Host "  OK - gym-go.exe (sin consola)" -ForegroundColor Green
Write-Host ""

# 3. Crear el instalador con NSIS
Write-Host "[3/3] Creando instalador con NSIS..." -ForegroundColor Green
& $nsisPath installer.nsi
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la creacion del instalador" -ForegroundColor Red
    exit 1
}
Write-Host "  OK - Instalador creado" -ForegroundColor Green
Write-Host ""

# Mostrar resultado
if (Test-Path "Gym-Go-Installer.exe") {
    $size = (Get-Item "Gym-Go-Installer.exe").Length / 1MB
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host "  BUILD COMPLETADO" -ForegroundColor Green
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Instalador: Gym-Go-Installer.exe" -ForegroundColor Yellow
    Write-Host "  Tamano: $([math]::Round($size, 2)) MB" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Contenido:" -ForegroundColor White
    Write-Host "    - gym-go.exe (backend + frontend embebido)" -ForegroundColor White
    Write-Host "    - scripts de inicio y parada" -ForegroundColor White
    Write-Host ""
    Write-Host "  Para instalar: .\Gym-Go-Installer.exe" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Estructura instalada:" -ForegroundColor White
    Write-Host "    C:\Program Files\Gym-Go\" -ForegroundColor Gray
    Write-Host "      gym-go.exe" -ForegroundColor Gray
    Write-Host "      scripts\" -ForegroundColor Gray
    Write-Host "        launch-gym.vbs (inicio sin consolas)" -ForegroundColor Gray
    Write-Host "        stop-gym.vbs (detener)" -ForegroundColor Gray
    Write-Host "    C:\ProgramData\Gym-Go\" -ForegroundColor Gray
    Write-Host "      gym-go.db (base de datos)" -ForegroundColor Gray
    Write-Host "      uploads\ (logo y fotos)" -ForegroundColor Gray
}
else {
    Write-Host "ERROR: No se pudo crear el instalador" -ForegroundColor Red
    exit 1
}

