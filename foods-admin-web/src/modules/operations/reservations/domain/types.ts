/** `seated` = «En mesa»; `attended` = «Asistió», solo de lectura: el backend lo proyecta cuando termina el tiempo de una reserva en mesa. */
export type ReservationStatus="pending"|"confirmed"|"seated"|"attended"|"cancelled"|"no_show";
/**
 * Cobro de la garantía en Caja: none (sin garantía), pending (por cobrar), paid (cobrada,
 * espera al cliente), applied (descontada de su cuenta), retained (quedó en el local) o refunded.
 */
export type ReservationDepositStatus="none"|"pending"|"paid"|"applied"|"retained"|"refunded";
/** Qué hacer con la garantía cobrada al cancelar la reserva. */
export type ReservationDepositDecision="refund"|"retain";
export type Reservation={
  id:string;customerId?:string|null;customerName:string;customerPhone:string;startsAt:string;guests:number;durationMinutes:number;
  /** Minutos que se espera al cliente después de la hora; vencidos, pasa sola a «No asistió». */
  toleranceMinutes:number;
  /** Garantía de la reserva («0.00» = sin costo). */
  depositAmount:string;
  depositStatus:ReservationDepositStatus;
  /** Medio con que se cobró la garantía. */
  depositMethodName:string;
  /** Serie-correlativo de la boleta de una garantía retenida (p. ej. B001-245). */
  depositDocument:string;
  tableId?:string|null;tableName:string;status:ReservationStatus;
  /** true si el backend la pasó a «No asistió» al vencer la tolerancia. */
  expired:boolean;notes:string;createdAt:string;
};
/** depositLocked: la garantía ya se cobró y su importe no se edita. */
export type ReservationDraft={id?:string;customerName:string;customerPhone:string;date:string;time:string;guests:string;durationMinutes:string;toleranceMinutes:string;depositEnabled:boolean;depositAmount:string;depositLocked?:boolean;tableId:string;notes:string};
/** defaultToleranceMinutes: tolerancia del local (Locales) que toma una reserva nueva. */
export type ReservationList={items:Reservation[];total:number;page:number;pageSize:number;defaultToleranceMinutes:number;defaultDepositAmount:string};
