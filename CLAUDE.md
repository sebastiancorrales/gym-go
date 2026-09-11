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

## Fiado (store credit)

Fiado is **a payment method type, not a special case**: `PaymentTypeCredit` (`internal/domain/entities/payment_method.go`) sits alongside `cash`/`card`/`transfer`/`other`, with a seeded "Fiado" method. A `payment_lines` row using it counts toward a sale's total exactly like any other line — `ValidatePaymentLines` never distinguishes types — which is what lets a fiado line combine with cash/card in the same sale for free, through the payment-split machinery that already existed.

What makes it a receivable instead of just another payment leg: a line whose method is `credit` also opens an `entities.Debt` (`internal/domain/entities/debt.go`) in the **same transaction** as the sale (`SaleUseCase.CreateSale` → `buildDebtForSale`). The debtor is either a linked member (`DebtorUserID`) or a free-typed name (`DebtorName`, always kept as a display snapshot even when linked) — a sale can carry at most one credit line, so at most one `Debt` per sale. **There are no partial payments**: a `Debt` is pending or collected, full stop; `DebtRepository` has no ledger of partial amounts because the business rule makes one unnecessary.

**Collecting a debt** (`POST /debts/:id/collect`, `DebtUseCase.CollectDebt`) writes a *new* `payment_lines` row with a fourth `PayableKind`, `PayableDebt`, dated **today** in the gym's timezone — not the day of the original sale. That's the whole mechanism behind "a fiado sold Monday and collected Tuesday shows up in each day's cuadre correctly": Monday's `/payments/summary` shows the Fiado bucket (revenue recognized, not cash), Tuesday's shows the real method (cash actually landed). Collecting with another `credit`-type method is rejected (`ErrCannotPayDebtWithCredit`) — you can't pay a fiado with another IOU. `AggregatePaymentSummary` needed one addition to support this: a `PayableDebt` case feeding `DebtsCollectedTotal`/`Count`, alongside the existing `SalesTotal`/`SubsTotal` — the `sum(Methods) == Total` invariant holds automatically for any new `PayableKind` because the totals are summed unconditionally; the switch only feeds display sub-totals.

**Voiding a sale with a fiado defers to the debt's state**: pending → the debt is cascaded to `voided` in the same `uow.Do` as the void (the mirror payment line negation already handles the credit leg generically, no special code needed); already collected → the void is **rejected** (`ErrDebtAlreadyCollected`), same principle as `VoidExpense` blocking a purchase whose stock already sold — the money left in a different day's cuadre and undoing the sale would leave a mismatched mirror. `PaymentUseCase.AmendPayments` rejects a sale that has *any* `Debt` (pending, collected, or voided) outright (`ErrCannotAmendCreditSale`): rewriting payment lines through that path would desynchronize them from the debt they opened.

**`GET /debts/summary`** groups pending debts by debtor — by `DebtorUserID` when linked, else by `LOWER(TRIM(DebtorName))` — computed in the database (`DebtRepository.SummaryByDebtor`), same principle as `SumByMethod`. Two different free-typed names for the same real person do **not** merge automatically; that's a deliberate limitation until someone asks for a real `Debtor` entity — linking to a member (`PATCH /debts/:id/link-debtor`) is the intended fix path today.

Permissions: `PermDebtsRead`/`PermDebtsCollect` sit at the same tier as `PermSalesCreate` (RECEPCIONISTA/STAFF have both) — collecting a fiado is the same trust act as ringing up cash. Selling fiado needs no new permission; it rides on `PermSalesCreate` since it's just another line on `POST /sales`. Reverting a wrongful collection (`POST /debts/:id/uncollect`) reuses `PermPaymentsAmend` — it's the same "correct a payment after the fact" operation as `AmendPayments`, so it stays admin-only rather than inventing a third permission.

**`ensurePaymentMethodCatalog` (`migrations/backfill_payments.go`) must be called unconditionally in `Migrate()`, not only from inside `backfillPaymentLines`.** That function returns early when there are no pending sales/subscriptions to backfill — which is *always* true on a fresh install or in steady state — so a method added to `defaultCatalog` after the initial rollout (this bit "Otro", then "Fiado") would never actually get seeded outside the one-time backfill window. Caught only by spinning up a clean database and checking `GET /payment-methods` for real.

