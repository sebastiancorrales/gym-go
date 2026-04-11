# Backend GO Contexto Compacto

## Objetivo

Este archivo sirve para darle contexto a una IA con pocos tokens.

Importante:
- El backend real actual no coincide 100% con todos los `.md` grandes.
- `docs/ARCHITECTURE.md`, `docs/DATABASE_DESIGN.md` y `docs/API_ENDPOINTS.md` mezclan estado actual + roadmap SaaS futuro.
- La fuente de verdad del funcionamiento actual es principalmente `main.go`, `internal/usecases`, `internal/domain` y `internal/infrastructure`.

---

## Resumen Ultra Corto

Proyecto de gimnasio con backend monolítico en Go usando Gin + GORM + SQLite.

Arquitectura real:
- `internal/domain`: entidades y contratos de repositorio.
- `internal/usecases`: lógica de negocio.
- `internal/infrastructure/http`: handlers, middleware y DTOs.
- `internal/infrastructure/persistence`: repositorios SQLite y algunos repositorios en memoria.

Módulos activos:
- auth con JWT
- usuarios
- gym/configuración
- planes
- suscripciones
- control de acceso/check-in
- biometría/huellas
- clases
- asistencia
- inventario/productos
- métodos de pago
- ventas
- notificaciones por email
- uploads

Persistencia real actual:
- SQLite local (`gym-go.db`)
- migraciones con `AutoMigrate`
- no PostgreSQL multi-tenant en producción actual del repo

Seguridad actual:
- JWT access + refresh
- middleware por rol
- bloqueo temporal tras 5 intentos fallidos

Roles:
- `SUPER_ADMIN`
- `ADMIN_GYM`
- `RECEPCIONISTA`
- `STAFF`
- `MEMBER`

Frontend:
- servido por el mismo binario desde `frontend/dist` o embebido con `embed`

---

## Diferencia Entre Documentación y Código Real

Lo que sí existe hoy en código:
- monolito Go
- Gin
- GORM
- SQLite
- JWT
- rutas agrupadas por módulo
- repositorios SQLite
- expiración automática de suscripciones cada hora
- frontend servido por el backend

Lo que aparece documentado pero se ve más como roadmap o diseño futuro:
- multi-tenancy por schema
- PostgreSQL principal
- Redis
- Nginx
- cola de mensajes
- MinIO/S3
- particionado avanzado
- hardware edge/offline enterprise

Si una IA va a proponer cambios, debe asumir primero el stack real actual, no el roadmap.

---

## Flujo Real de Arquitectura

Flujo típico:
1. `main.go` crea config, DB, migraciones y seeds.
2. Inicializa repositorios.
3. Inicializa use cases.
4. Inicializa handlers HTTP.
5. Registra rutas Gin con middleware JWT y roles.
6. Atiende API y frontend SPA desde el mismo proceso.

Patrón dominante:
`handler -> usecase -> repository -> SQLite`

---

## Capas

### 1. Domain

Ubicación:
- `internal/domain/entities`
- `internal/domain/repositories`

Responsabilidad:
- entidades de negocio
- enums/estados/roles
- reglas simples dentro de las entidades
- interfaces de persistencia

Entidades principales:
- `User`
- `Gym`
- `Plan`
- `Subscription`
- `SubscriptionMember`
- `SubscriptionAuditLog`
- `AccessLog`
- `Fingerprint`
- `Product`
- `Sale`
- `SaleDetail`
- `SalePaymentMethod`
- `Class`
- `Attendance`

### 2. Use Cases

Ubicación:
- `internal/usecases`

Responsabilidad:
- reglas de negocio y coordinación entre repositorios

Use cases activos:
- `UserUseCase`
- `PlanUseCase`
- `SubscriptionUseCase`
- `AccessUseCase`
- `BiometricService`
- `ProductUseCase`
- `PaymentMethodUseCase`
- `SaleUseCase`
- `ClassUseCase`
- `AttendanceUseCase`

### 3. Infrastructure HTTP

Ubicación:
- `internal/infrastructure/http/handlers`
- `internal/infrastructure/http/middleware`
- `internal/infrastructure/http/dto`

