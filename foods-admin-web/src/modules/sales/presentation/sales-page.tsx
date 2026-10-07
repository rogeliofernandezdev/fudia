"use client";
import "./sales.css";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,FormField,Icon,Input,PageHeader,Pagination,RowActionButton,Status} from "@/design-system";
import {useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {getSaleDetail,listSales} from "../infrastructure/sales-api";
import type {Sale} from "../domain/types";
import {SaleDetailDialog} from "./sale-detail-dialog";

const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};
const paymentMethodsLabel=(sale:Sale)=>(sale.paymentMethods??[]).join(" · ")||"—";

export function SalesPage(){
  const{location}=useSession();
  const settings=useSettings();
  const[q,setQ]=useState("");
  const debouncedQ=useDebouncedValue(q);
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[detailId,setDetailId]=useState<string|null>(null);
  const[from,setFrom]=useState("");
  const[to,setTo]=useState("");
  const invalidRange=Boolean(from&&to&&from>to);
  const hasFilters=Boolean(q||from||to);
  const sales=useQuery({queryKey:["sales",debouncedQ,from,to,page,size],queryFn:()=>listSales({q:debouncedQ,page,pageSize:size,from,to}),enabled:!invalidRange,refetchInterval:30000});
  const detail=useQuery({queryKey:["sale-detail",detailId],queryFn:()=>getSaleDetail(detailId!),enabled:Boolean(detailId)});
  const money=(value:string|number)=>{
    const n=Number(value||0);
    const formatted=formatRegionalNumber(Math.abs(n),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=n<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  };
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  return <><PageHeader eyebrow="CONTROL" title="Ventas" description="Historial real de pedidos completamente cobrados del local."/>
    <section className="panel management standardized-management">
      <SalesFilters q={q} from={from} to={to} invalidRange={invalidRange} search={value=>{setQ(value);setPage(1)}} start={value=>{setFrom(value);setPage(1)}} end={value=>{setTo(value);setPage(1)}} clear={()=>{setFrom("");setTo("");setPage(1)}}/>
      {invalidRange?null:sales.isLoading?<SalesTableSkeleton/>
      :sales.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las ventas</b><p>{sales.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>void sales.refetch()}>Reintentar</Button></div>
      :!sales.data?.items.length?<div className="catalog-state"><span><Icon name="receipt"/></span><b>{hasFilters?"No hay ventas para estos filtros":"Aún no hay ventas cobradas"}</b><p>{hasFilters?"Prueba otra búsqueda o cambia el periodo.":"Las ventas aparecerán aquí cuando un pedido quede totalmente pagado."}</p></div>
      :<SalesResults items={sales.data.items} money={money} date={date} view={setDetailId}/>}
      {!invalidRange&&!sales.isLoading&&!sales.isError&&<Pagination page={page} size={size} total={sales.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
    {detailId&&<SaleDetailDialog loading={detail.isLoading} error={detail.error?.message} data={detail.data} money={money} date={date} close={()=>setDetailId(null)} retry={()=>void detail.refetch()}/>}
  </>;
}

function SalesFilters({q,from,to,invalidRange,search,start,end,clear}:{q:string;from:string;to:string;invalidRange:boolean;search:(value:string)=>void;start:(value:string)=>void;end:(value:string)=>void;clear:()=>void}){
  return <div className="sales-filters">
    <FormField label="Buscar" className="sales-search"><div className="sales-search-control"><Icon name="search" size={18}/><Input value={q} onChange={event=>search(event.target.value)} placeholder="Pedido, cliente o mesa..." aria-label="Buscar pedido, cliente o mesa"/></div></FormField>
    <FormField label="Fecha de inicio"><Input type="date" value={from} min="0001-01-01" max={to||"9999-12-31"} onChange={event=>start(event.target.value)}/></FormField>
    <FormField label="Fecha de fin" error={invalidRange?"La fecha final no puede ser anterior a la inicial.":undefined}><Input type="date" value={to} min={from||"0001-01-01"} max="9999-12-31" onChange={event=>end(event.target.value)}/></FormField>
    {(from||to)&&<Button kind="ghost" icon="refresh" onClick={clear}>Limpiar fechas</Button>}
  </div>;
}

function SalesResults({items,money,date,view}:{items:Sale[];money:(value:string|number)=>string;date:(value:string)=>string;view:(id:string)=>void}){
  const action=(sale:Sale)=><div className="standard-actions"><RowActionButton action="view" label={`Ver detalle de la venta ${sale.code}`} onClick={()=>view(sale.id)}/></div>;
  return <>
    <div className="table-wrap hover-scroll sales-table-wrap"><table><thead><tr><th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>MEDIO DE PAGO</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{items.map((sale,i)=><tr className={i%2?"alternate":""} key={sale.id}><td><span className="row-icon"><Icon name="receipt"/></span><b>{sale.code}</b></td><td><b>{sale.tableName||sale.customerName||"Venta directa"}</b>{sale.customerName&&sale.tableName?<small>{sale.customerName}</small>:null}</td><td>{channelLabel[sale.channel]??sale.channel}</td><td><span className="sales-payment-methods">{paymentMethodsLabel(sale)}</span></td><td>{date(sale.createdAt)}</td><td><b>{money(sale.total)}</b></td><td><Status tone="green">Pagada</Status></td><td>{action(sale)}</td></tr>)}</tbody></table></div>
    <div className="management-cards sales-cards">{items.map(sale=><article key={sale.id}>
      <header><span className="row-icon"><Icon name="receipt"/></span><div><b>{sale.code}</b><small>{sale.tableName||sale.customerName||"Venta directa"}</small></div><Status tone="green">Pagada</Status></header>
      <dl><div><dt>Canal</dt><dd>{channelLabel[sale.channel]??sale.channel}</dd></div><div><dt>Total</dt><dd>{money(sale.total)}</dd></div><div className="sales-card-methods"><dt>Medio de pago</dt><dd>{paymentMethodsLabel(sale)}</dd></div><div className="sales-card-date"><dt>Fecha</dt><dd>{date(sale.createdAt)}</dd></div></dl>
      {sale.customerName&&sale.tableName&&<p>{sale.customerName}</p>}<footer>{action(sale)}</footer>
    </article>)}</div>
  </>;
}

function SalesTableSkeleton(){
  return <div aria-label="Cargando ventas" aria-busy="true"><div className="table-wrap hover-scroll sales-table-wrap">
    <table>
      <thead><tr><th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>MEDIO DE PAGO</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
      <tbody>{Array.from({length:6},(_,index)=><tr className={`sales-skeleton${index%2?" alternate":""}`} key={index}>
        <td><div className="sales-skeleton-name"><span/><div><b/><i/></div></div></td>
        <td><div className="sales-skeleton-customer"><b/><i/></div></td>
        <td><i/></td>
        <td><i/></td>
        <td><i/></td>
        <td><i/></td>
        <td><span className="sales-skeleton-status"/></td>
        <td><span className="sales-skeleton-action"/></td>
      </tr>)}</tbody>
    </table>
  </div><div className="management-cards sales-cards" aria-hidden="true">{Array.from({length:3},(_,index)=><article className="sales-skeleton" key={index}><header><span className="sales-skeleton-status"/><div className="sales-skeleton-customer"><b/><i/></div><span className="sales-skeleton-status"/></header><dl>{Array.from({length:4},(_,cell)=><div className={cell===2?"sales-card-methods":cell===3?"sales-card-date":undefined} key={cell}><i className="sales-skeleton-status"/></div>)}</dl><footer><span className="sales-skeleton-action"/></footer></article>)}</div></div>;
}
