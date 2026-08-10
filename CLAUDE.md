# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend
```bash
# Run (dev)
go run main.go

# Build production binary (embeds frontend/dist)
cd frontend && npm run build && cd ..
go build -o gym-go.exe .

# Run tests
go test ./...
go test ./internal/usecases/...   # single package
```

### Frontend (`cd frontend` first)
```bash
npm run dev      # dev server on :5173, proxies /api to :8080
npm run build    # outputs to frontend/dist — required before go build
npm run lint     # eslint
```

### Dev workflow
Run both processes simultaneously: `go run main.go` (backend on :8080) and `npm run dev` (frontend on :5173). Vite proxies `/api` to the backend automatically.

## Architecture

**Clean Architecture in three layers:**

```
domain/entities/        → structs + business rules (no external deps)
internal/usecases/      → orchestration logic, depend only on repo interfaces
internal/infrastructure/
  persistence/          → SQLite/GORM repos implementing domain interfaces
  http/handlers/        → Gin handlers, depend only on use cases
  http/middleware/      → auth (JWT), CORS, role guard, gym timezone
  http/dto/             → request/response types (separate from entities)
```

**Important:** `internal/infrastructure/http/router.go` is an **outdated stub** and is not used. All real routes are registered in `main.go` directly.

**Database:** SQLite via GORM. Migrations run automatically on startup via `migrations.Migrate()` (GORM AutoMigrate). No manual migration files — schema is derived from entity structs. Two repos (`MemberRepository`, `InstructorRepository`) are still in-memory stubs.

**Frontend:** React 19 SPA embedded into the Go binary via `//go:embed all:frontend/dist`. In production the binary serves `/` (admin dashboard) and `/checkin` (kiosk). The frontend communicates exclusively with `/api/v1/*`.

## Adding a new feature

The pattern for every new module:
1. Add/update entity in `internal/domain/entities/`
2. Define repository interface in `internal/domain/repositories/`
3. Implement SQLite repo in `internal/infrastructure/persistence/sqlite_<name>_repository.go`
4. Write use case in `internal/usecases/<name>_usecase.go`
5. Write handler in `internal/infrastructure/http/handlers/<name>_handler.go`
6. Add DTO in `internal/infrastructure/http/dto/<name>_dto.go`
7. Register repo, use case, handler and routes in `main.go`
8. Add the permission(s) the routes need to `internal/domain/entities/permissions.go` (constant + `allPermissions` + the roles that get it), and guard the routes with `middleware.RequirePermission(...)`. If the endpoint takes a date range over money, add `middleware.RestrictReportRange()` too.

## Auth, roles & permissions

JWT tokens stored in browser `localStorage` (`access_token`, `refresh_token`). Middleware order in protected routes: `AuthMiddleware` → `GymTimezoneMiddleware` → `RequirePermission(...)`.

Roles: `SUPER_ADMIN`, `ADMIN_GYM`, `RECEPCIONISTA`, `STAFF`, `MEMBER` (the last one does not enter the panel).

**The single source of truth is `internal/domain/entities/permissions.go`** — a role → permission matrix. Routes declare the *capability* they need (`middleware.RequirePermission(entities.PermSalesCreate)`), never a role, so changing what a role can do means editing that one matrix. `RequireRole` still exists but is legacy; prefer `RequirePermission` for new routes.

The frontend does not duplicate the matrix: the backend ships the permission list in the login response and in `/auth/me`, and `frontend/src/utils/permissions.js` reads it to build the sidebar. Hiding a menu entry is cosmetic — the API check is what protects the data.

### Report scoping

Four separate permissions implement "staff closes its day but cannot look backwards":

| Permission | Meaning |
|---|---|
| `reports:daily` | May query the report endpoints **only with today's date** |
| `reports:history` | May query any date range |
| `reports:accounting` | Sees the *Reportes Contables* screen |
| `reports:inventory` | Sees the inventory *Reportes* screen |

