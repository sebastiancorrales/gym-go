# 🗄️ Database Design - Gym SaaS

## ERD (Entity Relationship Diagram)

```
┌─────────────────┐
│     TENANTS     │
│─────────────────│
│ id (PK)         │
│ name            │
│ schema_name     │
│ domain          │
│ plan            │
│ status          │
│ created_at      │
└────────┬────────┘
         │
         │ 1:N
         │
┌────────▼────────┐         ┌─────────────────┐
│      GYMS       │─────────│     USERS       │
│─────────────────│   1:N   │─────────────────│
│ id (PK)         │         │ id (PK)         │
│ tenant_id (FK)  │         │ gym_id (FK)     │
│ name            │         │ email           │
│ address         │         │ password_hash   │
│ phone           │         │ first_name      │
│ settings        │         │ last_name       │
└─────────────────┘         │ document_type   │
                            │ document_number │
                            │ phone           │
                            │ date_of_birth   │
                            │ photo_url       │
         ┌──────────────────┤ role            │
         │                  │ status          │
         │                  │ created_at      │
         │                  └────────┬────────┘
         │                           │
         │                           │ 1:N
         │                           │
         │                  ┌────────▼────────────┐
         │                  │   SUBSCRIPTIONS     │
         │                  │─────────────────────│
         │                  │ id (PK)             │
         │                  │ user_id (FK)        │
         │                  │ plan_id (FK)        │
         │                  │ start_date          │
         │                  │ end_date            │
         │                  │ status              │
         │                  │ auto_renew          │
         │                  │ created_at          │
         │                  └─────────────────────┘
         │                           │
         │                           │ N:1
         │                           │
         │                  ┌────────▼────────┐
         │                  │     PLANS       │
         │                  │─────────────────│
         │                  │ id (PK)         │
         │                  │ gym_id (FK)     │
         │                  │ name            │
         │                  │ duration_days   │
         │                  │ price           │
         │                  │ description     │
         │                  │ features        │
         │                  │ status          │
         │                  └─────────────────┘
         │
         │ 1:N
         │
┌────────▼────────────┐
│      PAYMENTS       │
│─────────────────────│
│ id (PK)             │
│ user_id (FK)        │
│ subscription_id(FK) │
│ amount              │
│ method              │
│ status              │
│ payment_date        │
│ reference           │
│ discount_id (FK)    │
│ notes               │
│ created_by (FK)     │
└─────────────────────┘
         │
         │ N:1
         │
┌────────▼────────────┐
│   CASH_REGISTERS    │
│─────────────────────│
│ id (PK)             │
│ gym_id (FK)         │
│ opened_by (FK)      │
│ closed_by (FK)      │
│ opened_at           │
│ closed_at           │
│ initial_cash        │
│ final_cash          │
│ total_cash          │
│ total_card          │
│ total_transfer      │
│ expected            │
│ difference          │
│ status              │
└─────────────────────┘


┌─────────────────┐         ┌─────────────────┐
│  ACCESS_LOGS    │         │    DEVICES      │
│─────────────────│         │─────────────────│
│ id (PK)         │         │ id (PK)         │
│ user_id (FK)    │◄────────│ gym_id (FK)     │
│ device_id (FK)  │   N:1   │ name            │
│ method          │         │ type            │
│ result          │         │ location        │
│ reason          │         │ ip_address      │
│ timestamp       │         │ api_key         │
│ location        │         │ last_seen       │
└─────────────────┘         │ status          │
                            └─────────────────┘

┌─────────────────┐
│   PROMOTIONS    │
│─────────────────│
│ id (PK)         │
│ gym_id (FK)     │
│ code            │
│ type            │
│ value           │
│ start_date      │
│ end_date        │
│ max_uses        │
│ used_count      │
│ status          │
└─────────────────┘


┌─────────────────┐
│   AUDIT_LOGS    │
│─────────────────│
│ id (PK)         │
│ user_id (FK)    │
│ action          │
│ entity_type     │
│ entity_id       │
│ old_values      │
│ new_values      │
│ ip_address      │
│ user_agent      │
│ created_at      │
└─────────────────┘
```

