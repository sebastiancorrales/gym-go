# 🚀 Frontend React + Tailwind CSS

## ✅ Completado

Se ha creado un frontend moderno con:
- ✅ React 18 (JavaScript sin TypeScript)
- ✅ Vite como build tool
- ✅ Tailwind CSS para estilos
- ✅ Conexión a API REST en localhost:8080
- ✅ Sin Wails (frontend web puro)

## 🌐 Acceso

**Frontend React**: http://localhost:5173
**Backend API**: http://localhost:8080

## 📂 Estructura

```
frontend/
├── src/
│   ├── components/
│   │   ├── MembersTab.jsx      # Gestión de miembros
│   │   ├── ClassesTab.jsx      # Gestión de clases
│   │   └── AttendanceTab.jsx   # Registro de asistencias
│   ├── App.jsx                  # Componente principal
│   ├── main.jsx                 # Entry point
│   └── index.css                # Tailwind imports
├── tailwind.config.js           # Configuración Tailwind
├── postcss.config.js            # PostCSS config
└── package.json                 # Dependencias npm
```

## 🎯 Funcionalidades

### Tab Miembros
- ✅ Registrar nuevos miembros
- ✅ Ver lista de miembros
- ✅ Formulario con validación
- ✅ Tabla con datos en tiempo real

### Tab Clases
- ✅ Crear nuevas clases
- ✅ Ver clases disponibles
- ✅ Definir horarios y capacidad
- ✅ Asignar instructores

### Tab Asistencias
- ✅ Registrar check-in de miembros
- ✅ Seleccionar miembro y clase desde dropdown
- ✅ Ver historial completo
- ✅ Estados: Presente, Ausente, Tarde

## 🔧 Comandos

### Iniciar Desarrollo

```bash
# Terminal 1: Backend Go
cd C:\Proyectos\gym-go
go run main.go

# Terminal 2: Frontend React
cd C:\Proyectos\gym-go\frontend
npm run dev
```

### Build para Producción

```bash
cd frontend
npm run build
```

Los archivos compilados estarán en `frontend/dist/`

### Instalar Dependencias (si es necesario)

```bash
cd frontend
npm install
```

## 🎨 Características de UI

- **Diseño Responsivo**: Mobile-first con Tailwind
- **Indicador de API**: Muestra estado en tiempo real (cada 10s)
- **Tabs Interactivos**: Navegación fluida entre módulos
- **Mensajes Flash**: Feedback visual de operaciones
- **Tablas Modernas**: Listado de datos con styling profesional
- **Formularios**: Validación HTML5 y estilos consistentes

## 🔄 Flujo de Datos

```
React Component → fetch() → API REST (Go/Gin) → Repository → In-Memory Storage
                  ↓
              JSON Response
                  ↓
              State Update
                  ↓
              Re-render UI
```

## 📡 API Endpoints Utilizados

```javascript
// Miembros
GET    /api/v1/members
POST   /api/v1/members

// Clases
GET    /api/v1/classes
POST   /api/v1/classes

// Asistencias
GET    /api/v1/attendance
POST   /api/v1/attendance

// Health Check
GET    /health
```

## 🐛 Troubleshooting

### El frontend no carga
1. Verifica que Vite esté corriendo: `npm run dev`
2. Abre http://localhost:5173 en tu navegador
3. Revisa la consola del navegador (F12)

### API no responde
1. Verifica que Go esté corriendo: `go run main.go`
2. Prueba http://localhost:8080/health
3. Revisa los logs del terminal

### Errores de CORS
El backend ya tiene CORS configurado en `internal/infrastructure/http/middleware/cors.go`

### No se ven los estilos de Tailwind
1. Verifica que `index.css` tenga los imports de Tailwind
2. Recarga la página con Ctrl+F5
3. Borra caché del navegador

## 🚀 Próximos Pasos

- [ ] Agregar loading states
- [ ] Implementar paginación en tablas
- [ ] Agregar búsqueda/filtros
- [ ] Modal para editar registros
- [ ] Confirmación antes de eliminar
- [ ] Dashboard con estadísticas
- [ ] Gráficos con Chart.js
- [ ] Dark mode toggle

## 📦 Dependencias

```json
{
  "react": "^19.0.0",
  "react-dom": "^19.0.0",
  "tailwindcss": "^3.4.17",
  "vite": "^7.3.1"
}
```

## 💡 Tips

- **Hot Module Replacement**: Guarda archivos y ve cambios al instante
- **Componentes reutilizables**: Cada tab es un componente independiente
- **Tailwind IntelliSense**: Instala la extensión de VS Code para autocompletado
- **React DevTools**: Usa la extensión de navegador para debug

## 🔗 Archivos Importantes

- [App.jsx](src/App.jsx) - Layout principal y navegación de tabs
- [MembersTab.jsx](src/components/MembersTab.jsx) - CRUD de miembros
- [ClassesTab.jsx](src/components/ClassesTab.jsx) - CRUD de clases
- [AttendanceTab.jsx](src/components/AttendanceTab.jsx) - Sistema de check-in
- [tailwind.config.js](tailwind.config.js) - Configuración de Tailwind

---

**✨ Frontend listo y funcionando en http://localhost:5173**
