export type HourlySale={hour:number;total:string};
export type TopProduct={name:string;qty:string;revenue:string};

export type DashboardData={
  salesNet:string;
  paidOrders:number;
  averageTicket:string;
  openOrders:number;
  criticalStock:number;
  purchasesToApprove:number;
  reservationsToday:number;
  kitchenPending:number;
  hourlySales:HourlySale[];
  topProducts:TopProduct[];
};