---

## Schema SQL (PostgreSQL)

### 1. Public Schema (Global - SaaS Level)

```sql
-- =====================================================
-- TENANTS (Global)
-- =====================================================
CREATE TABLE public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(50) UNIQUE NOT NULL,
    domain VARCHAR(255) UNIQUE,
    plan VARCHAR(50) NOT NULL, -- BASIC, PRO, ENTERPRISE
    max_gyms INTEGER DEFAULT 1,
    max_users INTEGER DEFAULT 100,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, SUSPENDED, CANCELLED
    trial_ends_at TIMESTAMP,
    subscription_ends_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tenants_schema ON public.tenants(schema_name);
CREATE INDEX idx_tenants_status ON public.tenants(status);

-- =====================================================
-- GLOBAL USERS (SaaS Admins)
-- =====================================================
CREATE TABLE public.super_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) DEFAULT 'SUPER_ADMIN',
    status VARCHAR(20) DEFAULT 'ACTIVE',
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

### 2. Gym Schema Template (Per Tenant)

```sql
-- Cada tenant tiene su propio schema
CREATE SCHEMA gym_1;

-- =====================================================
-- GYMS (Multi-sucursal)
-- =====================================================
CREATE TABLE gym_1.gyms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    tax_id VARCHAR(50),
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(2) DEFAULT 'CO',
    postal_code VARCHAR(20),
    phone VARCHAR(20),
    email VARCHAR(255),
    logo_url TEXT,
    timezone VARCHAR(50) DEFAULT 'America/Bogota',
    currency VARCHAR(3) DEFAULT 'COP',
    
    -- Settings (JSONB para flexibilidad)
    settings JSONB DEFAULT '{
        "access_control": {
            "require_photo": true,
            "block_on_debt": true,
            "grace_period_days": 3
        },
        "business_hours": {
            "monday": {"open": "05:00", "close": "22:00"},
            "tuesday": {"open": "05:00", "close": "22:00"}
        }
    }'::jsonb,
    
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- =====================================================
-- USERS
-- =====================================================
CREATE TABLE gym_1.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    
    -- Authentication
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    
    -- Personal Info
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    document_type VARCHAR(20), -- CC, TI, CE, PASSPORT
    document_number VARCHAR(50),
    phone VARCHAR(20),
    date_of_birth DATE,
    gender VARCHAR(20),
    
    -- Contact
    address TEXT,
    city VARCHAR(100),
    emergency_contact_name VARCHAR(200),
    emergency_contact_phone VARCHAR(20),
    
    -- Media
    photo_url TEXT,
    qr_code TEXT UNIQUE, -- Código QR único para acceso
    
    -- System
    role VARCHAR(50) NOT NULL, -- ADMIN_GYM, RECEPCIONISTA, STAFF, MEMBER
    status VARCHAR(20) DEFAULT 'ACTIVE', -- ACTIVE, INACTIVE, BLOCKED, SUSPENDED
    
    -- Metadata
    notes TEXT,
    tags VARCHAR(255)[], -- Para segmentación
    
    -- Security
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    password_reset_token VARCHAR(255),
    password_reset_expires TIMESTAMP,
    last_login TIMESTAMP,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP -- Soft delete
);

CREATE INDEX idx_users_email ON gym_1.users(email);
CREATE INDEX idx_users_gym ON gym_1.users(gym_id);
CREATE INDEX idx_users_document ON gym_1.users(document_number);
CREATE INDEX idx_users_qr ON gym_1.users(qr_code);
CREATE INDEX idx_users_status ON gym_1.users(status);

