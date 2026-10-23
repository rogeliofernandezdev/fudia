/**
 * Subtotal de una línea igual al backend: `round(quantity * unit_cost, 2)` con la
 * cantidad en 3 decimales y el costo en 4. El total de la orden es la suma de los
 * subtotales ya redondeados, como `purchase_orders.total`.
 */
export function purchaseLineTotal(quantity:string,unitCost:string){
  const q=Number(quantity),c=Number(unitCost);
  if(!Number.isFinite(q)||!Number.isFinite(c)||q<=0||c<0)return 0;
  // Enteros exactos (milésimas × diezmilésimas) evitan errores de coma flotante al redondear.
  const scaled=Math.round(q*1000)*Math.round(c*10000);
  if(Number.isSafeInteger(scaled))return Math.floor((scaled+50000)/100000)/100;
  return Math.round(q*c*100)/100;
}

export function purchaseOrderTotal(lines:Array<{quantity:string;unitCost:string}>){
  return Math.round(lines.reduce((sum,line)=>sum+purchaseLineTotal(line.quantity,line.unitCost),0)*100)/100;
}

/** Las presentaciones distintas de la unidad de inventario se compran, reciben y devuelven enteras. */
export function requiresWholeQuantity(presentationType:string){
  return presentationType!=="unit";
}
