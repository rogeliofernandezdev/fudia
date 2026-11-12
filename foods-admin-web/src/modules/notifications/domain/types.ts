/** Tipos de aviso que redacta el backend; el texto llega listo en title/detail. */
export type NoticeKind="table_order"|"counter_order"|"delivery_order"|"pickup_order"|"order_ready"|"bill_requested"|"assistance_requested"|"stale_account"|"delivery_unassigned";

export type StaffNotice={id:string;kind:NoticeKind;title:string;detail:string;orderId?:string;tableId?:string;createdAt:string;read:boolean};

/** Campana: avisos de las últimas 24 horas y cuántos siguen sin leer para el usuario. */
export type NoticeFeed={items:StaffNotice[];unread:number};