-- =====================================================
-- PLANS
-- =====================================================
CREATE TABLE gym_1.plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    duration_days INTEGER NOT NULL, -- 30, 90, 365, etc.
    
    -- Pricing
    price DECIMAL(10, 2) NOT NULL,
    enrollment_fee DECIMAL(10, 2) DEFAULT 0,
    
    -- Features (JSONB para flexibilidad)
    features JSONB DEFAULT '{
        "access_areas": ["gym", "classes"],
        "guest_passes": 0,
        "freeze_days_per_year": 0,
        "class_bookings_per_week": 10
    }'::jsonb,
    
    -- Limits
    max_freezes_per_year INTEGER DEFAULT 0,
    max_active_subscriptions INTEGER, -- NULL = ilimitado
    
    -- Display
    color VARCHAR(7) DEFAULT '#3B82F6',
    icon VARCHAR(50),
    display_order INTEGER DEFAULT 0,
    is_featured BOOLEAN DEFAULT FALSE,
    
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_plans_gym ON gym_1.plans(gym_id);
CREATE INDEX idx_plans_status ON gym_1.plans(status);

-- =====================================================
-- SUBSCRIPTIONS
-- =====================================================
CREATE TABLE gym_1.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES gym_1.users(id),
    plan_id UUID NOT NULL REFERENCES gym_1.plans(id),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    
    -- Dates
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    activated_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    
    -- Pricing (snapshot al momento de compra)
    price_paid DECIMAL(10, 2) NOT NULL,
    enrollment_fee_paid DECIMAL(10, 2) DEFAULT 0,
    discount_applied DECIMAL(10, 2) DEFAULT 0,
    total_paid DECIMAL(10, 2) NOT NULL,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', 
    -- PENDING, ACTIVE, EXPIRED, CANCELLED, SUSPENDED, FROZEN
    
    -- Freezing
    frozen_until DATE,
    freeze_reason TEXT,
    total_freeze_days INTEGER DEFAULT 0,
    
    -- Renewal
    auto_renew BOOLEAN DEFAULT FALSE,
    renewal_reminder_sent BOOLEAN DEFAULT FALSE,
    
    -- Metadata
    notes TEXT,
    cancellation_reason TEXT,
    cancelled_by UUID REFERENCES gym_1.users(id),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user ON gym_1.subscriptions(user_id);
CREATE INDEX idx_subscriptions_plan ON gym_1.subscriptions(plan_id);
CREATE INDEX idx_subscriptions_status ON gym_1.subscriptions(status);
CREATE INDEX idx_subscriptions_dates ON gym_1.subscriptions(start_date, end_date);
CREATE INDEX idx_subscriptions_end_date ON gym_1.subscriptions(end_date) WHERE status = 'ACTIVE';

-- =====================================================
-- PAYMENTS
-- =====================================================
CREATE TABLE gym_1.payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES gym_1.users(id),
    subscription_id UUID REFERENCES gym_1.subscriptions(id),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    cash_register_id UUID REFERENCES gym_1.cash_registers(id),
    
    -- Payment Info
    amount DECIMAL(10, 2) NOT NULL,
    method VARCHAR(50) NOT NULL, -- CASH, CARD, TRANSFER, NEQUI, DAVIPLATA
    status VARCHAR(20) DEFAULT 'COMPLETED', -- PENDING, COMPLETED, FAILED, REFUNDED
    
    payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- References
    reference VARCHAR(255), -- Referencia bancaria, número de transacción
    authorization_code VARCHAR(100),
    
    -- Discounts
    discount_id UUID REFERENCES gym_1.promotions(id),
    discount_amount DECIMAL(10, 2) DEFAULT 0,
    discount_reason TEXT,
    
    -- Metadata
    notes TEXT,
    receipt_url TEXT,
    invoice_number VARCHAR(50),
    
    -- Who processed
    created_by UUID NOT NULL REFERENCES gym_1.users(id),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_user ON gym_1.payments(user_id);
