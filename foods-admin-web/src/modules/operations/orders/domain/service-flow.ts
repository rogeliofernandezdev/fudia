export type ServiceItem={id:string;orderItemId:string;productId:string;name:string;qty:string;destination:"kitchen"|"bar"|"direct";destinationLabel:string;status:string;statusLabel:string};
export type TableAccount={id:string;channel:string;status:string;completedAt?:string;billClosedAt?:string;accountState?:"open"|"awaiting_payment"|"paid";waiterId?:string;paidAmount?:string;remainingAmount?:string;total:string;serviceItems?:ServiceItem[]};

export function canAddConsumption(order:TableAccount){
 return order.channel==="salon"&&!order.completedAt&&!order.billClosedAt&&order.status!=="cancelado"&&order.status!=="nuevo"&&order.accountState!=="paid"&&!(Number(order.paidAmount)>0&&Number(order.remainingAmount)<=0);
}
function allProductsDelivered(order:Pick<TableAccount,"status"|"serviceItems">){
 return order.status==="entregado"&&(!order.serviceItems?.length||order.serviceItems.every(item=>item.status==="entregado"));
}
export function canCloseAccount(order:TableAccount){
 return order.channel==="salon"&&!order.completedAt&&!order.billClosedAt&&allProductsDelivered(order);
}
export function canChargeAccount(order:Pick<TableAccount,"channel"|"status"|"billClosedAt"|"completedAt"|"serviceItems">){
 return !order.completedAt&&order.status!=="cancelado"&&order.status!=="nuevo"&&(order.channel==="salon"?Boolean(order.billClosedAt)&&allProductsDelivered(order):["listo","en_camino","entregado"].includes(order.status));
}