## Hikvision access terminals (ISAPI)

A Hikvision terminal (tested against a **DS-K1T8003MF, firmware V1.4.1**) is registered as a `Device` row with `DeviceType == DeviceTypeHikvisionTerminal`. Its IP lives in the existing `Device.IPAddress`; the port, admin credentials and door number live as JSON in the existing `Device.Configuration` (`entities.HikvisionConfig`) — no new table, no migration. `redactDeviceSecret` strips the password before any device row is serialized, and `applyConnectionFields` *merges* the config so a save that omits the password keeps the stored one.

Everything talks plain HTTP over the LAN via `internal/infrastructure/hikvision` (Digest auth implemented in `digest_client.go`, no external dependency). Unlike the DigitalPersona USB reader, no C# helper and no CGO are involved. `POST /devices/:id/trigger` opens the door for both device kinds — serial relay or terminal — branching on `DeviceType`.

Live tests against real hardware live behind a build tag so `go test ./...` needs none:

```bash
HIK_HOST=192.168.1.x HIK_USER=admin HIK_PASS=... go test -tags device ./internal/infrastructure/hikvision/ -v
```

### The firmware's traps

Every one of these was found by probing a real device; Hikvision's ISAPI reference is behind a partner-portal login that takes days to approve, and the device's own `/capabilities` **lies** in several places.

