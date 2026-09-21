"use client";
import "./inventory.css";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon,PageHeader,Select,Status} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {listInventoryProducts,listStockMovements} from "../infrastructure/inventory-api";
import type {StockMovement} from "../domain/types";

const movementLabels:Record<Exclude<StockMovement["movementType"],"inventory_adjustment">,string>={
 entry:"Entrada",
 sale:"Venta",
 sale_reversal:"Reversa de venta",
 sale_adjustment:"Ajuste por edición",
};

const reasonLabels:Record<string,string>={
 surplus_adjustment:"Ajuste por sobrante",
 shortage_adjustment:"Ajuste por faltante",
 waste:"Merma",
 expiration:"Vencimiento",
 other_exit:"Otro motivo de salida",
};

function movementLabel(item:StockMovement){
 if(item.movementType==="inventory_adjustment"){
  return item.adjustmentType==="exit"?"Ajuste de salida":"Ajuste de entrada";
 }
 return movementLabels[item.movementType];
}

function traceLabel(item:StockMovement){
 if(item.reason)return reasonLabels[item.reason]??item.reason;
 return item.sourceReference||"Sin referencia";
}

export function KardexPage(){
 const{location}=useSession();
 const[inventoryItemId,setInventoryItemId]=useState("");
 const products=useQuery({queryKey:["inventory-products","kardex"],queryFn:()=>listInventoryProducts()});
 const movements=useQuery({queryKey:["inventory-movements",inventoryItemId],queryFn:()=>listStockMovements(inventoryItemId)});
 const items=movements.data?.items??[];

 return <>
  <PageHeader eyebrow="ABASTECIMIENTO" title="Kárdex" description="Trazabilidad de movimientos y ajustes de la existencia física del local."/>
  <section className="panel standardized-management inventory-panel">
   <div className="inventory-toolbar">
    <Select value={inventoryItemId} onChange={event=>setInventoryItemId(event.target.value)} disabled={products.isLoading} aria-label="Filtrar Kárdex por artículo">
     <option value="">Todos los artículos</option>
     {products.data?.items.map(item=><option value={item.id} key={item.id}>{item.name}{item.kind==="ingredient"?" · Insumo":""}</option>)}
    </Select>
    <p><Icon name="receipt" size={15}/>Cada movimiento conserva saldo anterior, saldo resultante y usuario.</p>
   </div>
   {movements.isLoading?<KardexSkeleton/>:movements.isError?
    <div className="inventory-state"><Icon name="alert" size={24}/><b>No pudimos cargar el Kárdex</b><p>{movements.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>movements.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="inventory-state"><Icon name="receipt" size={24}/><b>Sin movimientos</b><p>Aún no hay movimientos registrados para este filtro.</p></div>
   :<div className="table-wrap hover-scroll inventory-table-wrap"><table className="kardex-table">
    <thead><tr><th>FECHA</th><th>ARTÍCULO</th><th>TIPO</th><th>MOTIVO</th><th>CANTIDAD</th><th>STOCK ANTERIOR</th><th>STOCK RESULTANTE</th><th>USUARIO</th></tr></thead>
    <tbody>{items.map((item,index)=>{
     const delta=Number(item.quantityDelta);
     const adjustment=item.movementType==="inventory_adjustment";
     return <tr className={index%2?"alternate":""} key={item.id}>
      <td>{formatRegionalDateTime(item.createdAt,{country:location?.country,timeZone:location?.timezone})}</td>
      <td><b>{item.itemName}</b></td>
      <td><Status tone={adjustment?(item.adjustmentType==="entry"?"green":"orange"):(delta>0?"green":"blue")}>{movementLabel(item)}</Status></td>
      <td><div className="kardex-reason"><b>{traceLabel(item)}</b>{item.note&&<small>{item.note}</small>}</div></td>
      <td><b className="inventory-quantity">{delta>0?"+":""}{formatRegionalNumber(delta,location?.country,{maximumFractionDigits:3})}</b></td>
      <td><span className="kardex-balance">{formatRegionalNumber(Number(item.balanceBefore),location?.country,{maximumFractionDigits:3})}</span></td>
      <td><span className="kardex-balance">{formatRegionalNumber(Number(item.balanceAfter),location?.country,{maximumFractionDigits:3})}</span></td>
      <td><div className="kardex-trace"><b>{item.createdByName||"Usuario no disponible"}</b></div></td>
     </tr>;
    })}</tbody>
   </table></div>}
  </section>
 </>;
}

function KardexSkeleton(){
 return <div className="inventory-table-wrap"><table className="kardex-table" aria-label="Cargando Kárdex"><thead><tr><th>FECHA</th><th>ARTÍCULO</th><th>TIPO</th><th>MOTIVO</th><th>CANTIDAD</th><th>STOCK ANTERIOR</th><th>STOCK RESULTANTE</th><th>USUARIO</th></tr></thead><tbody>{Array.from({length:7},(_,index)=><tr className="inventory-skeleton" key={index}>{Array.from({length:8},(_,cell)=><td key={cell}><i/></td>)}</tr>)}</tbody></table></div>;
}