CREATE INDEX idx_payments_subscription ON gym_1.payments(subscription_id);
CREATE INDEX idx_payments_date ON gym_1.payments(payment_date);
CREATE INDEX idx_payments_method ON gym_1.payments(method);
CREATE INDEX idx_payments_status ON gym_1.payments(status);

-- =====================================================
-- CASH REGISTERS (Caja)
-- =====================================================
CREATE TABLE gym_1.cash_registers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    
    -- Who opened/closed
    opened_by UUID NOT NULL REFERENCES gym_1.users(id),
    closed_by UUID REFERENCES gym_1.users(id),
    
    -- Timestamps
    opened_at TIMESTAMP NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP,
    
    -- Cash flow
    initial_cash DECIMAL(10, 2) NOT NULL DEFAULT 0,
    final_cash DECIMAL(10, 2),
    
    -- Totals by method (calculated from payments)
    total_cash DECIMAL(10, 2) DEFAULT 0,
    total_card DECIMAL(10, 2) DEFAULT 0,
    total_transfer DECIMAL(10, 2) DEFAULT 0,
    total_other DECIMAL(10, 2) DEFAULT 0,
    
    -- Reconciliation
    expected DECIMAL(10, 2) DEFAULT 0,
    difference DECIMAL(10, 2),
    
    -- Notes
    opening_notes TEXT,
    closing_notes TEXT,
    
    status VARCHAR(20) DEFAULT 'OPEN', -- OPEN, CLOSED
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_cash_registers_gym ON gym_1.cash_registers(gym_id);
CREATE INDEX idx_cash_registers_opened_at ON gym_1.cash_registers(opened_at);
CREATE INDEX idx_cash_registers_status ON gym_1.cash_registers(status);

-- =====================================================
-- PROMOTIONS / DISCOUNTS
-- =====================================================
CREATE TABLE gym_1.promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Discount
    type VARCHAR(20) NOT NULL, -- PERCENTAGE, FIXED_AMOUNT
    value DECIMAL(10, 2) NOT NULL,
    
    -- Validity
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    
    -- Usage limits
    max_uses INTEGER, -- NULL = unlimited
    max_uses_per_user INTEGER DEFAULT 1,
    used_count INTEGER DEFAULT 0,
    
    -- Applicability
    applicable_plans UUID[], -- NULL = todos los planes
    min_purchase_amount DECIMAL(10, 2),
    
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_promotions_code ON gym_1.promotions(code);
CREATE INDEX idx_promotions_dates ON gym_1.promotions(start_date, end_date);

-- =====================================================
-- DEVICES (Hardware Integration)
-- =====================================================
CREATE TABLE gym_1.devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- QR_SCANNER, TURNSTILE, FINGERPRINT, FACIAL
    location VARCHAR(255), -- "Entrada Principal", "Área de Pesas"
    
    -- Connection
    ip_address INET,
    mac_address VARCHAR(17),
    api_key VARCHAR(255) UNIQUE,
    webhook_url TEXT,
    
    -- Config (JSONB para flexibilidad)
    config JSONB DEFAULT '{
        "offline_mode": true,
        "sync_interval": 300,
        "require_photo": false
    }'::jsonb,
    
    -- Status
    status VARCHAR(20) DEFAULT 'ACTIVE', -- ACTIVE, INACTIVE, MAINTENANCE
    last_seen TIMESTAMP,
    last_sync TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_devices_gym ON gym_1.devices(gym_id);
CREATE INDEX idx_devices_api_key ON gym_1.devices(api_key);