- **A person without door rights is silently useless.** `POST /UserInfo/Record` happily stores someone with `doorRight: ""` and `planTemplateNo: ""`. Their fingerprint then enrolls fine, the terminal reads their finger fine — and denies entry, which on the screen looks exactly like a failed fingerprint. `personPayload` always sends `doorRight: "1"` and `RightPlan: [{doorNo: 1, planTemplateNo: "1"}]`.
- **No validity date from 2038 onwards.** The terminal keeps time in a signed 32-bit counter, so `Valid.endTime` past `2037-12-31T23:59:59` is refused — as `timeFormatError` on `UserInfo/Record`, and as a bare `badParameters` on `UserInfo/Modify`, which is why `UpsertPerson` reports *both* failures instead of just the second. `clampToDeviceRange` caps it.
- **`employeeNo` is alphanumeric only, max 32.** A hyphen or underscore is rejected as `illegalEmployeeNo` despite `/capabilities` advertising `characterType "any"`. A UUID with its hyphens stripped is exactly 32 hex characters, which is why `EmployeeNoForUser` is a plain `ReplaceAll` and needs no mapping table. The device's own admin account is `employeeNo` `"1"`, so never assume an employeeNo came from gym-go.
- **Formats are inconsistent between neighbouring endpoints.** `CaptureFingerPrint` accepts **XML only** — the same request as JSON with `?format=json` fails with `MessageParametersLack`. `FingerPrint/SetUp` and `FingerPrint/Delete` accept **JSON only** (XML there fails with `badXmlFormat`).
- **The device emits base64 its own parser rejects.** A captured template came back 685 characters long — not a multiple of 4. `normalizeBase64` re-pads it. **Never "fix" it by trimming bytes**: a truncated template enrolls with no error at all and then never matches the finger it came from. That dead end cost hours.
- **`cardReaderRecvStatus` is not an error code.** `0` = added, `1` = overwrote that slot, `5` = *this finger already belongs to somebody else*, with the owner's `employeeNo` in `errorMsg`. Only 5 is a refusal, and it is the answer to "can two members enroll the same finger?" — they cannot; the first wins and `DuplicateFingerprintError` carries the owner so the UI can name them.
- **Deleting is asynchronous and the ack lies.** `FingerPrint/Delete` returns `{"status":"success"}` while the templates are still in the reader, so an immediate re-enroll bounces as a duplicate. The real outcome is only in `FingerPrint/DeleteProcess`, which `DeleteFingerprints` polls.
- **Only `mode: "byCardReader"` works for deletion**, with `CardReaderDetail` as a single **object**. The `byEmployeeNo` mode that `/capabilities` advertises is rejected as `badParameters` in every shape. Likewise `isSupportSetUp: true` under `FingerPrintCfg` points at an endpoint (`FingerPrintCfg/SetUp`) that answers `notSupport` to everything — the working one is `FingerPrint/SetUp`.
- **Capture blocks for ~11 seconds and that is the device's whole patience.** `CaptureFingerPrint` holds the connection open waiting for a finger, then answers `Device Error`. Two consequences. The HTTP client used for it must outlast that wait — a 10-second timeout against an 11-second wait never sees the answer and reports a *connection failure*, which is how "nobody presented a finger" reached the gym disguised as "error de comunicación con el huellero". And eleven seconds is not enough when the reader is across the room from the screen (the first installation has it three metres away, out of sight), so `CaptureFingerprintWithin` chains attempts to cover the walk; the reader is busy for that whole window, so it is sized for a walk and not longer.
- **Every search answers at most ten records, whatever `maxResults` says.** Both `UserInfo/Search` and `AcsEvent` cap a page at ten — the device even documents it, `maxResults` is capped at 10 in `UserInfo/capabilities` and reports the real size in `totalMatches`, so paging must follow that field. Reading "I got fewer than I asked for, so that was the last page" stops at person ten and reports everybody after them as *missing from the device* — which made the reconciler try to create them on every pass, forever, and produced the write burst that broke it. Both `ListPersons` and the event ingester now advance by what came back and stop at `totalMatches`; `TestListPersonsPagesPastTheDeviceCap` and `TestSearchEventsReturnsWhatTheDeviceActuallyHas` guard them. On the event side the same mistake loses attendance instead: it skips every event between one page and the next, which only shows up once more than ten are pending — a busy morning, or a backend that was off for a while.
- **`updateDatabaseError` means "busy", not "broken".** The terminal keeps its people in a small embedded database and refuses a write that lands while it is busy, answering HTTP 400 with `Device Error` / `updateDatabaseError`. The identical request succeeds moments later. It is retried (`withRetry`, `IsTransient`); note the reason lives only in the JSON body, so `jsonStatusCall` parses the body *before* judging the HTTP status — reading it as a transport failure is what once hid it behind raw response text.
- **Validity is written in the gym's local wall clock.** `Valid` is `timeType: "local"` with no zone, so an instant must be converted with `.In(time.Local)` first. Sending a UTC wall clock as if it were local shifts every window by the offset (five hours in Bogotá) *and* makes the value read back different from the one written, so the reconciler rewrites the same member on every pass forever. `deviceLocalTime` does the conversion; `maxDeviceTime` is therefore a local instant too.

### Subscription enforcement lives in the device, not in the event

**The terminal decides by itself and never asks the server.** By the time an access event reaches gym-go the door has already opened, so validating a subscription *against the event* only produces a report, never a refusal. Enforcement therefore works by pushing the rule into the device ahead of time: `Valid.endTime` on the person record is set to the subscription's end date, and the terminal denies expired members on its own — with the network down, with gym-go down.

`HikvisionSyncUseCase` owns this. Its `AccessWindowFor` computes the window from `findActiveSubscription`, **the same lookup the kiosk's `RecordEntry` uses** — shared deliberately, because if the door and the check-in screen ever compute "is this member up to date" differently they start telling the member contradictory stories. Staff and admins are not gated; a frozen or expired subscription closes the window, as does having none.

A closed window is an *expired* span (year 2000), never `Valid.enable: false` — "disabled validity" can read as "no time restriction" on Hikvision firmwares, which would fail open. The person and their fingerprints stay on the device, so renewing reopens the window without the member re-scanning.

Two mechanisms, and the order matters:

- **The hourly reconciler is the source of truth** (`SyncAll`, run right after the auto-expire/auto-unfreeze pass in `main.go`). Most expirations happen because time passed, with no event to hook, and the terminal may have been unreachable when something did happen.
- **Write-through is only a latency optimization** (`SubscriptionUseCase.SetAccessSyncHook` → `SyncMemberBestEffort`, fired on create/renew/cancel/freeze/unfreeze/date-change). Somebody who just paid walks to the door in seconds. It is **best-effort by design: an unplugged terminal must never fail a sale**, which is exactly why the reconciler and not this is authoritative. `POST /devices/sync-access` forces a pass by hand after an outage.


