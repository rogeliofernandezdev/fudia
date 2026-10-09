import type {IconName} from "@/design-system";

export const paymentMethodMeta:Record<string,{label:string;icon:IconName}>={
  cash:{label:"Efectivo",icon:"cash"},
  card:{label:"Tarjeta",icon:"payment"},
  transfer:{label:"Transferencia",icon:"transfer"},
  other:{label:"Otro",icon:"wallet"},
};
