import type {LinkableProduct} from "./types";

/** Cómo controla hoy su cantidad un producto de la carta que aún no usa Inventario. */
export function linkableControlLabel(product:LinkableProduct){
  return product.quantityControl==="portions"?"Usa cupo del día":"Sin control de cantidad";
}
