export type ServiceMode="counter"|"dine_in"|"mixed"|"";

export type RestaurantSetup={
  serviceMode:ServiceMode;
  completedAt:string|null;
  coreReady:boolean;
  counts:{
    categories:number;
    products:number;
    tables:number;
    cashRegisters:number;
    users:number;
    inventoryItems:number;
    recipes:number;
    suppliers:number;
  };
  modules:Record<string,boolean>;
};