`middleware.RestrictReportRange()` enforces the date limit: for any role without `reports:history` it rejects (403) a request whose `start_date`/`end_date`/`from`/`to` is not today in the **gym's timezone**. It rejects instead of silently clamping, so nobody reads a one-day total believing it covers a month. Any new endpoint that exposes historical money data must be wired through it.

`RECEPCIONISTA` and `STAFF` have `reports:daily` and land on the *Cuadre del Día* screen (`frontend/src/components/DailyClose.jsx`); only `ADMIN_GYM` and `SUPER_ADMIN` have the other three.

## Stock is a ledger, not a counter

`products.stock` is the running balance of `stock_movements` (`internal/domain/entities/stock_movement.go`). **The only supported way to move it is `ProductRepository.ApplyStockDelta`**, which does a conditional `UPDATE ... WHERE stock >= ?` and returns the new balance; the guard lives in the WHERE because read-then-write is a race that drove stock negative when two tills sold the last unit.

Every mutation writes its movement **in the same `uow.Do` transaction** — sales, voids, purchases, manual adjustments, physical counts. `StockMovement.StockAfter` is the balance read back inside that transaction, which makes the ledger self-verifying: if `stock_after[n] + signed[n+1] ≠ stock_after[n+1]`, someone wrote stock without going through the ledger.

Two rules that are easy to break:

- **`stock` is not in `sqlite_product_repository.Update`'s column list, and not in `UpdateProductRequest`.** It used to be, via `Save()`, and a `PUT /products/:id` without a `stock` field silently zeroed it (`Stock int` — the Go zero is indistinguishable from "set to zero"). Don't add it back; use `PATCH /products/:id/stock` (count) or `POST /products/:id/stock/adjust` (signed delta + reason).
- **Movements built for a transaction must be copied per attempt.** `uow.Do` retries on `SQLITE_BUSY`, and `StockAfter` is the one field computed inside; without the `copy()` a half-failed attempt leaves stale balances. See `CreateSale` in `internal/usecases/sale_usecase.go`.

The ledger has **no history before its deployment**: `backfillInitialStockMovements` seeds one `initial` entry per product with stock, and does *not* replay `sale_details` (`products.stock` already reflects those sales, so replaying would double-count). The UI says so.

## Expenses & suppliers

`Expense` + `ExpenseLine` (`internal/domain/entities/expense.go`) covers **both** a general expense (rent) and a merchandise purchase — the only difference is whether a line carries a `product_id`. A line with a product writes a `purchase` stock movement and recomputes the product's weighted average cost via `ProductRepository.ApplyPurchase`, all inside one `uow.Do`.

**An expense always has at least one line.** The UI's "simple" mode synthesises one from description + amount, so `Subtotal == SUM(lines.total)` always holds and no report needs two code paths.

`VoidExpense` mirrors `VoidSale`: it writes a negative twin rather than editing or deleting. That is why `ExpenseRepository.Report` can sum every row in the range without filtering by status — voids net themselves out. **Voiding a purchase whose goods were already sold fails** with `ErrInsufficientStock`, deliberately: allowing negative stock there would break the guard that prevents overselling at the till. The weighted average cost is *not* reverted on void (it has already blended with later purchases); the movement's note says so.

`Product.CostPrice` is weighted-average, not last-cost — one urgent overpriced order must not rewrite the margin on stock already in the warehouse.

### Margin leak — the easy mistake

`cost_price` and `StockMovement.UnitCost` reveal the business's margin, and `GET /products` is consumed by the POS, which RECEPCIONISTA and STAFF use. Both fields are `*float64` with `omitempty` and are only populated when the caller has `PermInventoryCost` — see `canSeeCost` in `product_handler.go`. **Adding a plain `float64` cost field to any response leaks margins to the whole staff.** `TestInformacionContableNoLlegaAlMostrador` guards this.

