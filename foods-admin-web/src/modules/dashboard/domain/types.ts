export type HourlySale={hour:number;total:string};
export type TopProduct={name:string;qty:string;revenue:string};
export type SalesBreakdown={value:string;label:string;total:string;count:number};
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
  /** Mismo día de la semana anterior (hoy) o ventana anterior de igual duración. */
  previous:{salesNet:string;paidOrders:number;averageTicket:string;from:string;to:string};
  trend:{granularity:"hour"|"day";points:DashboardTrendPoint[]};
  salesByChannel:SalesBreakdown[];
  salesByPaymentMethod:SalesBreakdown[];
  salesByCategory:CategorySale[];
};
