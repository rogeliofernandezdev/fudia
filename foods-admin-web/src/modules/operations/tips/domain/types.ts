export type TipSettings={mode:"disabled"|"suggest"|"collect";percentage:number;showSuggestion:boolean;allowFixed:boolean;channels:string[]};
export type TipPolicy=TipSettings&{inherited:boolean;country:string;currency:string;decimals:number;channelOptions:{value:string;label:string}[]};
export type TipInput={kind:"percentage"|"fixed";value:number;directToStaff:boolean};
export type TipReceipt={id:string;paymentId:string;orderCode:string;amount:string;methodName:string;affectsCash:boolean;directToStaff:boolean;createdByName:string;createdAt:string};
export type TipPayout={id:string;recipientId:string;recipientName:string;amount:string;method:"cash"|"transfer";note:string;createdByName:string;createdAt:string};
export type TipReport={cashCollected:string;otherCollected:string;direct:string;cashPaidOut:string;otherPaidOut:string;pending:string;receipts:TipReceipt[];payouts:TipPayout[]};
export type TipWorkspace={report:TipReport;available:string;recipients:{id:string;name:string}[]};
export type TipPayoutDraft={recipientId:string;amount:number;method:"cash"|"transfer";note:string;requestKey:string};
