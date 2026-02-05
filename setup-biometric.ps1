# Script para verificar y configurar el sistema biométrico
# Ejecutar como: .\setup-biometric.ps1

Write-Host "[*] Verificando configuracion del sistema biometrico..." -ForegroundColor Cyan

# Verificar si el dispositivo está conectado
Write-Host "`n[*] Verificando dispositivo USB..." -ForegroundColor Yellow
$device = Get-PnpDevice | Where-Object {$_.FriendlyName -like "*DigitalPersona*" -or $_.FriendlyName -like "*U.are.U*"}

if ($device) {
    Write-Host "[OK] Dispositivo encontrado: $($device.FriendlyName)" -ForegroundColor Green
    Write-Host "   Estado: $($device.Status)" -ForegroundColor Green
} else {
    Write-Host "[ERROR] No se encontro el lector DigitalPersona U.are.U 4500" -ForegroundColor Red
    Write-Host "   Por favor, conecta el dispositivo y reinicia este script" -ForegroundColor Yellow
    exit 1
}

# Verificar instalación del SDK
Write-Host "`n[*] Verificando SDK de DigitalPersona..." -ForegroundColor Yellow
$sdkPath = "C:\Program Files\DigitalPersona\U.are.U SDK"
$sdkPathAlt = "C:\Program Files (x86)\DigitalPersona\U.are.U SDK"

$sdkFound = $false
if (Test-Path $sdkPath) {
    Write-Host "[OK] SDK encontrado en: $sdkPath" -ForegroundColor Green
    $sdkFound = $true
} elseif (Test-Path $sdkPathAlt) {
    Write-Host "[OK] SDK encontrado en: $sdkPathAlt" -ForegroundColor Green
    $sdkFound = $true
} else {
    Write-Host "[INFO] SDK de DigitalPersona no encontrado (OPCIONAL)" -ForegroundColor Cyan
    Write-Host "   Tu lector usa Windows Biometric Framework (WBF)" -ForegroundColor Cyan
    Write-Host "   No necesitas el SDK - usa el servicio C# en docs/BiometricServiceWBF.cs" -ForegroundColor Yellow
    $sdkFound = $true  # No es requerido para WBF
}

# Verificar base de datos
Write-Host "`n[*] Verificando base de datos..." -ForegroundColor Yellow
$dbPath = ".\gym-go.db"
if (Test-Path $dbPath) {
    Write-Host "[OK] Base de datos encontrada: $dbPath" -ForegroundColor Green
    
    # Verificar tablas biométricas
    Write-Host "   Verificando tablas de biometría..." -ForegroundColor Cyan
    # Aquí podrías agregar una consulta SQLite si tienes el CLI instalado
} else {
    Write-Host "[WARN] Base de datos no encontrada" -ForegroundColor Yellow
    Write-Host "   Se creará automáticamente al iniciar el servidor" -ForegroundColor Yellow
}

# Verificar migraciones
Write-Host "`n[*] Verificando migraciones..." -ForegroundColor Yellow
$migrationFile = ".\migrations\002_biometric_system.sql"
if (Test-Path $migrationFile) {
    Write-Host "[OK] Migracion biometrica encontrada" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Migracion no encontrada" -ForegroundColor Red
    Write-Host "   Archivo esperado: $migrationFile" -ForegroundColor Yellow
}

# Verificar Go
Write-Host "`n[*] Verificando Go..." -ForegroundColor Yellow
$goVersion = go version 2>$null
if ($goVersion) {
    Write-Host "[OK] Go instalado: $goVersion" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Go no encontrado" -ForegroundColor Red
    Write-Host "   Descarga desde: https://go.dev/dl/" -ForegroundColor Yellow
}

# Verificar Node.js
Write-Host "`n[*] Verificando Node.js..." -ForegroundColor Yellow
$nodeVersion = node --version 2>$null
if ($nodeVersion) {
    Write-Host "[OK] Node.js instalado: $nodeVersion" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Node.js no encontrado" -ForegroundColor Red
    Write-Host "   Descarga desde: https://nodejs.org/" -ForegroundColor Yellow
}

# Resumen
Write-Host "`n" + ("="*60) -ForegroundColor Cyan
Write-Host "RESUMEN DE CONFIGURACION" -ForegroundColor Cyan
Write-Host ("="*60) -ForegroundColor Cyan

if ($device -and $sdkFound -and $goVersion -and $nodeVersion) {
    Write-Host "`n[OK] Todo listo para usar el sistema biometrico!" -ForegroundColor Green
    Write-Host "`nProximos pasos:" -ForegroundColor Yellow
    Write-Host "`n  OPCION A - Con captura real de huellas (RECOMENDADO):" -ForegroundColor Cyan
    Write-Host "  1. Compilar servicio C# WBF (ver QUICKSTART_WBF.md)" -ForegroundColor White
    Write-Host "     cd docs && dotnet new console -n BiometricServiceWBF" -ForegroundColor White
    Write-Host "  2. Copiar BiometricServiceWBF.cs y ejecutar: dotnet run" -ForegroundColor White
    Write-Host "  3. En otra terminal: go run main.go" -ForegroundColor White
    Write-Host "  4. Acceder a: http://localhost:8080/check-in" -ForegroundColor White
    Write-Host "`n  OPCION B - Solo modo manual (sin huellas):" -ForegroundColor Cyan
    Write-Host "  1. go run main.go" -ForegroundColor White
    Write-Host "  2. Acceder a: http://localhost:8080/check-in" -ForegroundColor White
    Write-Host "     (Solo funcionara el metodo manual de documento)" -ForegroundColor Gray
    Write-Host "`n[INFO] Guia rapida: QUICKSTART_WBF.md" -ForegroundColor Cyan
    Write-Host "[INFO] Documentacion completa: docs\BIOMETRIC_SETUP.md" -ForegroundColor Cyan
} else {
    Write-Host "`n[WARN] Faltan algunos componentes" -ForegroundColor Yellow
    Write-Host "   Revisa los errores arriba y completa la instalacion" -ForegroundColor Yellow
    Write-Host "`n[INFO] Guia completa en: docs\BIOMETRIC_SETUP.md" -ForegroundColor Cyan
}

Write-Host ""
