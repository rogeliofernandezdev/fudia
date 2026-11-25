/** Matriz de estados de stock (FUDIA DS 1.0): suficiente verde (éxito), bajo ámbar
 * (advertencia) y agotado rojo (peligro). La semántica es la del backend: agotado ≤ 0;
 * bajo ≤ mínimo (>0). No recolorear por pantalla. */
export type StockStatusKey="ok"|"low"|"out";
export type StockStatus={key:StockStatusKey;tone:"green"|"orange"|"red";label:string;hint:string};

export function stockStatus(quantity:number,minimumStock:number):StockStatus{
  if(!Number.isFinite(quantity)||quantity<=0)return {key:"out",tone:"red",label:"Sin stock",hint:"Reponlo para poder vender los platos que lo consumen."};
  if(Number.isFinite(minimumStock)&&minimumStock>0&&quantity<=minimumStock)return {key:"low",tone:"orange",label:"Stock bajo",hint:"Queda poco: repone pronto para no bloquear pedidos."};
  return {key:"ok",tone:"green",label:"Saludable",hint:""};
}