Supplier NIT uniqueness is validated in the use case, **not** with a unique index: SQLite compares `''` as a value, so a `UNIQUE(gym_id, tax_id)` would forbid a second supplier without a NIT. Suppliers are deactivated, never deleted — an expense referencing one must not lose its history.

Neither `CreateExpense` nor `CreateSupplier` requires a non-nil `GymID`: the seeded super admin has `uuid.Nil`, and it is exactly who registers the first expense on a fresh install.

## Payments are lines, not a column

A sale, a subscription and an expense can each be paid with more than one method (part cash, part card). The single source of truth is the polymorphic `payment_lines` table (`internal/domain/entities/payment_line.go`, `payable_kind` + `payable_id` — not nullable FKs, because SQLite can't `ALTER TABLE ADD CONSTRAINT` on an already-deployed base). `Sale.PaymentMethodID` and `Subscription.PaymentMethod` still exist and stay filled as the "primary" (largest) line — five and nine call sites respectively still read them — but they are derived, not authoritative.

`PaymentLine.Amount` **carries the sign of its document**: a void's mirror lines are negative, exactly like `Sale.Total`. That is what makes `SumByMethod` (`PaymentLineRepository`) a plain `GROUP BY payment_method_id` — voids cancel themselves out instead of needing a status filter, and it is the fix for a real bug where a day with a void never summed to its total.

**`GET /payments/summary?start_date=&end_date=`** (`internal/usecases/payment_summary.go`, `AggregatePaymentSummary`) is the *one* place that turns `payment_lines` rows into a per-method breakdown. It backs both the Cuadre del Día screen and the daily-close email (`notification_usecase.go`) — before this endpoint existed there were six separate places reconciling sales vs. subscriptions **by matching the payment-method name as a string**, and they only agreed by coincidence. Any new screen or report that needs "how much came in by method" should call this endpoint (or `AggregatePaymentSummary` directly, server-side) instead of adding a seventh reconciliation site.

**Icons and labels are keyed by the payment method's `type`** (`cash`/`card`/`transfer`/`other`), never by its name — an admin-created method like "Nequi" breaks any lookup keyed by the literal string.

Change (`vuelto`) is a property of the cash line, not the sale: `CashReceived` and `ChangeGiven` live on the `payment_lines` row whose method is of type `cash`. The **server always recalculates `ChangeGiven`**; it never trusts what the client sends.

**`PATCH /sales/:id/payments`, `/subscriptions/:id/payments`, `/expenses/:id/payments`** (`internal/usecases/payment_usecase.go`, `PermPaymentsAmend` — only `ADMIN_GYM`/`SUPER_ADMIN`) let someone correct how a document was paid after the fact, with the change written to `ChangeLog`. **Correcting a voided sale cascades to its void mirror**, rewriting the mirror's lines as the exact negation of the new set while keeping the mirror's original `Date`/`Hour` — otherwise the void's lines would sit in the desglose as a self-inconsistent phantom.

`middleware.RestrictReportRange()` only inspects params literally named `start_date`/`end_date`/`from`/`to`, and **does nothing if none of them are present** — any new date-ranged money endpoint must require those params (400 if missing), not just declare the middleware and assume it's enough.

## Configuration (env vars)

| Variable | Default | Notes |
|---|---|---|
| `SERVER_PORT` | `8080` | |
| `DATABASE_PATH` | `gym-go.db` | Overridden to `%PROGRAMDATA%\Gym-Go\` when installed in Program Files |
| `JWT_ACCESS_SECRET` | (insecure default) | Must be changed in production |
| `DEFAULT_TIMEZONE` | `America/Bogota` | Fallback when gym has no timezone |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | (empty) | Required for email notifications |

## Background goroutines (main.go)

- **Auto-expire subscriptions** — runs every 1 hour
- **DB backup** — daily at 02:00 local time, keeps last 7 days in `backups/`
- **Daily-close email** — daily at 23:00, sends end-of-day report to each gym's recipients
