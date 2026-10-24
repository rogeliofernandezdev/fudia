"use client";
import "./sales.css";
import {useState} from "react";
import dynamic from "next/dynamic";
import {useQuery} from "@tanstack/react-query";
import {Button,DataTable,EmptyState,ErrorState,FormField,Icon,Input,PageHeader,Pagination,RowActionButton,SearchField,Status} from "@/design-system";
import {useMoney,useSession} from "@/providers";
import {formatRegionalCalendarDate,formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {getSaleDetail,listSales} from "../infrastructure/sales-api";
import type {Sale} from "../domain/types";
import {localToday,periodRange,salesPeriods,type DateRange,type SalesPeriod} from "../domain/sales-periods";
import {SaleDetailDialog} from "./sale-detail-dialog";
const PaymentTicketDialog=dynamic(()=>import("./payment-ticket-dialog").then(module=>module.PaymentTicketDialog),{ssr:false});

const channelLabel:Record<string,string>={salon:"Salón",mostrador:"Mostrador",recojo:"Recojo",delivery:"Delivery",whatsapp:"WhatsApp"};
const paymentMethodsLabel=(sale:Sale)=>(sale.paymentMethods??[]).join(" · ")||"—";

export function SalesPage(){
  const{location}=useSession();
  const[q,setQ]=useState("");
  const debouncedQ=useDebouncedValue(q);
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[detailId,setDetailId]=useState<string|null>(null);
  const[from,setFrom]=useState("");
  const[to,setTo]=useState("");
  const[ticketId,setTicketId]=useState<string|null>(null);
  const[period,setPeriod]=useState<SalesPeriod>("today");
  // Los periodos rápidos se recalculan con la fecha del local en cada render: al pasar la medianoche local, «Hoy» avanza solo.
  const range:DateRange=period==="custom"?{from,to}:periodRange(period,localToday(location?.timezone));
  const invalidRange=Boolean(range.from&&range.to&&range.from>range.to);
  const hasFilters=Boolean(q||range.from||range.to);
  const sales=useQuery({queryKey:["sales",debouncedQ,range.from,range.to,page,size],queryFn:()=>listSales({q:debouncedQ,page,pageSize:size,from:range.from,to:range.to}),enabled:!invalidRange,refetchInterval:30000});
  // Editar una fecha a mano parte del rango visible y pasa a «Personalizado».
  const custom=(next:Partial<DateRange>)=>{setFrom(next.from??range.from);setTo(next.to??range.to);setPeriod("custom");setPage(1)};
  const detail=useQuery({queryKey:["sale-detail",detailId],queryFn:()=>getSaleDetail(detailId!),enabled:Boolean(detailId)});
  const money=useMoney();
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  return <><PageHeader title="Ventas" description="Historial real de pedidos completamente cobrados del local."/>
    <section className="panel management standardized-management sales-panel">
      <SalesPeriods period={period} choose={value=>{if(value==="custom"){custom({})}else{setPeriod(value);setPage(1)}}}/>
      <SalesFilters q={q} period={period} from={range.from} to={range.to} invalidRange={invalidRange} search={value=>{setQ(value);setPage(1)}} start={value=>custom({from:value})} end={value=>custom({to:value})} clear={()=>custom({from:"",to:""})}/>
      {!invalidRange&&<SalesPeriodSummary period={period} range={range} loading={sales.isLoading} error={sales.isError} count={sales.data?.total??0} amount={sales.data?.amountTotal} money={money} day={value=>formatRegionalCalendarDate(value,location?.country,{day:"numeric",month:"short"})}/>}
      {invalidRange?null:sales.isLoading?<SalesTableSkeleton/>
      :sales.isError?<ErrorState title="No pudimos cargar las ventas" text={sales.error.message} onRetry={()=>void sales.refetch()} retrying={sales.isFetching}/>
      :!sales.data?.items.length?(hasFilters?<EmptyState icon="search" title="Sin coincidencias" text="No hay ventas para estos filtros. Prueba otra búsqueda o cambia el periodo."/>:<EmptyState icon="receipt" title="Aún no hay ventas cobradas" text="Las ventas aparecerán aquí cuando un pedido quede totalmente pagado."/>)
      :<SalesResults items={sales.data.items} money={money} date={date} view={setDetailId}/>}
      {!invalidRange&&!sales.isLoading&&!sales.isError&&<Pagination page={page} size={size} total={sales.data?.total??0} onPage={setPage} onSize={v=>{setSize(v);setPage(1)}}/>}
    </section>
    {detailId&&<SaleDetailDialog loading={detail.isLoading} error={detail.error?.message} data={detail.data} money={money} date={date} close={()=>setDetailId(null)} retry={()=>void detail.refetch()} ticket={()=>{setDetailId(null);setTicketId(detailId)}}/>}
    {ticketId&&<PaymentTicketDialog orderId={ticketId} close={()=>setTicketId(null)}/>}
  </>;
}

/** Periodos rápidos: hijo directo del panel para quedar fijos en móvil mientras se recorre la lista. */
function SalesPeriods({period,choose}:{period:SalesPeriod;choose:(value:SalesPeriod)=>void}){
  return <div className="sales-periods" role="radiogroup" aria-label="Periodo de ventas">
    {salesPeriods.map(option=><button type="button" key={option.value} role="radio" aria-checked={period===option.value} className={`sales-period${period===option.value?" active":""}`} onClick={()=>choose(option.value)}>{option.label}</button>)}
  </div>;
}

function SalesFilters({q,period,from,to,invalidRange,search,start,end,clear}:{q:string;period:SalesPeriod;from:string;to:string;invalidRange:boolean;search:(value:string)=>void;start:(value:string)=>void;end:(value:string)=>void;clear:()=>void}){
  const custom=period==="custom";
  // Las fechas solo aparecen en «Personalizado»: con un periodo rápido, la franja del total ya nombra el periodo.
  return <div className={`sales-filters${custom?" is-custom":""}`}>
    <FormField as="div" label="Buscar" className="sales-search"><SearchField value={q} onChange={event=>search(event.target.value)} placeholder="Pedido, cliente o mesa…" label="Buscar pedido, cliente o mesa"/></FormField>
    {custom&&<>
     <FormField label="Desde"><Input type="date" value={from} min="0001-01-01" max={to||"9999-12-31"} onChange={event=>start(event.target.value)}/></FormField>
     <FormField label="Hasta" help={to?undefined:"Vacío: hasta hoy"} error={invalidRange?"La fecha final no puede ser anterior a la inicial.":undefined}><Input type="date" value={to} min={from||"0001-01-01"} max="9999-12-31" onChange={event=>end(event.target.value)}/></FormField>
     {(from||to)&&<Button kind="ghost" icon="refresh" onClick={clear}>Limpiar fechas</Button>}
    </>}
  </div>;
}

/** Total del periodo filtrado: suma de todo el filtro (no solo la página), número de ventas y ticket promedio. */
function SalesPeriodSummary({period,range,loading,error,count,amount,money,day}:{period:SalesPeriod;range:DateRange;loading:boolean;error:boolean;count:number;amount?:string;money:(value:string|number)=>string;day:(value:string)=>string}){
  if(error)return null;
  const title=period!=="custom"?salesPeriods.find(option=>option.value===period)!.summary
   :range.from&&range.to?`Ventas del ${day(range.from)} al ${day(range.to)}`
   :range.from?`Ventas del ${day(range.from)} hasta hoy`
   :range.to?`Ventas hasta el ${day(range.to)}`
   :"Todas las ventas";
  if(loading)return <div className="sales-period-summary" role="status" aria-busy="true" aria-label="Cargando total del periodo"><span>{title}</span><i className="sales-summary-skeleton"/></div>;
  if(amount===undefined)return null;
  const total=Number(amount);
  return <div className="sales-period-summary" role="status" aria-live="polite">
    <span>{title}</span>
    <strong className="tabular">{money(amount)}</strong>
    <small>{count===1?"1 venta":`${count} ventas`}{count>0&&<> · Ticket promedio <b className="tabular">{money(total/count)}</b></>}</small>
  </div>;
}

function SalesResults({items,money,date,view}:{items:Sale[];money:(value:string|number)=>string;date:(value:string)=>string;view:(id:string)=>void}){
  const action=(sale:Sale)=><div className="standard-actions"><RowActionButton action="view" label={`Ver detalle de la venta ${sale.code}`} onClick={()=>view(sale.id)}/></div>;
  return <>
    <DataTable label="Historial de ventas" className="sales-table" containerClassName="hover-scroll sales-table-wrap"><thead><tr><th>Pedido</th><th>Cliente / mesa</th><th>Canal</th><th>Medio de pago</th><th>Fecha</th><th>Total</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>{items.map((sale,i)=><tr className={i%2?"alternate":""} key={sale.id}><td><span className="row-icon"><Icon name="receipt"/></span><b>{sale.code}</b></td><td><b>{sale.tableName||sale.customerName||"Venta directa"}</b>{sale.customerName&&sale.tableName?<small>{sale.customerName}</small>:null}</td><td>{channelLabel[sale.channel]??sale.channel}</td><td><span className="sales-payment-methods">{paymentMethodsLabel(sale)}</span></td><td>{date(sale.createdAt)}</td><td><b>{money(sale.total)}</b></td><td><Status tone="green">Pagada</Status></td><td>{action(sale)}</td></tr>)}</tbody></DataTable>
  </>;
}

function SalesTableSkeleton(){
  return <div aria-label="Cargando ventas" aria-busy="true"><div className="table-wrap hover-scroll sales-table-wrap">
    <table>
      <thead><tr><th>Pedido</th><th>Cliente / mesa</th><th>Canal</th><th>Medio de pago</th><th>Fecha</th><th>Total</th><th>Estado</th><th>Acciones</th></tr></thead>
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
  </div></div>;
}
