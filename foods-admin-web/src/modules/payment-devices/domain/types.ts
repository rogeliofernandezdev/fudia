/** Fudia Pay: dispositivos vinculados por QR y pagos detectados que Caja revisa. */

export type PayPlatform="ANDROID"|"IOS";
export type PayProvider="yape"|"plin";
export type PayNotificationStatus="detected"|"associated"|"confirmed"|"dismissed";

export type PayDevice={
  id:string;
  name:string;
  platform:PayPlatform;
  model?:string;
  tokenSuffix:string;
  active:boolean;
  lastSeenAt?:string;
  createdAt:string;
};

/** Código temporal de vinculación: un solo uso, expira; nunca trae credenciales. */
export type PayPairingDraft={code:string;expiresAt:string};

export type PayNotification={
  id:string;
  deviceId:string;
  deviceName:string;
  provider:PayProvider;
  eventKey:string;
  senderName:string;
  /** Monto como texto decimal, como el resto de montos del sistema. */
  amount:string;
  currency:string;
  occurredAt:string;
  status:PayNotificationStatus;
  orderId?:string;
  paymentId?:string;
  confirmedBy?:string;
  confirmedAt?:string;
  createdAt:string;
};

/** Referencia estrecha de pedido para asociar o confirmar un pago detectado. */
export type PayOrderRef={id:string;code:string;channel?:string;customerName?:string;total?:string;paymentStatus?:string};
export type PayPaymentRef={id:string;amount:string;method:string;createdAt:string};
