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
  revokedAt?:string;
};
/** Filtro de la lista: activos (por defecto), revocados o todos (""). */
export type PayDeviceStatusFilter="active"|"revoked"|"";

/** Código temporal de vinculación: un solo uso, expira; nunca trae credenciales. */
export type PayPairingDraft={id:string;code:string;expiresAt:string};
/** Estado del código que Locales muestra: esperando, escaneado por la app, vinculado o vencido. */
export type PayPairingStatus={status:"pending"|"scanned"|"linked"|"expired";model?:string;deviceName?:string;expiresAt:string};

export type PayNotification={
  id:string;
  deviceId:string;
  deviceName:string;
  provider:PayProvider;
  eventKey:string;
  /** Número bancario informado por el proveedor; nunca es la clave técnica del evento. */
  operationNumber?:string;
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
/** Pedido candidato para asociar: el backend ya envía mesa, saldo y creación para reconocerlo sin abrirlo. */
export type PayOrderRef={id:string;code:string;channel?:string;customerName?:string;tableName?:string;total?:string;remainingAmount?:string;paymentStatus?:string;createdAt?:string};
export type PayPaymentRef={id:string;amount:string;method:string;createdAt:string};