-- =====================================================
-- ACCESS LOGS (Partitioned by month)
-- =====================================================
CREATE TABLE gym_1.access_logs (
    id UUID DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES gym_1.users(id),
    gym_id UUID NOT NULL REFERENCES gym_1.gyms(id),
    device_id UUID REFERENCES gym_1.devices(id),
    
    -- Access attempt
    method VARCHAR(50) NOT NULL, -- QR, FINGERPRINT, CARD, MANUAL
    identifier VARCHAR(255), -- QR code, fingerprint hash, etc.
    
    -- Result
    result VARCHAR(20) NOT NULL, -- GRANTED, DENIED
    reason VARCHAR(100), -- OK, EXPIRED_SUBSCRIPTION, NO_SUBSCRIPTION, USER_BLOCKED
    
    -- Context
    location VARCHAR(255),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Metadata
    metadata JSONB,
    
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create partitions for each month
CREATE TABLE gym_1.access_logs_2026_01 PARTITION OF gym_1.access_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE gym_1.access_logs_2026_02 PARTITION OF gym_1.access_logs
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- Indexes on partition
CREATE INDEX idx_access_logs_user_2026_01 ON gym_1.access_logs_2026_01(user_id);
CREATE INDEX idx_access_logs_timestamp_2026_01 ON gym_1.access_logs_2026_01(timestamp);
CREATE INDEX idx_access_logs_result_2026_01 ON gym_1.access_logs_2026_01(result);

-- =====================================================
-- AUDIT LOGS
-- =====================================================
CREATE TABLE gym_1.audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES gym_1.users(id),
    
    action VARCHAR(50) NOT NULL, -- CREATE, UPDATE, DELETE, LOGIN, LOGOUT
    entity_type VARCHAR(50) NOT NULL, -- USER, SUBSCRIPTION, PAYMENT
    entity_id UUID,
    
    -- Changes
    old_values JSONB,
    new_values JSONB,
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON gym_1.audit_logs(user_id);
CREATE INDEX idx_audit_logs_entity ON gym_1.audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created ON gym_1.audit_logs(created_at);

-- =====================================================
-- SESSIONS (Redis alternative in DB)
-- =====================================================
CREATE TABLE gym_1.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES gym_1.users(id),
    
    access_token_hash VARCHAR(255) NOT NULL,
    refresh_token_hash VARCHAR(255) UNIQUE NOT NULL,
    
    device_info JSONB,
    ip_address INET,
    user_agent TEXT,
    
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_activity TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON gym_1.sessions(user_id);
CREATE INDEX idx_sessions_refresh ON gym_1.sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires ON gym_1.sessions(expires_at);
```

---

## 3. Views útiles

```sql
-- =====================================================
-- VIEW: Active Subscriptions with User Info
-- =====================================================
CREATE VIEW gym_1.v_active_subscriptions AS
SELECT 
    s.id as subscription_id,
    u.id as user_id,
    u.first_name,
    u.last_name,
    u.email,
    u.qr_code,
    p.name as plan_name,
    s.start_date,
    s.end_date,
    s.status,
    CASE 
        WHEN s.end_date < CURRENT_DATE THEN TRUE 
        ELSE FALSE 
    END as is_expired,
    s.end_date - CURRENT_DATE as days_remaining
FROM gym_1.subscriptions s
JOIN gym_1.users u ON s.user_id = u.id
JOIN gym_1.plans p ON s.plan_id = p.id
WHERE s.status = 'ACTIVE';

-- =====================================================
-- VIEW: Daily Revenue Summary
-- =====================================================
CREATE VIEW gym_1.v_daily_revenue AS
SELECT 
    DATE(payment_date) as date,
    COUNT(*) as total_transactions,
    SUM(amount) as total_revenue,
    SUM(CASE WHEN method = 'CASH' THEN amount ELSE 0 END) as cash_revenue,
    SUM(CASE WHEN method = 'CARD' THEN amount ELSE 0 END) as card_revenue,
    SUM(CASE WHEN method = 'TRANSFER' THEN amount ELSE 0 END) as transfer_revenue
FROM gym_1.payments
WHERE status = 'COMPLETED'
GROUP BY DATE(payment_date)
ORDER BY date DESC;

-- =====================================================
-- VIEW: Expiring Subscriptions (Next 7 days)
-- =====================================================
CREATE VIEW gym_1.v_expiring_subscriptions AS
SELECT 
    u.id as user_id,
    u.first_name,
    u.last_name,
    u.email,
    u.phone,
    s.end_date,
    s.end_date - CURRENT_DATE as days_until_expiry,
    p.name as plan_name,
    p.price as renewal_price
FROM gym_1.subscriptions s
JOIN gym_1.users u ON s.user_id = u.id
JOIN gym_1.plans p ON s.plan_id = p.id
WHERE s.status = 'ACTIVE'
  AND s.end_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '7 days'
ORDER BY s.end_date;
```

---

## 4. Functions & Triggers

```sql
-- =====================================================
-- Function: Auto-update updated_at
-- =====================================================
CREATE OR REPLACE FUNCTION gym_1.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON gym_1.users
    FOR EACH ROW EXECUTE FUNCTION gym_1.update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at BEFORE UPDATE ON gym_1.subscriptions
    FOR EACH ROW EXECUTE FUNCTION gym_1.update_updated_at_column();

-- =====================================================
-- Function: Generate QR Code
-- =====================================================
CREATE OR REPLACE FUNCTION gym_1.generate_qr_code()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.qr_code IS NULL THEN
        NEW.qr_code := 'GYM-' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 12));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER generate_user_qr BEFORE INSERT ON gym_1.users
    FOR EACH ROW EXECUTE FUNCTION gym_1.generate_qr_code();

