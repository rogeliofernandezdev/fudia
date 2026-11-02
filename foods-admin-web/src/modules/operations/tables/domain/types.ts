/** Tipo de atención: mesa tradicional o venta por mostrador (pedidos independientes, sin cuenta abierta). */
export type TableServiceType="table"|"counter";
export type Table={id:string;name:string;seats:number;zone:string;active:boolean;qrToken:string;qrEnabled:boolean;serviceType:TableServiceType};
export type Zone={id:string;name:string;sortOrder:number;active:boolean};
export type List<T>={items:T[];total:number;page?:number;pageSize?:number};
export type RowDraft={id?:string;name:string;seats:string;zone:string;active:boolean;qrEnabled?:boolean;serviceType:TableServiceType;isNew?:boolean};
export type ZoneDraft={id?:string;name:string;sortOrder:number;active:boolean};
