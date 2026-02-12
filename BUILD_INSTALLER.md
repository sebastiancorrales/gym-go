# 📦 Guía para Crear el Instalador de Gym-Go

Esta guía te ayudará a crear un instalador ejecutable (.exe) para Windows que empaqueta tanto el backend (Go) como el frontend (React).

## 📋 Requisitos Previos

### 1. **NSIS (Nullsoft Scriptable Install System)**
Descarga e instala NSIS desde: https://nsis.sourceforge.io/Download

- Versión recomendada: 3.x o superior
- Ruta de instalación por defecto: `C:\Program Files (x86)\NSIS\`

### 2. **Go**
- Versión 1.21 o superior
- Verificar: `go version`

### 3. **Node.js y npm**
- Versión 18 o superior
- Verificar: `node --version` y `npm --version`

## 🚀 Compilar el Instalador

### Opción 1: Script PowerShell (Recomendado)

```powershell
.\build-installer.ps1
```

### Opción 2: Script Batch

```cmd
build-installer.bat
```

### Opción 3: Manual

```powershell
# 1. Compilar backend
go build -o gym-go.exe -ldflags="-s -w" .

# 2. Compilar frontend
cd frontend
npm install
npm run build
cd ..

# 3. Crear instalador
"C:\Program Files (x86)\NSIS\makensis.exe" installer.nsi
```

## 📁 Estructura del Instalador

El instalador empaqueta los siguientes componentes:

```
Gym-Go-Installer.exe
├── Backend
│   ├── gym-go.exe (Servidor Go - se ejecuta como servicio)
│   └── config/
├── Frontend
│   └── frontend/dist/ (Build de React)
├── Migrations
│   └── migrations/*.sql
├── Documentación
│   ├── README.md
│   ├── DEPLOYMENT.md
│   └── LICENSE
└── Scripts
    ├── open-gym.bat (Abre navegador y verifica servicio)
    ├── stop-service.bat (Detiene servicio)
    ├── restart-service.bat (Reinicia servicio)
    └── uninstall-service.bat (Desinstala servicio)
```

## 🎯 Opciones de Instalación

El instalador ofrece 3 componentes:

### 1. **Gym-Go (Requerido)**
- Archivos principales del sistema
- Backend y frontend
- Migraciones de base de datos

### 2. **Crear Accesos Directos**
- Menú Inicio: Abrir Gym-Go, Detener Servicio, Reiniciar Servicio
- Opciones de gestión del servicio

### 3. **Instalar como Servicio de Windows (Recomendado)**
- ✅ Se instala automáticamente como servicio
- ✅ Se inicia con Windows
- ✅ Corre siempre en segundo plano (sin ventanas de consola)
- ✅ Solo necesitas abrir el navegador - el servidor ya está corriendo
- 🎯 **Funciona como una aplicación profesional**

## 🚀 Inicio Automático

Al finalizar la instalación, el instalador ofrece:

1. **Iniciar Gym-Go ahora** (marcado por defecto)
   - Si instalaste el servicio: Lo inicia automáticamente
   - Abre el navegador en `http://localhost:8080`
   - ¡Listo para usar en segundos!

2. **Crear acceso directo en el escritorio**
   - Acceso rápido desde el escritorio
   - Solo abre el navegador (el servicio ya está corriendo)
   - Si el servicio no está activo, lo inicia automáticamente

## 📍 Ubicación de Instalación por Defecto

```
C:\Program Files\Gym-Go\
├── gym-go.exe (Corre como servicio de Windows)
├── open-gym.bat
├── stop-service.bat
├── restart-service.bat
├── frontend\
│   └── dist\ (archivos del build)
├── migrations\
├── data\
│   └── gym.db (creado en primer uso)
└── docs\
```

## 🔧 Servicio de Windows

Cuando instalas como servicio:

- **Nombre del Servicio**: `GymGoService`
- **Nombre para Mostrar**: "Gym-Go Service"
- **Tipo de Inicio**: Automático (se inicia con Windows)
- **Estado**: En ejecución (siempre activo)

### Ventajas del Servicio:
- ✅ Sin ventanas de consola
- ✅ Inicia automáticamente con Windows
- ✅ Siempre disponible en segundo plano
- ✅ Experiencia de aplicación profesional
- ✅ Solo abres el navegador cuando lo necesitas

## 🔧 Uso de Gym-Go

### Si Instalaste como Servicio (Recomendado):

El servicio está **SIEMPRE corriendo en segundo plano**. Solo necesitas:

**Desde el Menú Inicio:**
- **Abrir Gym-Go**: Abre el navegador (el servidor ya está corriendo)
- **Detener Servicio**: Detiene el servicio de Windows
- **Reiniciar Servicio**: Reinicia el servicio (útil tras actualizaciones)

**Desde el Escritorio** (si creaste el acceso directo):
- Doble clic en "Gym-Go" → Abre el navegador

**Manualmente:**
```cmd
# Abrir navegador (verifica que el servicio esté corriendo)
C:\Program Files\Gym-Go\open-gym.bat

# Gestionar servicio
sc start GymGoService    # Iniciar
sc stop GymGoService     # Detener
sc query GymGoService    # Ver estado
```

### Si NO Instalaste como Servicio:

Necesitas iniciar el servidor manualmente cada vez:

```cmd
# Iniciar servidor
C:\Program Files\Gym-Go\gym-go.exe

# En otra ventana o después de iniciar
start http://localhost:8080
```

**💡 Recomendación:** Reinstala seleccionando "Instalar como Servicio de Windows" para una mejor experiencia.

## 🎨 Personalizar el Instalador

### Cambiar el Icono

Edita `installer.nsi`:

```nsis
!define MUI_ICON "ruta\a\tu\icono.ico"
!define MUI_UNICON "ruta\a\tu\icono-uninstall.ico"
```

### Cambiar el Nombre

```nsis
Name "Tu Nombre de App"
OutFile "Tu-Instalador.exe"
```

### Agregar Archivos Adicionales

```nsis
Section "Gym-Go (requerido)" SecCore
  ; ... código existente ...
  
  ; Tus archivos adicionales
  File "tu-archivo.txt"
  File /r "tu-carpeta\*.*"
SectionEnd
```

## � Arquitectura de Ubicación de Archivos

El instalador utiliza la estructura recomendada de Windows:

### Archivos de Programa:
```
C:\Program Files\Gym-Go\
├── gym-go.exe          (Ejecutable principal)
├── dist\               (Frontend compilado)
├── open-gym.bat        (Script de inicio)
├── stop-service.bat    (Detener servicio)
└── uninstall.exe       (Desinstalador)
```

### Datos de Aplicación:
```
C:\ProgramData\Gym-Go\
└── gym-go.db           (Base de datos SQLite)
```

**¿Por qué esta separación?**
- 🔒 **Program Files**: Solo lectura para usuarios normales (seguridad)
- ✍️ **ProgramData**: Tiene permisos de escritura para la base de datos
- 🎯 Esta arquitectura permite que el servicio de Windows funcione correctamente sin requerir permisos elevados

El ejecutable detecta automáticamente si está instalado en `Program Files` y ajusta la ruta de la base de datos a `ProgramData\Gym-Go\gym-go.db`.

## �📦 Reducir el Tamaño del Instalador

### 1. Compilar Go con flags de optimización:

```bash
go build -o gym-go.exe -ldflags="-s -w" .
```

Flags:
- `-s`: Elimina la tabla de símbolos
- `-w`: Elimina información de debug

### 2. Optimizar el build del frontend:

El `vite build` ya optimiza automáticamente:
- Minificación de JS/CSS
- Tree-shaking
- Compresión de assets

### 3. Comprimir el instalador con UPX (Opcional):

```bash
# Instalar UPX: https://upx.github.io/
upx --best gym-go.exe
```

## 🧪 Probar el Instalador

### En tu Máquina de Desarrollo:

```cmd
# Ejecutar el instalador
Gym-Go-Installer.exe

# Verificar la instalación
cd "C:\Program Files\Gym-Go"
start-server.bat
```

### En Máquina Virtual (Recomendado):

1. Crear una VM limpia de Windows
2. Instalar solo los requisitos del usuario final (nada de Go, Node, etc.)
3. Ejecutar el instalador
4. Probar todas las funcionalidades

## 🔍 Solución de Problemas

### Error: "NSIS no está instalado"
- Instalar NSIS desde https://nsis.sourceforge.io/Download
- Verificar que la ruta sea: `C:\Program Files (x86)\NSIS\makensis.exe`

### Error: "Falló la compilación del backend"
- Verificar que Go esté instalado: `go version`
- Revisar errores de compilación en la consola
- Asegurarse que todas las dependencias estén instaladas: `go mod download`

### Error: "Falló la compilación del frontend"
- Verificar que Node.js esté instalado: `node --version`
- Limpiar y reinstalar dependencias:
  ```bash
  cd frontend
  rm -rf node_modules package-lock.json
  npm install
  npm run build
  ```

### El instalador se crea pero es muy pequeño (< 1MB)
- Es probable que falten archivos
- Verificar que existan:
  - `gym-go.exe` en la raíz
  - `frontend/dist/` con archivos
  - `migrations/*.sql`

### Puerto 8080 ya en uso
- Cambiar el puerto en `config/config.go` antes de compilar
- O detener el proceso que usa el puerto 8080

## 📊 Tamaño Esperado del Instalador

- **Backend (gym-go.exe)**: ~15-25 MB
- **Frontend (dist/)**: ~2-5 MB
- **Total (instalador)**: ~20-35 MB

Si el instalador es significativamente más grande o pequeño, revisar los archivos incluidos.

## 🚀 Distribución

### Para Usuarios Finales:

1. Compartir solo: `Gym-Go-Installer.exe`
2. No se necesita Go, Node, ni herramientas de desarrollo
3. Solo ejecutar el instalador

### Para Actualizar:

1. Desinstalar versión anterior (opcional si se quiere limpiar)
2. Ejecutar nuevo instalador
3. Mantiene la base de datos en `C:\Program Files\Gym-Go\data\`

## 📝 Checklist Pre-Release

Antes de distribuir el instalador:

- [ ] Compilar en modo release (sin debug)
- [ ] Probar instalación en VM limpia
- [ ] Verificar que el servidor inicie correctamente
- [ ] Probar login con credenciales por defecto
- [ ] Verificar todas las funcionalidades principales
- [ ] Probar desinstalación completa
- [ ] Verificar que no queden archivos residuales
- [ ] Documentar versión y fecha de build
- [ ] Actualizar CHANGELOG.md

## 🔐 Firma Digital (Opcional pero Recomendado)

Para evitar advertencias de Windows SmartScreen:

1. Obtener certificado de firma de código
2. Firmar el ejecutable:
   ```cmd
   signtool sign /f certificado.pfx /p password /t http://timestamp.server.com gym-go.exe
   signtool sign /f certificado.pfx /p password /t http://timestamp.server.com Gym-Go-Installer.exe
   ```

## 📞 Soporte

Si encuentras problemas:
1. Revisar logs en `C:\Program Files\Gym-Go\logs\`
2. Verificar permisos de carpetas
3. Ejecutar como Administrador si es necesario

---

**¡Listo para distribuir tu aplicación! 🎉**
