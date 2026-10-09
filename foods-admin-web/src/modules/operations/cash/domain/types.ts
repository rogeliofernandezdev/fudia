export type CashShiftStatus="open"|"closed";
export type CashReportSale={id:string;orderCode:string;customer:string;name:string;details:string;quantity:string;unitPrice:string;total:string;createdAt:string;waiterName:string};
export type CashReportPayment={id:string;orderCode:string;methodName:string;amount:string;reference:string;createdByName:string;createdAt:string};
export type CashShiftReport={
 version:1;persisted:boolean;generatedAt:string;organizationName:string;locationName:string;
 timezone:string;country:string;currency:string;currencySymbol:string;currencyPosition:"before"|"after";currencyDecimals:number;
 shift:CashShift;sales:CashReportSale[];payments:CashReportPayment[];refunds:CashReportPayment[];
 methods:{name:string;amount:string}[];counts:{denomination:string;quantity:number;total:string}[];
 collectedAmount:string;refundedAmount:string;netCollectedAmount:string;
};
export type CashMovementType="income"|"expense";

export type CashMovement={
  id:string;
  movementType:CashMovementType;
  sourceType:"manual"|"cash_sale"|"cash_refund"|"cash_pull"|"transfer_in"|"transfer_out"|"deposit"|"adjustment";
  sourceId:string|null;
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
  blindClose:boolean;
  expectedVisible:boolean;
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
  activeUserNames:string[];
  businessDate:string;
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
  blindClose:boolean;
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
  blindClose:boolean;
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

export type CashCountLine={
  denomination:string;
  quantity:string;
};

export type CloseCashShiftDraft={
  countedAmount:string;
  note:string;
  counts:CashCountLine[];
};

export type CashShiftUser={
  userId:string;
  name:string;
  assignedAt:string;
};

export type CashUserOption={
  id:string;
  name:string;
  assignedShiftId:string|null;
};

export type CashOperationType="cash_pull"|"deposit"|"transfer";

export type CashOperationDraft={
  operationType:CashOperationType;
  targetShiftId:string;
  amount:string;
  reason:string;
  note:string;
};

export type CashOperation={
  id:string;
  operationType:CashOperationType;
  amount:string;
  reason:string;
  note:string;
  createdAt:string;
};
