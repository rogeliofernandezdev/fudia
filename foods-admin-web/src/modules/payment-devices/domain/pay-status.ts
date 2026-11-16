import type {PayNotificationStatus} from "./types";

/** Estados del pago detectado con la matriz aprobada: espera suave para lo
 * operativo pendiente (naranja), registro (azul), confirmado (verde) y
 * descartado (gris). Nunca rojo: descartar no es un error. */
export const payStatusMeta:Record<PayNotificationStatus,{label:string;tone:"orange"|"blue"|"green"|"gray"}>={
  detected:{label:"Detectado",tone:"orange"},
  associated:{label:"Asociado",tone:"blue"},
  confirmed:{label:"Confirmado",tone:"green"},
  dismissed:{label:"Descartado",tone:"gray"},
};

export const payProviderLabel:Record<"yape"|"plin",string>={yape:"Yape",plin:"Plin"};
