import type {Customer} from "./types";

/** Agrupa un celular peruano (+51 9XX XXX XXX) para leerlo de un vistazo; otros números se muestran tal cual. */
export function formatCustomerPhone(phone:string):string{
  const compact=phone.replace(/[\s-]+/g,"");
  const peru=/^(\+?51)?(9\d{2})(\d{3})(\d{3})$/.exec(compact);
  if(!peru)return phone.trim();
  return `${peru[1]?`+${peru[1].replace("+","")} `:""}${peru[2]} ${peru[3]} ${peru[4]}`;
}

const consentLabels:Record<NonNullable<Customer["marketingConsentStatus"]>,string>={
  granted:"Acepta",
  declined:"No acepta",
  withdrawn:"Retiró su permiso",
  asked:"Consultado, sin respuesta",
  pending:"Sin consultar",
};

/** El permiso vigente manda; el último evento solo matiza por qué no hay permiso. */
export function marketingConsentLabel(customer:Pick<Customer,"marketingConsent"|"marketingConsentStatus">):string{
  if(customer.marketingConsent)return consentLabels.granted;
  const status=customer.marketingConsentStatus;
  return status&&status!=="granted"?consentLabels[status]:consentLabels.declined;
}

/** Hay relación de compra si registra visitas, consumo o una última compra. */
export function hasPurchases(customer:Pick<Customer,"visitCount"|"totalSpent"|"lastPurchaseAt">):boolean{
  return customer.visitCount>0||Number(customer.totalSpent)>0||Boolean(customer.lastPurchaseAt);
}
