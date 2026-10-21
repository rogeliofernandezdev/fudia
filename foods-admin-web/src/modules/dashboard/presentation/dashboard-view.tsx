"use client";
import "./dashboard.css";
import Link from "next/link";
import {useState} from "react";
import {keepPreviousData,useQuery} from "@tanstack/react-query";
import {Icon,type IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useSession} from "@/providers/session-context";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {pageRoutes} from "@/shared/routing/page-routes";
import {canOpenNavigationItem,navigationItemForPath} from "@/shell/navigation";
import {getDashboard} from "../infrastructure/dashboard-api";
import type {DashboardData,DashboardPeriodKey} from "../domain/types";
import {BarList,Sparkline,TrendChart,type BarListItem,type TrendPoint} from "./dashboard-charts";

const periods:Array<{key:DashboardPeriodKey;label:string;title:string}>=[
  {key:"today",label:"Hoy",title:"Hoy"},
  {key:"7d",label:"7 días",title:"Últimos 7 días"},
  {key:"30d",label:"30 días",title:"Últimos 30 días"},
];

function day(value:string,options:Intl.DateTimeFormatOptions){return new Intl.DateTimeFormat("es",{...options,timeZone:"UTC"}).format(new Date(value+"T12:00:00Z"))}
function formatDate(value:string){return day(value,{day:"numeric",month:"long",weekday:"long"})}
const capitalize=(text:string)=>text.charAt(0).toUpperCase()+text.slice(1);

type Change={text:string;direction:"up"|"down"|"flat"};
type AlertItem={count:number;title:string;detail:string;href:string;tone:"warning"|"info";icon:IconName};
type AreaPanel={title:string;icon:IconName;href:string;rows:{label:string;value:string;tone?:"primary"|"green"}[]};

