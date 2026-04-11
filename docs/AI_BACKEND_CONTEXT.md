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
- cierre diario automático a las 23:00 con envío de correo
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
- `NotificationRecipient`  ← destinatarios configurables por gimnasio y tipo

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
- `NotificationUseCase`  ← cierre diario, recordatorios y gestión de destinatarios

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
- SQLite para usuarios, suscripciones, acceso, planes, huellas, inventario, ventas, métodos de pago, gym, destinatarios de notificaciones
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

### Notificaciones por email

#### Tipos de notificación (`NotificationType`)

| Constante | Descripción |
|-----------|-------------|
| `DAILY_CLOSE` | Cierre diario con adjuntos PDF + Excel |
| `ACCOUNTING_REPORT` | Reporte contable periódico |
| `SUBSCRIPTION_REMINDER` | Recordatorio de vencimiento al miembro |

#### Destinatarios configurables

- Tabla `notification_recipients` en SQLite
- Cada fila: `gym_id`, `notification_type`, `name`, `email`, `active`
- Múltiples destinatarios por tipo (e.g. contador + admin para `DAILY_CLOSE`)
- Gestionados via endpoints REST (ver rutas)

#### Cierre diario (`DAILY_CLOSE`)

Fuentes de datos:
- Ventas: `saleRepo.GetByDateRange` — filtra por `sale_date` entre inicio y fin del día/rango en UTC
- Suscripciones nuevas: `subscriptionRepo.FindByGymIDAndDateRange` — filtra por `created_at` del período

Nota: `Sale` no tiene `gym_id`; el cierre usa todas las ventas del período (correcto para instalaciones de un solo gimnasio).

Contenido del email:
- Cuerpo HTML con tabla de totales (ventas, suscripciones, total del período)
- Valores monetarios formateados con `FmtAmt` — formato colombiano `$ 1.234.567`
- Desglose por método de pago (nombre, cantidad de transacciones, total)
- Adjunto **Excel** (1 hoja "Reporte") generado con `excelize`, secciones:
  1. Tabla de suscripciones (10 columnas: usuario, plan, método, fechas, precio, matrícula, descuento, total)
  2. SUBTOTAL SUSCRIPCIONES
  3. PLANES VENDIDOS (plan, cantidad, total) — ordenados por cantidad desc
  4. PRODUCTOS VENDIDOS (producto, cantidad, total) — agregado, no línea por línea
  5. VENTAS INVENTARIO (transacciones, bruto, descuentos, neto)
  6. METODOS DE PAGO (método, suscripciones, ventas inventario, total)
  7. TOTAL GENERAL
- Adjunto **PDF** (A4 vertical) generado con `go-pdf/fpdf`, secciones:
  1. Header oscuro con nombre del gym y período
  2. 3 cajas resumen (verde: suscripciones, azul: ventas inventario, oscuro: total general)
  3. Sección 1: tabla de suscripciones + desglose por plan
  4. Sección 2: ventas inventario (bruto/descuentos/neto)
  5. Sección 2b: productos vendidos (si hay)
  6. Sección 3: desglose por método de pago (suscripciones vs ventas vs total)
  7. Caja final TOTAL GENERAL
  8. Paginación automática con `{nb}`

Soporte de rango de fechas:
- Un solo día: asunto y header dicen "Cierre del dia DD/MM/YYYY"
- Rango: asunto y header dicen "Cierre del DD/MM al DD/MM/YYYY"

`DailyCloseReport` struct (campos clave):
- `Date`, `EndDate` — período
- `TotalSalesAmount`, `TotalSalesCount`, `SalesGross`, `SalesDiscount`
- `TotalSubsAmount`, `TotalSubsCount`, `TotalRevenue`
- `PaymentMethods []PaymentMethodSummary` — con SubsTotal/SubsCount/SalesTotal/SalesCount/Total
- `Plans []PlanSummary` — nombre, cantidad, revenue; ordenados por qty desc
- `Products []ProductSummary` — nombre, cantidad, revenue; agregado por nombre normalizado (lowercase), ordenados por revenue desc
- `SaleItems []SaleLineItem`, `SubscriptionItems []SubscriptionLineItem`

