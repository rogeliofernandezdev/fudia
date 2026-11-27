/**
 * API pública liviana de Cobros para otras verticales: Ventas devuelve pagos de pedidos aún
 * abiertos y Reservas cobra la garantía con los mismos medios y la misma caja.
 */
// El diálogo de devolución usa los estilos de Cobros también cuando se abre desde Ventas.
import "./presentation/pos.css";
export type {Payment,PaymentMethodOption,RefundDraft} from "./domain/types";
export {getPOSCollectionAccess,listSalesPaymentMethods,refundPayment} from "./infrastructure/pos-api";
export {collectionShift} from "./domain/collection-shift";
export {RefundDialog} from "./presentation/pos-dialogs";
export {PaymentMethodPicker} from "./presentation/payment-method-picker";
