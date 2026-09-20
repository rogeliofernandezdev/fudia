"use client";

import { Button } from "@/components/ui/controls";


import { Icon } from "@/components/icon";

const kpis = [
  { label: "Ventas hoy", value: "S/ 2,847.50", delta: "+12%", tone: "green", icon: "cash" as const },
  { label: "Tickets hoy", value: "47", delta: "+8", tone: "green", icon: "receipt" as const },
  { label: "Ticket promedio", value: "S/ 60.58", delta: "+3%", tone: "green", icon: "pos" as const },
  { label: "Tiempo de espera", value: "14 min", delta: "-2 min", tone: "green", icon: "clock" as const },
];

const topProducts = [
  { name: "Lomo saltado", qty: 28, revenue: "S/ 672.00", trend: "up" },
  { name: "Ceviche clásico", qty: 22, revenue: "S/ 704.00", trend: "up" },
  { name: "Ají de gallina", qty: 18, revenue: "S/ 396.00", trend: "down" },
  { name: "Arroz con pollo", qty: 15, revenue: "S/ 360.00", trend: "up" },
  { name: "Chicha morada", qty: 42, revenue: "S/ 336.00", trend: "up" },
];

const hourlyData = [
  { hour: "12p", value: 45 }, { hour: "1p", value: 78 }, { hour: "2p", value: 62 },
  { hour: "3p", value: 28 }, { hour: "4p", value: 15 }, { hour: "5p", value: 22 },
  { hour: "6p", value: 55 }, { hour: "7p", value: 85 }, { hour: "8p", value: 92 },
  { hour: "9p", value: 68 }, { hour: "10p", value: 38 }, { hour: "11p", value: 18 },
];

const channels = [
  { label: "Salón", value: "S/ 1,720", pct: 60, tone: "blue" },
  { label: "Delivery", value: "S/ 683", pct: 24, tone: "violet" },
  { label: "Recojo", value: "S/ 285", pct: 10, tone: "green" },
  { label: "WhatsApp", value: "S/ 159", pct: 6, tone: "orange" },
];

const payments = [
  { label: "Efectivo", value: "S/ 1,138", pct: 40 },
  { label: "Tarjeta", value: "S/ 996", pct: 35 },
  { label: "Yape/Plin", value: "S/ 569", pct: 20 },
  { label: "Por cobrar", value: "S/ 142", pct: 5 },
];

export default function ReportsPage() {
  const maxHour = Math.max(...hourlyData.map(d => d.value));
  return <div className="reports-page">
    <header className="reports-header">
      <div>
        <span className="reports-eyebrow">ANÁLISIS</span>
        <h1>Reportes</h1>
        <p>Ventas, productos y rendimiento del día.</p>
      </div>
      <div className="reports-period">
        <Button className="active">Hoy</Button>
        <Button>Semana</Button>
        <Button>Mes</Button>
      </div>
    </header>

    <div className="reports-kpis">
      {kpis.map(k => <article className="reports-kpi" key={k.label}>
        <div className={"reports-kpi-icon " + k.tone}><Icon name={k.icon} size={20}/></div>
        <div className="reports-kpi-info">
          <small>{k.label}</small>
          <strong>{k.value}</strong>
          <span className={"reports-kpi-delta " + k.tone}>{k.delta}</span>
        </div>
      </article>)}
    </div>

    <div className="reports-grid">
      <section className="reports-card reports-chart">
        <header><h2>Ventas por hora</h2><small>Tráfico del día</small></header>
        <div className="reports-bars">
          {hourlyData.map(d => <div className="reports-bar" key={d.hour}>
            <div className="reports-bar-fill" style={{ height: `${(d.value / maxHour) * 100}%` }}/>
            <small>{d.hour}</small>
          </div>)}
        </div>
      </section>

      <section className="reports-card reports-top">
        <header><h2>Productos top</h2><small>Más vendidos hoy</small></header>
        <div className="reports-top-list">
          {topProducts.map((p, i) => <div className="reports-top-item" key={p.name}>
            <b className="reports-top-rank">{i + 1}</b>
            <div className="reports-top-info">
              <strong>{p.name}</strong>
              <small>{p.qty} pedidos · {p.revenue}</small>
            </div>
            <span className={"reports-top-trend " + p.trend}>{p.trend === "up" ? "↑" : "↓"}</span>
          </div>)}
        </div>
      </section>
    </div>

    <div className="reports-grid">
      <section className="reports-card">
        <header><h2>Por canal</h2><small>Distribución de ventas</small></header>
        <div className="reports-breakdown">
          {channels.map(c => <div className="reports-breakdown-row" key={c.label}>
            <div className="reports-breakdown-label">
              <i className={"reports-dot " + c.tone}/>
              <span>{c.label}</span>
            </div>
            <div className="reports-breakdown-bar"><i className={"reports-bar-fill-h " + c.tone} style={{ width: `${c.pct}%` }}/></div>
            <b>{c.value}</b>
          </div>)}
        </div>
      </section>

      <section className="reports-card">
        <header><h2>Medios de pago</h2><small>Cómo pagan los clientes</small></header>
        <div className="reports-breakdown">
          {payments.map(p => <div className="reports-breakdown-row" key={p.label}>
            <div className="reports-breakdown-label">
              <i className="reports-dot blue"/>
              <span>{p.label}</span>
            </div>
            <div className="reports-breakdown-bar"><i className="reports-bar-fill-h blue" style={{ width: `${p.pct}%` }}/></div>
            <b>{p.value}</b>
          </div>)}
        </div>
      </section>
    </div>
  </div>;
}
