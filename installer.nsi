; Gym-Go Installer Script
; NSIS Modern User Interface

!include "MUI2.nsh"
!include "FileFunc.nsh"

; --------------------------------
; General Configuration
; --------------------------------
Name "Gym-Go"
OutFile "Gym-Go-Installer.exe"
InstallDir "$PROGRAMFILES64\Gym-Go"
InstallDirRegKey HKLM "Software\Gym-Go" "Install_Dir"
RequestExecutionLevel admin

; --------------------------------
; Interface Settings
; --------------------------------
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Finish page settings
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Iniciar Gym-Go ahora"
!define MUI_FINISHPAGE_RUN_FUNCTION "LaunchApplication"
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Crear acceso directo en el escritorio"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION "CreateDesktopShortcut"

; --------------------------------
; Pages
; --------------------------------
!insertmacro MUI_PAGE_LICENSE "LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; --------------------------------
; Languages
; --------------------------------
!insertmacro MUI_LANGUAGE "Spanish"

; --------------------------------
; Installer Sections
; --------------------------------
Section "Gym-Go (requerido)" SecCore
  SectionIn RO
  
  ; Set output path to the installation directory
  SetOutPath "$INSTDIR"
  
  ; Backend - Ejecutable de Go
  File "gym-go.exe"
  
  ; Create data directory in ProgramData (writable location)
  CreateDirectory "$PROGRAMDATA\Gym-Go"
  
  ; Give write permissions to Users group on ProgramData\Gym-Go using icacls
  nsExec::ExecToLog 'icacls "$PROGRAMDATA\Gym-Go" /grant Users:(OI)(CI)M /T'
  
  ; Frontend - Archivos build (en frontend/dist para que el exe los encuentre)
  SetOutPath "$INSTDIR\frontend\dist"
  File /r "frontend\dist\*.*"
  
  ; Migrations
  SetOutPath "$INSTDIR\migrations"
  File "migrations\*.sql"
  
  ; Documentación
  SetOutPath "$INSTDIR\docs"
  File "README.md"
  File "DEPLOYMENT.md"
  File "COMO_USAR.md"
  File "LICENSE"
  
  ; Write the installation path into the registry
  WriteRegStr HKLM "Software\Gym-Go" "Install_Dir" "$INSTDIR"
  
  ; Write the uninstall keys for Windows
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "DisplayName" "Gym-Go - Sistema de Gestión de Gimnasios"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "NoRepair" 1
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
SectionEnd

; --------------------------------
; Shortcuts Section
; --------------------------------
Section "Crear Accesos Directos" SecShortcuts
  
  ; Start Menu shortcuts
  CreateDirectory "$SMPROGRAMS\Gym-Go"
  CreateShortcut "$SMPROGRAMS\Gym-Go\Cómo Usar Gym-Go.lnk" "$INSTDIR\docs\COMO_USAR.md" "" "" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Desinstalar.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  
SectionEnd

; --------------------------------
; Windows Service Section (Opcional)
; --------------------------------
Section "Instalar como Servicio de Windows" SecService
  
  ; Instalar el servicio automáticamente
  nsExec::ExecToLog 'sc create GymGoService binPath= "$INSTDIR\gym-go.exe" start= auto DisplayName= "Gym-Go Service"'
  nsExec::ExecToLog 'sc description GymGoService "Sistema de Gestión de Gimnasios - Backend Server"'
  nsExec::ExecToLog 'sc start GymGoService'
  
  ; Crear script para gestionar el servicio
  FileOpen $0 "$INSTDIR\restart-service.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "echo Reiniciando servicio Gym-Go...$\r$\n"
  FileWrite $0 "sc stop GymGoService$\r$\n"
  FileWrite $0 "timeout /t 2 /nobreak >nul$\r$\n"
  FileWrite $0 "sc start GymGoService$\r$\n"
  FileWrite $0 "echo Servicio reiniciado.$\r$\n"
  FileWrite $0 "timeout /t 2$\r$\n"
  FileClose $0
  
  ; Crear script para desinstalar servicio
  FileOpen $0 "$INSTDIR\uninstall-service.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "sc stop GymGoService$\r$\n"
  FileWrite $0 "sc delete GymGoService$\r$\n"
  FileWrite $0 "echo Servicio desinstalado.$\r$\n"
  FileWrite $0 "pause$\r$\n"
  FileClose $0
  
  CreateShortcut "$SMPROGRAMS\Gym-Go\Reiniciar Servicio.lnk" "$INSTDIR\restart-service.bat" "" "" 0
  
