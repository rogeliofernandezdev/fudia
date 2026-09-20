import { ActionLink, Input, Label, Select } from "@/components/ui/controls";
import Link from "next/link";
import { Icon } from "@/components/icon";
import { Logo } from "@/components/logo";
export default function ShiftPage() {
  return <main className="setup-page"><header><Logo/><Link href="/login"><Icon name="logout"/> Salir</Link></header>
    <section className="setup-card"><div className="setup-card-accent" aria-hidden="true"/><div className="setup-progress"><i className="done"><Icon name="check"/></i><span/><i className="active">2</i><span/><i>3</i></div>
      <div className="setup-intro"><div className="setup-icon"><Icon name="store" size={23}/></div><div><span className="kicker">PREPARA TU JORNADA</span><h1>Abre tu turno</h1><p>Confirma dónde trabajarás y el fondo inicial de caja.</p></div></div>
      <div className="setup-selection-grid"><Label htmlFor="shift-location">Local<div className="shift-field"><Icon name="store"/><Select className="shift-control" id="shift-location" defaultValue="current"><option value="current">Sabor Criollo · Miraflores</option></Select></div></Label>
      <Label htmlFor="shift-register">Caja<div className="shift-field"><Icon name="cash"/><Select className="shift-control" id="shift-register" defaultValue="current"><option value="current">Caja principal</option></Select></div></Label></div>
      <div className="opening-grid"><Label htmlFor="opening-balance">Fondo inicial<div className="shift-field shift-money"><span>S/</span><Input className="shift-control" id="opening-balance" inputMode="decimal" defaultValue="200.00"/></div></Label><Label htmlFor="shift-owner">Responsable<div className="shift-field"><Icon name="user"/><Input className="shift-control" id="shift-owner" value="Rogelio Fernández" readOnly/></div></Label></div>
      <div className="shift-note"><Icon name="check" size={15}/><span><b>Arqueo anterior cuadra</b><small>Cierre de ayer: S/ 812.00 sin diferencias.</small></span></div>
      <div className="setup-action"><ActionLink tone="primary" className="wide" href="/inicio">Abrir turno y comenzar <Icon name="fingerprint"/></ActionLink><p className="setup-help"><Icon name="clock" size={16}/> El turno registrará ventas desde este momento.</p></div>
    </section>
  </main>;
}
