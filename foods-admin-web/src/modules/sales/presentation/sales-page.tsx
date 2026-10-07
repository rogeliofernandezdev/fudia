"use client";
import "./sales.css";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RowActionButton,Status} from "@/design-system";
import {useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {getSaleDetail,listSales} from "../infrastructure/sales-api";
import type {Sale} from "../domain/types";
import {SaleDetailDialog} from "./sale-detail-dialog";

const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};

export function SalesPage(){
  const{location}=useSession();
  const settings=useSettings();
  const[q,setQ]=useState("");
  const debouncedQ=useDebouncedValue(q);
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[detailId,setDetailId]=useState<string|null>(null);
  const sales=useQuery({queryKey:["sales",debouncedQ,page,size],queryFn:()=>listSales({q:debouncedQ,page,pageSize:size}),refetchInterval:30000});
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
      <div className="toolbar"><label><Icon name="search" size={18}/><Input value={q} onChange={e=>{setQ(e.target.value);setPage(1)}} placeholder="Buscar pedido, cliente o mesa..."/></label></div>
      {sales.isLoading?<SalesTableSkeleton/>
      :sales.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las ventas</b><p>{sales.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>void sales.refetch()}>Reintentar</Button></div>
      :!sales.data?.items.length?<div className="catalog-state"><span><Icon name="receipt"/></span><b>Aún no hay ventas cobradas</b><p>Las ventas aparecerán aquí cuando un pedido quede totalmente pagado.</p></div>
      :<SalesResults items={sales.data.items} money={money} date={date} view={setDetailId}/>}
      {!sales.isLoading&&!sales.isError&&<Pagination page={page} size={size} total={sales.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
    {detailId&&<SaleDetailDialog loading={detail.isLoading} error={detail.error?.message} data={detail.data} money={money} date={date} close={()=>setDetailId(null)} retry={()=>void detail.refetch()}/>}
  </>;
}

function SalesResults({items,money,date,view}:{items:Sale[];money:(value:string|number)=>string;date:(value:string)=>string;view:(id:string)=>void}){
  const action=(sale:Sale)=><div className="standard-actions"><RowActionButton action="view" label={`Ver detalle de la venta ${sale.code}`} onClick={()=>view(sale.id)}/></div>;
  return <>
    <div className="table-wrap hover-scroll sales-table-wrap"><table><thead><tr><th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{items.map((sale,i)=><tr className={i%2?"alternate":""} key={sale.id}><td><span className="row-icon"><Icon name="receipt"/></span><b>{sale.code}</b></td><td><b>{sale.tableName||sale.customerName||"Venta directa"}</b>{sale.customerName&&sale.tableName?<small>{sale.customerName}</small>:null}</td><td>{channelLabel[sale.channel]??sale.channel}</td><td>{date(sale.createdAt)}</td><td><b>{money(sale.total)}</b></td><td><Status tone="green">Pagada</Status></td><td>{action(sale)}</td></tr>)}</tbody></table></div>
    <div className="management-cards sales-cards">{items.map(sale=><article key={sale.id}>
      <header><span className="row-icon"><Icon name="receipt"/></span><div><b>{sale.code}</b><small>{sale.tableName||sale.customerName||"Venta directa"}</small></div><Status tone="green">Pagada</Status></header>
      <dl><div><dt>Canal</dt><dd>{channelLabel[sale.channel]??sale.channel}</dd></div><div><dt>Total</dt><dd>{money(sale.total)}</dd></div><div className="sales-card-date"><dt>Fecha</dt><dd>{date(sale.createdAt)}</dd></div></dl>
      {sale.customerName&&sale.tableName&&<p>{sale.customerName}</p>}<footer>{action(sale)}</footer>
    </article>)}</div>
  </>;
}

function SalesTableSkeleton(){
  return <div aria-label="Cargando ventas" aria-busy="true"><div className="table-wrap hover-scroll">
    <table>
      <thead><tr><th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
      <tbody>{Array.from({length:6},(_,index)=><tr className={`sales-skeleton${index%2?" alternate":""}`} key={index}>
        <td><div className="sales-skeleton-name"><span/><div><b/><i/></div></div></td>
        <td><div className="sales-skeleton-customer"><b/><i/></div></td>
        <td><i/></td>
        <td><i/></td>
        <td><i/></td>
        <td><span className="sales-skeleton-status"/></td>
        <td><span className="sales-skeleton-action"/></td>
      </tr>)}</tbody>
    </table>
  </div><div className="management-cards sales-cards" aria-hidden="true">{Array.from({length:3},(_,index)=><article className="sales-skeleton" key={index}><header><span className="sales-skeleton-status"/><div className="sales-skeleton-customer"><b/><i/></div><span className="sales-skeleton-status"/></header><dl>{Array.from({length:3},(_,cell)=><div key={cell}><i className="sales-skeleton-status"/></div>)}</dl><footer><span className="sales-skeleton-action"/></footer></article>)}</div></div>;
}
