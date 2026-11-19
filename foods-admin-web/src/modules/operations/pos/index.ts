/** API pública liviana de Cobros para otras verticales (Ventas): devolver pagos de pedidos aún abiertos. */
// El diálogo de devolución usa los estilos de Cobros también cuando se abre desde Ventas.
import "./presentation/pos.css";
export type {Payment,RefundDraft} from "./domain/types";
export {getPOSCollectionAccess,refundPayment} from "./infrastructure/pos-api";
export {RefundDialog} from "./presentation/pos-dialogs";