Normalización de métodos de pago:
- Clave interna: `strings.ToLower(strings.TrimSpace(name))` para evitar duplicados "EFECTIVO"/"Efectivo"

`NotificationUseCase` dependencias (10 en total):
- `recipientRepo`, `saleRepo`, `saleDetailRepo`, `subscriptionRepo`, `planRepo`, `gymRepo`, `userRepo`, `paymentMethodRepo`, `productRepo`, `emailSender`

`FmtAmt` (exportada desde `report_builder.go`):
- Formato colombiano con puntos como separadores de miles: `$ 1.234.567`
- Usada tanto en PDF/Excel como en el cuerpo HTML del correo

Scheduler:
- `scheduleDailyClose(23, 0, fn)` en `main.go`
- Dispara a las 23:00 hora local del servidor
- Itera todos los gimnasios; usa el campo `Gym.Timezone` para la zona horaria

Trigger manual:
- `POST /api/v1/notifications/send-daily-close`
- Body opcional: `{"date": "2024-01-15", "date_end": "2024-01-17"}` — si omite `date_end`, es igual a `date`

#### Infraestructura de email

Paquete: `internal/infrastructure/email`

| Archivo | Responsabilidad |
|---------|----------------|
| `sender.go` | SMTP con `Send` (sin adjuntos) y `SendWithAttachments` (MIME multipart) |
| `templates.go` | Templates HTML: `RenderDailyCloseEmail`, `RenderSubscriptionReminderEmail` |
| `report_builder.go` | `DailyCloseReport` struct + `BuildExcelReport` + `BuildPDFReport` |

Patrón de fallback SMTP:
1. Si el gimnasio tiene SMTP propio configurado (`Gym.SMTPHost`), lo usa
2. Si no, usa el SMTP global de variables de entorno (`SMTP_*`)

#### Archivos clave de notificaciones

```
internal/domain/entities/notification_recipient.go
internal/domain/repositories/repository.go            (interface NotificationRecipientRepository)
internal/infrastructure/email/sender.go
internal/infrastructure/email/templates.go
internal/infrastructure/email/report_builder.go
internal/infrastructure/persistence/sqlite_notification_recipient_repository.go
internal/usecases/notification_usecase.go
internal/infrastructure/http/handlers/notification_handler.go
```

---

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
- `/notifications/*`  (send-expiring, send-daily-close, test-email, recipients CRUD)
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
- incluye campos SMTP propios por gimnasio (`smtp_host`, `smtp_port`, `smtp_username`, `smtp_password`, `smtp_from`)
- el repositorio usa raw SQL `UPDATE ... WHERE id=?` en vez de `gorm.Save()` para evitar el bug de INSERT por UUID nil

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
- envío de recordatorios de vencimiento de suscripción
- test de email SMTP
- **cierre diario**: genera reporte con totales de ventas y suscripciones del día, adjunta PDF y Excel, envía a destinatarios configurados
- gestión de destinatarios por tipo de notificación (CRUD)
- scheduler automático a las 23:00 hora local por cada gimnasio

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

Si vas a tocar notificaciones / cierre diario:
- `docs/AI_BACKEND_CONTEXT.md`
- `main.go`
- `internal/usecases/notification_usecase.go`
- `internal/infrastructure/http/handlers/notification_handler.go`
- `internal/infrastructure/email/report_builder.go`
- `internal/infrastructure/email/templates.go`
- `internal/domain/entities/notification_recipient.go`

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
- módulos: usuarios, planes, suscripciones, acceso, biometría, clases, asistencia, inventario, ventas, notificaciones (cierre diario PDF+Excel, recordatorios, destinatarios configurables)
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

Gym-Go hoy es un backend Go monolítico con Gin, JWT, GORM y SQLite, organizado en `domain/usecases/infrastructure`, con módulos reales para auth, usuarios, suscripciones, acceso, biometría, clases, asistencia, inventario, ventas y un sistema completo de notificaciones por email (cierre diario con PDF+Excel que incluyen desglose por plan, productos agregados y métodos de pago; soporte de rango de fechas; valores formateados en moneda local; recordatorios de vencimiento; destinatarios configurables por tipo).