SectionEnd

; --------------------------------
; Create Start Script
; --------------------------------
Section "-CreateStartScript" SecStartScript
  
  ; Script para abrir el navegador (verifica que el servicio esté corriendo)
  FileOpen $0 "$INSTDIR\open-gym.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "REM Verificar si el servicio está corriendo$\r$\n"
  FileWrite $0 "sc query GymGoService 2>nul | find $\"RUNNING$\" >nul$\r$\n"
  FileWrite $0 "if %errorlevel% == 0 ($\r$\n"
  FileWrite $0 "  REM Servicio corriendo - solo abrir navegador$\r$\n"
  FileWrite $0 "  start http://localhost:8080$\r$\n"
  FileWrite $0 "  exit$\r$\n"
  FileWrite $0 ")$\r$\n"
  FileWrite $0 "REM Servicio no está corriendo - iniciar ejecutable directamente$\r$\n"
  FileWrite $0 "cd /d $\"$INSTDIR$\"$\r$\n"
  FileWrite $0 "REM Verificar si ya hay una instancia corriendo$\r$\n"
  FileWrite $0 "tasklist /FI $\"IMAGENAME eq gym-go.exe$\" 2>nul | find /I /N $\"gym-go.exe$\">nul$\r$\n"
  FileWrite $0 "if %errorlevel% == 0 ($\r$\n"
  FileWrite $0 "  REM Ya hay una instancia - solo abrir navegador$\r$\n"
  FileWrite $0 "  start http://localhost:8080$\r$\n"
  FileWrite $0 "  exit$\r$\n"
  FileWrite $0 ")$\r$\n"
  FileWrite $0 "REM Iniciar el servidor en segundo plano$\r$\n"
  FileWrite $0 "start $\"$\" /B gym-go.exe$\r$\n"
  FileWrite $0 "REM Esperar 3 segundos$\r$\n"
  FileWrite $0 "timeout /t 3 /nobreak >nul$\r$\n"
  FileWrite $0 "REM Abrir navegador$\r$\n"
  FileWrite $0 "start http://localhost:8080$\r$\n"
  FileWrite $0 "exit$\r$\n"
  FileClose $0
  
  ; Script para detener el servicio o proceso
  FileOpen $0 "$INSTDIR\stop-gym.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "echo Deteniendo Gym-Go...$\r$\n"
  FileWrite $0 "REM Intentar detener el servicio primero$\r$\n"
  FileWrite $0 "sc query GymGoService 2>nul | find $\"RUNNING$\" >nul$\r$\n"
  FileWrite $0 "if %errorlevel% == 0 ($\r$\n"
  FileWrite $0 "  echo Deteniendo servicio...$\r$\n"
  FileWrite $0 "  net stop GymGoService$\r$\n"
  FileWrite $0 "  if %errorlevel% neq 0 ($\r$\n"
  FileWrite $0 "    echo Nota: Se requieren permisos de administrador para detener el servicio.$\r$\n"
  FileWrite $0 "    echo Puede ejecutar este script como administrador o detener el proceso.$\r$\n"
  FileWrite $0 "  )$\r$\n"
  FileWrite $0 ") else ($\r$\n"
  FileWrite $0 "  REM Detener proceso directamente$\r$\n"
  FileWrite $0 "  taskkill /IM gym-go.exe /F >nul 2>&1$\r$\n"
  FileWrite $0 "  if %errorlevel% == 0 ($\r$\n"
  FileWrite $0 "    echo Gym-Go detenido.$\r$\n"
  FileWrite $0 "  ) else ($\r$\n"
  FileWrite $0 "    echo Gym-Go no estaba en ejecucion.$\r$\n"
  FileWrite $0 "  )$\r$\n"
  FileWrite $0 ")$\r$\n"
  FileWrite $0 "timeout /t 3$\r$\n"
  FileClose $0
  
  ; Script para detener el servicio (con admin)
  FileOpen $0 "$INSTDIR\stop-service.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "echo Deteniendo servicio Gym-Go...$\r$\n"
  FileWrite $0 "net stop GymGoService 2>nul$\r$\n"
  FileWrite $0 "if %errorlevel% == 0 ($\r$\n"
  FileWrite $0 "  echo Servicio detenido exitosamente.$\r$\n"
  FileWrite $0 ") else ($\r$\n"
  FileWrite $0 "  echo No se pudo detener el servicio.$\r$\n"
  FileWrite $0 "  echo Ejecute este script como Administrador.$\r$\n"
  FileWrite $0 ")$\r$\n"
  FileWrite $0 "timeout /t 3$\r$\n"
  FileClose $0
  
  CreateShortcut "$SMPROGRAMS\Gym-Go\Abrir Gym-Go.lnk" "$INSTDIR\open-gym.bat" "" "$INSTDIR\gym-go.exe" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Detener Gym-Go.lnk" "$INSTDIR\stop-gym.bat" "" "" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Detener Servicio (Admin).lnk" "$INSTDIR\stop-service.bat" "" "" 0
  
