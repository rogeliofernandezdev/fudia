"use client";
import "./dashboard.css";
import Link from "next/link";
import {useState,type CSSProperties} from "react";
import {keepPreviousData,useQuery} from "@tanstack/react-query";
import {Icon} from "@/design-system/icons";
import {IconTile} from "@/design-system/icon-tile";
import {PageHeader} from "@/design-system/page-header";
import {Tabs} from "@/design-system/tabs";
import {useSession} from "@/providers/session-context";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {pageRoutes} from "@/shared/routing/page-routes";
import {canOpenNavigationItem,navigationItemForPath} from "@/shell/navigation";
import {getDashboard} from "../infrastructure/dashboard-api";
import type {DashboardData,DashboardPeriodKey} from "../domain/types";
import {BarList,CountUp,DonutChart,RankTrack,TrendChart,type BarListItem,type DashTone,type TrendPoint} from "./dashboard-charts";

const periods:Array<{key:DashboardPeriodKey;label:string;title:string}>=[
  {key:"today",label:"Hoy",title:"Hoy"},
  {key:"7d",label:"7 días",title:"Últimos 7 días"},
  {key:"30d",label:"30 días",title:"Últimos 30 días"},
];

function day(value:string,options:Intl.DateTimeFormatOptions){return new Intl.DateTimeFormat("es",{...options,timeZone:"UTC"}).format(new Date(value+"T12:00:00Z"))}
function formatDate(value:string){return day(value,{day:"numeric",month:"long",weekday:"long"})}
const capitalize=(text:string)=>text.charAt(0).toUpperCase()+text.slice(1);
/** Combinación elegida por el dueño (2026-11-20): canales en índigo; medios de pago con el efectivo en violeta y los
 * digitales en petróleo y celeste según su orden en el catálogo (después índigo, marino y durazno). El color sigue
 * al medio, nunca a su posición en el anillo. */
const digitalTones:DashTone[]=["teal","sky","indigo","navy","peach"];
function methodTone(item:{affectsCash?:boolean;digitalIndex?:number}):DashTone{return item.affectsCash?"violet":item.digitalIndex===undefined?"gray":digitalTones[item.digitalIndex%digitalTones.length]}
/** Duración legible: «45 min», «1 h», «1 h 35 min». */
function formatMinutes(total:number){const hours=Math.floor(total/60),minutes=total%60;return hours?`${hours} h${minutes?` ${minutes} min`:""}`:`${minutes} min`}
/** «04» → «Mesa 04»; un nombre que ya empieza con «Mesa» se respeta tal cual. */
function tableTitle(name:string){const trimmed=name.trim();return /^mesa\b/i.test(trimmed)?trimmed:`Mesa ${trimmed}`}

type Change={value:string;direction:"up"|"down"|"flat"};
type RankItem={key:string;name:string;qty:number;revenue:number;ranked:boolean};
/** Cada cifra de «Ahora mismo» lleva el color de su estado (versión de texto, con contraste sobre blanco). */
type AreaTone="primary"|"indigo"|"orange"|"purple"|"green"|"navy";
type AreaPanel={title:string;href:string;rows:{label:string;value:string;tone?:AreaTone}[]};
type ReviewItem={count:number;title:string;href:string};
type Ranking="products"|"categories";

