"use client";
import "./inventory.css";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,Select,Status} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {listInventoryProducts,listStockMovements} from "../infrastructure/inventory-api";
import type {StockMovement} from "../domain/types";

const movementLabels:Record<StockMovement["movementType"],string>={
 entry:"Entrada",sale:"Venta",sale_reversal:"Reversa de venta",sale_adjustment:"Ajuste por edición",
 inventory_adjustment:"Ajuste de inventario",supplier_return:"Devolución a proveedor",receipt_correction:"Corrección de recepción",
 transfer_out:"Transferencia salida",transfer_in:"Transferencia entrada",recipe_consumption:"Consumo por receta",recipe_reversal:"Reversa de receta",
};
const reasonLabels:Record<string,string>={surplus_adjustment:"Ajuste por sobrante",shortage_adjustment:"Ajuste por faltante",waste:"Merma",expiration:"Vencimiento",other_exit:"Otro motivo de salida"};
function movementLabel(item:StockMovement){if(item.movementType==="inventory_adjustment")return item.adjustmentType==="exit"?"Ajuste de salida":"Ajuste de entrada";return movementLabels[item.movementType]}
function traceLabel(item:StockMovement){if(item.reason)return reasonLabels[item.reason]??item.reason;return item.sourceReference||"Sin referencia"}

export function KardexPage(){
 const{location}=useSession();
 const[inventoryItemId,setInventoryItemId]=useState("");
 const[movementType,setMovementType]=useState("");
 const[sourceType,setSourceType]=useState("");
 const[from,setFrom]=useState("");const[to,setTo]=useState("");
 const[page,setPage]=useState(1);const[size,setSize]=useState(25);
 const products=useQuery({queryKey:["inventory-products","kardex"],queryFn:()=>listInventoryProducts()});
 const movements=useQuery({queryKey:["inventory-movements",inventoryItemId,movementType,sourceType,from,to,page,size],queryFn:()=>listStockMovements({inventoryItemId,movementType,sourceType,from,to,page,pageSize:size})});
 const items=movements.data?.items??[];
 return <>
  <PageHeader eyebrow="ABASTECIMIENTO" title="Kárdex" description="Trazabilidad física y valorizada del inventario por local."/>
  <section className="panel standardized-management inventory-panel">
   <div className="inventory-toolbar" style={{flexWrap:"wrap"}}>
    <Select value={inventoryItemId} onChange={e=>{setInventoryItemId(e.target.value);setPage(1)}} disabled={products.isLoading} aria-label="Filtrar Kárdex por artículo"><option value="">Todos los artículos</option>{products.data?.items.map(item=><option value={item.id} key={item.id}>{item.name}{item.kind==="ingredient"?" · Insumo":""}</option>)}</Select>
    <Select value={movementType} onChange={e=>{setMovementType(e.target.value);setPage(1)}} aria-label="Tipo de movimiento"><option value="">Todos los movimientos</option>{Object.entries(movementLabels).map(([v,l])=><option key={v} value={v}>{l}</option>)}</Select>
    <Select value={sourceType} onChange={e=>{setSourceType(e.target.value);setPage(1)}} aria-label="Origen"><option value="">Todos los orígenes</option><option value="purchase_receipt">Recepción de compra</option><option value="purchase_return">Devolución/corrección</option><option value="inventory_transfer">Transferencia</option><option value="order">Pedido / receta</option><option value="inventory_adjustment">Ajuste</option><option value="inventory_entry">Entrada manual</option></Select>
    <Input type="date" value={from} onChange={e=>{setFrom(e.target.value);setPage(1)}} aria-label="Desde"/>
    <Input type="date" value={to} onChange={e=>{setTo(e.target.value);setPage(1)}} aria-label="Hasta"/>
    <p><Icon name="receipt" size={15}/>Cada movimiento conserva cantidad, costo, valor, saldo y usuario.</p>
   </div>
   {movements.isLoading?<KardexSkeleton/>:movements.isError?
    <div className="inventory-state"><Icon name="alert" size={24}/><b>No pudimos cargar el Kárdex</b><p>{movements.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>movements.refetch()}>Reintentar</Button></div>
   :!items.length?<div className="inventory-state"><Icon name="receipt" size={24}/><b>Sin movimientos</b><p>Aún no hay movimientos registrados para estos filtros.</p></div>
   :<div className="table-wrap hover-scroll inventory-table-wrap"><table className="kardex-table">
    <thead><tr><th>FECHA</th><th>ARTÍCULO</th><th>TIPO</th><th>REFERENCIA</th><th>CANTIDAD</th><th>SALDO ANT.</th><th>SALDO</th><th>COSTO U.</th><th>VALOR MOV.</th><th>VALOR SALDO</th><th>USUARIO</th></tr></thead>
    <tbody>{items.map((item,index)=>{const delta=Number(item.quantityDelta);return <tr className={index%2?"alternate":""} key={item.id}>
      <td>{formatRegionalDateTime(item.createdAt,{country:location?.country,timeZone:location?.timezone})}</td><td><b>{item.itemName}</b></td>
      <td><Status tone={delta<0?"orange":"green"}>{movementLabel(item)}</Status></td>
      <td><div className="kardex-reason"><b>{traceLabel(item)}</b>{item.note&&<small>{item.note}</small>}</div></td>
      <td><b className="inventory-quantity">{delta>0?"+":""}{formatRegionalNumber(delta,location?.country,{maximumFractionDigits:3})}</b></td>
      <td>{formatRegionalNumber(Number(item.balanceBefore),location?.country,{maximumFractionDigits:3})}</td>
      <td>{formatRegionalNumber(Number(item.balanceAfter),location?.country,{maximumFractionDigits:3})}</td>
      <td>{formatRegionalNumber(Number(item.unitCost),location?.country,{minimumFractionDigits:2,maximumFractionDigits:4})}</td>
      <td>{formatRegionalNumber(Number(item.valueDelta),location?.country,{minimumFractionDigits:2,maximumFractionDigits:2})}</td>
      <td><b>{formatRegionalNumber(Number(item.balanceValueAfter),location?.country,{minimumFractionDigits:2,maximumFractionDigits:2})}</b></td>
      <td>{item.createdByName||"Usuario no disponible"}</td>
    </tr>})}</tbody>
   </table></div>}
   {!movements.isLoading&&!movements.isError&&<Pagination page={page} size={size} total={movements.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
  </section>
 </>;
}
function KardexSkeleton(){return <div className="inventory-table-wrap"><table className="kardex-table" aria-label="Cargando Kárdex"><thead><tr>{Array.from({length:11},(_,i)=><th key={i}>…</th>)}</tr></thead><tbody>{Array.from({length:7},(_,index)=><tr className="inventory-skeleton" key={index}>{Array.from({length:11},(_,cell)=><td key={cell}><i/></td>)}</tr>)}</tbody></table></div>}
