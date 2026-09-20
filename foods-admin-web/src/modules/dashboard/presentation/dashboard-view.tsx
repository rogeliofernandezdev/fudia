"use client";
import {Icon} from "@/design-system/icons";
import {Button, PageHeader, Status} from "@/design-system/page-header";
import {useSession} from "@/providers/session-context";

const kpis = [
  {label: "VENTAS NETAS", value: "S/ 12,840.50", note: "+12.5% vs. ayer", icon: "sales", tone: "green"},
  {label: "PEDIDOS", value: "186", note: "Ticket medio S/ 69.03", icon: "receipt", tone: "blue"},
  {label: "COSTO DE VENTA", value: "31.8%", note: "Meta menor a 33%", icon: "stock", tone: "violet"},
  {label: "MARGEN ESTIMADO", value: "S/ 8,757.20", note: "68.2% de las ventas", icon: "grid", tone: "green"},
] as const;

function greeting() {
  const h = new Date().getHours();
  if (h < 12) return "Buenos días";
  if (h < 19) return "Buenas tardes";
  return "Buenas noches";
}

function formatDate() {
  const d = new Date();
  const days = ["Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"];
  const months = ["enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"];
  return `${days[d.getDay()]}, ${d.getDate()} de ${months[d.getMonth()]}`;
}

export function DashboardView() {
  const {user, location} = useSession();
  const name = user?.name ?? "Admin";
  const locName = location?.name ?? "el local";
  return (
    <>
      <PageHeader eyebrow="CONTROL DEL NEGOCIO" title={`${greeting()}, ${name.split(" ")[0]}`} description={`Lo importante de ${locName}, actualizado hace un minuto.`} action={<Button icon="download" kind="secondary">Descargar reporte</Button>} />
      <section className="period"><div><button className="active">Hoy</button><button>7 días</button><button>30 días</button></div><span><Icon name="clock" size={16} /> {formatDate()}</span></section>
      <section className="kpi-grid">{kpis.map(k => <article key={k.label}><span className={k.tone}><Icon name={k.icon} /></span><div><small>{k.label}</small><strong>{k.value}</strong><em>{k.note}</em></div></article>)}</section>
      <section className="dashboard-grid">
        <article className="panel chart-panel"><header><div><small>RENDIMIENTO</small><h2>Ventas por hora</h2></div><select aria-label="Canal"><option>Todos los canales</option><option>Salón</option><option>Delivery</option></select></header><div className="chart"><div className="y"><span>S/ 3k</span><span>S/ 2k</span><span>S/ 1k</span><span>S/ 0</span></div><div className="bars">{[25, 38, 32, 58, 78, 65, 88, 70, 52, 42, 28].map((v, i) => <i key={i} style={{height: `${v}%`}}><b /></i>)}</div><div className="x"><span>10 am</span><span>12 pm</span><span>2 pm</span><span>4 pm</span><span>6 pm</span><span>8 pm</span></div></div></article>
        <article className="panel alerts"><header><div><small>REQUIERE ATENCIÓN</small><h2>Alertas operativas</h2></div><b>4</b></header>{[{t: "Stock crítico", d: "4 insumos necesitan reposición", tone: "orange"}, {t: "Comprobantes pendientes", d: "2 documentos esperan respuesta", tone: "blue"}, {t: "Compra por aprobar", d: "OC-00128 · S/ 1,450.00", tone: "green"}].map(a => <button key={a.t}><span className={a.tone}><Icon name="alert" /></span><b>{a.t}<small>{a.d}</small></b><Icon name="chevron" size={16} /></button>)}<a>Ver todas las alertas</a></article>
      </section>
      <section className="panel top-products"><header><div><small>DESEMPEÑO DEL MENÚ</small><h2>Productos más vendidos</h2></div><a>Ver todos los productos</a></header>{[{n: "Lomo saltado", q: "32", v: "S/ 1,280.00"}, {n: "Pollo a la brasa", q: "28", v: "S/ 1,064.00"}, {n: "Ceviche mixto", q: "19", v: "S/ 855.00"}, {n: "Ají de gallina", q: "15", v: "S/ 525.00"}].map((p, i) => <div className="top-product" key={p.n}><span className={`rank r${i + 1}`}>{i + 1}</span><b>{p.n}</b><span className="qty">{p.q} pedidos</span><Status tone="green">{p.v}</Status></div>)}</section>
    </>
  );
}
