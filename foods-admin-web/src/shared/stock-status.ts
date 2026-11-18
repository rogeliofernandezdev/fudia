/** Matriz de estados de stock aprobada (mismos tonos y etiquetas de la tabla de
 * Inventario): saludable verde, bajo naranja y sin stock gris. La semántica es
 * la del backend: sin stock ≤ 0; bajo ≤ mínimo (>0). No recolorear por pantalla. */
export type StockStatusKey="ok"|"low"|"out";
export type StockStatus={key:StockStatusKey;tone:"green"|"orange"|"gray";label:string;hint:string};

export function stockStatus(quantity:number,minimumStock:number):StockStatus{
 if(!Number.isFinite(quantity)||quantity<=0)return {key:"out",tone:"gray",label:"Sin stock",hint:"Reponlo para poder vender los platos que lo consumen."};
 if(Number.isFinite(minimumStock)&&minimumStock>0&&quantity<=minimumStock)return {key:"low",tone:"orange",label:"Stock bajo",hint:"Queda poco: repone pronto para no bloquear pedidos."};
 return {key:"ok",tone:"green",label:"Saludable",hint:""};
}
