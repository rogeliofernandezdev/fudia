"use client";
import "./dashboard.css";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {Icon,type IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useSession} from "@/providers/session-context";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {pageRoutes} from "@/shared/routing/page-routes";
import {canOpenNavigationItem,navigationItemForPath} from "@/shell/navigation";
import {getDashboard} from "../infrastructure/dashboard-api";

function formatDate(day:string){return new Intl.DateTimeFormat("es",{day:"numeric",month:"long",weekday:"long",timeZone:"UTC"}).format(new Date(day+"T12:00:00Z"))}

type Kpi={label:string;value:string;note:string;icon:IconName;tone:"primary"|"blue"|"green"|"violet"};
type AlertItem={count:number;title:string;detail:string;href:string;tone:"warning"|"info"|"ok";icon:IconName};
type OperationPanel={title:string;icon:IconName;href:string;rows:{label:string;value:string;tone?:"primary"|"green"}[]};

export function DashboardView(){
  const session=useSession();
  const{user,location,organization,modules}=session;
  const enabled=(module:string)=>Boolean(user?.platformAdmin||modules?.[module]);
  const canOpen=(href:string)=>{const item=navigationItemForPath(href);return Boolean(item&&user&&canOpenNavigationItem(item,{...session,user,modules:modules??{}}))};
  const settings=useSettings();
  const query=useQuery({queryKey:["dashboard",organization?.id,location?.id],queryFn:getDashboard,refetchInterval:30000});
  const data=query.data;
  const money=(value:string|number)=>{
    const n=Number(value||0);
    const formatted=formatRegionalNumber(Math.abs(n),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=n<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  };
  const compact=(value:string|number)=>{
    const n=Number(value||0);
    return formatRegionalNumber(n,location?.country,{maximumFractionDigits:0});
  };
  const firstName=(user?.name??"Admin").split(" ")[0];
  const header=<PageHeader eyebrow="CONTROL DEL NEGOCIO" title={`Hola, ${firstName}`} description={`${data?.businessDate?formatDate(data.businessDate)+" · ":""}${location?.name??"tu local"} · se actualiza cada 30 s`} action={canOpen(pageRoutes.sales)?<Link href={pageRoutes.sales} className="button secondary"><Icon name="sales" size={17}/><span>Ver ventas</span></Link>:undefined}/>;

  if(query.isLoading)return <>{header}<DashboardSkeleton/></>;
  if(query.isError)return <><PageHeader eyebrow="CONTROL DEL NEGOCIO" title="No pudimos cargar el panel" description="Los datos no se reemplazan por valores simulados."/><section className="panel catalog-state error"><span><Icon name="alert"/></span><b>Error de lectura</b><p>{query.error.message}</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  if(!data?.operations)return <>{header}<section className="panel catalog-state error"><Icon name="alert"/><b>Resumen no disponible</b><p>No pudimos leer el estado del restaurante. Inténtalo nuevamente.</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  const operations=data.operations;

  const kpis:Kpi[]=[
    {label:"Ventas del día",value:money(data.salesNet),note:"Cobrado hoy",icon:"sales",tone:"primary"},
    {label:"Pedidos cobrados",value:String(data.paidOrders),note:`Promedio por pedido ${money(data.averageTicket)}`,icon:"orders",tone:"green"},
    {label:"Saldo por cobrar",value:money(operations.pendingBalance),note:`${operations.unpaidOrders+operations.partialOrders} pedidos pendientes · ${operations.partialOrders} con pago parcial`,icon:"payment",tone:"violet"},
    {label:"Pedidos en atención",value:String(data.openOrders),note:data.kitchenPending?`${data.kitchenPending} en cocina`:"Sin pedidos en cocina",icon:"orders",tone:"blue"},
  ];
  const panels:OperationPanel[]=[];
  if(enabled("pedidos")||enabled("mesas")||enabled("reservas"))panels.push({title:"Atención",icon:"tables",href:pageRoutes.diningRoom,rows:[
    ...(enabled("mesas")?[{label:"Mesas ocupadas",value:`${operations.tablesOccupied} de ${operations.tablesTotal}`}]:[]),
    ...(enabled("reservas")?[{label:"Reservas de hoy",value:String(data.reservationsToday)}]:[]),
    {label:"Sin pagar",value:String(operations.unpaidOrders)},
  ]});
  if(enabled("cocina"))panels.push({title:"Cocina",icon:"kitchen",href:pageRoutes.kitchen,rows:[
    {label:"Por preparar",value:String(operations.kitchenConfirmed)},
    {label:"En preparación",value:String(operations.kitchenPreparing)},
    {label:"Listos para entregar",value:String(operations.readyOrders),tone:"green"},
  ]});
  if(enabled("delivery"))panels.push({title:"Delivery",icon:"truck",href:pageRoutes.delivery,rows:[
    {label:"Por despachar",value:String(operations.deliveryPending)},
    {label:"En camino",value:String(operations.deliveryInTransit)},
  ]});
  if(enabled("caja"))panels.push({title:"Caja",icon:"cash",href:pageRoutes.cash,rows:[
    {label:"Turnos abiertos",value:`${operations.openCashShifts} de ${operations.activeCashRegisters}`},
    {label:"Efectivo en caja",value:operations.cashBalance===null?"Importe reservado":money(operations.cashBalance),tone:"primary"},
  ]});
  const hourly=data.hourlySales.map(item=>({hour:item.hour,total:Number(item.total)||0}));
  const max=Math.max(1,...hourly.map(item=>item.total));
  const peak=hourly.reduce<{hour:number;total:number}|null>((best,item)=>!best||item.total>best.total?item:best,null);
  const alerts=([
    {count:data.criticalStock,title:"Stock crítico",detail:"Insumos en mínimo o agotados",href:pageRoutes.inventory,tone:"warning",icon:"stock"},
    {count:data.purchasesToApprove,title:"Compras por aprobar",detail:"Órdenes pendientes de aprobación",href:pageRoutes.purchases,tone:"info",icon:"cart"},
    {count:operations.soldOutProducts,title:"Productos agotados",detail:"Sin disponibilidad para vender hoy",href:pageRoutes.availability,tone:"warning",icon:"utensils"},
  ] satisfies AlertItem[]).filter(item=>item.count>0&&enabled(navigationItemForPath(item.href)?.module??""));
  const topMax=Math.max(1,...data.topProducts.map(p=>Number(p.revenue)||0));

  return <>
    {header}
    <section className="kpi-grid" aria-label="Indicadores de hoy">
      {kpis.map(k=><article key={k.label} className={"kpi "+k.tone}>
        <span className="kpi-icon"><Icon name={k.icon} size={20}/></span>
        <div><small>{k.label}</small><strong>{k.value}</strong><em>{k.note}</em></div>
      </article>)}
    </section>

    {panels.length>0&&<section className="dashboard-operations" aria-label="Estado actual del restaurante">
      {panels.map(panel=><article className="panel dashboard-operation" key={panel.title}>
        <header><span className="panel-icon"><Icon name={panel.icon} size={18}/></span><h2>{panel.title}</h2>{canOpen(panel.href)&&<Link href={panel.href} className="dashboard-operation-link" aria-label={`Ver ${panel.title.toLowerCase()}`} title={`Ver ${panel.title.toLowerCase()}`}><Icon name="chevron" size={18}/></Link>}</header>
        <dl>{panel.rows.map(row=><div key={row.label}><dt>{row.label}</dt><dd className={row.tone??""}>{row.value}</dd></div>)}</dl>
      </article>)}
    </section>}

    <section className="dashboard-grid">
      <article className="panel chart-panel">
        <header><span className="panel-icon"><Icon name="sales" size={18}/></span><div><small>RENDIMIENTO</small><h2>Ventas cobradas por hora</h2></div>{peak&&peak.total>0&&<span className="chart-peak">Pico {String(peak.hour).padStart(2,"0")}:00 · {money(peak.total)}</span>}</header>
        {hourly.length?<div className="chart">
          <div className="chart-y" aria-hidden="true"><span>{compact(max)}</span><span>{compact(max/2)}</span><span>0</span></div>
          <div className="chart-plot">
            <div className="bars" role="img" aria-label="Ventas cobradas por hora">
              {hourly.map(item=><div className={"bar"+(peak&&item.hour===peak.hour&&item.total>0?" peak":"")} key={item.hour} style={{height:`${item.total>0?Math.max(3,(item.total/max)*100):0}%`}} title={`${String(item.hour).padStart(2,"0")}:00 · ${money(item.total)}`}><i/></div>)}
            </div>
            <div className="x" aria-hidden="true">{hourly.map(item=><span key={item.hour}>{String(item.hour).padStart(2,"0")}</span>)}</div>
          </div>
        </div>:<div className="dashboard-empty"><Icon name="sales" size={20}/><b>Aún no hay ventas cobradas hoy</b><p>El gráfico aparecerá con el primer cobro.</p></div>}
      </article>

      <article className="panel alerts">
        <header><span className="panel-icon warning"><Icon name="alert" size={18}/></span><div><small>REQUIERE ATENCIÓN</small><h2>Pendientes por revisar</h2></div>{alerts.length>0&&<b>{alerts.length}</b>}</header>
        {alerts.length?<div className="alert-list">{alerts.map(a=>{const content=<><span className="alert-icon"><Icon name={a.icon} size={17}/></span><span className="alert-copy"><b>{a.title}</b><small>{a.detail}</small></span><span className="alert-count">{a.count}</span></>;return canOpen(a.href)?<Link className={"alert-row "+a.tone} key={a.title} href={a.href}>
          {content}
          <Icon name="chevron" size={16}/>
        </Link>:<div className={"alert-row "+a.tone} key={a.title}>{content}</div>})}</div>:<div className="dashboard-empty ok"><Icon name="check" size={20}/><b>Sin pendientes por revisar</b><p>Sin alertas de disponibilidad o abastecimiento.</p></div>}
      </article>
    </section>

    <section className="panel top-products">
      <header><span className="panel-icon"><Icon name="utensils" size={18}/></span><div><small>CARTA DEL RESTAURANTE</small><h2>Productos vendidos hoy</h2></div>{canOpen(pageRoutes.products)&&<Link href={pageRoutes.products} className="panel-link">Ver carta<Icon name="chevron" size={14}/></Link>}</header>
      {data.topProducts.length?<ol className="top-list">{data.topProducts.map((p,i)=>{
        const revenue=Number(p.revenue)||0;
        return <li className="top-product" key={p.name}>
          <span className="rank">{i+1}</span>
          <div className="top-copy"><div><b>{p.name}</b><span className="qty">{Number(p.qty)} {Number(p.qty)===1?"unidad":"unidades"}</span><strong>{money(revenue)}</strong></div><div className="top-track"><i style={{width:`${(revenue/topMax)*100}%`}}/></div></div>
        </li>;
      })}</ol>:<div className="dashboard-empty"><Icon name="utensils" size={20}/><b>Sin productos vendidos todavía</b><p>Los productos aparecerán después de registrar cobros.</p></div>}
    </section>
  </>;
}

function DashboardSkeleton(){
  return <div className="dashboard-skeleton" aria-label="Cargando reportes" aria-busy="true">
    <section className="kpi-grid">
      {Array.from({length:4},(_,index)=><article className="kpi dashboard-skeleton-kpi" key={index}><i className="sk" style={{width:"var(--size-40)",height:"var(--size-40)",borderRadius:"var(--radius-12)"}}/><div><i className="sk" style={{width:"var(--size-90)",height:"var(--size-10)"}}/><i className="sk" style={{width:"55%",height:"var(--size-26)"}}/><i className="sk" style={{width:"70%",height:"var(--size-10)"}}/></div></article>)}
    </section>
    <section className="dashboard-operations">
      {Array.from({length:4},(_,index)=><article className="panel dashboard-operation dashboard-skeleton-operation" key={index}><header><i className="sk" style={{width:"var(--size-32)",height:"var(--size-32)"}}/><i className="sk" style={{width:"var(--size-80)",height:"var(--size-14)"}}/></header><dl>{Array.from({length:3},(_,row)=><div key={row}><dt><i className="sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/></dt><dd><i className="sk" style={{width:"var(--size-32)",height:"var(--size-14)"}}/></dd></div>)}</dl></article>)}
    </section>
    <section className="dashboard-grid">
      <article className="panel chart-panel dashboard-skeleton-panel">
        <header><i className="sk" style={{width:"var(--size-32)",height:"var(--size-32)",borderRadius:"var(--radius-10)"}}/><div><i className="sk" style={{width:"var(--size-80)",height:"var(--size-9)"}}/><i className="sk" style={{width:"var(--size-170)",height:"var(--size-14)"}}/></div></header>
        <div className="dashboard-skeleton-chart">{[34,48,63,46,72,58,82,40,55].map((h,i)=><span className="sk" key={i} style={{height:h+"%"}}/>)}</div>
      </article>
      <article className="panel alerts dashboard-skeleton-panel">
        <header><i className="sk" style={{width:"var(--size-32)",height:"var(--size-32)",borderRadius:"var(--radius-10)"}}/><div><i className="sk" style={{width:"var(--size-100)",height:"var(--size-9)"}}/><i className="sk" style={{width:"var(--size-140)",height:"var(--size-14)"}}/></div></header>
        <div className="alert-list">{Array.from({length:3},(_,index)=><div className="alert-row dashboard-skeleton-alert" key={index}><i className="sk" style={{width:"var(--size-36)",height:"var(--size-36)",borderRadius:"var(--radius-10)"}}/><span className="alert-copy"><i className="sk" style={{width:"var(--size-110)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-150)",height:"var(--size-9)"}}/></span><i className="sk" style={{width:"var(--size-28)",height:"var(--size-22)",borderRadius:"var(--radius-99)"}}/></div>)}</div>
      </article>
    </section>
    <section className="panel top-products dashboard-skeleton-panel">
      <header><i className="sk" style={{width:"var(--size-32)",height:"var(--size-32)",borderRadius:"var(--radius-10)"}}/><div><i className="sk" style={{width:"var(--size-110)",height:"var(--size-9)"}}/><i className="sk" style={{width:"var(--size-160)",height:"var(--size-14)"}}/></div><i className="sk" style={{width:"var(--size-70)",height:"var(--size-11)"}}/></header>
      <ol className="top-list">{Array.from({length:4},(_,index)=><li className="top-product dashboard-skeleton-product" key={index}><i className="sk" style={{width:"var(--size-26)",height:"var(--size-26)",borderRadius:"var(--radius-8)"}}/><div className="top-copy"><div><i className="sk" style={{width:"var(--size-150)",height:"var(--size-11)"}}/><i className="sk" style={{width:"var(--size-70)",height:"var(--size-9)",marginLeft:"auto"}}/><i className="sk" style={{width:"var(--size-70)",height:"var(--size-11)"}}/></div><i className="sk" style={{width:"100%",height:"var(--size-4)",borderRadius:"var(--radius-99)"}}/></div></li>)}</ol>
    </section>
  </div>;
}
