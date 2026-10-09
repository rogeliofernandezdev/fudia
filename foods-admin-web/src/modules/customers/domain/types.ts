export type Option={value:string;label:string};

export type Address={
  id?:string;
  label:string;
  address:string;
  reference:string;
  district:string;
  city:string;
  countryCode:string;
  default:boolean;
};

export type Customer={
  id:string;
  code:string;
  customerType:"person"|"company";
  displayName:string;
  documentType:string;
  documentNumber:string;
  phone:string;
  email:string;
  preferredChannel:string;
  preferences:string;
  notes:string;
  marketingConsent:boolean;
  vipOverride:boolean;
  visitCount:number;
  totalSpent:string;
  lastPurchaseAt:string|null;
  active:boolean;
  segment:string;
  addresses?:Address[];
};

export type CustomersResponse={
  items:Customer[];
  total:number;
  channelOptions:Option[];
  segmentOptions:Option[];
  customerTypeOptions:Option[];
  documentTypeOptions:Option[];
};

export type CustomerDraft=Pick<Customer,
  "customerType"|"displayName"|"documentType"|"documentNumber"|"phone"|"email"|
  "preferredChannel"|"preferences"|"notes"|"marketingConsent"|"vipOverride"
>&{id?:string;addresses:Address[]};
