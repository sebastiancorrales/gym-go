# 🏗️ Arquitectura del Sistema - Gym SaaS

## 1. Visión General

Sistema SaaS multi-tenant para gestión de gimnasios con arquitectura escalable, preparado para integración con hardware de control de acceso.

### Stack Tecnológico

**Backend:**
- Go 1.21+
- Gin Framework
- PostgreSQL 15+
- Redis 7+
- JWT (Access + Refresh Tokens)
- GORM / sqlx
- Swagger/OpenAPI

**Frontend:**
- React 18
- React Query / TanStack Query
- React Router v6
- Tailwind CSS
- Recharts / Chart.js
- Axios

**Infraestructura:**
- Docker + Docker Compose
- Nginx (Reverse Proxy)
- PostgreSQL (Master-Slave replication)
- Redis Cluster
- MinIO / S3 (almacenamiento)

---

## 2. Diagrama de Arquitectura General

```
┌─────────────────────────────────────────────────────────────────┐
│                         FRONTEND LAYER                          │
├─────────────────────────────────────────────────────────────────┤
│  React SPA          │  Mobile App (Future)  │  Hardware Client  │
│  - Dashboard        │  - iOS/Android        │  - Torniquetes    │
│  - Admin Panel      │  - QR Scanner         │  - Lectores QR    │
│  - Reports          │                       │  - Biométricos    │
└──────────────┬──────────────────────────────────────────────────┘
               │
               │ HTTPS/WSS
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      REVERSE PROXY (Nginx)                      │
│  - SSL/TLS Termination                                          │
│  - Rate Limiting                                                │
│  - Load Balancing                                               │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API GATEWAY                             │
│  - Authentication Middleware                                    │
│  - Request Validation                                           │
│  - Logging & Monitoring                                         │
│  - CORS                                                         │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      BACKEND SERVICES (Go)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐        │
│  │   Auth      │  │ Subscription │  │  Access        │        │
│  │   Service   │  │   Service    │  │  Control       │        │
│  └─────────────┘  └──────────────┘  └────────────────┘        │
│                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐        │
│  │  Payment    │  │   Reports    │  │   User         │        │
│  │  Service    │  │   Service    │  │   Management   │        │
│  └─────────────┘  └──────────────┘  └────────────────┘        │
│                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐        │
│  │  Gym        │  │  Notifications│  │   Audit        │        │
│  │  Management │  │   Service    │  │   Service      │        │
│  └─────────────┘  └──────────────┘  └────────────────┘        │
│                                                                 │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                       DATA LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐         ┌─────────────────┐             │
│  │   PostgreSQL     │         │     Redis       │             │
│  │   (Master/Slave) │         │   - Sessions    │             │
│  │   - Users        │         │   - Cache       │             │
│  │   - Subscriptions│         │   - Tokens      │             │
│  │   - Payments     │         │   - Rate Limit  │             │
│  │   - Access Logs  │         └─────────────────┘             │
│  └──────────────────┘                                          │
│                                                                 │
│  ┌──────────────────┐         ┌─────────────────┐             │
│  │   MinIO/S3       │         │   Message Queue │             │
│  │   - Documents    │         │   (RabbitMQ)    │             │
│  │   - Reports      │         │   - Emails      │             │
│  │   - Backups      │         │   - Webhooks    │             │
│  └──────────────────┘         └─────────────────┘             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Clean Architecture (Backend)

```
gym-go/
├── cmd/
│   ├── api/                    # Main API server
│   ├── worker/                 # Background jobs
│   └── migrations/             # DB migrations
│
├── internal/
│   ├── domain/                 # Enterprise Business Rules
│   │   ├── entities/
│   │   │   ├── user.go
│   │   │   ├── gym.go
│   │   │   ├── subscription.go
│   │   │   ├── plan.go
│   │   │   ├── payment.go
│   │   │   ├── access_log.go
│   │   │   ├── device.go
│   │   │   └── audit.go
│   │   │
│   │   └── repositories/       # Repository interfaces
│   │       ├── user_repository.go
│   │       ├── subscription_repository.go
│   │       ├── payment_repository.go
│   │       └── access_repository.go
│   │
│   ├── usecases/               # Application Business Rules
│   │   ├── auth/
│   │   │   ├── login.go
│   │   │   ├── refresh_token.go
│   │   │   ├── logout.go
│   │   │   └── password_recovery.go
│   │   │
│   │   ├── subscription/
│   │   │   ├── create_subscription.go
│   │   │   ├── renew_subscription.go
│   │   │   ├── cancel_subscription.go
│   │   │   └── check_expiration.go
│   │   │
│   │   ├── access/
│   │   │   ├── validate_entry.go
│   │   │   ├── register_access.go
│   │   │   └── check_permissions.go
│   │   │
│   │   ├── payment/
│   │   │   ├── process_payment.go
│   │   │   ├── apply_discount.go
│   │   │   └── cash_register.go
│   │   │
│   │   └── reports/
│   │       ├── revenue_report.go
│   │       ├── members_report.go
│   │       └── access_report.go
│   │
│   ├── infrastructure/         # Interface Adapters
│   │   ├── http/
│   │   │   ├── handlers/
│   │   │   │   ├── auth_handler.go
│   │   │   │   ├── subscription_handler.go
│   │   │   │   ├── payment_handler.go
│   │   │   │   ├── access_handler.go
│   │   │   │   └── report_handler.go
│   │   │   │
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── rbac.go
│   │   │   │   ├── rate_limiter.go
│   │   │   │   ├── logger.go
│   │   │   │   └── tenant.go
│   │   │   │
│   │   │   └── router.go
│   │   │
│   │   ├── persistence/
│   │   │   ├── postgres/
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── subscription_repository.go
│   │   │   │   └── payment_repository.go
│   │   │   │
│   │   │   └── redis/
│   │   │       ├── session_store.go
│   │   │       ├── token_store.go
│   │   │       └── cache.go
│   │   │
│   │   ├── security/
│   │   │   ├── jwt.go
│   │   │   ├── bcrypt.go
│   │   │   └── rbac.go
│   │   │
│   │   └── external/
│   │       ├── email/
│   │       ├── sms/
│   │       └── payment_gateway/
│   │
│   └── config/
│       ├── config.go
│       └── database.go
│
├── pkg/                        # Shared libraries
│   ├── logger/
│   ├── validator/
│   ├── errors/
│   └── utils/
│
├── migrations/                 # SQL migrations
├── docs/                       # Documentation
└── tests/                      # Tests
```

---

## 4. Arquitectura Multi-Tenant

### Estrategia: Schema per Tenant (Recomendada)

```sql
-- Cada gimnasio tiene su propio schema
CREATE SCHEMA gym_1;
CREATE SCHEMA gym_2;

