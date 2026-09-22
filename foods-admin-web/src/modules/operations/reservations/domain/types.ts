export type ReservationStatus="pending"|"confirmed"|"seated"|"cancelled"|"no_show";
export type Reservation={
  id:string;customerId?:string|null;customerName:string;customerPhone:string;startsAt:string;guests:number;
  tableId?:string|null;tableName:string;status:ReservationStatus;notes:string;createdAt:string;
};
export type ReservationDraft={id?:string;customerName:string;customerPhone:string;startsAt:string;guests:string;tableId:string;notes:string};
export type ReservationList={items:Reservation[];total:number;page:number;pageSize:number};
