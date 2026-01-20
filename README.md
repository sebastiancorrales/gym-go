# Gym Management System API

Sistema de gestión de gimnasio construido con arquitectura limpia y principios SOLID en Go.

## 🏗️ Arquitectura

Este proyecto sigue una **arquitectura limpia (Clean Architecture)** que separa las responsabilidades en capas:

```
gym-go/
├── cmd/                          # Punto de entrada de la aplicación
├── config/                       # Configuración de la aplicación
├── internal/
│   ├── domain/                  # Capa de dominio (Entidades y lógica de negocio)
│   │   ├── entities/           # Entidades del dominio
│   │   └── repositories/       # Interfaces de repositorios (DIP)
│   ├── usecases/               # Casos de uso (Lógica de aplicación)
│   └── infrastructure/         # Capa de infraestructura
│       ├── http/
│       │   ├── handlers/      # Controladores HTTP
│       │   ├── middleware/    # Middlewares HTTP
│       │   └── dto/           # Data Transfer Objects
│       └── persistence/       # Implementaciones de repositorios
└── pkg/                        # Paquetes compartidos
    ├── errors/                 # Errores personalizados
    └── utils/                  # Utilidades

```

## 🎯 Principios SOLID Aplicados

### Single Responsibility Principle (SRP)
- Cada entidad, caso de uso y handler tiene una única responsabilidad
- Los handlers solo manejan peticiones HTTP
- Los casos de uso solo contienen lógica de negocio
- Los repositorios solo manejan persistencia

### Open/Closed Principle (OCP)
- Las entidades pueden extenderse sin modificar código existente
- Los middlewares se pueden agregar sin modificar handlers
- Nuevos casos de uso se pueden agregar sin modificar los existentes

### Liskov Substitution Principle (LSP)
- Las implementaciones de repositorios son intercambiables
- Se pueden usar repositorios en memoria o PostgreSQL sin cambiar código

### Interface Segregation Principle (ISP)
- Interfaces de repositorios específicas para cada entidad
- DTOs separados por funcionalidad

### Dependency Inversion Principle (DIP)
- Los casos de uso dependen de interfaces, no de implementaciones concretas
- Inyección de dependencias en toda la aplicación
- Los handlers dependen de casos de uso, no de repositorios directamente

## 🚀 Características

- ✅ Gestión de miembros (crear, actualizar, activar, suspender)
- ✅ Gestión de clases (crear, iniciar, completar, cancelar)
- ✅ Sistema de asistencia (check-in/check-out)
- ✅ Asignación de membresías
- ✅ Middlewares (logging, CORS, recovery)
- ✅ Manejo de errores centralizado
- ✅ Respuestas estandarizadas

## 📦 Instalación

```bash
# Clonar el repositorio
git clone <repository-url>
cd gym-go

# Instalar dependencias
go mod download

# Ejecutar la aplicación
go run main.go
```

## 🔧 Configuración

La aplicación se configura mediante variables de entorno:

```bash
# Servidor
SERVER_HOST=localhost
SERVER_PORT=8080

# Base de datos (cuando se implemente)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=gym_db
DB_SSL_MODE=disable
```

## 📚 API Endpoints

### Miembros
- `POST /api/v1/members` - Crear miembro
- `GET /api/v1/members/{id}` - Obtener miembro
- `PUT /api/v1/members/{id}` - Actualizar miembro
- `POST /api/v1/members/{id}/membership` - Asignar membresía
- `POST /api/v1/members/{id}/suspend` - Suspender miembro
- `POST /api/v1/members/{id}/activate` - Activar miembro

### Clases
- `POST /api/v1/classes` - Crear clase
- `GET /api/v1/classes/{id}` - Obtener clase
- `POST /api/v1/classes/{id}/start` - Iniciar clase
- `POST /api/v1/classes/{id}/complete` - Completar clase
- `POST /api/v1/classes/{id}/cancel` - Cancelar clase

### Asistencia
- `POST /api/v1/attendance/checkin` - Hacer check-in
- `POST /api/v1/attendance/{member_id}/checkout` - Hacer check-out

### Health Check
- `GET /health` - Estado del servidor

## 🧪 Testing

```bash
# Ejecutar tests
go test ./...

# Con cobertura
go test -cover ./...
```

## 🔄 Próximos Pasos

1. **Base de Datos**
   - Implementar migraciones con `golang-migrate`
   - Configurar PostgreSQL
   - Completar implementación de repositorios

2. **Autenticación y Autorización**
   - Implementar JWT
   - Roles y permisos
   - Middleware de autenticación

3. **Testing**
   - Tests unitarios para casos de uso
   - Tests de integración para handlers
   - Mocks de repositorios

4. **Mejoras**
   - Validación de DTOs con `go-playground/validator`
   - Documentación con Swagger
   - Rate limiting
   - Métricas y monitoreo

## 📖 Estructura de Capas

### Domain Layer (Capa de Dominio)
Contiene las entidades y reglas de negocio core. No depende de ninguna otra capa.

### Use Cases Layer (Capa de Aplicación)
Orquesta el flujo de datos entre el dominio y la infraestructura. Implementa los casos de uso específicos.

### Infrastructure Layer (Capa de Infraestructura)
Implementa detalles técnicos como persistencia, HTTP, etc. Depende del dominio pero el dominio no depende de ella.

## 🤝 Contribuir

Las contribuciones son bienvenidas. Por favor:
1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📄 Licencia

Este proyecto está bajo la licencia MIT.
