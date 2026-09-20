import { ActionLink } from "@/components/ui/controls";
import Link from "next/link";
import { Icon } from "@/components/icon";
import { PageHeading } from "@/components/app-shell";

const quick = [
  { href: "/mesas", name: "Mesas", detail: "5 disponibles · 2 por cobrar", icon: "tables", tone: "green" },
  { href: "/pos", name: "Punto de venta", detail: "Nueva venta en segundos", icon: "pos", tone: "blue" },
  { href: "/cocina", name: "Comandas", detail: "3 por preparar", icon: "kitchen", tone: "indigo" },
  { href: "/pedidos", name: "WhatsApp", detail: "2 nuevos pedidos", icon: "whatsapp", tone: "violet" },
] as const;

const activity = [
  { id: "04", label: "Mesa 04", person: "Lucía", state: "En cocina", tone: 0, total: "S/ 86.00", href: "/pos" },
  { id: "11", label: "Mesa 11", person: "Marco", state: "Por cobrar", tone: 1, total: "S/ 124.50", href: "/pos/pago" },
  { id: "W", label: "WhatsApp #18", person: "Canal digital", state: "Confirmado", tone: 2, total: "S/ 49.90", href: "/pedidos" },
];

export default function HomePage() {
  return <div className="home-page">
    <PageHeading eyebrow="LUNES, 31 DE AGOSTO" title="Buenas tardes, Rogelio" description="Resumen operativo de Sabor Criollo · Miraflores." action={<ActionLink tone="primary" href="/pos"><Icon name="plus"/> Nueva venta</ActionLink>}/>
    <div className="home-section-head"><div><span>ACCESOS RÁPIDOS</span><h2>Continúa con la operación</h2></div><small><i/> Turno activo · 04 h 32 min</small></div>
    <section className="quick-grid">{quick.map(x => <Link className={"quick " + x.tone} href={x.href} key={x.name}><span><Icon name={x.icon}/></span><div><b>{x.name}</b><small>{x.detail}</small></div><span className="quick-arrow"><Icon name="chevron" size={16}/></span></Link>)}</section>
    <section className="dashboard-grid">
      <article className="panel"><header><div><span className="section-kicker">SERVICIO EN CURSO</span><h2>Mesas y pedidos activos</h2></div><div className="panel-head-actions"><em>3 activos</em><Link href="/mesas">Ver todos</Link></div></header>
        <div className="activity-list">{activity.map((r, i) => <Link className="activity" href={r.href} key={r.label}><i className={i === 2 ? "digital" : ""}>{r.id}</i><span><b>{r.label}</b><small>{r.person}</small></span><em className={"state s" + r.tone}>{r.state}</em><strong>{r.total}</strong><Icon name="chevron"/></Link>)}</div>
      </article>
      <article className="panel shift-summary"><div className="summary-title"><i><Icon name="cash"/></i><span><small>CAJA PRINCIPAL</small><h2>Turno abierto</h2></span><em>EN LÍNEA</em></div><div className="shift-amount"><small>VENTAS DEL TURNO</small><strong>S/ 1,284.50</strong></div><dl><div><dt>Pedidos atendidos</dt><dd>27</dd></div><div><dt>Ticket promedio</dt><dd>S/ 47.57</dd></div><div><dt>Inicio</dt><dd>10:00 a. m.</dd></div></dl><Link className="soft-button" href="/caja">Ver detalle de caja <Icon name="chevron"/></Link></article>
    </section>
    <Link className="alert-banner" href="/cocina"><span><Icon name="clock"/></span><div><small>ATENCIÓN EN COMANDAS</small><b>1 comanda supera el tiempo esperado</b><p>Mesa 04 lleva 18 minutos en preparación.</p></div><strong>Ver comandas <Icon name="chevron"/></strong></Link>
  </div>;
}
