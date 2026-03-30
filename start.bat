@echo off
chcp 65001 >nul
title Gym-Go - Iniciando servicios...
color 0A

echo ============================================
echo        GYM-GO - Inicio Completo
echo ============================================
echo.

:: -----------------------------------------------
:: 1. Build del frontend (Vite)
:: -----------------------------------------------
echo [1/3] Compilando frontend (Vite build)...
cd /d "%~dp0frontend"
call npm run build
if %ERRORLEVEL% NEQ 0 (
    color 0C
    echo ERROR: Fallo el build del frontend!
    pause
    exit /b 1
)
echo [OK] Frontend compilado en frontend\dist
echo.

cd /d "%~dp0"

:: -----------------------------------------------
:: 2. Servicio biometrico C# - HTTP en puerto 5001
:: -----------------------------------------------
echo [2/3] Iniciando servicio biometrico C# (HTTP puerto 5001)...
start "Gym-Go Biometric Service" cmd /k "cd /d "%~dp0" && echo === SERVICIO BIOMETRICO === && echo API HTTP: http://localhost:5001 && echo. && biometric-service\bin\Debug\net8.0-windows\BiometricPOC.exe"
timeout /t 3 /nobreak >nul
echo [OK] Servicio biometrico iniciado (ver ventana aparte)
echo.

:: -----------------------------------------------
:: 3. Backend Go (en ventana aparte con logs visibles)
:: -----------------------------------------------
echo [3/3] Iniciando backend Go (puerto 8080)...
start "Gym-Go Backend" cmd /k "cd /d "%~dp0" && echo === BACKEND GO === && echo Puerto: 8080 && echo Frontend: http://localhost:8080 && echo. && go build -o gym-go.exe . && gym-go.exe"
timeout /t 2 /nobreak >nul
echo [OK] Backend Go iniciado (ver ventana aparte)
echo.

:: -----------------------------------------------
:: Listo
:: -----------------------------------------------
color 0A
echo ============================================
echo   TODOS LOS SERVICIOS INICIADOS
echo ============================================
echo.
echo   Frontend:    http://localhost:8080
echo   Backend Go:  puerto 8080
echo   Biometrico:  http://localhost:5001
echo.
echo   Cada servicio tiene su propia ventana
echo   con logs visibles.
echo.
echo   Cierra esta ventana o presiona una tecla
echo   para salir (los servicios siguen corriendo).
echo ============================================
pause
