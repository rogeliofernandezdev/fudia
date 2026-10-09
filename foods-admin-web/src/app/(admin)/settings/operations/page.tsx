"use client";
import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

export default function OperacionPage(){
  return <><PageHeader eyebrow="CONFIGURACIÓN" title="Mesas, cajas y estaciones" description="Estructura operativa de cada local."/>
  <section className="panel management"><div className="catalog-state"><span><Icon name="menu" size={24}/></span><b>Próximamente</b><p>La configuración de mesas, cajas y estaciones estará disponible pronto.</p></div></section>
  </>;
}
