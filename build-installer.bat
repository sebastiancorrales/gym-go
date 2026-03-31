@echo off
echo ================================
echo   Gym-Go - Build Installer
echo ================================
echo.

REM Verificar NSIS
if not exist "C:\Program Files (x86)\NSIS\makensis.exe" (
    echo ERROR: NSIS no esta instalado
    echo Descarga NSIS desde: https://nsis.sourceforge.io/Download
    pause
    exit /b 1
)

REM 1. Build Backend (Go - sin consola)
echo [1/4] Construyendo backend (Go)...
go build -o gym-go.exe -ldflags="-s -w -H windowsgui" .
if errorlevel 1 (
    echo ERROR: Fallo la compilacion del backend
    pause
    exit /b 1
)
echo   OK - gym-go.exe (sin consola)
echo.

REM 2. Build Frontend
echo [2/4] Construyendo frontend (React)...
cd frontend
call npm install --silent 2>nul
call npm run build
if errorlevel 1 (
    echo ERROR: Fallo la compilacion del frontend
    cd ..
    pause
    exit /b 1
)
cd ..
echo   OK - frontend/dist/
echo.

REM 3. Publish Servicio Biometrico (self-contained)
echo [3/4] Publicando servicio biometrico (.NET)...
if exist "build\biometric" rmdir /s /q "build\biometric"
mkdir "build\biometric" 2>nul

dotnet publish biometric-service/BiometricPOC.csproj -c Release -r win-x64 --self-contained -p:PublishSingleFile=false -p:IncludeNativeLibrariesForSelfExtract=true -o build/biometric
if errorlevel 1 (
    echo ERROR: Fallo la publicacion del servicio biometrico
    pause
    exit /b 1
)

REM Copiar DLLs de DigitalPersona si no estan
for %%f in (DPFPShrNET.dll DPFPEngNET.dll DPFPVerNET.dll DPFPDevNET.dll DPFPCtlXTypeLibNET.dll DPFPCtlXWrapperNET.dll) do (
    if exist "biometric-service\lib\%%f" (
        if not exist "build\biometric\%%f" copy "biometric-service\lib\%%f" "build\biometric\%%f" >nul
    )
)
echo   OK - BiometricService.exe (self-contained)
echo.

REM 4. Crear instalador
echo [4/4] Creando instalador con NSIS...
"C:\Program Files (x86)\NSIS\makensis.exe" installer.nsi
if errorlevel 1 (
    echo ERROR: Fallo la creacion del instalador
    pause
    exit /b 1
)
echo   OK - Instalador creado
echo.

echo ================================
echo   BUILD COMPLETADO
echo ================================
echo.
echo Instalador: Gym-Go-Installer.exe
echo.
echo Contenido:
echo   - gym-go.exe (backend + frontend embebido)
echo   - BiometricService.exe (servicio biometrico)
echo   - biometric.hta (captura de huella)
echo   - Migraciones SQL
echo.
pause
