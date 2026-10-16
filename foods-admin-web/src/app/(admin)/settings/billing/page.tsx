"use client";
import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

export default function FacturacionPage(){
  return <><PageHeader eyebrow="CONFIGURACIÓN" title="Facturación electrónica" description="Series, comprobantes y proveedor SUNAT."/>
  <section className="panel management"><div className="catalog-state"><span><Icon name="receipt" size={24}/></span><b>Próximamente</b><p>La configuración de facturación electrónica estará disponible pronto.</p></div></section>
  </>;
}
