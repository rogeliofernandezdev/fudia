import {PageHeader} from "@/design-system/page-header";
import {EmptyState} from "@/design-system/states";

export function ReceiptsPage(){
  return <><PageHeader title="Comprobantes" description="La facturación electrónica se habilitará en una fase posterior al MVP operativo."/>
    <section className="panel management"><EmptyState icon="receipt" title="Fuera del MVP actual" text="No se muestran comprobantes simulados. Los pedidos y pagos reales continúan disponibles en Ventas y Punto de venta."/></section>
  </>;
}
