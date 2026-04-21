' Gym-Go Stop - Detiene todos los servicios

Dim WshShell
Set WshShell = CreateObject("WScript.Shell")

' Detener gym-go.exe
WshShell.Run "taskkill /F /IM gym-go.exe", 0, True

' Detener BiometricPOC.exe
WshShell.Run "taskkill /F /IM BiometricPOC.exe", 0, True
