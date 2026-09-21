export type CashShiftStatus="open"|"closed";
export type CashMovementType="income"|"expense";

export type CashMovement={
  id:string;
  movementType:CashMovementType;
  amount:string;
  reason:string;
  note:string;
  createdByName:string;
  createdAt:string;
};

export type CashShift={
  id:string;
  code:string;
  cashRegisterId:string;
  cashRegisterName:string;
  status:CashShiftStatus;
  openingAmount:string;
  incomeAmount:string;
  expenseAmount:string;
  expectedAmount:string;
  closingExpectedAmount:string|null;
  closingCountedAmount:string|null;
  varianceAmount:string|null;
  openingNote:string;
  closingNote:string;
  openedByName:string;
  closedByName:string;
  openedAt:string;
  closedAt:string|null;
  movementCount:number;
  movements?:CashMovement[];
};

export type CashRegister={
  id:string;
  code:string;
  name:string;
  active:boolean;
  openShift:CashShift|null;
};

export type CashShiftList={
  items:CashShift[];
  total:number;
  page:number;
  pageSize:number;
};

export type CashRegisterDraft={
  name:string;
};

export type OpenCashShiftDraft={
  openingAmount:string;
  note:string;
};

export type CashMovementDraft={
  movementType:CashMovementType;
  amount:string;
  reason:string;
  note:string;
};

export type CloseCashShiftDraft={
  countedAmount:string;
  note:string;
};
