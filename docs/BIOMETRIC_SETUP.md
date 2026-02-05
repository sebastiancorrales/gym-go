# 📱 Guía de Integración del Lector Biométrico Digital Persona U.are.U 4500

Esta guía te ayudará a configurar e integrar el lector de huellas digitales Digital Persona U.are.U 4500 con tu sistema GYM-GO.

## 📋 Requisitos Previos

### Hardware
- ✅ Lector de huellas **Digital Persona U.are.U 4500** (o U.are.U 4500 Reader)
- ✅ Puerto USB disponible
- ✅ PC con Windows 10/11 (recomendado)

### Software
- ✅ Go 1.21 o superior
- ✅ Node.js 18+ (para el frontend)
- ✅ Git

## 🔧 Instalación del Hardware y Drivers

### Paso 1: Instalar Drivers del Dispositivo

1. **Descargar el SDK de DigitalPersona**
   - Visita: https://www.crossmatch.com/ o https://www.hidglobal.com/
   - Busca "DigitalPersona U.are.U SDK" o "HID DigitalPersona SDK"
   - Descarga la versión más reciente compatible con Windows

2. **Instalar el SDK**
   ```bash
   # Ejecuta el instalador descargado
   # Típicamente: DigitalPersona_UareU_SDK_x.x.x.exe
   # Acepta los términos y condiciones
   # Instala en la ruta por defecto: C:\Program Files\DigitalPersona\
   ```

3. **Verificar la instalación**
   - Conecta el lector U.are.U 4500 al puerto USB
   - Abre el **Administrador de dispositivos** de Windows
   - Busca en "Dispositivos biométricos" o "Human Interface Devices"
   - Deberías ver: "DigitalPersona U.are.U 4500 Fingerprint Reader"

4. **Probar el dispositivo**
   - El SDK incluye una aplicación de prueba: `UareUSampleJava` o `SampleApp`
   - Ejecútala para verificar que el lector funciona correctamente
   - Intenta capturar una huella de prueba

### Paso 2: Instalar Dependencias de Go

El SDK de DigitalPersona proporciona bibliotecas nativas en C/C++. Para usarlas en Go, necesitas CGO o una biblioteca de bindings.

#### Opción A: Usar CGO (Recomendado para producción)

1. **Instalar MinGW-w64 (compilador C para Windows)**
   ```powershell
   # Usando Chocolatey
   choco install mingw
   
   # O descarga desde: https://www.mingw-w64.org/
   ```

2. **Configurar variables de entorno**
   ```powershell
   # Agregar al PATH la ruta del SDK
   $env:CGO_CFLAGS = "-IC:\Program Files\DigitalPersona\U.are.U SDK\Include"
   $env:CGO_LDFLAGS = "-LC:\Program Files\DigitalPersona\U.are.U SDK\Lib"
   ```

#### Opción B: Crear un servicio wrapper en C# (Alternativa más simple)

Si CGO es complicado, puedes crear un pequeño servicio en C#/.NET que use el SDK y se comunique con Go via HTTP/gRPC.

## 🚀 Configuración del Sistema

### Paso 3: Ejecutar Migraciones de Base de Datos

```bash
# Desde el directorio raíz del proyecto
cd c:\Proyectos\gym-go

# Las migraciones se ejecutan automáticamente al iniciar el servidor
# Pero también puedes ejecutarlas manualmente:

# Verificar que existe la migración de biometría
ls migrations\002_biometric_system.sql
```

### Paso 4: Configurar el Backend

```bash
# Instalar dependencias de Go
go mod tidy

# Compilar el proyecto
go build -o gym-go.exe main.go
```

### Paso 5: Iniciar el Servidor

```bash
# Iniciar el servidor backend
.\gym-go.exe

# Deberías ver:
# ✅ Server running on http://localhost:8080
# ✅ Biometric services initialized
```

### Paso 6: Configurar el Frontend

```bash
# Entrar al directorio del frontend
cd frontend

# Instalar dependencias
npm install

# Compilar para producción
npm run build

# O iniciar en modo desarrollo
npm run dev
```

## 🔌 Implementación del SDK (Código Nativo)

### Plantilla de Integración en Go

Crea el archivo: `internal/infrastructure/biometric/digitalpersona.go`

```go
package biometric

// #cgo CFLAGS: -I"C:/Program Files/DigitalPersona/U.are.U SDK/Include"
// #cgo LDFLAGS: -L"C:/Program Files/DigitalPersona/U.are.U SDK/Lib" -ldpfpdd -ldpfj
// #include <dpfpdd.h>
// #include <dpfj.h>
import "C"
import (
    "errors"
    "unsafe"
)

type DigitalPersonaReader struct {
    deviceHandle C.DPFPDD_DEV
}

func NewDigitalPersonaReader() (*DigitalPersonaReader, error) {
    // Inicializar el SDK
    result := C.dpfpdd_init()
    if result != C.DPFPDD_SUCCESS {
        return nil, errors.New("failed to initialize DigitalPersona SDK")
    }
    
    return &DigitalPersonaReader{}, nil
}

func (r *DigitalPersonaReader) CheckDeviceStatus() (bool, error) {
    var count C.unsigned_int
    result := C.dpfpdd_query_devices(&count)
    
    if result != C.DPFPDD_SUCCESS {
        return false, errors.New("failed to query devices")
    }
    
    return count > 0, nil
}

func (r *DigitalPersonaReader) CaptureFinger() ([]byte, int, error) {
    // Abrir el primer dispositivo
    result := C.dpfpdd_open(0, &r.deviceHandle)
    if result != C.DPFPDD_SUCCESS {
        return nil, 0, errors.New("failed to open device")
    }
    defer C.dpfpdd_close(r.deviceHandle)
    
    // Capturar huella
    var captureResult C.DPFPDD_CAPTURE_RESULT
    var image C.DPFPDD_CAPTURE_DATA
    
    // Implementar lógica de captura...
    // (Ver documentación del SDK para detalles completos)
    
    return nil, 0, errors.New("not implemented")
}

func (r *DigitalPersonaReader) Cleanup() {
    C.dpfpdd_exit()
}
```

