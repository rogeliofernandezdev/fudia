"use client";
import Link from "next/link";
import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

export default function Page(){
  return <><PageHeader eyebrow="NEGOCIO" title="CRM y fidelización" description="Segmentación, campañas y fidelización de clientes."/>
  <Link href="/dashboard" className="settings-back"><Icon name="chevronLeft" size={16}/>Volver al inicio</Link>
  <section className="panel management"><div className="catalog-state"><span><Icon name="settings" size={24}/></span><b>Próximamente</b><p>Este módulo está en desarrollo. Pronto estará disponible para tu empresa.</p></div></section>
  </>;
}
