export type PaymentMethod={
 code:string;
 name:string;
 description:string;
 active:boolean;
 salesEnabled:boolean;
 expensesEnabled:boolean;
 affectsCash:boolean;
 sortOrder:number;
};

export type PaymentMethodDraft={
 code:string;
 name:string;
 description:string;
 salesEnabled:boolean;
 expensesEnabled:boolean;
 affectsCash:boolean;
 sortOrder:number;
};

export type PaymentMethodsResponse={
 items:PaymentMethod[];
 total:number;
 page:number;
 pageSize:number;
};
