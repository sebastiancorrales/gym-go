; Gym-Go Installer Script
; NSIS Modern User Interface - Con Servicio Biometrico

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
  
  ; Detener procesos existentes si estan corriendo
  nsExec::ExecToLog 'taskkill /F /IM gym-go.exe'
  nsExec::ExecToLog 'taskkill /F /IM BiometricPOC.exe'
  
  ; ================================
  ; Backend Go + Frontend embebido
  ; ================================
  SetOutPath "$INSTDIR"
  File "gym-go.exe"
  
  ; Frontend embebido en el exe, pero tambien copiar dist
  SetOutPath "$INSTDIR\frontend\dist"
  File /r "frontend\dist\*.*"
  
  ; ================================
  ; Servicio Biometrico
  ; ================================
  SetOutPath "$INSTDIR\biometric"
  File /r "build\biometric\*.*"
  
  ; ================================
  ; Migrations
  ; ================================
  SetOutPath "$INSTDIR\migrations"
  File "migrations\*.sql"
  
  ; ================================
  ; Scripts de ejecucion
  ; ================================
  SetOutPath "$INSTDIR\scripts"
  File "scripts\launch-gym.vbs"
  File "scripts\stop-gym.vbs"
  
  ; ================================
  ; Documentacion
  ; ================================
  SetOutPath "$INSTDIR\docs"
  File "README.md"
  File "COMO_USAR.md"
  File "LICENSE"
  
  ; ================================
  ; Directorio de datos (ProgramData)
  ; ================================
  CreateDirectory "$PROGRAMDATA\Gym-Go"
  CreateDirectory "$PROGRAMDATA\Gym-Go\uploads"
  
  ; Permisos de escritura para usuarios
  nsExec::ExecToLog 'icacls "$PROGRAMDATA\Gym-Go" /grant Users:(OI)(CI)M /T'
  
  ; ================================
  ; Registro de Windows
  ; ================================
  WriteRegStr HKLM "Software\Gym-Go" "Install_Dir" "$INSTDIR"
  WriteRegStr HKLM "Software\Gym-Go" "Version" "1.0.0"
  
  ; Registro de desinstalacion
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "DisplayName" "Gym-Go - Sistema de Gestion de Gimnasios"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "DisplayVersion" "1.0.0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "Publisher" "Gym-Go"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go" "NoRepair" 1
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
SectionEnd

; --------------------------------
; Shortcuts Section
; --------------------------------
Section "Crear Accesos Directos" SecShortcuts
  
  ; Start Menu
  CreateDirectory "$SMPROGRAMS\Gym-Go"
  CreateShortcut "$SMPROGRAMS\Gym-Go\Gym-Go.lnk" "wscript.exe" '"$INSTDIR\scripts\launch-gym.vbs"' "$INSTDIR\gym-go.exe" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Detener Gym-Go.lnk" "wscript.exe" '"$INSTDIR\scripts\stop-gym.vbs"' "" 0
  CreateShortcut "$SMPROGRAMS\Gym-Go\Desinstalar.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  
SectionEnd

; --------------------------------
; Autostart Section (Opcional)
; --------------------------------
Section /o "Iniciar con Windows" SecAutostart
  
  ; Agregar al registro de inicio
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Gym-Go" '"wscript.exe" "$INSTDIR\scripts\launch-gym.vbs"'
  
SectionEnd

; --------------------------------
; Section Descriptions
; --------------------------------
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Archivos principales de Gym-Go con servicio biometrico (requerido)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecShortcuts} "Crear accesos directos en el menu inicio"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAutostart} "Iniciar Gym-Go automaticamente al encender la computadora"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; --------------------------------
; Launch Functions
; --------------------------------
Function LaunchApplication
  Exec 'wscript.exe "$INSTDIR\scripts\launch-gym.vbs"'
FunctionEnd

Function CreateDesktopShortcut
  CreateShortcut "$DESKTOP\Gym-Go.lnk" "wscript.exe" '"$INSTDIR\scripts\launch-gym.vbs"' "$INSTDIR\gym-go.exe" 0
FunctionEnd

; --------------------------------
; Uninstaller Section
; --------------------------------
Section "Uninstall"
  
  ; Detener todos los procesos
  nsExec::ExecToLog 'taskkill /F /IM gym-go.exe'
  nsExec::ExecToLog 'taskkill /F /IM BiometricPOC.exe'
  
  ; Quitar del autostart
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Gym-Go"
  
  ; Quitar registro
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Gym-Go"
  DeleteRegKey HKLM "Software\Gym-Go"
  
  ; Eliminar archivos
  Delete "$INSTDIR\gym-go.exe"
  Delete "$INSTDIR\uninstall.exe"
  
  RMDir /r "$INSTDIR\biometric"
  RMDir /r "$INSTDIR\frontend"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\scripts"
  RMDir /r "$INSTDIR\docs"
  
  RMDir "$INSTDIR"
  
  ; Preguntar por la base de datos
  MessageBox MB_YESNO "¿Desea eliminar tambien la base de datos?$\n(Se perderan todos los datos)$\n$\nUbicacion: $PROGRAMDATA\Gym-Go" IDYES DeleteDB IDNO KeepDB
  DeleteDB:
    RMDir /r "$PROGRAMDATA\Gym-Go"
  KeepDB:
  
  ; Eliminar accesos directos
  Delete "$SMPROGRAMS\Gym-Go\*.*"
  RMDir "$SMPROGRAMS\Gym-Go"
  Delete "$DESKTOP\Gym-Go.lnk"
  
SectionEnd
