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

# El servicio biometrico es OPCIONAL para construir el instalador. Necesita dos
# cosas que no viven en el repositorio:
#   - el SDK de .NET 8
#   - biometric-service\lib\DPUruNet.dll (el SDK de DigitalPersona; *.dll esta en
#     .gitignore, asi que quien clone el repo no la tiene)
# Sin el servicio la app instalada funciona con check-in manual; solo se pierde el
# check-in por huella. Antes la ausencia de cualquiera de las dos cosas abortaba
# todo el build.
$dpDll = "biometric-service\lib\DPUruNet.dll"
$buildBiometric = $true
$biometricSkipReason = ""

if (-not (Get-Command dotnet -ErrorAction SilentlyContinue)) {
    $buildBiometric = $false
    $biometricSkipReason = "el SDK de .NET 8 no esta instalado"
} elseif (-not (Test-Path $dpDll)) {
    $buildBiometric = $false
    $biometricSkipReason = "falta $dpDll (SDK de DigitalPersona, no se versiona)"
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

# 3. Publish del Servicio Biometrico (C# self-contained) - OPCIONAL
Write-Host "[3/4] Publicando servicio biometrico (.NET)..." -ForegroundColor Green

# Limpiar build anterior
if (Test-Path "build\biometric") {
    Remove-Item -Recurse -Force "build\biometric"
}
New-Item -ItemType Directory -Path "build\biometric" -Force | Out-Null

if (-not $buildBiometric) {
    Write-Host "  OMITIDO: $biometricSkipReason" -ForegroundColor Yellow
    Write-Host "  El instalador se creara SIN el servicio biometrico." -ForegroundColor Yellow
    Write-Host "  La app funcionara con check-in manual; no habra check-in por huella." -ForegroundColor Yellow
    # NSIS necesita que el directorio no este vacio para su 'File /r'.
    "El servicio biometrico no se incluyo en este build: $biometricSkipReason" |
        Out-File -FilePath "build\biometric\LEEME.txt" -Encoding utf8
} else {
    # Publish self-contained para Windows x64 (no requiere .NET instalado en destino)
    # El proyecto es BiometricPOC.csproj; el nombre BiometricService.csproj que habia
    # aqui no existe y hacia fallar este paso siempre.
    dotnet publish biometric-service/BiometricPOC.csproj `
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

    # DLL del SDK de DigitalPersona, por si el publish no la copio. La lista anterior
    # (DPFPShrNET, DPFPEngNET, ...) pertenecia a un prototipo abandonado; la
    # dependencia real del csproj es DPUruNet.dll.
    if (-not (Test-Path "build\biometric\DPUruNet.dll")) {
        Copy-Item $dpDll "build\biometric\DPUruNet.dll"
    }

    if (-not (Test-Path "build\biometric\BiometricPOC.exe")) {
        Write-Host "ERROR: no se genero BiometricPOC.exe" -ForegroundColor Red
        exit 1
    }
    Write-Host "  OK - BiometricPOC.exe (self-contained)" -ForegroundColor Green
}
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
    Write-Host "    - BiometricPOC.exe (servicio biometrico, $([math]::Round($bioSize, 1)) MB self-contained)" -ForegroundColor White
    Write-Host "    - Migraciones SQL" -ForegroundColor White
    Write-Host ""
    Write-Host "  Para instalar: .\Gym-Go-Installer.exe" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Estructura instalada:" -ForegroundColor White
    Write-Host "    C:\Program Files\Gym-Go\" -ForegroundColor Gray
    Write-Host "      gym-go.exe" -ForegroundColor Gray
    Write-Host "      biometric\" -ForegroundColor Gray
    Write-Host "        BiometricPOC.exe + DLLs" -ForegroundColor Gray
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

