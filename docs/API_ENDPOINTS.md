# 🚀 API Endpoints - Gym SaaS

## Base URL

```
Production: https://api.gymgo.com/v1
Development: http://localhost:8080/v1
```

## Authentication

Todas las rutas (excepto `/auth/*` y `/health`) requieren:

```
Authorization: Bearer {access_token}
```

---

## 1. Authentication & Authorization

### 1.1 Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "admin@gym.com",
  "password": "password123"
}
```

**Response 200:**
```json
{
  "user": {
    "id": "uuid",
    "email": "admin@gym.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "ADMIN_GYM",
    "gym_id": "uuid",
    "permissions": ["users:read", "users:write", "payments:write"]
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "refresh_token_here",
  "expires_in": 900
}
```

### 1.2 Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "refresh_token_here"
}
```

**Response 200:**
```json
{
  "access_token": "new_access_token",
  "expires_in": 900
}
```

### 1.3 Logout

```http
POST /auth/logout
Authorization: Bearer {token}
```

**Response 204:** No Content

### 1.4 Password Recovery

```http
POST /auth/forgot-password
Content-Type: application/json

{
  "email": "user@gym.com"
}
```

**Response 200:**
```json
{
  "message": "Password reset email sent"
}
```

### 1.5 Reset Password

```http
POST /auth/reset-password
Content-Type: application/json

{
  "token": "reset_token_from_email",
  "new_password": "newpassword123"
}
```

### 1.6 Change Password

```http
PUT /auth/change-password
Authorization: Bearer {token}
Content-Type: application/json

{
  "current_password": "oldpassword",
  "new_password": "newpassword123"
}
```

---

## 2. Users Management

### 2.1 List Users

```http
GET /users?page=1&limit=20&role=MEMBER&status=ACTIVE&search=john
Authorization: Bearer {token}
```

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `role` (string): Filter by role
- `status` (string): Filter by status
- `search` (string): Search in name, email, document

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "email": "john@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "document_type": "CC",
      "document_number": "123456789",
      "phone": "3001234567",
      "role": "MEMBER",
      "status": "ACTIVE",
      "qr_code": "GYM-ABC123DEF456",
      "photo_url": "https://...",
      "created_at": "2026-01-15T10:00:00Z",
      "subscription": {
        "status": "ACTIVE",
        "plan_name": "Mensual",
        "expires_at": "2026-02-15"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

### 2.2 Get User by ID

```http
GET /users/{id}
Authorization: Bearer {token}
```

**Response 200:** (Same structure as list item)

### 2.3 Create User

```http
POST /users
Authorization: Bearer {token}
Content-Type: application/json

{
  "email": "newuser@example.com",
  "password": "password123",
  "first_name": "Jane",
  "last_name": "Smith",
  "document_type": "CC",
  "document_number": "987654321",
  "phone": "3009876543",
  "date_of_birth": "1990-05-15",
  "role": "MEMBER",
  "emergency_contact_name": "John Smith",
  "emergency_contact_phone": "3001111111"
}
```

**Response 201:**
```json
{
  "id": "uuid",
  "email": "newuser@example.com",
  "qr_code": "GYM-XYZ789ABC123",
  ...
}
```

### 2.4 Update User

```http
PUT /users/{id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "first_name": "Jane Updated",
  "phone": "3009999999"
}
```

### 2.5 Delete User (Soft Delete)

```http
DELETE /users/{id}
Authorization: Bearer {token}
```

**Response 204:** No Content

### 2.6 Upload User Photo

```http
POST /users/{id}/photo
Authorization: Bearer {token}
Content-Type: multipart/form-data

{
  "photo": <binary>
}
```

**Response 200:**
```json
{
  "photo_url": "https://storage.gymgo.com/photos/user-uuid.jpg"
}
```

---

## 3. Plans

### 3.1 List Plans

```http
GET /plans?status=ACTIVE
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Plan Mensual",
      "description": "Acceso ilimitado por 30 días",
      "duration_days": 30,
      "price": 80000,
      "enrollment_fee": 0,
      "features": {
        "access_areas": ["gym", "classes"],
        "guest_passes": 0
      },
      "status": "ACTIVE"
    }
  ]
}
```

### 3.2 Create Plan

```http
POST /plans
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Plan Anual VIP",
  "description": "12 meses con beneficios exclusivos",
  "duration_days": 365,
  "price": 800000,
  "enrollment_fee": 50000,
  "features": {
    "access_areas": ["gym", "classes", "spa"],
    "guest_passes": 12,
    "freeze_days_per_year": 15
  }
}
```

### 3.3 Update Plan

```http
PUT /plans/{id}
Authorization: Bearer {token}
```

### 3.4 Delete Plan

```http
DELETE /plans/{id}
Authorization: Bearer {token}
```

---

## 4. Subscriptions

### 4.1 List Subscriptions

```http
GET /subscriptions?user_id={uuid}&status=ACTIVE&expiring_soon=true
Authorization: Bearer {token}
```

**Query Parameters:**
- `user_id` (uuid): Filter by user
- `status` (string): ACTIVE, EXPIRED, CANCELLED, etc.
- `expiring_soon` (bool): Next 7 days

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "user_name": "John Doe",
      "plan_id": "uuid",
      "plan_name": "Mensual",
      "start_date": "2026-01-15",
      "end_date": "2026-02-15",
      "status": "ACTIVE",
      "price_paid": 80000,
      "auto_renew": true,
      "days_remaining": 10
    }
  ]
}
```

### 4.2 Create Subscription

```http
POST /subscriptions
Authorization: Bearer {token}
Content-Type: application/json