### Alternativa: Servicio C# (Más simple para empezar)

Si CGO es complicado, crea un pequeño servicio en C#:

```csharp
// BiometricService.cs
using DPUruNet;
using System;
using System.Net;
using System.Net.Sockets;
using System.Text;

namespace BiometricService
{
    class Program
    {
        static void Main()
        {
            // Iniciar servidor TCP en puerto 9000
            TcpListener listener = new TcpListener(IPAddress.Loopback, 9000);
            listener.Start();
            
            Console.WriteLine("Biometric Service listening on port 9000...");
            
            while (true)
            {
                TcpClient client = listener.AcceptTcpClient();
                // Manejar comandos: CAPTURE, VERIFY, STATUS
                HandleClient(client);
            }
        }
        
        static void HandleClient(TcpClient client)
        {
            // Implementar comunicación con Go via TCP
            // Comandos: {"command": "capture"}, {"command": "status"}, etc.
        }
    }
}
```

Luego modifica el servicio Go para comunicarse con este servicio:

```go
// En biometric_usecase.go
func (s *BiometricService) CaptureFingerprint(ctx context.Context) ([]byte, int, error) {
    // Conectar al servicio C# en localhost:9000
    conn, err := net.Dial("tcp", "localhost:9000")
    if err != nil {
        return nil, 0, err
    }
    defer conn.Close()
    
    // Enviar comando
    json.NewEncoder(conn).Encode(map[string]string{"command": "capture"})
    
    // Recibir respuesta
    var response struct {
        Success  bool   `json:"success"`
        Template string `json:"template"`
        Quality  int    `json:"quality"`
    }
    json.NewDecoder(conn).Decode(&response)
    
    if !response.Success {
        return nil, 0, errors.New("capture failed")
    }
    
    template, _ := base64.StdEncoding.DecodeString(response.Template)
    return template, response.Quality, nil
}
```

## 🧪 Pruebas

### Probar el estado del lector

```bash
# Usando curl o Postman
curl -X GET http://localhost:8080/api/v1/biometric/status \
  -H "Authorization: Bearer YOUR_TOKEN"

# Respuesta esperada:
# {
#   "success": true,
#   "data": {
#     "reader_connected": true,
#     "reader_model": "DigitalPersona U.are.U 4500",
#     "status": "connected"
#   }
# }
```

### Registrar una huella

```bash
# 1. Primero captura la huella
curl -X POST http://localhost:8080/api/v1/biometric/capture \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"timeout": 30}'

# 2. Luego registra la huella para un usuario
curl -X POST http://localhost:8080/api/v1/biometric/enroll \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "finger_index": "right_index",
    "template_data": "BASE64_TEMPLATE_FROM_CAPTURE",
    "quality": 85
  }'
```

### Verificar acceso con huella

```bash
curl -X POST http://localhost:8080/api/v1/biometric/verify \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_data": "BASE64_TEMPLATE",
    "device_id": "front-desk-reader"
  }'
```

## 📝 Configuración en la Interfaz Web

1. **Accede al sistema** como administrador
2. **Navega a** `/check-in`
3. **Verifica** que el indicador del lector muestre "Conectado"
4. **Selecciona** el método "Huella"
5. **Haz clic** en "Iniciar Escaneo"
6. **Coloca** tu dedo en el lector

## 🐛 Troubleshooting

### Problema: "Lector no disponible"

**Solución:**
```powershell
# Verificar que el dispositivo está conectado
Get-PnpDevice | Where-Object {$_.FriendlyName -like "*DigitalPersona*"}

# Reiniciar el servicio del dispositivo
Restart-Service "nombre_del_servicio"
```

### Problema: "Fingerprint capture not implemented"

**Solución:**
- Necesitas implementar la integración con el SDK nativo
- Sigue las instrucciones en "Implementación del SDK" arriba
- O usa la alternativa del servicio C#

### Problema: Calidad de captura baja

**Solución:**
- Limpia el sensor del lector con un paño suave
- Asegúrate de que el dedo esté limpio y seco
- Presiona firmemente pero sin exceso
- Ajusta el umbral de calidad en el código (actualmente 50%)

## 📚 Recursos Adicionales

- [Documentación SDK DigitalPersona](https://www.crossmatch.com/documentation/)
- [Guía de integración U.are.U SDK](https://www.hidglobal.com/support)
- [Foro de desarrolladores](https://community.hidglobal.com/)

## 🔒 Consideraciones de Seguridad

1. **Nunca almacenes imágenes de huellas**, solo templates (minutiae)
2. **Usa HTTPS** en producción para transmitir datos biométricos
3. **Implementa rate limiting** en los endpoints de captura
4. **Registra todos los intentos** de verificación para auditoría
5. **Cumple con regulaciones** locales de protección de datos biométricos

## 📞 Soporte

Si tienes problemas con la integración:
1. Revisa los logs del servidor: `logs/gym-go.log`
2. Verifica el estado del dispositivo en el Administrador de dispositivos
3. Consulta la documentación oficial del SDK
4. Contacta al soporte técnico de DigitalPersona/HID Global

---

**Nota**: Esta es una implementación base. Para producción, deberás:
- Implementar el código nativo de captura y verificación
- Agregar manejo robusto de errores
- Implementar timeouts y reintentos
- Agregar logs detallados
- Realizar pruebas exhaustivas de seguridad y rendimiento
