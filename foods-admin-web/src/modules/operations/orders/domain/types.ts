export type Option={value:string;label:string};

export type OrderItemSelection={
  groupId:string;
  groupName:string;
  productId:string;
  name:string;
  surcharge:string;
};

export type OrderItem={
  id:string;
  productId:string;
  name:string;
  qty:string;
  unitPrice:string;
  note:string;
  itemType?:"product"|"combo";
  selections?:OrderItemSelection[];
};

export type Order={
  id:string;
  code:string;
  channel:string;
  status:string;
  customerId:string;
  customerName:string;
  customerPhone:string;
  address:string;
  reference:string;
  tableId:string;
  tableName:string;
  notes:string;
  subtotal:string;
  deliveryFee:string;
  total:string;
  createdAt:string;
  updatedAt:string;
  items?:OrderItem[];
  paidAmount?:string;
  remainingAmount?:string;
  paymentStatus?:"pending"|"partial"|"paid";
};

export type OrdersResponse={
  items:Order[];
  total:number;
  channelCounts:Record<string,number>;
  channelOptions:Option[];
  statusOptions:Option[];
};
