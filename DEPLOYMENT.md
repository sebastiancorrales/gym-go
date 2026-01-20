# 🚀 Guía Completa de Deployment

## 📦 Para Uso LOCAL (.exe Windows)

### Usa: **Wails**
```bash
# Instalar
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Crear proyecto
wails init -n gym-desktop -t react

# Compilar
wails build
# Resultado: gym-desktop.exe
```

**Ventajas:**
- ✅ Un solo .exe para distribuir
- ✅ No requiere instalación
- ✅ Interfaz nativa

---

## 🌐 Para SERVIDOR (Internet)

### **Opción 1: API + Frontend Separados** ⭐ (Recomendado)

#### Backend (API Go):
```bash
# Compilar para Linux
GOOS=linux GOARCH=amd64 go build -o gym-api main.go

# Subir a servidor
scp gym-api user@servidor:/app/
ssh user@servidor
./gym-api
```

**Hostings recomendados:**
- Railway.app (gratis)
- Render.com (gratis)
- DigitalOcean ($5/mes)
- Fly.io (gratis)

#### Frontend (web/index.html):
```bash
# Subir a hosting estático GRATIS:
netlify deploy --prod
# o
vercel deploy --prod
```

**Cambiar API_URL en index.html:**
```javascript
const API_URL = 'https://tu-api.railway.app/api/v1';
```

---

### **Opción 2: Todo Integrado (Go sirve frontend)** 

Usar `main-with-frontend.go` que te creé:

```bash
# Compilar con frontend embebido
go build -o gym-api main-with-frontend.go

# Un solo binario para desplegar
# Frontend en: http://servidor:8080/app/
# API en: http://servidor:8080/api/v1/
```

**Ventajas:**
- ✅ Un solo servidor
- ✅ Un solo binario
- ✅ Más fácil de mantener

**Desventajas:**
- ❌ CDN no optimizado para frontend
- ❌ Menos escalable

---

## 📋 Resumen

| Escenario | Solución | Costo |
|-----------|----------|-------|
| **Escritorio Windows** | Wails | Gratis |
| **Servidor Web Simple** | Go embed + Railway | Gratis |
| **Servidor Web Escalable** | Go API + Vercel frontend | Gratis |
| **Producción Enterprise** | Go API + React + CDN | Variable |

---

## 🎯 Mi Recomendación para tu Caso:

1. **Desarrollo/Testing Local:** Usa `web/index.html` (actual)
2. **Distribución Escritorio:** Migra a Wails
3. **Servidor Web:** Railway (backend) + Netlify (frontend)

¿Quieres que te ayude a configurar alguna de estas opciones?
