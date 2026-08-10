// Package money contiene las operaciones de comparación y redondeo de importes.
//
// El proyecto guarda el dinero en float64. Convertirlo todo a centavos enteros
// sería más correcto en abstracto, pero tocaría Sale, SaleDetail, Subscription,
// Plan, Product, Expense, seis sitios del frontend y el generador de reportes; y
// mientras tanto convivirían dos unidades, donde un ×100 olvidado multiplica la
// recaudación por cien en silencio. Mezclar unidades es peor que el redondeo que
// arreglaría.
//
// En pesos colombianos el problema además es teórico: los importes son enteros y
// float64 los representa exactamente hasta 2^53. No hay divisiones ni repartos
// porcentuales — el cajero teclea cada monto.
package money

import "math"

// epsilon es medio centavo.
//
// Deliberadamente NO es un peso: un peso de tolerancia por transacción es
// exactamente el descuadre que el cuadre de caja existe para detectar. Esto solo
// absorbe el error de representación binaria.
const epsilon = 0.005

// Round redondea a dos decimales.
func Round(v float64) float64 {
	return math.Round(v*100) / 100
}

// Equal indica si dos importes son el mismo dinero.
func Equal(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// IsZero indica si un importe es cero.
func IsZero(v float64) bool {
	return math.Abs(v) < epsilon
}

// Sum suma importes redondeando el resultado, para que una suma de valores
// tecleados no arrastre residuos binarios al JSON.
func Sum(values ...float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return Round(total)
}
