' Gym-Go Stop - Detiene todos los servicios

Dim WshShell
Set WshShell = CreateObject("WScript.Shell")

' Detener gym-go.exe
WshShell.Run "taskkill /F /IM gym-go.exe", 0, True

' Detener BiometricService.exe
WshShell.Run "taskkill /F /IM BiometricService.exe", 0, True