**Write only what changed.** `SyncGym` reads the whole roster once (`ListPersons`) and skips anybody whose window and name already match (`Person.SameWindowAs`). Rewriting every member every hour is pointless traffic against an embedded database, and a burst of it is what collides with the ten-second event poller and comes back as `updateDatabaseError` — the failure that reached the gym as "a veces no registra huellas", since enrollment starts by upserting the person too. In steady state a pass now writes nothing: the log line reports `actualizadas / ya estaban al día / fallidas`.

For the same reason the window given to staff (anyone who is not a `MEMBER`) is two **fixed** dates, not `now ± years`: a window computed from the current time differs on every pass, never matches what the device holds, and would rewrite every staff member hourly — precisely the people at the desk. `TestAccessWindow_PersonalEsEstableEntrePasadas` guards it.

### Ingesting the door's own log

`HikvisionEventUseCase` polls `AcsEvent` every 10 seconds and turns each entry into an `AccessLog`, so doors opened at the terminal appear in the panel like kiosk check-ins. Polling rather than the device's event push: nothing has to be exposed or configured on the device, and a gym-go that was switched off simply catches up from its cursor.

Two minor codes carry the meaning, both confirmed by scanning a valid member and an expired one:

| `minor` | Meaning |
|---|---|
| `38` | Fingerprint accepted — the door opened |
| `8` | Recognised, but the validity window has closed → *"Suscripción vencida"* |

Everything else (door buttons, remote opens, tamper) has an empty `employeeNoString` and is skipped, as is any `employeeNo` that is not a gym-go UUID — the device's own admin account is `"1"`.

**The ingester records what happened at the door, never what the backend would rule now.** A granted event stays granted even if the member looks lapsed today: they physically walked in, and that mismatch is the evidence that the validity sync fell behind. Recomputing the verdict would erase it.

Dedup is `AccessEventCursor.LastSerialNo` — the device numbers events monotonically. It lives in its own table rather than on `Device` because the ingester writes it every few seconds while an admin may be saving the device's configuration, and `DeviceRepository.Update` writes the whole row: on `Device` the two would silently overwrite each other. A terminal being read for the first time only imports the last 24 hours (`firstRunLookback`), so adopting a device that has been in service for months does not import its whole history as today's attendance.

**Event timestamps are only as correct as the device's clock.** A unit out of the box is set to `CST-8` (China): the instant is right but every wall-clock reading — and every `Valid` window comparison, which uses naive local times — lands 13 hours off. Set the terminal's timezone before trusting any of this (`PUT /ISAPI/System/time`, timezone written POSIX-style inverted: Bogotá is `CST+5:00:00`).

`GET /access/recent` returns the newest entries with the member's name resolved in one query, and marks each one `from_terminal`. The kiosk polls it to announce door events — in particular the expiry refusal, which on the terminal's own screen is a generic error the member cannot act on; the firmware's message is not configurable, so the readable version has to come from a screen gym-go controls.

### Fingerprint templates have a source

`fingerprints.Source` (`digitalpersona` | `hikvision`) exists because the two formats are mutually unreadable. **`FingerprintRepository.GetAllTemplates` must keep filtering to DigitalPersona** — it feeds the C# matcher, and handing it terminal templates makes every kiosk verification match against garbage. Rows predating the column are empty and belong to DigitalPersona (`SourceOrDefault`).

Terminal templates are stored locally only as a backup copy: the terminal does its own matching. That copy is what makes `RemoveFinger` possible — the device can only delete *all* of a person's fingers, so removing one wipes the person and re-pushes the survivors from the stored copies. For the same reason, deleting a terminal enrollment must go through `DELETE /biometric/terminal/user/:user_id/finger/:id`; the generic `DELETE /biometric/:id` only deactivates the local row and would leave the member still able to open the door.

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
