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
  /** Opciones elegidas del plato (término, acompañamiento, adicionales). */
  modifiers?:{groupId:string;groupName:string;optionId:string;name:string;surcharge:string}[];
  /** Estado del plato en su estación; ausente en comandas históricas (sin marcado por plato). */
  status?:KitchenStatus;
  /** Tiempo de preparación del producto en minutos; ausente si el producto no lo define. */
  prepMinutes?:number;
  /** Hora en que la estación empezó a preparar el plato; ausente mientras está por preparar. */
  preparingSince?:string;
};

export type KitchenTicket={
 destination?:"kitchen"|"bar";
  id:string;
  orderId:string;
  roundNumber:number;
  code:string;
  channel:string;
  /** "counter" en la venta por mostrador. */
  origin?:string;
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
  destinationOptions?:KitchenOption[];
};
