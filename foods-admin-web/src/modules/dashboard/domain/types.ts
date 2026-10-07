export type HourlySale={hour:number;total:string};
export type TopProduct={name:string;qty:string;revenue:string};
export type DashboardOperations={
  pendingBalance:string;unpaidOrders:number;partialOrders:number;
  tablesTotal:number;tablesOccupied:number;
  kitchenConfirmed:number;kitchenPreparing:number;readyOrders:number;
  deliveryPending:number;deliveryInTransit:number;
  activeCashRegisters:number;openCashShifts:number;cashBalance:string|null;
  soldOutProducts:number;
};

export type DashboardData={
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
  hourlySales:HourlySale[];
  topProducts:TopProduct[];
};
