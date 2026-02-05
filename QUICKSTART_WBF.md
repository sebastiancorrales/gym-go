# 🚀 Inicio Rápido - Windows Biometric Framework

Tu lector **U.are.U 4500** está funcionando con Windows Biometric Framework (WBF). Esto significa que puedes usarlo sin instalar el SDK de DigitalPersona.

## ✅ Lo que ya tienes:

```
✅ Lector U.are.U 4500 conectado (WBF)
✅ Backend Go completo con API biométrica
✅ Frontend React con UI de huellas
✅ Base de datos configurada
✅ Go y Node.js instalados
```

## 🔧 Dos opciones para empezar:

### Opción 1: Servicio C# (RECOMENDADO) ⭐

El servicio C# usa la API nativa de Windows (WinBio) y funciona directamente con tu lector.

**Pasos:**

```powershell
# 1. Crear proyecto .NET
cd docs
dotnet new console -n BiometricServiceWBF
cd BiometricServiceWBF

# 2. Reemplazar Program.cs con el contenido de BiometricServiceWBF.cs
Copy-Item ..\BiometricServiceWBF.cs .\Program.cs -Force

# 3. Compilar
dotnet build -c Release

# 4. Ejecutar el servicio
dotnet run

# Deberías ver:
# ✅ Windows Biometric Framework disponible
# ✅ Servicio escuchando en localhost:9000
```

### Opción 2: Modo de Desarrollo (SIN HARDWARE)

Para desarrollo/pruebas sin integración real del lector:

```powershell
# Simplemente inicia tu aplicación
go build -o gym-go.exe main.go
.\gym-go.exe

# El sistema funcionará pero mostrará "Lector desconectado"
# Podrás usar solo el método manual de check-in
```

## 🎯 Prueba rápida del servicio C#:

Una vez que el servicio C# esté corriendo, pruébalo:

```powershell
# En otra terminal PowerShell
$client = New-Object System.Net.Sockets.TcpClient("localhost", 9000)
$stream = $client.GetStream()
$writer = New-Object System.IO.StreamWriter($stream)
$reader = New-Object System.IO.StreamReader($stream)

# Enviar comando de estado
$writer.WriteLine('{"command":"status"}')
$writer.Flush()

# Leer respuesta
$response = $reader.ReadLine()
Write-Host $response

$client.Close()
```

## 🔗 Integración con tu app Go:

El código Go ya está listo para comunicarse con el servicio. Solo actualiza el servicio biométrico:

```go
// En internal/usecases/biometric_usecase.go
// La función CaptureFingerprint() se conectará automáticamente
// al servicio C# en localhost:9000
```

## 📝 Flujo completo:

```
1. Usuario en check-in → Selecciona "Huella"
2. Frontend React → POST /api/v1/biometric/capture
3. Backend Go → Conecta TCP localhost:9000
4. Servicio C# → Captura huella del dispositivo WBF
5. C# → Retorna template base64
6. Go → Guarda/verifica en BD
7. Frontend → Muestra resultado
```

## 🐛 Solución de problemas:

**"Windows Biometric Framework no disponible"**
```powershell
# Habilitar Windows Hello
# Configuración → Cuentas → Opciones de inicio de sesión → Windows Hello
```

**"No biometric devices found"**
```powershell
# Verificar dispositivo
Get-PnpDevice | Where-Object {$_.FriendlyName -like "*Fingerprint*"}

# Debe mostrar "OK" en Status
```

**Puerto 9000 en uso**
```powershell
# Ver qué está usando el puerto
netstat -ano | findstr :9000

# Cambiar puerto en BiometricServiceWBF.cs si es necesario
```

## 🎬 Comenzar AHORA:

```powershell
# Terminal 1: Servicio biométrico
cd docs
dotnet new console -n BiometricServiceWBF
cd BiometricServiceWBF
Copy-Item ..\BiometricServiceWBF.cs .\Program.cs -Force
dotnet run

# Terminal 2: Backend Go
cd c:\Proyectos\gym-go
go run main.go

# Terminal 3: Frontend (si es necesario)
cd c:\Proyectos\gym-go\frontend
npm run dev

# Acceder a: http://localhost:8080/check-in
```

## 📚 Siguiente paso:

1. **Ahora**: Compila y ejecuta el servicio C#
2. **Luego**: Inicia tu app Go
3. **Finalmente**: Prueba el check-in con huella en la web

¿Necesitas ayuda con algún paso? Solo pregunta! 🚀
