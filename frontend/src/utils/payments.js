/**
 * Deja las lineas del editor listas para el backend: numeros de verdad y sin el
 * campo de efectivo cuando esta vacio.
 *
 * `change_given` NO se envia nunca — lo calcula el servidor, que es lo unico
 * sensato para un importe monetario.
 *
 * Vive fuera de PaymentSplitEditor.jsx porque un modulo que exporta a la vez un
 * componente y una funcion rompe el fast refresh de Vite.
 */
export function toPaymentsPayload(lines) {
  return lines
    .filter(l => l.payment_method_id && (parseFloat(l.amount) || 0) !== 0)
    .map(l => {
      const line = {
        payment_method_id: l.payment_method_id,
        amount: parseFloat(l.amount) || 0,
      };
      const received = parseFloat(l.cash_received);
      if (received > 0) line.cash_received = received;
      return line;
    });
}