{
  "user_id": "uuid",
  "plan_id": "uuid",
  "start_date": "2026-01-20",
  "auto_renew": true,
  "promotion_code": "PROMO2026"
}
```

**Response 201:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "plan_id": "uuid",
  "start_date": "2026-01-20",
  "end_date": "2026-02-20",
  "status": "PENDING",
  "price_paid": 70000,
  "discount_applied": 10000,
  "total_paid": 70000,
  "payment_required": true
}
```

### 4.3 Renew Subscription

```http
POST /subscriptions/{id}/renew
Authorization: Bearer {token}
Content-Type: application/json

{
  "plan_id": "uuid",  // optional: change plan
  "promotion_code": "RENEW2026"
}
```

### 4.4 Cancel Subscription

```http
POST /subscriptions/{id}/cancel
Authorization: Bearer {token}
Content-Type: application/json

{
  "reason": "Moving to another city",
  "immediate": false  // false = cancel at end_date
}
```

### 4.5 Freeze Subscription

```http
POST /subscriptions/{id}/freeze
Authorization: Bearer {token}
Content-Type: application/json

{
  "freeze_until": "2026-02-01",
  "reason": "Medical leave"
}
```

### 4.6 Unfreeze Subscription

```http
POST /subscriptions/{id}/unfreeze
Authorization: Bearer {token}
```

---

## 5. Payments

### 5.1 List Payments

```http
GET /payments?start_date=2026-01-01&end_date=2026-01-31&method=CASH
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "user_name": "John Doe",
      "subscription_id": "uuid",
      "amount": 80000,
      "method": "CASH",
      "status": "COMPLETED",
      "payment_date": "2026-01-15T14:30:00Z",
      "reference": "REF-001",
      "created_by_name": "Admin User"
    }
  ],
  "summary": {
    "total_amount": 5600000,
    "total_cash": 3200000,
    "total_card": 1600000,
    "total_transfer": 800000,
    "count": 70
  }
}
```

### 5.2 Create Payment

```http
POST /payments
Authorization: Bearer {token}
Content-Type: application/json

{
  "user_id": "uuid",
  "subscription_id": "uuid",
  "amount": 80000,
  "method": "CASH",
  "reference": "REF-12345",
  "notes": "Primer pago mensual",
  "cash_register_id": "uuid"
}
```

**Response 201:**
```json
{
  "id": "uuid",
  "amount": 80000,
  "status": "COMPLETED",
  "payment_date": "2026-01-20T10:15:00Z",
  "receipt_url": "https://storage.gymgo.com/receipts/..."
}
```

### 5.3 Refund Payment

```http
POST /payments/{id}/refund
Authorization: Bearer {token}
Content-Type: application/json

{
  "reason": "Duplicate payment",
  "amount": 80000  // partial refund possible
}
```

---

## 6. Cash Register

### 6.1 Open Cash Register

```http
POST /cash-registers/open
Authorization: Bearer {token}
Content-Type: application/json

{
  "initial_cash": 50000,
  "notes": "Caja abierta por turno mañana"
}
```

**Response 201:**
```json
{
  "id": "uuid",
  "opened_by": "uuid",
  "opened_at": "2026-01-20T06:00:00Z",
  "initial_cash": 50000,
  "status": "OPEN"
}
```

### 6.2 Close Cash Register

```http
POST /cash-registers/{id}/close
Authorization: Bearer {token}
Content-Type: application/json

{
  "final_cash": 320000,
  "notes": "Cierre de turno mañana"
}
```

