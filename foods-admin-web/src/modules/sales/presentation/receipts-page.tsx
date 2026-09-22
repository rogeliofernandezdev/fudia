import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

export function ReceiptsPage(){
  return <><PageHeader eyebrow="VENTA" title="Comprobantes" description="La facturación electrónica se habilitará en una fase posterior al MVP operativo."/>
    <section className="panel management"><div className="catalog-state"><span><Icon name="receipt" size={24}/></span><b>Fuera del MVP actual</b><p>No se muestran comprobantes simulados. Los pedidos y pagos reales continúan disponibles en Ventas y Punto de venta.</p></div></section>
  </>;
}