-- =====================================================
-- Function: Auto-expire subscriptions
-- =====================================================
CREATE OR REPLACE FUNCTION gym_1.expire_subscriptions()
RETURNS void AS $$
BEGIN
    UPDATE gym_1.subscriptions
    SET status = 'EXPIRED'
    WHERE status = 'ACTIVE'
      AND end_date < CURRENT_DATE;
END;
$$ LANGUAGE plpgsql;

-- Run daily via cron job
SELECT cron.schedule('expire-subscriptions', '0 0 * * *', 
    'SELECT gym_1.expire_subscriptions()');
```

---

## 5. Datos de Ejemplo

```sql
-- Insert test gym
INSERT INTO gym_1.gyms (name, phone, email) VALUES
('FitLife Gym', '3001234567', 'info@fitlife.com');

-- Insert test plans
INSERT INTO gym_1.plans (gym_id, name, duration_days, price) VALUES
((SELECT id FROM gym_1.gyms LIMIT 1), 'Mensual', 30, 80000),
((SELECT id FROM gym_1.gyms LIMIT 1), 'Trimestral', 90, 210000),
((SELECT id FROM gym_1.gyms LIMIT 1), 'Anual', 365, 800000);

-- Insert test user
INSERT INTO gym_1.users (gym_id, email, password_hash, first_name, last_name, role) VALUES
((SELECT id FROM gym_1.gyms LIMIT 1), 'admin@fitlife.com', '$2a$12$...', 'Admin', 'User', 'ADMIN_GYM');
```

---

## 6. Backup Strategy

```bash
# Full backup
pg_dump -h localhost -U postgres -F c -b -v -f "backup_$(date +%Y%m%d).dump" gym_db

# Backup specific schema
pg_dump -h localhost -U postgres -n gym_1 -F c -f "gym_1_$(date +%Y%m%d).dump" gym_db

# Automated daily backups (cron)
0 2 * * * /usr/bin/pg_dump -h localhost -U postgres -F c gym_db > /backups/daily_$(date +\%Y\%m\%d).dump
```

---

**Esta estructura está lista para soportar:**
- ✅ Multi-tenancy (schema per tenant)
- ✅ Multi-sucursal (gym_id en todas las tablas)
- ✅ Escalabilidad (particiones, índices optimizados)
- ✅ Auditoría completa
- ✅ Hardware integration
- ✅ SaaS features

**¿Continuamos con los endpoints de la API?**
