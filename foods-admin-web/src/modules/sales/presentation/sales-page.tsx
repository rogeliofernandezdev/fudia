"use client";
import "./sales.css";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Icon,Input,PageHeader,Pagination,Status} from "@/design-system";
import {useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {listSales} from "../infrastructure/sales-api";

const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};

export function SalesPage(){
  const{location}=useSession();
  const settings=useSettings();
  const[q,setQ]=useState("");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const sales=useQuery({queryKey:["sales",q,page,size],queryFn:()=>listSales({q,page,pageSize:size}),refetchInterval:30000});
  const money=(value:string|number)=>{
    const n=Number(value||0);
    const formatted=formatRegionalNumber(Math.abs(n),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=n<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  };
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  return <><PageHeader eyebrow="CONTROL" title="Ventas" description="Historial real de pedidos completamente cobrados del local."/>
    <section className="panel management standardized-management">
      <div className="toolbar"><label><Icon name="search" size={18}/><Input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar pedido, cliente o mesa..."/></label></div>
      {sales.isLoading?<SalesTableSkeleton/>
      :sales.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las ventas</b><p>{sales.error.message}</p></div>
      :!sales.data?.items.length?<div className="catalog-state"><span><Icon name="receipt"/></span><b>Aún no hay ventas cobradas</b><p>Las ventas aparecerán aquí cuando un pedido quede totalmente pagado.</p></div>
      :<div className="table-wrap hover-scroll"><table><thead><tr><th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th></tr></thead><tbody>{sales.data.items.map((sale,i)=><tr className={i%2?"alternate":""} key={sale.id}><td><span className="row-icon"><Icon name="receipt"/></span><b>{sale.code}</b></td><td><b>{sale.tableName||sale.customerName||"Venta directa"}</b>{sale.customerName&&sale.tableName?<small>{sale.customerName}</small>:null}</td><td>{channelLabel[sale.channel]??sale.channel}</td><td>{date(sale.createdAt)}</td><td><b>{money(sale.paidAmount)}</b></td><td><Status tone="green">Pagada</Status></td></tr>)}</tbody></table></div>}
      {!sales.isLoading&&!sales.isError&&<Pagination page={page} size={size} total={sales.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
  </>;
}


function SalesTableSkeleton(){
  return <div className="table-wrap hover-scroll" aria-label="Cargando ventas" aria-busy="true">
    <table>
      <thead><tr><th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th></tr></thead>
      <tbody>{Array.from({length:6},(_,index)=><tr className={`sales-skeleton${index%2?" alternate":""}`} key={index}>
        <td><div className="sales-skeleton-name"><span/><div><b/><i/></div></div></td>
        <td><div className="sales-skeleton-customer"><b/><i/></div></td>
        <td><i/></td>
        <td><i/></td>
        <td><i/></td>
        <td><span className="sales-skeleton-status"/></td>
      </tr>)}</tbody>
    </table>
  </div>;
}
