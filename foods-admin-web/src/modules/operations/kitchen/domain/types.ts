export type KitchenStatus="confirmado"|"preparando"|"listo";

export type KitchenOption={value:string;label:string};

export type KitchenSelection={
  groupId:string;
  groupName:string;
  productId:string;
  name:string;
  surcharge:string;
};

export type KitchenItem={
  id:string;
  productId:string;
  name:string;
  qty:string;
  unitPrice:string;
  note:string;
  itemType?:"product"|"combo";
  selections?:KitchenSelection[];
};

export type KitchenTicket={
  id:string;
  orderId:string;
  roundNumber:number;
  code:string;
  channel:string;
  status:KitchenStatus;
  customerName:string;
  tableName:string;
  notes:string;
  createdAt:string;
  updatedAt:string;
  targetMinutes:number|null;
  items:KitchenItem[];
};

export type KitchenResponse={
  items:KitchenTicket[];
  counts:Record<KitchenStatus,number>;
  channelOptions:KitchenOption[];
  serverTime:string;
};
