# 🎯 Cómo Usar Gym-Go

## Después de Instalar

### 🚀 Inicio Rápido

1. **Abre Gym-Go**
   - Haz doble clic en el icono de Gym-Go en tu escritorio
   - O busca "Gym-Go" en el menú inicio y haz clic en "Abrir Gym-Go"
   
2. **Listo**
   - Se abrirá tu navegador en `http://localhost:8080`
   - Si instalaste como servicio: El servidor ya está corriendo en segundo plano
   - Si NO instalaste como servicio: El script iniciará el servidor automáticamente

**El script es inteligente:**
- ✅ Si el servicio está corriendo → Solo abre el navegador
- ✅ Si ya hay una instancia del servidor → Solo abre el navegador
- ✅ Si nada está corriendo → Inicia el servidor y abre el navegador
- ✅ **NO requiere permisos de administrador**

### 📱 Uso Diario

**Para usar Gym-Go:**
- Solo abre el navegador en `http://localhost:8080`
- O usa el acceso directo "Abrir Gym-Go"

**No necesitas:**
- ❌ Permisos de administrador
- ❌ Abrir ventanas de consola
- ❌ Ejecutar comandos complicados

### 🔧 Gestión del Servicio

**Si instalaste como servicio:**

El servicio se gestiona automáticamente, pero si necesitas controlarlo:

**Desde el Menú Inicio → Gym-Go:**
- **Detener Gym-Go**: Detiene el servidor (proceso o servicio)
- **Detener Servicio (Admin)**: Detiene el servicio de Windows (requiere admin)
- **Reiniciar Servicio**: Reinicia el servicio (requiere admin)

**Desde PowerShell (como Administrador):**
```powershell
# Ver estado
sc query GymGoService

# Iniciar
sc start GymGoService
# O usar: net start GymGoService

# Detener
sc stop GymGoService
# O usar: net stop GymGoService

# Reiniciar
sc stop GymGoService
sc start GymGoService
```

**Si NO instalaste como servicio:**

El script "Abrir Gym-Go" inicia el servidor automáticamente cuando es necesario. Para detenerlo:
- Usa "Detener Gym-Go" del menú inicio
- O cierra el proceso desde el Administrador de tareas

### 🌐 Acceso desde Otros Dispositivos

Si instalaste Gym-Go en una PC y quieres acceder desde otras computadoras o tablets en la misma red:

1. Encuentra la IP de la PC donde está instalado:
   ```powershell
   ipconfig
   ```
   Busca la dirección IPv4 (ej: `192.168.1.100`)

2. Desde otros dispositivos, abre el navegador y ve a:
   ```
   http://192.168.1.100:8080
   ```

## ⚙️ Configuración Avanzada

### Cambiar el Puerto

Si el puerto 8080 está en uso, puedes cambiarlo:

1. Crea o edita: `C:\Program Files\Gym-Go\config.yaml`
   ```yaml
   server:
     port: 8081  # Tu nuevo puerto
   ```

2. Reinicia el servicio

### Backup de la Base de Datos

Tu base de datos está en:
```
C:\ProgramData\Gym-Go\gym-go.db
```

**Nota:** Esta ubicación tiene permisos de escritura para todos los usuarios, permitiendo que la aplicación funcione correctamente.

Para hacer backup:
- **Opción 1 (Recomendada):** Detén Gym-Go, copia el archivo `gym-go.db` a un lugar seguro, e inicia Gym-Go nuevamente
- **Opción 2:** Usa el script de backup incluido en el menú inicio
- **Importante:** Siempre detén el servicio antes de hacer backup para evitar corrupción de datos

### Desinstalar

1. Ve a "Agregar o quitar programas"
2. Busca "Gym-Go"
3. Haz clic en "Desinstalar"
4. El instalador preguntará si deseas conservar la base de datos

## 🆘 Problemas Comunes

### "Acceso denegado" al abrir Gym-Go

**No te preocupes, es normal.** El script intenta verificar el servicio pero no puede iniciarlo sin permisos de admin. Sin embargo:

✅ **El script inicia el servidor automáticamente de forma normal** (sin requerir admin)
✅ **No necesitas hacer nada especial**
✅ **El navegador se abrirá y todo funcionará correctamente**

Si quieres usar el servicio (para que inicie con Windows):
1. Abre PowerShell como Administrador
2. Ejecuta: `net start GymGoService`

### El navegador no abre la página

1. **Espera unos segundos** - El servidor tarda 2-3 segundos en iniciar

2. **Verifica que el proceso esté corriendo:**
   - Abre el Administrador de Tareas (Ctrl + Shift + Esc)
   - Busca "gym-go.exe" en la lista de procesos
   
3. **Si no está corriendo, usa el acceso directo:**
   - Menú Inicio → Gym-Go → Abrir Gym-Go

4. **Verifica manualmente:**
   - Abre el navegador
   - Ve a `http://localhost:8080`

### Puerto en uso

Si ves un error que el puerto 8080 está en uso:
- Detén el servicio: `sc stop GymGoService`
- Cambia el puerto (ver "Configuración Avanzada")
- Inicia el servicio: `sc start GymGoService`

### No puedo acceder desde otros dispositivos

1. Verifica que el firewall de Windows permita el puerto 8080:
   ```powershell
   # Como Administrador
   New-NetFirewallRule -DisplayName "Gym-Go" -Direction Inbound -LocalPort 8080 -Protocol TCP -Action Allow
   ```

2. Verifica que ambos dispositivos estén en la misma red

## 📞 Soporte

Para más ayuda, consulta la documentación completa en:
- `C:\Program Files\Gym-Go\docs\README.md`
- `C:\Program Files\Gym-Go\docs\DEPLOYMENT.md`