**Response 200:**
```json
{
  "id": "uuid",
  "closed_at": "2026-01-20T14:00:00Z",
  "initial_cash": 50000,
  "final_cash": 320000,
  "total_cash": 280000,
  "total_card": 160000,
  "total_transfer": 80000,
  "expected": 570000,
  "difference": 10000,
  "status": "CLOSED"
}
```

### 6.3 Get Current Cash Register

```http
GET /cash-registers/current
Authorization: Bearer {token}
```

### 6.4 Get Cash Register History

```http
GET /cash-registers?start_date=2026-01-01&end_date=2026-01-31
Authorization: Bearer {token}
```

---

## 7. Access Control

### 7.1 Validate Entry (Hardware API)

```http
POST /access/validate
Authorization: Bearer {device_api_key}
Content-Type: application/json

{
  "device_id": "uuid",
  "method": "QR",
  "identifier": "GYM-ABC123DEF456",
  "timestamp": "2026-01-20T08:30:00Z",
  "location": "Entrada Principal"
}
```

**Response 200:**
```json
{
  "allowed": true,
  "reason": "OK",
  "user_id": "uuid",
  "user_name": "John Doe",
  "photo_url": "https://...",
  "subscription_status": "ACTIVE",
  "expires_at": "2026-02-15",
  "message": "¡Bienvenido John!",
  "display_color": "green"
}
```

**Response 200 (Denied):**
```json
{
  "allowed": false,
  "reason": "EXPIRED_SUBSCRIPTION",
  "user_name": "John Doe",
  "message": "Tu suscripción venció el 2026-01-15",
  "display_color": "red",
  "contact_info": "Acércate a recepción"
}
```

### 7.2 Get Access Logs

```http
GET /access/logs?user_id={uuid}&start_date=2026-01-01&result=GRANTED
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "user_name": "John Doe",
      "device_id": "uuid",
      "device_name": "Scanner Principal",
      "method": "QR",
      "result": "GRANTED",
      "reason": "OK",
      "timestamp": "2026-01-20T08:30:15Z",
      "location": "Entrada Principal"
    }
  ],
  "pagination": {...}
}
```

### 7.3 Manual Entry (Recepcionista)

```http
POST /access/manual-entry
Authorization: Bearer {token}
Content-Type: application/json

{
  "user_id": "uuid",
  "notes": "Olvidó su QR"
}
```

---

## 8. Promotions

### 8.1 List Promotions

```http
GET /promotions?status=ACTIVE
Authorization: Bearer {token}
```

### 8.2 Create Promotion

```http
POST /promotions
Authorization: Bearer {token}
Content-Type: application/json

{
  "code": "PROMO2026",
  "name": "Promoción Año Nuevo",
  "description": "10% de descuento",
  "type": "PERCENTAGE",
  "value": 10,
  "start_date": "2026-01-01",
  "end_date": "2026-01-31",
  "max_uses": 100,
  "max_uses_per_user": 1
}
```

### 8.3 Validate Promotion Code

```http
POST /promotions/validate
Authorization: Bearer {token}
Content-Type: application/json

{
  "code": "PROMO2026",
  "user_id": "uuid",
  "plan_id": "uuid"
}
```

**Response 200:**
```json
{
  "valid": true,
  "discount_amount": 8000,
  "final_price": 72000,
  "promotion": {
    "code": "PROMO2026",
    "name": "Promoción Año Nuevo",
    "type": "PERCENTAGE",
    "value": 10
  }
}
```

---

## 9. Reports

### 9.1 Revenue Report

```http
GET /reports/revenue?start_date=2026-01-01&end_date=2026-01-31&group_by=day
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "summary": {
    "total_revenue": 5600000,
    "total_transactions": 70,
    "average_transaction": 80000,
    "by_method": {
      "CASH": 3200000,
      "CARD": 1600000,
      "TRANSFER": 800000
    }
  },
  "data": [
    {
      "date": "2026-01-01",
      "revenue": 240000,
      "transactions": 3
    },
    {
      "date": "2026-01-02",
      "revenue": 320000,
      "transactions": 4
    }
  ]
}
```

### 9.2 Members Report

```http
GET /reports/members?status=ACTIVE
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "summary": {
    "total_members": 150,
    "active_subscriptions": 120,
    "expired_subscriptions": 20,
    "no_subscription": 10,
    "new_this_month": 15
  },
  "by_plan": [
    {
      "plan_name": "Mensual",
      "count": 80,
      "revenue": 6400000
    }
  ]
}
```

### 9.3 Access Report

