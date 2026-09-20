import { ActionLink } from "@/components/ui/controls";
import { Button } from "@/components/ui/controls";
import { Icon } from "@/components/icon";
import { PageHeading } from "@/components/app-shell";

const pendingAccounts = [
  { table: "Mesa 07", total: "S/ 124.50", time: "32 min", urgent: true },
  { table: "Mesa 11", total: "S/ 86.00", time: "18 min", urgent: false },
  { table: "Recojo #32", total: "S/ 38.50", time: "5 min", urgent: false },
];

export default function CashPage() {
  return <>
    <PageHeading eyebrow="CAJA PRINCIPAL" title="Caja y cobros" description="Turno abierto desde las 10:00 a. m." action={<ActionLink tone="operational" href="/pos" className="button primary blue"><Icon name="plus"/>Nueva venta</ActionLink>}/>
    <section className="cash-kpis">
      <article><span className="blue"><Icon name="wallet"/></span><div><small>VENTAS DEL TURNO</small><b>S/ 1,284.50</b><em>27 operaciones · +12% vs. ayer</em></div></article>
      <article><span className="green"><Icon name="cash"/></span><div><small>EFECTIVO</small><b>S/ 624.50</b><em>14 operaciones · 48.6%</em></div></article>
      <article><span className="violet"><Icon name="card"/></span><div><small>TARJETAS Y YAPE</small><b>S/ 660.00</b><em>13 operaciones · 51.4%</em></div></article>
    </section>
    <section className="dashboard-grid cash-grid">
      <article className="panel">
        <header><div><span className="section-kicker">PENDIENTES</span><h2>Cuentas por cobrar</h2></div><b className="count">{pendingAccounts.length}</b></header>
        <div className="pay-list">{pendingAccounts.map(a => <div key={a.table}><span><b>{a.table}</b><small>{a.time} de atención</small></span><strong>{a.total}</strong><ActionLink tone="operational" href="/pos/pago" className="button primary blue">Cobrar</ActionLink></div>)}</div>
        <footer className="pay-note"><Icon name="clock" size={14}/>Las cuentas con más de 25 minutos aparecen marcadas.</footer>
      </article>
      <article className="panel close-card">
        <span className="close-icon"><Icon name="cash"/></span>
        <h2>Cierre de turno</h2>
        <p>Revisa movimientos y realiza el arqueo al finalizar la jornada.</p>
        <dl>
          <div><dt>Fondo inicial</dt><dd>S/ 200.00</dd></div>
          <div><dt>Ingresos en efectivo</dt><dd>S/ 624.50</dd></div>
          <div><dt>Gastos del turno</dt><dd>-S/ 45.00</dd></div>
          <div className="expected"><dt>Efectivo esperado</dt><dd>S/ 779.50</dd></div>
        </dl>
        <Button className="button outline wide">Iniciar cierre de caja</Button>
      </article>
    </section>
  </>;
}
