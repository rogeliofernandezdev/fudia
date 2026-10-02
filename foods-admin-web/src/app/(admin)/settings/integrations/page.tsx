"use client";
import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

export default function IntegracionesPage(){
  return <><PageHeader eyebrow="CONFIGURACIÓN" title="Integraciones" description="WhatsApp, pagos, impresión y notificaciones."/>
  <section className="panel management"><div className="catalog-state"><span><Icon name="settings" size={24}/></span><b>Próximamente</b><p>La configuración de integraciones estará disponible pronto.</p></div></section>
  </>;
}
