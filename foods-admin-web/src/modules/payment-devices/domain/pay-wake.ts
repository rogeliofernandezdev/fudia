/**
 * Cada cuánto Caja pide al backend despertar los teléfonos de Fudia Pay del
 * local. Algunos fabricantes (HONOR/MagicOS) congelan la app y solo la
 * descongelan con un push: sin este aviso, el Yape esperaba hasta abrir el
 * teléfono. El backend limita el push a uno cada 10 s por dispositivo.
 */
export const PAY_WAKE_CHARGING_MS=10_000;
/** Caja abierta sin cobro en curso: el pago aparece en menos de 30 s. */
export const PAY_WAKE_STANDBY_MS=30_000;
/** El local no tiene teléfonos con push: solo se revisa por si vinculan uno. */
export const PAY_WAKE_IDLE_MS=300_000;

/** Próximo aviso según el último resultado; sin respuesta (red caída) se
 * conserva el ritmo normal para no perder pagos. */
export function nextPayWakeDelay(devices:number|null,charging:boolean){
  if(devices===0)return PAY_WAKE_IDLE_MS;
  return charging?PAY_WAKE_CHARGING_MS:PAY_WAKE_STANDBY_MS;
}
