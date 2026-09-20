"use client";
import "./inventory.css";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon,PageHeader,Select,Status} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {listInventoryProducts,listStockMovements} from "../infrastructure/inventory-api";

const movementLabels={
 entry:"Entrada",
 sale:"Venta",
 sale_reversal:"Reversa de venta",
 sale_adjustment:"Ajuste por edición",
};
export function KardexPage(){
 const{location}=useSession();
 const[productId,setProductId]=useState("");
 const products=useQuery({queryKey:["inventory-products","kardex"],queryFn:()=>listInventoryProducts()});
 const movements=useQuery({queryKey:["inventory-movements",productId],queryFn:()=>listStockMovements(productId)});
 const items=movements.data?.items??[];

 return <>
  <PageHeader eyebrow="ABASTECIMIENTO" title="Kárdex" description="Trazabilidad de entradas, ventas, reversas y ajustes de la existencia física del local."/>
  <section className="panel standardized-management inventory-panel">
   <div className="inventory-toolbar">
    <Select value={productId} onChange={event=>setProductId(event.target.value)} disabled={products.isLoading} aria-label="Filtrar Kárdex por producto">
     <option value="">Todos los productos</option>
     {products.data?.items.filter(product=>product.quantityControl==="inventory").map(product=><option value={product.id} key={product.id}>{product.name} · {product.sku}</option>)}
    </Select>
    <p><Icon name="receipt" size={15}/>Cada venta física deja su movimiento y saldo resultante.</p>
   </div>
   {movements.isLoading?<KardexSkeleton/>:movements.isError?
    <div className="inventory-state"><Icon name="alert" size={24}/><b>No pudimos cargar el Kárdex</b><p>{movements.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>movements.refetch()}>Reintentar</Button></div>
   :!items.length?
    <div className="inventory-state"><Icon name="receipt" size={24}/><b>Sin movimientos</b><p>Aún no hay entradas o ventas registradas para este filtro.</p></div>
   :<div className="table-wrap hover-scroll inventory-table-wrap"><table>
    <thead><tr><th>FECHA</th><th>PRODUCTO</th><th>MOVIMIENTO</th><th>CANTIDAD</th><th>SALDO</th><th>ORIGEN</th></tr></thead>
    <tbody>{items.map((item,index)=>{
     const delta=Number(item.quantityDelta);
     return <tr className={index%2?"alternate":""} key={item.id}>
      <td>{formatRegionalDateTime(item.createdAt,{country:location?.country,timeZone:location?.timezone})}</td>
      <td><b>{item.productName}</b><small>{item.productId.slice(0,8)}</small></td>
      <td><Status tone={delta>0?"green":"blue"}>{movementLabels[item.movementType]}</Status></td>
      <td><b className="inventory-quantity">{delta>0?"+":""}{formatRegionalNumber(delta,location?.country,{maximumFractionDigits:3})}</b></td>
      <td>{formatRegionalNumber(Number(item.balanceAfter),location?.country,{maximumFractionDigits:3})}</td>
      <td>{item.sourceType==="order"?"Pedido":"Entrada"}<small>{item.sourceId.slice(0,8)}</small></td>
     </tr>;
    })}</tbody>
   </table></div>}
  </section>
 </>;
}

function KardexSkeleton(){
 return <div className="inventory-table-wrap"><table aria-label="Cargando Kárdex"><thead><tr><th>FECHA</th><th>PRODUCTO</th><th>MOVIMIENTO</th><th>CANTIDAD</th><th>SALDO</th><th>ORIGEN</th></tr></thead><tbody>{Array.from({length:7},(_,index)=><tr className="inventory-skeleton" key={index}>{Array.from({length:6},(_,cell)=><td key={cell}><i/></td>)}</tr>)}</tbody></table></div>;
}
