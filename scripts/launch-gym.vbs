' Gym-Go Launcher - Inicia todos los servicios sin mostrar consolas
' y abre el navegador automaticamente

Dim WshShell, fso, installDir, gymExe, bioExe

Set WshShell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

' Detectar directorio de instalacion
installDir = fso.GetParentFolderName(WScript.ScriptFullName)
' Si estamos en scripts\, subir un nivel
If LCase(fso.GetFileName(installDir)) = "scripts" Then
    installDir = fso.GetParentFolderName(installDir)
End If

gymExe = installDir & "\gym-go.exe"
bioExe = installDir & "\biometric\BiometricPOC.exe"

' Verificar que existan los ejecutables
If Not fso.FileExists(gymExe) Then
    MsgBox "No se encontro gym-go.exe en:" & vbCrLf & gymExe, vbCritical, "Gym-Go"
    WScript.Quit 1
End If

' Verificar si ya estan corriendo
Dim gymRunning, bioRunning
gymRunning = IsProcessRunning("gym-go.exe")
bioRunning = IsProcessRunning("BiometricPOC.exe")

' Iniciar servicio biometrico (si existe y no esta corriendo)
If fso.FileExists(bioExe) And Not bioRunning Then
    WshShell.Run """" & bioExe & """", 0, False
    WScript.Sleep 2000
End If

' Iniciar backend Go (si no esta corriendo)
'
' ENVIRONMENT=production hace dos cosas: pone Gin en modo release y deja de enviar
' el detalle tecnico de los errores al navegador (sigue yendo al log). Sin esta
' variable la app instalada corria en modo desarrollo.
'
' No se carga ningun .env, asi que la unica forma de pasar configuracion es el
' entorno del proceso. Se define en el entorno del proceso hijo, no del sistema.
If Not gymRunning Then
    WshShell.Environment("PROCESS")("ENVIRONMENT") = "production"
    WshShell.Run """" & gymExe & """", 0, False
    WScript.Sleep 3000

    ' Si arranco y murio (por ejemplo, el puerto 8080 ocupado por otra instancia),
    ' avisar en vez de abrir el navegador contra la nada. El motivo exacto queda en
    ' gym-go.log, junto a la base de datos.
    If Not IsProcessRunning("gym-go.exe") Then
        MsgBox "Gym-Go no pudo iniciarse." & vbCrLf & vbCrLf & _
               "Lo mas habitual es que ya haya otra instancia corriendo o que el " & _
               "puerto 8080 este ocupado." & vbCrLf & vbCrLf & _
               "El detalle esta en:" & vbCrLf & _
               "%PROGRAMDATA%\Gym-Go\gym-go.log", vbExclamation, "Gym-Go"
        WScript.Quit 1
    End If
End If

' Abrir navegador
WshShell.Run "http://localhost:8080", 1, False

Function IsProcessRunning(processName)
    Dim objWMI, colProcesses
    Set objWMI = GetObject("winmgmts:\\.\root\cimv2")
    Set colProcesses = objWMI.ExecQuery("SELECT Name FROM Win32_Process WHERE Name='" & processName & "'")
    IsProcessRunning = (colProcesses.Count > 0)
End Function
