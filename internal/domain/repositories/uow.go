package repositories

import "context"

// Repos es el conjunto de repositorios ligados a una misma transacción.
// Sólo incluye los que participan en flujos transaccionales.
type Repos struct {
	Users         UserRepository
	Gyms          GymRepository
	Subscriptions SubscriptionRepository
	Members       SubscriptionMemberRepository
	Audit         SubscriptionAuditLogRepository
	Products      ProductRepository
	Sales         SaleRepository
	SaleDetails   SaleDetailRepository
	// StockMovements entra aquí porque el movimiento y el cambio de stock tienen
	// que ser atómicos: un stock movido sin su asiento deja el libro descuadrado
	// y sin forma de saber quién lo movió.
	StockMovements StockMovementRepository
	// Un gasto con líneas de producto mueve inventario: la cabecera, sus líneas y
	// los asientos del kardex tienen que entrar juntos o no entrar.
	Expenses     ExpenseRepository
	ExpenseLines ExpenseLineRepository
	// PaymentLines entra aquí porque las patas del pago se escriben junto al
	// documento que pagan: un cobro registrado sin sus líneas es dinero que
	// entró y no aparece en el desglose de la caja.
	PaymentLines PaymentLineRepository
	// ChangeLogs entra aquí porque la corrección y su registro son una sola cosa:
	// un cobro corregido sin constancia de quién lo cambió es peor que no poder
	// corregirlo.
	ChangeLogs ChangeLogRepository
	// Debts entra aquí porque fiar es parte de la misma transacción que crea la
	// venta: una venta con una línea de crédito sin su Debt sería dinero que
	// nadie sabe a quién cobrarle.
	Debts DebtRepository
	// GiftCards entra aquí porque vender una tarjeta escribe sus líneas de
	// pago, y redimirla crea la suscripción: en los dos casos la tarjeta y lo
	// que la acompaña entran juntos o no entran.
	GiftCards GiftCardRepository
}

// UnitOfWork ejecuta una función dentro de una única transacción de base de datos.
//
// Existe porque no había ninguna transacción en el sistema: una venta de 5
// productos eran 12 transacciones implícitas independientes, de modo que si
// fallaba la cuarta la venta quedaba grabada con el stock a medio descontar. Y una
// suscripción grupal podía guardarse sin sus beneficiarios devolviendo 201 OK.
//
// La implementación reintenta ante contención de locks, así que la función DEBE ser
// idempotente: genera los UUID antes de entrar y no mutes estructuras en memoria de
// forma acumulativa (restas, contadores, append a slices compartidos), porque un
// reintento las aplicaría dos veces.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(r Repos) error) error
}
