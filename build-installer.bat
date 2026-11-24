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

REM El servicio biometrico de DigitalPersona ya no se construye: la puerta la
REM maneja el terminal Hikvision por la red. Pedia .NET 8 y una DLL que no esta
REM en el repo, y era lo que hacia fallar el build.

REM 1. Build Frontend PRIMERO: gym-go.exe incrusta frontend/dist al compilar.
REM    Con el backend primero, el instalador salia con la interfaz anterior.
echo [1/3] Construyendo frontend (React)...
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

REM 2. Build Backend (Go - sin consola, con el frontend recien compilado)
echo [2/3] Construyendo backend (Go)...
go build -o gym-go.exe -ldflags="-s -w -H windowsgui" .
if errorlevel 1 (
    echo ERROR: Fallo la compilacion del backend
    pause
    exit /b 1
)
echo   OK - gym-go.exe (sin consola)
echo.

REM 3. Crear instalador
echo [3/3] Creando instalador con NSIS...
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
echo   - scripts de inicio y parada
echo.
pause
