import type {OrganizationSubscription,PaymentStanding,SubscriptionPayment} from "./types";

/** Tonos homologados de `Status` en el design system. */
export type StatusTone="green"|"blue"|"orange"|"red"|"violet"|"gray";

type SubscriptionStatus=OrganizationSubscription["status"];
type PaymentStatus=SubscriptionPayment["status"];

/** Única fuente de etiquetas del estado contractual de una suscripción. */
export const subscriptionStatusLabels:Record<SubscriptionStatus,string>={trial:"Prueba",active:"Activa",past_due:"Pago pendiente",cancelled:"Cancelada"};

export const subscriptionStatusTones:Record<SubscriptionStatus,StatusTone>={trial:"violet",active:"green",past_due:"red",cancelled:"gray"};

export function subscriptionStatusLabel(status:SubscriptionStatus){return subscriptionStatusLabels[status]??status;}
export function subscriptionStatusTone(status:SubscriptionStatus):StatusTone{return subscriptionStatusTones[status]??"gray";}

export const paymentStatusLabels:Record<PaymentStatus,string>={paid:"Pagado",pending:"Pendiente",failed:"Fallido",refunded:"Reembolsado"};
const paymentStatusTones:Record<PaymentStatus,StatusTone>={paid:"green",pending:"orange",failed:"red",refunded:"gray"};

export function paymentStatusLabel(status:PaymentStatus){return paymentStatusLabels[status]??status;}
export function paymentStatusTone(status:PaymentStatus):StatusTone{return paymentStatusTones[status]??"gray";}

/** La situación de pago llega etiquetada desde el API; aquí solo se resuelve el tono. */
const standingTones:Record<PaymentStanding,StatusTone>={up_to_date:"green",due_soon:"orange",overdue:"red",trial:"violet",cancelled:"gray",none:"gray"};
export function paymentStandingTone(standing:PaymentStanding):StatusTone{return standingTones[standing]??"gray";}
