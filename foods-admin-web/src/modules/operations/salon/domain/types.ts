import type {ComboSelection} from "./catalog-types";

export type OrderItemSelection={groupId:string;groupName:string;productId:string;name:string;surcharge:string};
export type OrderItem={id:string;productId:string;name:string;qty:string;unitPrice:string;note:string;itemType:"product"|"combo";selections?:OrderItemSelection[]};
export type Order={
 id:string;code:string;channel:string;status:string;customerId:string;customerName:string;customerPhone:string;
 address:string;reference:string;tableId:string;tableName:string;notes:string;subtotal:string;deliveryFee:string;total:string;
 createdAt:string;updatedAt:string;itemCount?:number;items?:OrderItem[];
};
export type FloorTable={id:string;name:string;zone:string;seats:number;order:Order|null};
export type LineDraft={
 lineKey:string;sourceItemId?:string;repriceCombo?:boolean;itemType:"product"|"combo";productId:string;name:string;
 qty:number;unitPrice:number;note:string;selections:ComboSelection[];
};
export type Draft={
 channel:string;customerName:string;customerPhone:string;address:string;reference:string;tableId:string;notes:string;
 deliveryFee:string;lines:LineDraft[];
};