SectionEnd

; --------------------------------
; Section Descriptions
; --------------------------------
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Archivos principales de Gym-Go (requerido)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecShortcuts} "Crear accesos directos en el menú inicio"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Instalar como servicio de Windows - Se inicia automáticamente (recomendado)"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; --------------------------------
; Launch Functions
; --------------------------------
Function LaunchApplication
  ; Start the service if installed
  nsExec::ExecToLog 'sc query GymGoService'
  Pop $0
  ${If} $0 == 0
    ; Service exists, start it
    nsExec::ExecToLog 'sc start GymGoService'
  ${Else}
    ; No service, start the exe in the background
    Exec '"$INSTDIR\gym-go.exe"'
  ${EndIf}
  
  ; Wait 3 seconds for the server to start
  Sleep 3000
  
  ; Open browser to localhost:8080
  ExecShell "open" "http://localhost:8080"
FunctionEnd

Function CreateDesktopShortcut
  ; Create desktop shortcut that opens the browser and ensures service is running
  CreateShortcut "$DESKTOP\Gym-Go.lnk" "$INSTDIR\open-gym.bat" "" "$INSTDIR\gym-go.exe" 0
FunctionEnd

; --------------------------------
; Uninstaller Section
; --------------------------------
Section "Uninstall"
  
  ; Stop and remove the service if it exists
  nsExec::ExecToLog 'sc stop GymGoService'
  nsExec::ExecToLog 'sc delete GymGoService'
  
  ; Also kill any running instances (fallback)
  ExecWait 'taskkill /F /IM gym-go.exe'
  
  ; Remove registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go"
  DeleteRegKey HKLM "Software\Gym-Go"
  
  ; Remove files and directories
  Delete "$INSTDIR\gym-go.exe"
  Delete "$INSTDIR\open-gym.bat"
  Delete "$INSTDIR\stop-gym.bat"
  Delete "$INSTDIR\stop-service.bat"
  Delete "$INSTDIR\restart-service.bat"
  Delete "$INSTDIR\uninstall-service.bat"
  Delete "$INSTDIR\uninstall.exe"
  
  RMDir /r "$INSTDIR\frontend"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\docs"
  RMDir /r "$INSTDIR\config"
  
  ; Ask before deleting database in ProgramData
  MessageBox MB_YESNO "¿Desea eliminar también la base de datos? (Se perderán todos los datos)$\nUbicación: $PROGRAMDATA\Gym-Go" IDYES DeleteDB IDNO KeepDB
  DeleteDB:
    RMDir /r "$PROGRAMDATA\Gym-Go"
  KeepDB:
  
  RMDir "$INSTDIR"
  
  ; Remove shortcuts
  Delete "$SMPROGRAMS\Gym-Go\*.*"
  RMDir "$SMPROGRAMS\Gym-Go"
  Delete "$DESKTOP\Gym-Go.lnk"
  
SectionEnd
