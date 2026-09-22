"use client";
import "./dashboard.css";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {Icon} from "@/design-system/icons";
import {PageHeader,Status} from "@/design-system/page-header";
import {useSession} from "@/providers/session-context";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {getDashboard} from "../infrastructure/dashboard-api";

function greeting(){const h=new Date().getHours();if(h<12)return"Buenos días";if(h<19)return"Buenas tardes";return"Buenas noches"}
function formatDate(){const d=new Date();const days=["Domingo","Lunes","Martes","Miércoles","Jueves","Viernes","Sábado"];const months=["enero","febrero","marzo","abril","mayo","junio","julio","agosto","septiembre","octubre","noviembre","diciembre"];return `${days[d.getDay()]}, ${d.getDate()} de ${months[d.getMonth()]}`}

export function DashboardView(){
  const{user,location}=useSession();
  const settings=useSettings();
  const query=useQuery({queryKey:["dashboard"],queryFn:getDashboard,refetchInterval:30000});
  const data=query.data;
  const money=(value:string|number)=>{
    const n=Number(value||0);
    const formatted=formatRegionalNumber(Math.abs(n),location?.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
    const sign=n<0?"-":"";
    return settings.currencyPosition==="before"?sign+settings.currencySymbol+" "+formatted:sign+formatted+" "+settings.currencySymbol;
  };
  const kpis=data?[
    {label:"VENTAS NETAS",value:money(data.salesNet),note:"Cobros netos de hoy",icon:"sales" as const,tone:"green"},
    {label:"PEDIDOS COBRADOS",value:String(data.paidOrders),note:`Ticket promedio ${money(data.averageTicket)}`,icon:"receipt" as const,tone:"blue"},
    {label:"PEDIDOS ABIERTOS",value:String(data.openOrders),note:data.kitchenPending?`${data.kitchenPending} en cola de cocina`:"Sin cola pendiente",icon:"kitchen" as const,tone:"violet"},
    {label:"RESERVAS HOY",value:String(data.reservationsToday),note:"Pendientes o confirmadas",icon:"clock" as const,tone:"blue"},
  ]:[];
  const max=Math.max(1,...(data?.hourlySales??[]).map(item=>Number(item.total)||0));
  const alertItems=data?[
    {count:data.criticalStock,title:"Stock crítico",detail:"Insumos en mínimo o agotados",href:"/inventario",tone:"orange"},
    {count:data.purchasesToApprove,title:"Compras por aprobar",detail:"Órdenes pendientes de aprobación",href:"/compras",tone:"blue"},
    {count:data.kitchenPending,title:"Comandas activas",detail:"Confirmadas o en preparación",href:"/cocina",tone:"green"},
  ].filter(item=>item.count>0):[];
  if(query.isLoading)return <><PageHeader eyebrow="CONTROL DEL NEGOCIO" title="Cargando operación…" description="Calculando ventas y actividad del local."/><section className="panel catalog-state"><b>Cargando datos reales…</b></section></>;
  if(query.isError)return <><PageHeader eyebrow="CONTROL DEL NEGOCIO" title="No pudimos cargar el panel" description="Los datos no se reemplazan por valores simulados."/><section className="panel catalog-state error"><span><Icon name="alert"/></span><b>Error de lectura</b><p>{query.error.message}</p><button className="button secondary" onClick={()=>void query.refetch()}>Reintentar</button></section></>;
  return <>
    <PageHeader eyebrow="CONTROL DEL NEGOCIO" title={`${greeting()}, ${(user?.name??"Admin").split(" ")[0]}`} description={`Actividad real de ${location?.name??"tu local"}.`} action={<Link href="/ventas" className="button secondary"><Icon name="receipt" size={17}/><span>Ver ventas</span></Link>}/>
    <section className="period"><div><button className="active">Hoy</button></div><span><Icon name="clock" size={16}/> {formatDate()}</span></section>
    <section className="kpi-grid">{kpis.map(k=><article key={k.label}><span className={k.tone}><Icon name={k.icon}/></span><div><small>{k.label}</small><strong>{k.value}</strong><em>{k.note}</em></div></article>)}</section>
    <section className="dashboard-grid">
      <article className="panel chart-panel"><header><div><small>RENDIMIENTO</small><h2>Ventas cobradas por hora</h2></div></header>{data?.hourlySales.length?<div className="chart"><div className="bars">{data.hourlySales.map(item=><i key={item.hour} title={`${item.hour}:00 · ${money(item.total)}`} style={{height:`${Math.max(8,(Number(item.total)/max)*100)}%`}}><b/></i>)}</div><div className="x">{data.hourlySales.map(item=><span key={item.hour}>{String(item.hour).padStart(2,"0")}h</span>)}</div></div>:<div className="catalog-state"><b>Aún no hay ventas cobradas hoy</b><p>El gráfico aparecerá con el primer cobro.</p></div>}</article>
      <article className="panel alerts"><header><div><small>REQUIERE ATENCIÓN</small><h2>Alertas operativas</h2></div><b>{alertItems.length}</b></header>{alertItems.length?alertItems.map(a=><Link key={a.title} href={a.href}><span className={a.tone}><Icon name="alert"/></span><b>{a.title}<small>{a.count} · {a.detail}</small></b><Icon name="chevron" size={16}/></Link>):<div className="catalog-state"><span><Icon name="check"/></span><b>Sin alertas pendientes</b><p>La operación no tiene incidencias activas en este momento.</p></div>}</article>
    </section>
    <section className="panel top-products"><header><div><small>DESEMPEÑO DEL MENÚ</small><h2>Productos cobrados hoy</h2></div><Link href="/productos">Ver carta</Link></header>{data?.topProducts.length?data.topProducts.map((p,i)=><div className="top-product" key={p.name}><span className={`rank r${i+1}`}>{i+1}</span><b>{p.name}</b><span className="qty">{Number(p.qty)} unidades</span><Status tone="green">{money(p.revenue)}</Status></div>):<div className="catalog-state"><b>Sin productos vendidos todavía</b><p>Los productos aparecerán después de registrar cobros.</p></div>}</section>
  </>;
}
