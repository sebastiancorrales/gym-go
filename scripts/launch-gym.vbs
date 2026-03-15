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
bioExe = installDir & "\biometric\BiometricService.exe"

' Verificar que existan los ejecutables
If Not fso.FileExists(gymExe) Then
    MsgBox "No se encontro gym-go.exe en:" & vbCrLf & gymExe, vbCritical, "Gym-Go"
    WScript.Quit 1
End If

' Verificar si ya estan corriendo
Dim gymRunning, bioRunning
gymRunning = IsProcessRunning("gym-go.exe")
bioRunning = IsProcessRunning("BiometricService.exe")

' Iniciar servicio biometrico (si existe y no esta corriendo)
If fso.FileExists(bioExe) And Not bioRunning Then
    WshShell.Run """" & bioExe & """", 0, False
    WScript.Sleep 2000
End If

' Iniciar backend Go (si no esta corriendo)
If Not gymRunning Then
    WshShell.Run """" & gymExe & """", 0, False
    WScript.Sleep 3000
End If

' Abrir navegador
WshShell.Run "http://localhost:8080", 1, False

Function IsProcessRunning(processName)
    Dim objWMI, colProcesses
    Set objWMI = GetObject("winmgmts:\\.\root\cimv2")
    Set colProcesses = objWMI.ExecQuery("SELECT Name FROM Win32_Process WHERE Name='" & processName & "'")
    IsProcessRunning = (colProcesses.Count > 0)
End Function