-- Tabla global de tenants
CREATE TABLE public.tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    schema_name VARCHAR(50) UNIQUE,
    subscription_plan VARCHAR(50),
    status VARCHAR(20),
    created_at TIMESTAMP
);

-- Cada schema tiene las mismas tablas
CREATE TABLE gym_1.users (...);
CREATE TABLE gym_1.subscriptions (...);
CREATE TABLE gym_1.payments (...);
```

**Ventajas:**
- Aislamiento total de datos
- Backups individuales
- Fácil migración de clientes
- Cumplimiento GDPR

---

## 5. Sistema de Autenticación

### Flow de Login

```
┌──────────┐                 ┌──────────┐                ┌──────────┐
│  Client  │                 │   API    │                │  Redis   │
└────┬─────┘                 └────┬─────┘                └────┬─────┘
     │                            │                           │
     │  POST /auth/login          │                           │
     │  { email, password }       │                           │
     ├──────────────────────────► │                           │
     │                            │                           │
     │                            │ Validate credentials      │
     │                            │ Generate Access Token     │
     │                            │ Generate Refresh Token    │
     │                            │                           │
     │                            │ Store refresh token       │
     │                            ├─────────────────────────► │
     │                            │                           │
     │  { accessToken,            │                           │
     │    refreshToken,           │                           │
     │    user, permissions }     │                           │
     │ ◄────────────────────────── │                           │
     │                            │                           │
     │  Store in localStorage/    │                           │
     │  httpOnly cookie           │                           │
     │                            │                           │
