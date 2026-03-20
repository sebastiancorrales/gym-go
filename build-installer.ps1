# Script para construir el instalador de Gym-Go
# Requiere: Go, Node.js, .NET 8 SDK, NSIS instalados

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

# Verificar herramientas
$tools = @(
    @{ Name = "Go"; Cmd = "go version" },
    @{ Name = "Node.js"; Cmd = "node --version" },
    @{ Name = ".NET SDK"; Cmd = "dotnet --version" }
)
foreach ($tool in $tools) {
    try {
        Invoke-Expression $tool.Cmd | Out-Null
    } catch {
        Write-Host "ERROR: $($tool.Name) no esta instalado" -ForegroundColor Red
        exit 1
    }
}

# 1. Build del Backend (Go) - sin consola visible
Write-Host "[1/4] Construyendo backend (Go)..." -ForegroundColor Green
go build -o gym-go.exe -ldflags="-s -w -H windowsgui" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la compilacion del backend" -ForegroundColor Red
    exit 1
}
Write-Host "  OK - gym-go.exe (sin consola)" -ForegroundColor Green
Write-Host ""

# 2. Build del Frontend (React + Vite)
Write-Host "[2/4] Construyendo frontend (React)..." -ForegroundColor Green
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

# 3. Publish del Servicio Biometrico (C# self-contained)
Write-Host "[3/4] Publicando servicio biometrico (.NET)..." -ForegroundColor Green

# Limpiar build anterior
if (Test-Path "build\biometric") {
    Remove-Item -Recurse -Force "build\biometric"
}
New-Item -ItemType Directory -Path "build\biometric" -Force | Out-Null

# Publish self-contained para Windows x64 (no requiere .NET instalado en destino)
dotnet publish biometric-service/BiometricService.csproj `
    -c Release `
    -r win-x64 `
    --self-contained `
    -p:PublishSingleFile=false `
    -p:IncludeNativeLibrariesForSelfExtract=true `
    -o build/biometric
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Fallo la publicacion del servicio biometrico" -ForegroundColor Red
    exit 1
}

# Copiar DLLs del SDK de DigitalPersona (por si no se copiaron automaticamente)
$dpDlls = @(
    "DPFPShrNET.dll", "DPFPEngNET.dll", "DPFPVerNET.dll",
    "DPFPDevNET.dll", "DPFPCtlXTypeLibNET.dll", "DPFPCtlXWrapperNET.dll"
)
foreach ($dll in $dpDlls) {
    $src = "biometric-service\lib\$dll"
    $dst = "build\biometric\$dll"
    if ((Test-Path $src) -and (-not (Test-Path $dst))) {
        Copy-Item $src $dst
    }
}

Write-Host "  OK - BiometricService.exe (self-contained)" -ForegroundColor Green
Write-Host ""

# 4. Crear el instalador con NSIS
Write-Host "[4/4] Creando instalador con NSIS..." -ForegroundColor Green
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
    $bioSize = (Get-ChildItem "build\biometric" -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host "  BUILD COMPLETADO" -ForegroundColor Green
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Instalador: Gym-Go-Installer.exe" -ForegroundColor Yellow
    Write-Host "  Tamano: $([math]::Round($size, 2)) MB" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Contenido:" -ForegroundColor White
    Write-Host "    - gym-go.exe (backend + frontend embebido)" -ForegroundColor White
    Write-Host "    - BiometricService.exe (servicio biometrico, $([math]::Round($bioSize, 1)) MB self-contained)" -ForegroundColor White
    Write-Host "    - biometric.hta (captura de huella)" -ForegroundColor White
    Write-Host "    - Migraciones SQL" -ForegroundColor White
    Write-Host ""
    Write-Host "  Para instalar: .\Gym-Go-Installer.exe" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Estructura instalada:" -ForegroundColor White
    Write-Host "    C:\Program Files\Gym-Go\" -ForegroundColor Gray
    Write-Host "      gym-go.exe" -ForegroundColor Gray
    Write-Host "      biometric\" -ForegroundColor Gray
    Write-Host "        BiometricService.exe + DLLs" -ForegroundColor Gray
    Write-Host "        scripts\biometric.hta" -ForegroundColor Gray
    Write-Host "      scripts\" -ForegroundColor Gray
    Write-Host "        launch-gym.vbs (inicio sin consolas)" -ForegroundColor Gray
    Write-Host "        stop-gym.vbs (detener)" -ForegroundColor Gray
    Write-Host "    C:\ProgramData\Gym-Go\" -ForegroundColor Gray
    Write-Host "      gym-go.db (base de datos)" -ForegroundColor Gray
}
else {
    Write-Host "ERROR: No se pudo crear el instalador" -ForegroundColor Red
    exit 1
}