Responsabilidad:
- parseo request/response
- validación básica
- auth JWT
- autorización por roles

### 4. Persistence

Ubicación:
- `internal/infrastructure/persistence`

Responsabilidad:
- implementación concreta de repositorios
- mayormente SQLite con GORM
- algunos módulos legacy/demo siguen in-memory

Repos mixtos:
- SQLite para usuarios, suscripciones, acceso, planes, huellas, inventario, ventas, métodos de pago, gym
- In-memory para `member` e `instructor`

---

## Estado Técnico Actual

### Stack real

- Go
- Gin
- GORM
- SQLite con driver `modernc.org/sqlite`
- JWT propio en `pkg/security`
- bcrypt para passwords

### Configuración

Archivo clave:
- `internal/config/config.go`

Variables importantes:
- `SERVER_HOST`
- `SERVER_PORT`
- `ENVIRONMENT`
- `DATABASE_PATH`
- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM`

### Base de datos

Archivo clave:
- `internal/config/database.go`

Notas:
- SQLite local
- `PRAGMA foreign_keys = ON`
- `AutoMigrate` sobre entidades
- seed inicial crea super admin y métodos de pago básicos

Seed por defecto:
- usuario: `admin@gym-go.com`
- password: `admin123`

---

## Reglas de Negocio Importantes

### Auth

- login por email/password
- si falla el password, incrementa intentos
- a los 5 intentos bloquea 15 minutos
- genera access token y refresh token
- middleware mete en contexto: `user_id`, `gym_id`, `email`, `role`

### Suscripciones

- un titular no puede tener más de una suscripción activa
- algunos planes soportan grupo mediante `MaxMembers`
- si el plan grupal requiere extras, se valida la cantidad exacta
- al crear una suscripción se activa de inmediato
- el cobro de matrícula (`EnrollmentFee`) solo aplica en la primera suscripción
- renovación usa la fecha final más lejana del usuario para encadenar vigencias
- existe freeze/unfreeze
- existe auditoría manual de cambio de fechas
- cada hora corre auto-expiración

### Acceso al gimnasio

- para check-in se valida usuario
- busca suscripción activa directa
- si no hay, busca suscripción grupal como beneficiario
- si no hay suscripción válida, registra acceso denegado
- si hay vigencia válida, registra acceso concedido

### Ventas e inventario

- una venta exige usuario, método de pago y detalles
- valida que el método de pago exista y esté activo
- valida productos y stock
- calcula subtotales y total
- al vender descuenta inventario
- al anular venta crea una venta inversa y repone stock

### Planes

- tienen `DurationDays`
- soportan `BillingMode`
- `CALENDAR_MONTH`: suma meses calendario
- `30_DAYS`: suma exactamente 30 días
- soportan `MaxMembers` para planes grupales

---

## Rutas Reales Principales

Base:
- `/api/v1`

Públicas:
- `/health`
- `/auth/login`
- `/auth/refresh`
- `/auth/register`
- `/auth/unlock-admin`

Protegidas:
- `/auth/logout`
- `/auth/me`
- `/auth/password`
- `/upload/*`
- `/gym`
- `/users/*`
- `/plans/*`
- `/subscriptions/*`
- `/access/*`
- `/biometric/*`
- `/notifications/*`
- `/classes/*`
- `/attendance/*`
- `/products/*`
- `/payment-methods/*`
- `/sales/*`

---

## Mapa de Módulos

### Auth y usuarios
- login, refresh, logout, perfil
- cambio de contraseña
- roles y control de acceso

### Gym
- lectura y actualización de configuración del gimnasio

### Planes
- CRUD de planes
- soporte para duración, matrícula, miembros máximos y modo de cobro mensual

### Suscripciones
- crear
- listar
- renovar
- cancelar
- congelar/descongelar
- estadísticas
- reporte
- auditoría de fechas

### Acceso
- check-in
- check-out
- histórico
- hoy
- stats

### Biometría
- estado
- captura
- enrolamiento
- verificación
- borrado de huella

### Clases y asistencia
- crear clase
- iniciar/completar/cancelar
- registrar asistencia

### Inventario y ventas
- CRUD de productos
- búsqueda
- ajuste de stock
- métodos de pago
- ventas
- reportes de ventas
- anulación

### Notificaciones
- envío de recordatorios de vencimiento
- test de email SMTP

---

## Archivos Que Más Conviene Darle a una IA

Si quieres ahorrar tokens, normalmente no le mandes todo el repo. Prioriza este orden:

1. `docs/AI_BACKEND_CONTEXT.md`
2. `main.go`
3. el `usecase` del módulo que quieres tocar
4. el `handler` del endpoint
5. la `entity` involucrada
6. el repositorio SQLite del módulo

Ejemplos:

Si vas a tocar suscripciones:
- `docs/AI_BACKEND_CONTEXT.md`
- `main.go`
- `internal/usecases/subscription_usecase.go`
- `internal/infrastructure/http/handlers/subscription_handler.go`
- `internal/domain/entities/subscription.go`
- `internal/domain/entities/plan.go`

Si vas a tocar ventas:
- `docs/AI_BACKEND_CONTEXT.md`
- `main.go`
- `internal/usecases/sale_usecase.go`
- `internal/infrastructure/http/handlers/sale_handler.go`
- `internal/domain/entities/sale.go`
- `internal/domain/entities/sale_detail.go`

Si vas a tocar acceso:
- `docs/AI_BACKEND_CONTEXT.md`
- `main.go`
- `internal/usecases/access_usecase.go`
- `internal/infrastructure/http/handlers/access_handler.go`
- `internal/domain/entities/access_log.go`
- `internal/domain/entities/subscription.go`

---

## Prompt Corto Recomendado

Usa algo así:

```text
Toma como contexto este backend real:
- Go monolítico con Gin + GORM + SQLite
- patrón handler -> usecase -> repository
- auth JWT con roles
- módulos: usuarios, planes, suscripciones, acceso, biometría, clases, asistencia, inventario, ventas, notificaciones
- la documentación grande mezcla roadmap con estado actual; prioriza el código real

Archivos fuente de verdad para esta tarea:
- [pega aquí 3 a 6 archivos]

Necesito que:
- [cambio puntual]

No inventes PostgreSQL, Redis, multi-tenant ni microservicios si no aparecen en estos archivos.
```

---

## Prompt Ultra Corto

```text
Contexto: backend Go monolítico con Gin + GORM + SQLite. Arquitectura: handler -> usecase -> repository. Auth JWT por roles. La doc grande incluye roadmap; prioriza código real. Trabaja solo con estos archivos: [archivos]. Tarea: [tarea].
```

---

## Plantilla Por Tipo de Pregunta

### Para corregir bug

```text
Contexto: backend Go monolítico con Gin + GORM + SQLite, patrón handler -> usecase -> repository. Prioriza código real sobre docs de roadmap.

Archivos:
- ...

Bug:
- ...

Quiero:
- causa raíz
- cambio mínimo seguro
- qué endpoints o casos rompe si se cambia mal
```

### Para agregar feature

```text
Contexto: backend Go actual con Gin + GORM + SQLite. No propongas rediseño a microservicios.

Archivos:
- ...

Feature:
- ...

Necesito:
- dónde implementarlo
- cambios en handler/usecase/repository/entity
- riesgos de compatibilidad
```

### Para pedir review

```text
Revisa este cambio con contexto de backend Go monolítico Gin + GORM + SQLite. Prioriza bugs, regresiones y edge cases. No me hagas una explicación general si no es necesaria.
```

---

## Regla Práctica Para Ahorrar Tokens

No envíes:
- todos los `.md`
- todo `internal/`
- todo `main.go` + todo el frontend + toda la DB

Sí envía:
- este resumen
- 1 archivo de entrada
- 1 use case
- 1 handler
- 1 o 2 entities/repos relevantes

En la mayoría de tareas eso basta.

---

## Resumen de Una Línea

Gym-Go hoy es un backend Go monolítico con Gin, JWT, GORM y SQLite, organizado en `domain/usecases/infrastructure`, con módulos reales para auth, usuarios, suscripciones, acceso, biometría, clases, asistencia, inventario y ventas.
