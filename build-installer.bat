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

REM 1. Build Backend
echo [1/3] Construyendo backend (Go)...
go build -o gym-go.exe -ldflags="-s -w" .
if errorlevel 1 (
    echo ERROR: Fallo la compilacion del backend
    pause
    exit /b 1
)
echo OK - Backend compilado
echo.

REM 2. Build Frontend
echo [2/3] Construyendo frontend (React)...
cd frontend
call npm install
call npm run build
if errorlevel 1 (
    echo ERROR: Fallo la compilacion del frontend
    cd ..
    pause
    exit /b 1
)
cd ..
echo OK - Frontend compilado
echo.

REM 3. Crear instalador
echo [3/3] Creando instalador con NSIS...
"C:\Program Files (x86)\NSIS\makensis.exe" installer.nsi
if errorlevel 1 (
    echo ERROR: Fallo la creacion del instalador
    pause
    exit /b 1
)
echo OK - Instalador creado
echo.

echo ================================
echo   BUILD COMPLETADO
echo ================================
echo.
echo Instalador creado: Gym-Go-Installer.exe
echo.
pause
