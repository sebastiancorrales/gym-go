# 📦 Componentes de Inventario y Ventas

Módulo completo de inventarios y ventas para gym-go.

## Componentes

### 🏷️ ProductsManagement.jsx
CRUD completo de productos con búsqueda y gestión de stock.

### 💰 SalesTab.jsx
Punto de venta (POS) con carrito interactivo y proceso de pago.

### 📋 SalesHistory.jsx
Historial de ventas con filtros por fecha y opción de anulación.

### 📊 ReportsTab.jsx
Reportes de ventas por fecha y por producto con estadísticas visuales.

### 💳 PaymentMethodsManagement.jsx
Gestión de métodos de pago (solo administradores).

---

## Integración

Todos los componentes están integrados en `Dashboard.jsx` como tabs independientes:

- **Productos** → Gestión de inventario
- **Ventas** → Punto de venta
- **Historial** → Consulta de ventas
- **Reportes** → Análisis y estadísticas
- **Métodos de Pago** → Configuración (admin)

## Stack

- React 18
- TailwindCSS
- Cliente HTTP personalizado (api.js)

## API Backend

Integrado con todos los endpoints documentados en `INVENTORY_API_SUMMARY.md`
