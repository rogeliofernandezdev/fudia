/** Delivery Fudia: enlace/QR de cada local y despacho del pedido a las centrales (globales). */
/** Enlace/QR de delivery del local; token vacío si aún no se generó. */
export type DeliveryLink={locationId:string;token:string;active:boolean;updatedAt?:string};

export type DispatchStatus="searching"|"accepted"|"unassigned"|"picked_up"|"delivered"|"own"|"cancelled";
export type DispatchOffer={partnerName:string;round:number;priority:number;status:"pending"|"accepted"|"rejected"|"expired"|"withdrawn";createdAt:string;expiresAt?:string;respondedAt?:string};
export type DeliveryDispatch={orderId:string;status:DispatchStatus;round:number;partnerName?:string;acceptedAt?:string;pickedUpAt?:string;deliveredAt?:string;offers:DispatchOffer[]};
