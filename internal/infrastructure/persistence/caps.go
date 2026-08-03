package persistence

import "log"

// Topes defensivos para las consultas que no reciben un límite del llamador.
//
// Sin tope, un SELECT sobre una tabla append-only (sales, access_logs) devuelve
// todo el histórico: hoy son miles de filas, en dos años serán cientos de miles,
// y cargarlas en memoria mantiene el snapshot de lectura abierto más tiempo.
//
// El tope SIEMPRE avisa cuando se alcanza. Un recorte silencioso es peor que una
// consulta lenta: hace que un listado parezca completo cuando no lo es.
const (
	maxSalesRows     = 2000
	maxAccessLogRows = 5000
	maxProductRows   = 1000
	defaultUserRows  = 500
)

// warnIfCapped logs when a query came back exactly at its cap, which means rows
// were almost certainly left out.
func warnIfCapped(query string, got, cap int) {
	if got >= cap {
		log.Printf("⚠️  %s devolvió %d filas, el tope de la consulta. "+
			"Hay resultados sin mostrar: usa un filtro por fecha o paginación.", query, got)
	}
}