```

### Tokens

**Access Token (JWT):**
- Duración: 15 minutos
- Almacenado: localStorage
- Claims:
```json
{
  "sub": "user_id",
  "tenant_id": "gym_id",
  "role": "ADMIN_GYM",
  "permissions": ["users:read", "users:write"],
  "exp": 1234567890
}
```

**Refresh Token:**
- Duración: 7 días
- Almacenado: Redis + httpOnly cookie
- Rotación automática

---

## 6. Control de Acceso (RBAC)

### Matriz de Permisos

| Recurso | SUPER_ADMIN | ADMIN_GYM | RECEPCIONISTA | STAFF |
|---------|-------------|-----------|---------------|-------|
| gyms:* | ✅ | ❌ | ❌ | ❌ |
| users:read | ✅ | ✅ | ✅ | ❌ |
| users:write | ✅ | ✅ | ❌ | ❌ |
| subscriptions:* | ✅ | ✅ | ✅ | ❌ |
| payments:write | ✅ | ✅ | ✅ | ❌ |
| payments:read | ✅ | ✅ | ✅ | ❌ |
| access:validate | ✅ | ✅ | ✅ | ✅ |
| reports:* | ✅ | ✅ | ❌ | ❌ |
| settings:* | ✅ | ✅ | ❌ | ❌ |

### Implementación

```go
// Middleware RBAC
func RequirePermissions(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPermissions := c.GetStringSlice("permissions")
        
        for _, required := range permissions {
            if !contains(userPermissions, required) {
                c.JSON(403, gin.H{"error": "Forbidden"})
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}

// Uso
router.GET("/users", 
    auth.Required(),
    rbac.RequirePermissions("users:read"),
    handlers.GetUsers
)
```

---

## 7. Módulo de Control de Acceso (Hardware Ready)

### Arquitectura del Módulo

```
┌─────────────────────────────────────────────────────────────┐
│                    ACCESS CONTROL MODULE                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────┐         ┌──────────────────┐        │
│  │  Validation      │         │   Registration   │        │
│  │  Engine          │────────►│   Service        │        │
│  │                  │         │                  │        │
│  │  - Check sub     │         │  - Log access    │        │
│  │  - Check payment │         │  - Record time   │        │
│  │  - Check blocks  │         │  - Device info   │        │
│  └──────────────────┘         └──────────────────┘        │
│           │                            │                   │
│           ▼                            ▼                   │
│  ┌──────────────────────────────────────────────┐         │
│  │           Access Decision Service            │         │
│  │  - Rules Engine                              │         │
│  │  - Business Logic                            │         │
│  │  - Audit Trail                               │         │
│  └──────────────────────────────────────────────┘         │
│           │                                                │
│           ▼                                                │
│  ┌──────────────────────────────────────────────┐         │
│  │          Hardware Integration Layer          │         │
│  │  - QR Scanner API                            │         │
│  │  - Biometric API (Future)                    │         │
│  │  - Turnstile Controller (Future)             │         │
│  │  - Webhook Support                           │         │
│  └──────────────────────────────────────────────┘         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### API para Dispositivos

```go
// POST /api/v1/access/validate
type AccessRequest struct {
    DeviceID     string `json:"device_id"`      // ID del dispositivo
    Method       string `json:"method"`         // QR, FINGERPRINT, CARD
    Identifier   string `json:"identifier"`     // QR code, fingerprint hash, card number
    Timestamp    time.Time `json:"timestamp"`
    Location     string `json:"location"`       // Entrada principal, área pesas
}

type AccessResponse struct {
    Allowed      bool   `json:"allowed"`
    Reason       string `json:"reason"`
    UserName     string `json:"user_name"`
    ExpiresAt    string `json:"expires_at"`
    Message      string `json:"message"`         // Mostrar en pantalla
}
```

### Reglas de Validación

```go
func (s *AccessService) ValidateEntry(req AccessRequest) (*AccessResponse, error) {
    // 1. Identificar usuario
    user, err := s.identifyUser(req.Identifier, req.Method)
    if err != nil {
        return &AccessResponse{
            Allowed: false,
            Reason: "USER_NOT_FOUND",
            Message: "Usuario no encontrado",
        }, nil
    }
    
    // 2. Verificar suscripción activa
    sub, err := s.subscriptionRepo.GetActiveSubscription(user.ID)
    if err != nil || sub == nil {
        return &AccessResponse{
            Allowed: false,
            Reason: "NO_ACTIVE_SUBSCRIPTION",
            Message: "Sin suscripción activa",
        }, nil
    }
    
    // 3. Verificar vigencia
    if sub.ExpiresAt.Before(time.Now()) {
        return &AccessResponse{
            Allowed: false,
            Reason: "SUBSCRIPTION_EXPIRED",
            Message: "Suscripción vencida",
        }, nil
    }
    
    // 4. Verificar pagos pendientes
    hasPending, _ := s.paymentRepo.HasPendingPayments(user.ID)
    if hasPending {
        return &AccessResponse{
            Allowed: false,
            Reason: "PENDING_PAYMENT",
            Message: "Pago pendiente",
        }, nil
    }
    
    // 5. Verificar bloqueos
    if user.Status == "BLOCKED" {
        return &AccessResponse{
            Allowed: false,
            Reason: "USER_BLOCKED",
            Message: "Usuario bloqueado",
        }, nil
    }
    
    // 6. Registrar acceso
    s.registerAccess(user.ID, req.DeviceID, "GRANTED")
    
    return &AccessResponse{
        Allowed: true,
        Reason: "OK",
        UserName: user.FirstName + " " + user.LastName,
        ExpiresAt: sub.ExpiresAt.Format("2006-01-02"),
        Message: "¡Bienvenido!",
    }, nil
}
```

---

## 8. Integración con Hardware (Futuro)

### Dispositivos Soportados

1. **Lectores QR/Código de Barras**
   - Webhook HTTP
   - Polling cada 2 segundos
   - Modo offline con sincronización

2. **Torniquetes Inteligentes**
   - Protocolo Wiegand
   - API REST
   - Control de apertura/cierre

3. **Biometría (Huella/Facial)**
   - SDK del fabricante
   - Hash de huella almacenado
   - Validación en dispositivo + servidor

### Modo Offline (Edge Computing)

```
┌──────────────────┐
│   Dispositivo    │
│   (Edge)         │
│                  │
│  ┌────────────┐  │
│  │ Cache      │  │◄─── Sincronización cada 5 min
│  │ Local      │  │
│  │ - Users    │  │
│  │ - Subs     │  │
│  └────────────┘  │
│                  │
│  Validación      │
│  Local si        │
│  API down        │
└──────────────────┘
```

---

## 9. Sistema de Pagos

### Flow de Pago

```
┌──────────┐         ┌──────────┐         ┌──────────────┐
│  Client  │         │   API    │         │  Subscription│
└────┬─────┘         └────┬─────┘         └──────┬───────┘
     │                    │                       │
     │ POST /payments     │                       │
     ├───────────────────►│                       │
     │                    │                       │
     │                    │ Create payment        │
     │                    │                       │
     │                    │ Update subscription   │
     │                    ├──────────────────────►│
     │                    │                       │
     │                    │ Extend expiration     │
     │                    │◄──────────────────────┤
     │                    │                       │
     │                    │ Register in cash box  │
     │                    │                       │
     │ Payment confirmed  │                       │
     │◄───────────────────┤                       │
     │                    │                       │
```

### Caja Diaria

```go
type CashRegister struct {
    ID          uuid.UUID
    GymID       uuid.UUID
    OpenedBy    uuid.UUID
    ClosedBy    *uuid.UUID
    OpenedAt    time.Time
    ClosedAt    *time.Time
    InitialCash decimal.Decimal
    FinalCash   *decimal.Decimal
    TotalCash   decimal.Decimal
    TotalCard   decimal.Decimal
    TotalTransfer decimal.Decimal
    Expected    decimal.Decimal
    Difference  *decimal.Decimal
    Status      string // OPEN, CLOSED
}
```

---

## 10. Escalabilidad

### Horizontal Scaling

```
         Load Balancer (Nginx)
                │
    ┌───────────┼───────────┐
    │           │           │
    ▼           ▼           ▼
┌────────┐ ┌────────┐ ┌────────┐
│ API #1 │ │ API #2 │ │ API #3 │
└────────┘ └────────┘ └────────┘
    │           │           │
    └───────────┼───────────┘
                │
         ┌──────┴──────┐
         │             │
         ▼             ▼
    PostgreSQL      Redis
    (Master)        Cluster
         │
         ▼
    PostgreSQL
    (Replica)
```

### Caching Strategy

```go
// 1. Cache de sesiones (Redis)
session, err := redis.Get("session:" + token)

// 2. Cache de permisos (5 min TTL)
permissions, err := redis.Get("permissions:" + userID)

// 3. Cache de suscripciones activas (1 min TTL)
sub, err := redis.Get("subscription:" + userID)

// 4. Cache de configuración de gym (10 min TTL)
config, err := redis.Get("gym:config:" + gymID)
```

### Database Optimization

```sql
-- Índices críticos
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_subscriptions_user_status ON subscriptions(user_id, status);
CREATE INDEX idx_subscriptions_expires_at ON subscriptions(expires_at);
CREATE INDEX idx_access_logs_user_date ON access_logs(user_id, created_at);
CREATE INDEX idx_payments_date ON payments(payment_date);

-- Particionamiento de access_logs por mes
CREATE TABLE access_logs_2026_01 PARTITION OF access_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
```

---

## 11. Seguridad

### Checklist de Seguridad

✅ **Autenticación:**
- [ ] JWT con rotación de refresh tokens
- [ ] Tokens en httpOnly cookies
- [ ] Rate limiting en login (5 intentos/min)
- [ ] Bloqueo temporal tras 5 fallos
- [ ] 2FA opcional (TOTP)

✅ **Autorización:**
- [ ] RBAC estricto
- [ ] Validación de tenant en cada request
- [ ] Audit log de acciones críticas

✅ **Datos:**
- [ ] Encriptación en tránsito (TLS 1.3)
- [ ] Encriptación en reposo (PostgreSQL)
- [ ] Hash de contraseñas (bcrypt cost 12)
- [ ] PII encriptado (GDPR)

✅ **API:**
- [ ] Rate limiting global (100 req/min)
- [ ] Input validation (todas las entradas)
- [ ] SQL injection prevention (prepared statements)
- [ ] XSS prevention (sanitización)
- [ ] CSRF tokens

✅ **Infraestructura:**
- [ ] Firewall (solo puertos necesarios)
- [ ] VPC privada para DB
- [ ] Secrets en variables de entorno
- [ ] Backups encriptados diarios
- [ ] Logs centralizados

---

## 12. Monitoreo y Observabilidad

```
┌──────────────────────────────────────────┐
│           Prometheus                     │
│  - API metrics                           │
│  - DB connections                        │
│  - Redis hit rate                        │
│  - Response times                        │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│           Grafana                        │
│  - Dashboards                            │
│  - Alerts                                │
└──────────────────────────────────────────┘

┌──────────────────────────────────────────┐
│           ELK Stack                      │
│  - Elasticsearch                         │
│  - Logstash                              │
│  - Kibana                                │
└──────────────────────────────────────────┘
```

---

## 13. Roadmap Técnico

### Fase 1: MVP (3 meses)
- ✅ Auth + JWT
- ✅ Gestión de usuarios
- ✅ Suscripciones básicas
- ✅ Control de acceso (QR manual)
- ✅ Pagos en efectivo
- ✅ Dashboard básico

### Fase 2: Operacional (3 meses)
- ⏳ Múltiples métodos de pago
- ⏳ Reportes completos
- ⏳ Promociones y descuentos
- ⏳ API para dispositivos QR
- ⏳ Modo offline devices
- ⏳ App móvil (React Native)

### Fase 3: Enterprise (6 meses)
- 🔮 Integración torniquetes
- 🔮 Biometría (huella)
- 🔮 Multi-sucursal
- 🔮 Facturación electrónica
- 🔮 CRM integrado
- 🔮 Analytics avanzado

---

## 14. Estimación de Costos (AWS)

**Infraestructura inicial (hasta 5 gimnasios):**

| Servicio | Recurso | Costo/mes |
|----------|---------|-----------|
| EC2 | t3.medium (API) | $30 |
| RDS | PostgreSQL db.t3.micro | $15 |
| ElastiCache | Redis t3.micro | $12 |
| S3 | 50GB storage | $1 |
| CloudFront | CDN | $5 |
| Route53 | DNS | $1 |
| **Total** | | **~$64/mes** |

**Escalado (50 gimnasios):**
- EC2: t3.large x2 ($140)
- RDS: db.t3.medium ($60)
- ElastiCache: Redis cluster ($80)
- **Total: ~$300/mes**

---

## 15. Conclusiones

✅ **Arquitectura lista para:**
- Multi-tenancy
- Escalamiento horizontal
- Integración hardware
- Cumplimiento GDPR
- SaaS enterprise

✅ **Próximos pasos:**
1. Implementar entities y repositories
2. Configurar PostgreSQL + Redis
3. Implementar sistema de auth completo
4. Desarrollar módulo de suscripciones
5. Crear dashboard React

**¿Comenzamos con la implementación del sistema de autenticación?**