export function DashboardView(){
  const session=useSession();
  const{user,location,organization,modules}=session;
  const enabled=(module:string)=>Boolean(user?.platformAdmin||modules?.[module]);
  const canOpen=(href:string)=>{const item=navigationItemForPath(href);return Boolean(item&&user&&canOpenNavigationItem(item,{...session,user,modules:modules??{}}))};
  const settings=useSettings();
  const[period,setPeriod]=useState<DashboardPeriodKey>("today");
  const[ranking,setRanking]=useState<Ranking>("products");
  const query=useQuery({queryKey:["dashboard",organization?.id,location?.id,period],queryFn:()=>getDashboard(period),meta:{live:["orders","payments","cash","availability","reservations"],poll:30000},placeholderData:keepPreviousData});
  const data=query.data;
  const money=(value:string|number)=>{
    const n=Number(value||0);
    const formatted=formatRegionalNumber(Math.abs(n),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=n<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  };
  const compact=(value:string|number)=>formatRegionalNumber(Number(value||0),location?.country,{maximumFractionDigits:0});
  const firstName=(user?.name??"Admin").split(" ")[0];
  const header=<PageHeader title={`Hola, ${firstName}`} description={`${location?.name??"Tu local"}${data?.businessDate?" · "+capitalize(formatDate(data.businessDate)):""}`}/>;
  // La base de la comparación se dice una sola vez, junto al periodo que la define.
  const toolbar=(comparison?:string)=><div className="dash-toolbar">
    <div className="dash-periods" role="radiogroup" aria-label="Periodo del resumen">
      {periods.map(item=><button key={item.key} type="button" role="radio" aria-checked={period===item.key} className={period===item.key?"active":""} onClick={()=>setPeriod(item.key)}>{item.label}</button>)}
    </div>
    {comparison&&<span className="dash-compare">Comparado con {comparison}</span>}
  </div>;

  if(query.isLoading)return <>{header}{toolbar()}<DashboardSkeleton/></>;
  if(query.isError)return <><PageHeader title="No pudimos cargar el panel" description="Los datos no se reemplazan por valores simulados."/><section className="panel catalog-state error"><span><Icon name="alert"/></span><b>Error de lectura</b><p>{query.error.message}</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  if(!data?.operations)return <>{header}<section className="panel catalog-state error"><Icon name="alert"/><b>Resumen no disponible</b><p>No pudimos leer el estado del restaurante. Inténtalo nuevamente.</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  const operations=data.operations;
  // Las etiquetas siguen al periodo de los datos mostrados, también mientras llega el siguiente.
  const shown=periods.find(item=>item.key===data.period?.key)??periods.find(item=>item.key===period)??periods[0];

  // Comparación cortada a la misma hora local: mismo día de la semana pasada para hoy; ventana anterior para 7 y 30 días.
  const previous=data.previous;
  const isToday=data.period?.key==="today";
  const lastWeekday=isToday&&previous?.from?day(previous.from,{weekday:"long"}):"";
  const comparison=lastWeekday?`el ${lastWeekday} pasado a esta hora`:`los ${data.period?.days??0} días anteriores`;
  const change=(current:number,before:number|undefined):Change|undefined=>{
    if(before===undefined||before<=0)return undefined;
    const value=Math.round(((current-before)/before)*100);
    return {value:`${value>0?"+":""}${value}%`,direction:value>0?"up":value<0?"down":"flat"};
  };
  const salesNet=Number(data.salesNet)||0;
  const salesChange=change(salesNet,previous?Number(previous.salesNet)||0:undefined);
  const ordersChange=change(data.paidOrders,previous?.paidOrders);
  // Sin pedidos no hay promedio que comparar: se omite la variación.
  const ticketChange=data.paidOrders>0?change(Number(data.averageTicket)||0,previous?Number(previous.averageTicket)||0:undefined):undefined;
  const compared=Boolean(salesChange||ordersChange||ticketChange);

  // Tendencia: horas con movimiento hoy o el mismo día de la semana pasada, hasta la hora en curso (hoy),
  // o todos los días del periodo. La serie anterior va completa como referencia; la actual se detiene en «now».
  const rawPoints=data.trend?.points??[];
  const hourly=data.trend?.granularity==="hour";
  const nowHour=hourly&&data.trend?.now?Number(data.trend.now):undefined;
  const points:TrendPoint[]=hourly?(()=>{
    const byHour=new Map(rawPoints.map(point=>[Number(point.key),point]));
    if(!byHour.size)return [];
    const hours=Array.from(byHour.keys()).concat(nowHour===undefined?[]:[nowHour]);
    const single=new Set(hours).size===1;
    const first=Math.max(0,Math.min(...hours)-(single?1:0));
    const last=Math.min(23,Math.max(...hours)+(single?1:0));
    return Array.from({length:last-first+1},(_,index)=>{
      const hour=first+index;const point=byHour.get(hour);
      return {key:String(hour),tick:String(hour).padStart(2,"0"),title:`${String(hour).padStart(2,"0")}:00`,value:Number(point?.current)||0,previous:Number(point?.previous)||0,orders:point?.orders??0,future:nowHour!==undefined&&hour>nowHour};
    });
  })():rawPoints.map((point,index)=>({key:point.key,tick:index===rawPoints.length-1?"Hoy":data.period?.days===7?day(point.key,{weekday:"short"}):day(point.key,{day:"numeric"}),title:capitalize(day(point.key,{weekday:"long",day:"numeric",month:"short"})),value:Number(point.current)||0,previous:Number(point.previous)||0,orders:point.orders,future:false}));
  const hasTrend=points.some(point=>point.value!==0||point.previous!==0);
  const trendSeries={current:shown.key==="today"?"Hoy":shown.title,previous:lastWeekday?capitalize(`${lastWeekday} pasado`):`${data.period?.days??0} días anteriores`};
  const peak=points.reduce<TrendPoint|null>((best,point)=>!best||point.value>best.value?point:best,null);

  // Ritmo del día: lo cobrado hoy frente al día completo de la semana pasada (solo «Hoy»).
  const previousDayTotal=isToday?points.reduce((sum,point)=>sum+point.previous,0):0;
  const pace=previousDayTotal>0?Math.max(0,salesNet)/previousDayTotal:undefined;
  // Canal y medio de pago muestran importe y participación; el canal suma sus pedidos y promedio (su suma es «Pedidos»).
  const breakdown=(items:DashboardData["salesByChannel"]|undefined,tone:(item:DashboardData["salesByChannel"][number])=>DashTone):BarListItem[]=>(items??[]).map(item=>({key:item.value,label:item.label,value:Number(item.total)||0,tone:tone(item),detail:item.orders?`${item.orders} ${item.orders===1?"pedido":"pedidos"} · promedio ${money(item.averageTicket??0)}`:undefined}));
  const products:RankItem[]=data.topProducts.map(product=>({key:product.name,name:product.name,qty:Number(product.qty)||0,revenue:Number(product.revenue)||0,ranked:true}));
  const allCategories=(data.salesByCategory??[]).map(item=>({key:item.name,name:item.name,qty:Number(item.qty)||0,revenue:Number(item.revenue)||0,ranked:true}));
  const otherCategories=allCategories.slice(5);
  const categories:RankItem[]=[
    ...allCategories.slice(0,5),
    ...(otherCategories.length?[{key:"__other",name:`Otras ${otherCategories.length} categorías`,qty:otherCategories.reduce((sum,item)=>sum+item.qty,0),revenue:otherCategories.reduce((sum,item)=>sum+item.revenue,0),ranked:false}]:[]),
  ];
  const rankItems=ranking==="products"?products:categories;
  const rankMax=Math.max(1,...rankItems.map(item=>item.revenue));

  // Cada número de la operación aparece una sola vez: lo pendiente de cobro vive en «Saldo por cobrar».
  const areas:AreaPanel[]=[];
  if(enabled("pedidos")||enabled("mesas")||enabled("reservas"))areas.push({title:"Atención",href:enabled("mesas")?pageRoutes.diningRoom:pageRoutes.orders,rows:[
    {label:"Pedidos activos",value:String(data.openOrders),tone:"primary"},
    ...(enabled("reservas")?[{label:"Reservas de hoy",value:String(data.reservationsToday)}]:[]),
  ]});
  // Mesas con los mismos términos del Salón; los tiempos se miden desde el primer pedido aún activo de cada mesa.
  if(enabled("mesas"))areas.push({title:"Mesas",href:pageRoutes.diningRoom,rows:[
    {label:"Ocupadas",value:`${operations.tablesOccupied} de ${operations.tablesTotal}`,tone:"primary"},
    {label:"Libres",value:String(Math.max(0,operations.tablesTotal-operations.tablesOccupied)),tone:"green"},
    ...(operations.tableMinutesAverage!=null&&operations.tablesOccupied>0?[{label:"Tiempo promedio",value:formatMinutes(operations.tableMinutesAverage)}]:[]),
    ...(operations.oldestTable?[{label:"Más tiempo",value:`${tableTitle(operations.oldestTable.name)} · ${formatMinutes(operations.oldestTable.minutes)}`}]:[]),
  ]});
  if(enabled("cocina"))areas.push({title:"Cocina",href:pageRoutes.kitchen,rows:[
    {label:"Por preparar",value:String(operations.kitchenConfirmed),tone:"indigo"},
    {label:"En preparación",value:String(operations.kitchenPreparing),tone:"orange"},
    {label:"Por entregar",value:String(operations.readyOrders),tone:"purple"},
  ]});
  if(enabled("delivery"))areas.push({title:"Delivery",href:pageRoutes.delivery,rows:[
    {label:"Por despachar",value:String(operations.deliveryPending),tone:"orange"},
    {label:"En camino",value:String(operations.deliveryInTransit),tone:"navy"},
  ]});
  if(enabled("caja"))areas.push({title:"Caja",href:pageRoutes.cash,rows:[
    {label:"Turnos abiertos",value:`${operations.openCashShifts} de ${operations.activeCashRegisters}`},
    {label:"Efectivo en caja",value:operations.cashBalance===null?"Importe reservado":money(operations.cashBalance),tone:"green"},
  ]});
  const review=([
    {count:data.criticalStock,title:"Stock crítico",href:pageRoutes.inventory},
    {count:data.purchasesToApprove,title:"Compras por aprobar",href:pageRoutes.purchases},
    {count:operations.soldOutProducts,title:"Productos agotados",href:pageRoutes.availability},
  ] satisfies ReviewItem[]).filter(item=>item.count>0&&enabled(navigationItemForPath(item.href)?.module??""));
  const pendingOrders=operations.unpaidOrders+operations.partialOrders;

  // Sin base de comparación (el periodo anterior no vendió) no hay porcentaje que mostrar: se omite la línea.
  const changeLine=(item:Change|undefined,before?:string)=>item?<p className={"dash-change "+item.direction}><b><Icon name={item.direction==="up"?"arrowUpRight":item.direction==="down"?"arrowDownRight":"minus"} size={14}/>{item.value}</b>{before&&<span>{before}</span>}</p>:null;
  // Entrada escalonada en orden de lectura; la columna en vivo acompaña a la tendencia.
  const enter=(index:number):CSSProperties=>({animationDelay:`${index*50}ms`});

  return <>
    {header}
    {toolbar(compared?comparison:undefined)}
    <div className={"dash-board"+(query.isPlaceholderData?" refreshing":"")} aria-busy={query.isFetching}>
      <section className="dash-kpis" aria-label={`Indicadores · ${shown.title}`}>
        <article className="panel dash-kpi lead green dash-enter" style={enter(0)}>
          <div className="dash-kpi-head"><small>Ventas</small>{canOpen(pageRoutes.sales)&&<Link href={pageRoutes.sales} className="stat-card-link" aria-label="Ver detalle de ventas"><Icon name="arrowUpRight" size={16}/></Link>}<IconTile name="sales" tone="brand" appearance="vivid" className="dash-kpi-icon"/></div>
          <strong><CountUp value={salesNet} format={money}>{money(data.salesNet)}</CountUp></strong>
          {changeLine(salesChange,previous?`Antes: ${money(previous.salesNet)}`:undefined)}
          {pace!==undefined&&<div className="dash-pace"><p><b>{Math.round(pace*100)}%</b> de lo vendido el {lastWeekday} completo</p><RankTrack ratio={Math.min(1,pace)} index={0}/></div>}
        </article>
        <article className="panel dash-kpi blue dash-enter" style={enter(1)}>
          <div className="dash-kpi-head"><small>Pedidos</small><IconTile name="receipt" tone="primary" appearance="vivid" className="dash-kpi-icon"/></div>
          <strong><CountUp value={data.paidOrders} format={compact}>{String(data.paidOrders)}</CountUp></strong>
          {changeLine(ordersChange)}
        </article>
        <article className="panel dash-kpi violet dash-enter" style={enter(2)}>
          <div className="dash-kpi-head"><small>Promedio</small><IconTile name="payment" tone="digital" appearance="vivid" className="dash-kpi-icon"/></div>
          <strong><CountUp value={Number(data.averageTicket)||0} format={money}>{money(data.averageTicket)}</CountUp></strong>
          {changeLine(ticketChange)}
        </article>
      </section>

      <div className="dash-layout">
        <div className="dash-main">
          <section className="panel dash-card dash-trend-card dash-enter" style={enter(3)} aria-label={`Evolución de ventas · ${shown.title}`}>
            <header>
              <h2>Evolución de ventas</h2>
              {/* El importe del pico ya está rotulado en el gráfico: aquí solo cuándo ocurrió. */}
              {peak&&peak.value>0&&<p className="dash-trend-peak"><small>{hourly?"Hora pico":"Mejor día"}</small><b>{peak.title}</b></p>}
            </header>
            {hasTrend?<TrendChart label={`Ventas · ${shown.title}`} points={points} format={money} compact={compact} live={shown.key==="today"} series={trendSeries}/>:<div className="dashboard-empty"><Icon name="sales" size={20}/><b>{shown.key==="today"?"Aún no hay ventas cobradas hoy":"Sin ventas en este periodo"}</b><p>El gráfico aparecerá con el primer cobro.</p></div>}
          </section>

          <section className="dash-breakdowns" aria-label="Desglose de ventas">
            <article className="panel dash-card dash-enter" style={enter(4)}>
              <header><h2>Ventas por canal</h2></header>
              <BarList label="Ventas por canal" items={breakdown(data.salesByChannel,()=>"indigo")} format={money} empty={{icon:"orders",title:"Sin ventas por canal",text:"Aparecerán con el primer cobro del periodo."}}/>
            </article>
            <article className="panel dash-card dash-enter" style={enter(5)}>
              <header><h2>Medios de pago</h2></header>
              <DonutChart label="Cobros por medio de pago" items={breakdown(data.salesByPaymentMethod,methodTone)} format={money} empty={{icon:"payment",title:"Sin cobros registrados",text:"Los medios de pago aparecerán con el primer cobro."}}/>
            </article>
          </section>

          <section className="panel dash-card dash-ranking dash-enter" style={enter(6)} aria-label="Lo más vendido">
            <header>
              <h2>Lo más vendido</h2>
              <Tabs label="Ver ranking por" value={ranking} onChange={setRanking} items={[{value:"products",label:"Productos",icon:"utensils"},{value:"categories",label:"Categorías",icon:"layers"}]}/>
            </header>
            {rankItems.length?<ol className="top-list">{rankItems.map((item,index)=><li className="top-product" key={item.key}>
              <span className={"rank"+(item.ranked?"":" empty")} aria-hidden={!item.ranked||undefined}>{item.ranked?index+1:""}</span>
              <div className="top-copy"><div><b>{item.name}</b><span className="qty">{item.qty} {item.qty===1?"unidad":"unidades"}</span><span className="top-revenue">{money(item.revenue)}</span></div><RankTrack ratio={item.revenue/rankMax} index={index}/></div>
            </li>)}</ol>:<div className="dashboard-empty compact"><Icon name={ranking==="products"?"utensils":"layers"} size={20}/><b>{ranking==="products"?"Sin productos vendidos todavía":"Sin categorías vendidas"}</b><p>Aparecerán después de registrar cobros.</p></div>}
          </section>
        </div>

        <aside className="panel dash-now dash-enter" style={enter(3)} aria-label="Estado actual del restaurante">
          <header><h2>Ahora mismo</h2><span className="dash-live"><i aria-hidden="true"/>En vivo</span></header>
          <div className="dash-now-figure">
            <small>Saldo por cobrar</small>
            <b><CountUp value={Number(operations.pendingBalance)||0} format={money}>{money(operations.pendingBalance)}</CountUp></b>
            {pendingOrders>0&&<em>{`${pendingOrders} ${pendingOrders===1?"pedido":"pedidos"}${operations.partialOrders?` · ${operations.partialOrders} con pago parcial`:""}`}</em>}
          </div>
          {areas.length>0&&<div className="dash-now-areas">{areas.map(area=><div className="dash-area" key={area.title}>
            <div className="dash-area-title"><h3>{area.title}</h3>{canOpen(area.href)&&<Link href={area.href} className="dash-area-link" aria-label={`Ver ${area.title.toLowerCase()}`}><Icon name="chevron" size={16}/></Link>}</div>
            <dl>{area.rows.map(row=><div key={row.label}><dt>{row.label}</dt><dd className={row.tone??""}>{row.value}</dd></div>)}</dl>
          </div>)}</div>}
          <div className="dash-review">
            <h3>Por revisar</h3>
            {review.length?<ul>{review.map(item=>{const content=<><span>{item.title}</span><b>{item.count}</b></>;return <li key={item.title}>{canOpen(item.href)?<Link className="dash-review-row" href={item.href}>{content}<Icon name="chevron" size={16}/></Link>:<div className="dash-review-row">{content}</div>}</li>})}</ul>:<p className="dash-review-ok"><Icon name="check" size={16}/>Sin pendientes por revisar</p>}
          </div>
        </aside>
      </div>
    </div>
  </>;
}

function DashboardSkeleton(){
  const head=<header><i className="sk" style={{width:"var(--size-150)",height:"var(--size-14)"}}/></header>;
  return <div className="dash-board dashboard-skeleton" aria-label="Cargando reportes" aria-busy="true">
    <section className="dash-kpis">{Array.from({length:3},(_,index)=><article className={"panel dash-kpi dashboard-skeleton-kpi"+(index===0?" lead":"")} key={index}><div className="dash-kpi-head"><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-40)",height:"var(--size-40)",borderRadius:"50%",marginLeft:"auto"}}/></div><i className="sk" style={{width:index===0?"var(--size-180)":"var(--size-80)",height:index===0?"var(--size-36)":"var(--size-28)"}}/><i className="sk" style={{width:"var(--size-100)",height:"var(--size-10)"}}/></article>)}</section>
    <div className="dash-layout">
      <div className="dash-main">
        <section className="panel dash-card dash-trend-card">
          {head}
          <div className="dashboard-skeleton-chart">{[30,42,36,55,48,64,52,70,58,76,62,84].map((height,index)=><span className="sk" key={index} style={{height:height+"%"}}/>)}</div>
        </section>
        <section className="dash-breakdowns">{Array.from({length:2},(_,index)=><article className="panel dash-card" key={index}>{head}<ul className="dash-bars">{Array.from({length:3},(_,row)=><li key={row}><div className="dash-bar-copy"><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-60)",height:"var(--size-11)"}}/></div><i className="sk" style={{width:"100%",height:"var(--size-8)",borderRadius:"var(--radius-pill)"}}/></li>)}</ul></article>)}</section>
        <section className="panel dash-card dash-ranking">{head}<ol className="top-list">{Array.from({length:4},(_,index)=><li className="top-product dashboard-skeleton-product" key={index}><i className="sk" style={{width:"var(--size-26)",height:"var(--size-26)",borderRadius:"var(--radius-tile)"}}/><div className="top-copy"><div><i className="sk" style={{width:"var(--size-150)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-70)",height:"var(--size-11)",marginLeft:"auto"}}/></div><i className="sk" style={{width:"100%",height:"var(--size-4)",borderRadius:"var(--radius-pill)"}}/></div></li>)}</ol></section>
      </div>
      <aside className="panel dash-now">
        <header><i className="sk" style={{width:"var(--size-110)",height:"var(--size-14)"}}/></header>
        <div className="dash-now-figure"><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-150)",height:"var(--size-24)"}}/></div>
        <div className="dash-now-areas">{Array.from({length:4},(_,index)=><div className="dash-area dashboard-skeleton-operation" key={index}><i className="sk" style={{width:"var(--size-80)",height:"var(--size-12)"}}/><i className="sk" style={{width:"100%",height:"var(--size-10)"}}/><i className="sk" style={{width:"70%",height:"var(--size-10)"}}/></div>)}</div>
        <div className="dash-review dashboard-skeleton-review"><i className="sk" style={{width:"var(--size-80)",height:"var(--size-12)"}}/><i className="sk" style={{width:"100%",height:"var(--size-10)"}}/></div>
      </aside>
    </div>
  </div>;
}