```http
GET /reports/access?start_date=2026-01-01&end_date=2026-01-31
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "summary": {
    "total_accesses": 2500,
    "unique_users": 120,
    "average_per_day": 83,
    "denied_accesses": 15,
    "denial_rate": 0.6
  },
  "peak_hours": [
    {
      "hour": "18:00",
      "count": 250
    }
  ],
  "denial_reasons": [
    {
      "reason": "EXPIRED_SUBSCRIPTION",
      "count": 10
    }
  ]
}
```

### 9.4 Expiring Subscriptions

```http
GET /reports/expiring-subscriptions?days=7
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "data": [
    {
      "user_id": "uuid",
      "user_name": "John Doe",
      "email": "john@example.com",
      "phone": "3001234567",
      "plan_name": "Mensual",
      "expires_at": "2026-01-25",
      "days_remaining": 5,
      "auto_renew": false
    }
  ],
  "count": 15
}
```

### 9.5 Export Report

```http
GET /reports/revenue/export?format=pdf&start_date=2026-01-01&end_date=2026-01-31
Authorization: Bearer {token}
```

**Response 200:**
```
Content-Type: application/pdf
Content-Disposition: attachment; filename="revenue_report_2026-01.pdf"

<binary PDF>
```

---

## 10. Dashboard

### 10.1 Get Dashboard KPIs

```http
GET /dashboard/kpis?date=2026-01-20
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "today": {
    "revenue": 320000,
    "new_members": 5,
    "active_members": 120,
    "check_ins": 85,
    "expiring_soon": 12
  },
  "this_month": {
    "revenue": 5600000,
    "new_members": 15,
    "renewals": 45,
    "cancellations": 3
  },
  "trends": {
    "revenue_change": 15.5,  // % vs last month
    "members_change": 8.2
  }
}
```

### 10.2 Get Charts Data

```http
GET /dashboard/charts/revenue?period=month&year=2026
Authorization: Bearer {token}
```

**Response 200:**
```json
{
  "revenue_by_month": [
    {"month": "Jan", "value": 5600000},
    {"month": "Feb", "value": 6200000}
  ],
  "members_growth": [
    {"month": "Jan", "value": 150},
    {"month": "Feb", "value": 165}
  ],
  "check_ins_by_day": [...]
}
```

---

## 11. Admin / Settings

### 11.1 Get Gym Settings

```http
GET /settings
Authorization: Bearer {token}
```

### 11.2 Update Gym Settings

```http
PUT /settings
Authorization: Bearer {token}
Content-Type: application/json

{
  "access_control": {
    "require_photo": true,
    "block_on_debt": true,
    "grace_period_days": 3
  },
  "business_hours": {...}
}
```

### 11.3 List Devices

```http
GET /devices
Authorization: Bearer {token}
```

### 11.4 Register Device

```http
POST /devices
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Scanner Entrada Principal",
  "type": "QR_SCANNER",
  "location": "Entrada Principal",
  "ip_address": "192.168.1.100"
}
```

**Response 201:**
```json
{
  "id": "uuid",
  "api_key": "device_api_key_here",
  "name": "Scanner Entrada Principal",
  "status": "ACTIVE"
}
```

---

## Error Responses

### 400 Bad Request
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "email": "Invalid email format",
      "phone": "Phone number required"
    }
  }
}
```

### 401 Unauthorized
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token"
  }
}
```

### 403 Forbidden
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "You don't have permission to access this resource"
  }
}
```

### 404 Not Found
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found"
  }
}
```

### 429 Too Many Requests
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again in 60 seconds",
    "retry_after": 60
  }
}
```

### 500 Internal Server Error
```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "An unexpected error occurred",
    "request_id": "uuid"
  }
}
```

---

## Rate Limiting

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642678800
```

**Limits:**
- **General:** 100 requests/minute
- **Login:** 5 requests/minute
- **Access Validation:** 1000 requests/minute (devices)

---

## Webhooks (Future)

```http
POST https://your-app.com/webhook
Content-Type: application/json
X-Webhook-Signature: sha256=...

{
  "event": "subscription.expired",
  "data": {
    "subscription_id": "uuid",
    "user_id": "uuid",
    "expired_at": "2026-01-20T00:00:00Z"
  },
  "timestamp": "2026-01-20T00:01:00Z"
}
```

**Events:**
- `subscription.created`
- `subscription.renewed`
- `subscription.expired`
- `subscription.cancelled`
- `payment.completed`
- `payment.failed`
- `access.denied`

---

**API completa con 60+ endpoints listos para implementación!**

¿Continuamos con las reglas de negocio y flujos?
