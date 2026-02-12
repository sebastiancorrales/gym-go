-- Migración 003: Sistema de Inventarios y Ventas
-- Fecha: 2026-02-11
-- Descripción: Tablas para gestión de inventarios, ventas y métodos de pago

-- Tabla de productos
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    unit_price REAL NOT NULL CHECK(unit_price >= 0),
    stock INTEGER NOT NULL DEFAULT 0 CHECK(stock >= 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'inactive')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Índices para productos
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);
CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at);

-- Tabla de métodos de pago
CREATE TABLE IF NOT EXISTS payment_methods (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('cash', 'card', 'transfer')),
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'inactive')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Índices para métodos de pago
CREATE INDEX IF NOT EXISTS idx_payment_methods_status ON payment_methods(status);
CREATE INDEX IF NOT EXISTS idx_payment_methods_type ON payment_methods(type);

-- Tabla de ventas
CREATE TABLE IF NOT EXISTS sales (
    id TEXT PRIMARY KEY,
    sale_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    total REAL NOT NULL DEFAULT 0,
    total_discount REAL NOT NULL DEFAULT 0,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('normal', 'void')),
    status TEXT NOT NULL CHECK(status IN ('completed', 'voided', 'pending')),
    payment_method_id TEXT NOT NULL,
    voided_sale_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id),
    FOREIGN KEY (voided_sale_id) REFERENCES sales(id)
);

-- Índices para ventas
CREATE INDEX IF NOT EXISTS idx_sales_sale_date ON sales(sale_date);
CREATE INDEX IF NOT EXISTS idx_sales_user_id ON sales(user_id);
CREATE INDEX IF NOT EXISTS idx_sales_type ON sales(type);
CREATE INDEX IF NOT EXISTS idx_sales_status ON sales(status);
CREATE INDEX IF NOT EXISTS idx_sales_payment_method_id ON sales(payment_method_id);
CREATE INDEX IF NOT EXISTS idx_sales_voided_sale_id ON sales(voided_sale_id);
CREATE INDEX IF NOT EXISTS idx_sales_date_user ON sales(sale_date, user_id);
CREATE INDEX IF NOT EXISTS idx_sales_date_status ON sales(sale_date, status);

-- Tabla de detalles de venta
CREATE TABLE IF NOT EXISTS sale_details (
    id TEXT PRIMARY KEY,
    sale_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    unit_price REAL NOT NULL CHECK(unit_price >= 0),
    quantity INTEGER NOT NULL CHECK(quantity > 0),
    total_price REAL NOT NULL DEFAULT 0,
    discount REAL NOT NULL DEFAULT 0 CHECK(discount >= 0),
    subtotal REAL NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sale_id) REFERENCES sales(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Índices para detalles de venta
CREATE INDEX IF NOT EXISTS idx_sale_details_sale_id ON sale_details(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_details_product_id ON sale_details(product_id);
CREATE INDEX IF NOT EXISTS idx_sale_details_created_at ON sale_details(created_at);

-- Trigger para actualizar updated_at en products
CREATE TRIGGER IF NOT EXISTS update_products_timestamp 
AFTER UPDATE ON products
FOR EACH ROW
BEGIN
    UPDATE products SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Trigger para actualizar updated_at en payment_methods
CREATE TRIGGER IF NOT EXISTS update_payment_methods_timestamp 
AFTER UPDATE ON payment_methods
FOR EACH ROW
BEGIN
    UPDATE payment_methods SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Trigger para actualizar updated_at en sales
CREATE TRIGGER IF NOT EXISTS update_sales_timestamp 
AFTER UPDATE ON sales
FOR EACH ROW
BEGIN
    UPDATE sales SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Vista para reporte de ventas
CREATE VIEW IF NOT EXISTS v_sales_summary AS
SELECT 
    s.id,
    s.sale_date,
    s.total,
    s.total_discount,
    s.type,
    s.status,
    u.username as user_name,
    pm.name as payment_method_name,
    COUNT(sd.id) as items_count
FROM sales s
LEFT JOIN users u ON s.user_id = u.id
LEFT JOIN payment_methods pm ON s.payment_method_id = pm.id
LEFT JOIN sale_details sd ON s.id = sd.sale_id
GROUP BY s.id;

-- Vista para productos con bajo stock (menos de 10 unidades)
CREATE VIEW IF NOT EXISTS v_products_low_stock AS
SELECT 
    id,
    name,
    description,
    unit_price,
    stock,
    status
FROM products
WHERE stock < 10 AND status = 'active'
ORDER BY stock ASC;

-- Vista para ventas con detalles
CREATE VIEW IF NOT EXISTS v_sales_with_details AS
SELECT 
    s.id as sale_id,
    s.sale_date,
    s.total as sale_total,
    s.total_discount as sale_discount,
    s.type as sale_type,
    s.status as sale_status,
    sd.id as detail_id,
    sd.product_id,
    p.name as product_name,
    sd.quantity,
    sd.unit_price,
    sd.total_price,
    sd.discount,
    sd.subtotal
FROM sales s
LEFT JOIN sale_details sd ON s.id = sd.sale_id
LEFT JOIN products p ON sd.product_id = p.id;
