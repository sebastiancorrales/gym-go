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
