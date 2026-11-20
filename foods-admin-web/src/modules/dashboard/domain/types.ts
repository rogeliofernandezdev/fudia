export type HourlySale={hour:number;total:string};
export type TopProduct={name:string;qty:string;revenue:string};
/** `orders` y `averageTicket` solo llegan en ventas por canal (pedidos que se terminaron de pagar en el periodo). */
/** `affectsCash` y `digitalIndex` solo llegan en medios de pago: la misma identidad de color que Cobros. */
export type SalesBreakdown={value:string;label:string;total:string;count:number;orders?:number;averageTicket?:string;affectsCash?:boolean;digitalIndex?:number};
export type CategorySale={name:string;qty:string;revenue:string};
export type DashboardPeriodKey="today"|"7d"|"30d";
export type DashboardTrendPoint={key:string;current:string;previous:string;orders:number};

export type DashboardOperations={
  pendingBalance:string;unpaidOrders:number;partialOrders:number;
  tablesTotal:number;tablesOccupied:number;
  kitchenConfirmed:number;kitchenPreparing:number;readyOrders:number;
  deliveryPending:number;deliveryInTransit:number;
  activeCashRegisters:number;openCashShifts:number;cashBalance:string|null;
  soldOutProducts:number;
  /** Mesas de servicio ocupadas con la cuenta pedida (Por cobrar). */
  tablesBillRequested:number;
  /** Minutos desde el primer pedido de salón activo; null sin mesas ocupadas. */
  tableMinutesAverage:number|null;
  oldestTable:{name:string;minutes:number}|null;
};

export type DashboardData={
  /** Cobros menos devoluciones del periodo seleccionado. */
  salesNet:string;
  paidOrders:number;
  averageTicket:string;
  openOrders:number;
  criticalStock:number;
  purchasesToApprove:number;
  reservationsToday:number;
  kitchenPending:number;
  businessDate:string;
  operations:DashboardOperations;
  /** Siempre del día actual. */
  hourlySales:HourlySale[];
  topProducts:TopProduct[];
  period:{key:DashboardPeriodKey;days:number;from:string;to:string};
  /** Mismo día de la semana anterior (hoy) o ventana anterior de igual duración, cortada a la misma hora local. */
  previous:{salesNet:string;paidOrders:number;averageTicket:string;from:string;to:string};
  /** `now`: tramo en curso en la zona horaria del local (hora HH o fecha); los posteriores aún no ocurren. */
  trend:{granularity:"hour"|"day";now:string;points:DashboardTrendPoint[]};
  salesByChannel:SalesBreakdown[];
  salesByPaymentMethod:SalesBreakdown[];
  salesByCategory:CategorySale[];
};
