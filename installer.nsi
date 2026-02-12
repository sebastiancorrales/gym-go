; Gym-Go Installer Script
; NSIS Modern User Interface

!include "MUI2.nsh"

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
  
  ; Base de datos SQLite (crear directorio)
  CreateDirectory "$INSTDIR\data"
  
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
  CreateShortcut "$SMPROGRAMS\Gym-Go\Gym-Go Server.lnk" "$INSTDIR\gym-go.exe" "" "$INSTDIR\gym-go.exe" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Abrir Gym-Go.lnk" "http://localhost:8080" "" "" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Desinstalar.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  
  ; Desktop shortcut
  CreateShortcut "$DESKTOP\Gym-Go.lnk" "http://localhost:8080" "" "$INSTDIR\gym-go.exe" 0
  
SectionEnd

; --------------------------------
; Windows Service Section (Opcional)
; --------------------------------
Section "Instalar como Servicio de Windows" SecService
  
  ; Crear script para instalar servicio
  FileOpen $0 "$INSTDIR\install-service.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "sc create GymGoService binPath= $\"$INSTDIR\gym-go.exe$\" start= auto DisplayName= $\"Gym-Go Service$\"$\r$\n"
  FileWrite $0 "sc description GymGoService $\"Sistema de Gestión de Gimnasios$\"$\r$\n"
  FileWrite $0 "sc start GymGoService$\r$\n"
  FileWrite $0 "pause$\r$\n"
  FileClose $0
  
  ; Crear script para desinstalar servicio
  FileOpen $0 "$INSTDIR\uninstall-service.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "sc stop GymGoService$\r$\n"
  FileWrite $0 "sc delete GymGoService$\r$\n"
  FileWrite $0 "pause$\r$\n"
  FileClose $0
  
  CreateShortcut "$SMPROGRAMS\Gym-Go\Instalar Servicio.lnk" "$INSTDIR\install-service.bat" "" "" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Desinstalar Servicio.lnk" "$INSTDIR\uninstall-service.bat" "" "" 0
  
SectionEnd

; --------------------------------
; Create Start Script
; --------------------------------
Section "-CreateStartScript" SecStartScript
  
  ; Script para iniciar el servidor
  FileOpen $0 "$INSTDIR\start-server.bat" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "echo ================================$\r$\n"
  FileWrite $0 "echo   Gym-Go - Servidor Backend$\r$\n"
  FileWrite $0 "echo ================================$\r$\n"
  FileWrite $0 "echo.$\r$\n"
  FileWrite $0 "echo Iniciando servidor en http://localhost:8080$\r$\n"
  FileWrite $0 "echo.$\r$\n"
  FileWrite $0 "echo Presiona Ctrl+C para detener el servidor$\r$\n"
  FileWrite $0 "echo.$\r$\n"
  FileWrite $0 "cd /d $\"$INSTDIR$\"$\r$\n"
  FileWrite $0 "start http://localhost:8080$\r$\n"
  FileWrite $0 "gym-go.exe$\r$\n"
  FileClose $0
  
  CreateShortcut "$SMPROGRAMS\Gym-Go\Iniciar Gym-Go.lnk" "$INSTDIR\start-server.bat" "" "$INSTDIR\gym-go.exe" 0
  
SectionEnd

; --------------------------------
; Section Descriptions
; --------------------------------
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Archivos principales de Gym-Go (requerido)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecShortcuts} "Crear accesos directos en el menú inicio y escritorio"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Instalar Gym-Go como servicio de Windows (opcional)"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; --------------------------------
; Uninstaller Section
; --------------------------------
Section "Uninstall"
  
  ; Remove registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go"
  DeleteRegKey HKLM "Software\Gym-Go"
  
  ; Remove files and directories
  Delete "$INSTDIR\gym-go.exe"
  Delete "$INSTDIR\start-server.bat"
  Delete "$INSTDIR\install-service.bat"
  Delete "$INSTDIR\uninstall-service.bat"
  Delete "$INSTDIR\uninstall.exe"
  
  RMDir /r "$INSTDIR\frontend"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\docs"
  RMDir /r "$INSTDIR\config"
  
  ; Ask before deleting database
  MessageBox MB_YESNO "¿Desea eliminar también la base de datos? (Se perderán todos los datos)" IDYES DeleteDB IDNO KeepDB
  DeleteDB:
    RMDir /r "$INSTDIR\data"
  KeepDB:
  
  RMDir "$INSTDIR"
  
  ; Remove shortcuts
  Delete "$SMPROGRAMS\Gym-Go\*.*"
  RMDir "$SMPROGRAMS\Gym-Go"
  Delete "$DESKTOP\Gym-Go.lnk"
  
SectionEnd