export function DashboardView(){
  const session=useSession();
  const{user,location,organization,modules}=session;
  const enabled=(module:string)=>Boolean(user?.platformAdmin||modules?.[module]);
  const canOpen=(href:string)=>{const item=navigationItemForPath(href);return Boolean(item&&user&&canOpenNavigationItem(item,{...session,user,modules:modules??{}}))};
  const settings=useSettings();
  const[period,setPeriod]=useState<DashboardPeriodKey>("today");
  const query=useQuery({queryKey:["dashboard",organization?.id,location?.id,period],queryFn:()=>getDashboard(period),refetchInterval:30000,placeholderData:keepPreviousData});
  const data=query.data;
  const money=(value:string|number)=>{
    const n=Number(value||0);
    const formatted=formatRegionalNumber(Math.abs(n),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=n<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  };
  const compact=(value:string|number)=>formatRegionalNumber(Number(value||0),location?.country,{maximumFractionDigits:0});
  const firstName=(user?.name??"Admin").split(" ")[0];
  const selected=periods.find(item=>item.key===period)??periods[0];
  const header=<PageHeader eyebrow="CONTROL DEL NEGOCIO" title={`Hola, ${firstName}`} description={`${location?.name??"Tu local"}${data?.businessDate?" · "+capitalize(formatDate(data.businessDate)):""}`} action={canOpen(pageRoutes.sales)?<Link href={pageRoutes.sales} className="button secondary"><Icon name="sales" size={17}/><span>Ver ventas</span></Link>:undefined}/>;
  const toolbar=<div className="dash-toolbar">
    <div className="dash-periods" role="radiogroup" aria-label="Periodo del resumen">
      {periods.map(item=><button key={item.key} type="button" role="radio" aria-checked={period===item.key} className={period===item.key?"active":""} onClick={()=>setPeriod(item.key)}>{item.label}</button>)}
    </div>
    <span className="dash-refresh" aria-live="polite">{query.isFetching&&!query.isLoading?"Actualizando…":"Se actualiza cada 30 s"}</span>
  </div>;

  if(query.isLoading)return <>{header}{toolbar}<DashboardSkeleton/></>;
  if(query.isError)return <><PageHeader eyebrow="CONTROL DEL NEGOCIO" title="No pudimos cargar el panel" description="Los datos no se reemplazan por valores simulados."/><section className="panel catalog-state error"><span><Icon name="alert"/></span><b>Error de lectura</b><p>{query.error.message}</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  if(!data?.operations)return <>{header}<section className="panel catalog-state error"><Icon name="alert"/><b>Resumen no disponible</b><p>No pudimos leer el estado del restaurante. Inténtalo nuevamente.</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  const operations=data.operations;
  // Las etiquetas siguen al periodo de los datos mostrados, también mientras llega el siguiente.
  const shown=periods.find(item=>item.key===data.period?.key)??selected;

  // Comparación: mismo día de la semana pasada para hoy; ventana anterior para 7 y 30 días.
  const previous=data.previous;
  const comparison=data.period?.key==="today"&&previous?.from?`el ${day(previous.from,{weekday:"long"})} pasado`:`los ${data.period?.days??0} días anteriores`;
  const change=(current:number,before:number|undefined):Change|undefined=>{
    if(before===undefined||before<=0)return undefined;
    const value=Math.round(((current-before)/before)*100);
    return {text:`${value>0?"+":""}${value}% vs. ${comparison}`,direction:value>0?"up":value<0?"down":"flat"};
  };
  const salesNet=Number(data.salesNet)||0;
  const salesChange=change(salesNet,previous?Number(previous.salesNet)||0:undefined);
  const ordersChange=change(data.paidOrders,previous?.paidOrders);
  const ticketChange=change(Number(data.averageTicket)||0,previous?Number(previous.averageTicket)||0:undefined);

  // Tendencia: horas con movimiento (hoy) o todos los días del periodo.
  const rawPoints=data.trend?.points??[];
  const hourly=data.trend?.granularity==="hour";
  const points:TrendPoint[]=hourly?(()=>{
    const byHour=new Map(rawPoints.map(point=>[Number(point.key),point]));
    if(!byHour.size)return [];
    const hours=Array.from(byHour.keys());
    const first=Math.max(0,Math.min(...hours)-(hours.length===1?1:0));
    const last=Math.min(23,Math.max(...hours)+(hours.length===1?1:0));
    return Array.from({length:last-first+1},(_,index)=>{
      const hour=first+index;const point=byHour.get(hour);
      return {key:String(hour),tick:String(hour).padStart(2,"0"),title:`${String(hour).padStart(2,"0")}:00`,value:Number(point?.current)||0,orders:point?.orders??0};
    });
  })():rawPoints.map((point,index)=>({key:point.key,tick:index===rawPoints.length-1?"Hoy":data.period?.days===7?day(point.key,{weekday:"short"}):day(point.key,{day:"numeric"}),title:capitalize(day(point.key,{weekday:"long",day:"numeric",month:"short"})),value:Number(point.current)||0,orders:point.orders}));
  const hasTrend=points.some(point=>point.value!==0);
  const orderSeries=rawPoints.map(point=>point.orders);
  const ticketSeries=rawPoints.map(point=>point.orders>0?(Number(point.current)||0)/point.orders:0);
  const peak=points.reduce<TrendPoint|null>((best,point)=>!best||point.value>best.value?point:best,null);

  const breakdown=(items:DashboardData["salesByChannel"]|undefined):BarListItem[]=>(items??[]).map(item=>({key:item.value,label:item.label,value:Number(item.total)||0,detail:`${item.count} ${item.count===1?"cobro":"cobros"}`}));
  const categories=(data.salesByCategory??[]).map(item=>({name:item.name,qty:Number(item.qty)||0,revenue:Number(item.revenue)||0}));
  const otherCategories=categories.slice(5);
  const categoryItems:BarListItem[]=[
    ...categories.slice(0,5).map(item=>({key:item.name,label:item.name,value:item.revenue,detail:`${item.qty} ${item.qty===1?"unidad":"unidades"}`})),
    ...(otherCategories.length?[{key:"__other",label:`Otras ${otherCategories.length} categorías`,value:otherCategories.reduce((sum,item)=>sum+item.revenue,0),detail:`${otherCategories.reduce((sum,item)=>sum+item.qty,0)} unidades`}]:[]),
  ];
  const topMax=Math.max(1,...data.topProducts.map(product=>Number(product.revenue)||0));

  const areas:AreaPanel[]=[];
  if(enabled("pedidos")||enabled("mesas")||enabled("reservas"))areas.push({title:"Atención",icon:"tables",href:pageRoutes.diningRoom,rows:[
    ...(enabled("mesas")?[{label:"Mesas ocupadas",value:`${operations.tablesOccupied} de ${operations.tablesTotal}`}]:[]),
    ...(enabled("reservas")?[{label:"Reservas de hoy",value:String(data.reservationsToday)}]:[]),
    {label:"Sin pagar",value:String(operations.unpaidOrders)},
  ]});
  if(enabled("cocina"))areas.push({title:"Cocina",icon:"kitchen",href:pageRoutes.kitchen,rows:[
    {label:"Por preparar",value:String(operations.kitchenConfirmed)},
    {label:"En preparación",value:String(operations.kitchenPreparing)},
    {label:"Listos para entregar",value:String(operations.readyOrders),tone:"green"},
  ]});
  if(enabled("delivery"))areas.push({title:"Delivery",icon:"truck",href:pageRoutes.delivery,rows:[
    {label:"Por despachar",value:String(operations.deliveryPending)},
    {label:"En camino",value:String(operations.deliveryInTransit)},
  ]});
  if(enabled("caja"))areas.push({title:"Caja",icon:"cash",href:pageRoutes.cash,rows:[
    {label:"Turnos abiertos",value:`${operations.openCashShifts} de ${operations.activeCashRegisters}`},
    {label:"Efectivo en caja",value:operations.cashBalance===null?"Importe reservado":money(operations.cashBalance),tone:"primary"},
  ]});
  const alerts=([
    {count:data.criticalStock,title:"Stock crítico",detail:"Insumos en mínimo o agotados",href:pageRoutes.inventory,tone:"warning",icon:"stock"},
    {count:data.purchasesToApprove,title:"Compras por aprobar",detail:"Órdenes pendientes de aprobación",href:pageRoutes.purchases,tone:"info",icon:"cart"},
    {count:operations.soldOutProducts,title:"Productos agotados",detail:"Sin disponibilidad para vender hoy",href:pageRoutes.availability,tone:"warning",icon:"utensils"},
  ] satisfies AlertItem[]).filter(item=>item.count>0&&enabled(navigationItemForPath(item.href)?.module??""));

  return <>
    {header}
    {toolbar}
    <div className={"dash-layout"+(query.isPlaceholderData?" refreshing":"")} aria-busy={query.isFetching}>
      <div className="dash-main">
        <section className="panel dash-hero" aria-label={`Ventas · ${shown.title}`}>
          <header>
            <div className="dash-hero-figure">
              <small>Ventas · {shown.title}</small>
              <strong>{money(data.salesNet)}</strong>
              <div className="dash-hero-meta">
                {salesChange?<span className={"dash-change "+salesChange.direction}>{salesChange.text}</span>:<span className="dash-change flat">Sin ventas para comparar con {comparison}</span>}
                {previous&&<em>Antes: {money(previous.salesNet)}</em>}
              </div>
            </div>
            {peak&&peak.value>0&&<div className="dash-hero-peak"><small>{hourly?"Hora pico":"Mejor día"}</small><b>{peak.title}</b><em>{money(peak.value)}</em></div>}
          </header>
          {hasTrend?<TrendChart label={`Ventas · ${shown.title}`} points={points} format={money} compact={compact}/>:<div className="dashboard-empty"><Icon name="sales" size={20}/><b>{shown.key==="today"?"Aún no hay ventas cobradas hoy":"Sin ventas en este periodo"}</b><p>El gráfico aparecerá con el primer cobro.</p></div>}
        </section>

        <section className="dash-stats" aria-label="Indicadores del periodo">
          <article className="panel dash-stat">
            <div><small>Pedidos cobrados</small><strong>{data.paidOrders}</strong></div>
            <Sparkline values={orderSeries}/>
            {ordersChange?<span className={"dash-change "+ordersChange.direction}>{ordersChange.text}</span>:<em>Pagados por completo en el periodo</em>}
          </article>
          <article className="panel dash-stat">
            <div><small>Promedio por pedido</small><strong>{money(data.averageTicket)}</strong></div>
            <Sparkline values={ticketSeries}/>
            {ticketChange?<span className={"dash-change "+ticketChange.direction}>{ticketChange.text}</span>:<em>Por pedido cobrado</em>}
          </article>
          <article className="panel dash-stat">
            <div><small>Pedidos en atención</small><strong>{data.openOrders}</strong></div>
            <span className="dash-stat-icon"><Icon name="orders" size={20}/></span>
            <em>{data.kitchenPending?`${data.kitchenPending} en cocina`:"Sin pedidos en cocina"}</em>
          </article>
        </section>

        <section className="dash-breakdowns" aria-label="Desglose de ventas">
          <article className="panel dash-card">
            <header><span className="panel-icon"><Icon name="orders" size={18}/></span><div><small>{shown.title.toUpperCase()}</small><h2>Ventas por canal</h2></div></header>
            <BarList label="Ventas por canal" items={breakdown(data.salesByChannel)} format={money} empty={{icon:"orders",title:"Sin ventas por canal",text:"Aparecerán con el primer cobro del periodo."}}/>
          </article>
          <article className="panel dash-card">
            <header><span className="panel-icon"><Icon name="payment" size={18}/></span><div><small>{shown.title.toUpperCase()}</small><h2>Medios de pago</h2></div></header>
            <BarList label="Cobros por medio de pago" items={breakdown(data.salesByPaymentMethod)} format={money} empty={{icon:"payment",title:"Sin cobros registrados",text:"Los medios de pago aparecerán con el primer cobro."}}/>
          </article>
          <article className="panel dash-card">
            <header><span className="panel-icon"><Icon name="layers" size={18}/></span><div><small>{shown.title.toUpperCase()}</small><h2>Ventas por categoría</h2></div></header>
            <BarList label="Ventas por categoría" items={categoryItems} format={money} empty={{icon:"layers",title:"Sin categorías vendidas",text:"Se calculan con los pedidos pagados por completo."}}/>
          </article>
        </section>

        <section className="panel dash-card top-products">
          <header><span className="panel-icon"><Icon name="utensils" size={18}/></span><div><small>{shown.title.toUpperCase()}</small><h2>Productos más vendidos</h2></div>{canOpen(pageRoutes.products)&&<Link href={pageRoutes.products} className="panel-link">Ver carta<Icon name="chevron" size={14}/></Link>}</header>
          {data.topProducts.length?<ol className="top-list">{data.topProducts.map((product,index)=>{
            const revenue=Number(product.revenue)||0;
            return <li className="top-product" key={product.name}>
              <span className="rank">{index+1}</span>
              <div className="top-copy"><div><b>{product.name}</b><span className="qty">{Number(product.qty)} {Number(product.qty)===1?"unidad":"unidades"}</span><span className="top-revenue">{money(revenue)}</span></div><div className="top-track"><i style={{width:`${(revenue/topMax)*100}%`}}/></div></div>
            </li>;
          })}</ol>:<div className="dashboard-empty compact"><Icon name="utensils" size={20}/><b>Sin productos vendidos todavía</b><p>Los productos aparecerán después de registrar cobros.</p></div>}
        </section>
      </div>

      <aside className="dash-aside" aria-label="Estado actual del restaurante">
        <section className="panel dash-now">
          <header><h2>Ahora mismo</h2><span className="dash-live"><i aria-hidden="true"/>En vivo</span></header>
          <div className="dash-now-figure">
            <small>Saldo por cobrar</small>
            <b>{money(operations.pendingBalance)}</b>
            <em>{`${operations.unpaidOrders+operations.partialOrders} pedidos pendientes · ${operations.partialOrders} con pago parcial`}</em>
          </div>
          {areas.map(area=><div className="dash-area" key={area.title}>
            <div className="dash-area-title"><Icon name={area.icon} size={16}/><h3>{area.title}</h3>{canOpen(area.href)&&<Link href={area.href} className="dash-area-link" aria-label={`Ver ${area.title.toLowerCase()}`}><Icon name="chevron" size={16}/></Link>}</div>
            <dl>{area.rows.map(row=><div key={row.label}><dt>{row.label}</dt><dd className={row.tone??""}>{row.value}</dd></div>)}</dl>
          </div>)}
        </section>

        <section className="panel alerts">
          <header><span className="panel-icon warning"><Icon name="alert" size={18}/></span><div><small>REQUIERE ATENCIÓN</small><h2>Pendientes por revisar</h2></div>{alerts.length>0&&<b>{alerts.length}</b>}</header>
          {alerts.length?<div className="alert-list">{alerts.map(alert=>{const content=<><span className="alert-icon"><Icon name={alert.icon} size={17}/></span><span className="alert-copy"><b>{alert.title}</b><small>{alert.detail}</small></span><span className="alert-count">{alert.count}</span></>;return canOpen(alert.href)?<Link className={"alert-row "+alert.tone} key={alert.title} href={alert.href}>
            {content}
            <Icon name="chevron" size={16}/>
          </Link>:<div className={"alert-row "+alert.tone} key={alert.title}>{content}</div>})}</div>:<div className="dashboard-empty ok compact"><Icon name="check" size={20}/><b>Sin pendientes por revisar</b><p>Sin alertas de disponibilidad o abastecimiento.</p></div>}
        </section>
      </aside>
    </div>
  </>;
}

function DashboardSkeleton(){
  const head=<header><i className="sk" style={{width:"var(--size-32)",height:"var(--size-32)",borderRadius:"var(--radius-10)"}}/><div><i className="sk" style={{width:"var(--size-80)",height:"var(--size-9)"}}/><i className="sk" style={{width:"var(--size-150)",height:"var(--size-14)"}}/></div></header>;
  return <div className="dash-layout dashboard-skeleton" aria-label="Cargando reportes" aria-busy="true">
    <div className="dash-main">
      <section className="panel dash-hero dashboard-skeleton-kpi">
        <header><div className="dash-hero-figure"><i className="sk" style={{width:"var(--size-120)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-240)",height:"var(--size-48)"}}/><i className="sk" style={{width:"var(--size-180)",height:"var(--size-14)"}}/></div></header>
        <div className="dashboard-skeleton-chart">{[30,42,36,55,48,64,52,70,58,76,62,84].map((height,index)=><span className="sk" key={index} style={{height:height+"%"}}/>)}</div>
      </section>
      <section className="dash-stats">{Array.from({length:3},(_,index)=><article className="panel dash-stat dashboard-skeleton-kpi" key={index}><div><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-80)",height:"var(--size-26)"}}/><i className="sk" style={{width:"var(--size-140)",height:"var(--size-10)"}}/></div></article>)}</section>
      <section className="dash-breakdowns">{Array.from({length:3},(_,index)=><article className="panel dash-card" key={index}>{head}<ul className="dash-bars">{Array.from({length:3},(_,row)=><li key={row}><div className="dash-bar-copy"><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-60)",height:"var(--size-11)"}}/></div><i className="sk" style={{width:"100%",height:"var(--size-8)",borderRadius:"var(--radius-99)"}}/></li>)}</ul></article>)}</section>
      <section className="panel dash-card top-products">{head}<ol className="top-list">{Array.from({length:4},(_,index)=><li className="top-product dashboard-skeleton-product" key={index}><i className="sk" style={{width:"var(--size-26)",height:"var(--size-26)",borderRadius:"var(--radius-8)"}}/><div className="top-copy"><div><i className="sk" style={{width:"var(--size-150)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-70)",height:"var(--size-11)",marginLeft:"auto"}}/></div><i className="sk" style={{width:"100%",height:"var(--size-4)",borderRadius:"var(--radius-99)"}}/></div></li>)}</ol></section>
    </div>
    <aside className="dash-aside">
      <section className="panel dash-now">
        <header><i className="sk" style={{width:"var(--size-110)",height:"var(--size-14)"}}/></header>
        <div className="dash-now-figure"><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-150)",height:"var(--size-24)"}}/></div>
        {Array.from({length:4},(_,index)=><div className="dash-area dashboard-skeleton-operation" key={index}><i className="sk" style={{width:"var(--size-80)",height:"var(--size-12)"}}/><i className="sk" style={{width:"100%",height:"var(--size-10)"}}/><i className="sk" style={{width:"70%",height:"var(--size-10)"}}/></div>)}
      </section>
      <section className="panel alerts">{head}<div className="alert-list">{Array.from({length:3},(_,index)=><div className="alert-row dashboard-skeleton-alert" key={index}><i className="sk" style={{width:"var(--size-36)",height:"var(--size-36)",borderRadius:"var(--radius-10)"}}/><span className="alert-copy"><i className="sk" style={{width:"var(--size-110)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-150)",height:"var(--size-9)"}}/></span></div>)}</div></section>
    </aside>
  </div>;
}
