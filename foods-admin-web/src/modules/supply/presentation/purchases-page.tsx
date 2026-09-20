import {ManagementPage} from "@/shared/management";

export function PurchasesPage(){
  return <ManagementPage eyebrow="ABASTECIMIENTO" title="Compras" description="Planifica órdenes, aprobaciones y recepciones de proveedores." action="Nueva orden" columns={["ORDEN","PROVEEDOR","TOTAL"]} rows={[
    {name:"OC-00128",detail:"8 productos · Hoy",category:"Mercado Central SAC",value:"S/ 1,450.00",state:"Por aprobar",tone:"orange"},
    {name:"OC-00127",detail:"12 productos · Ayer",category:"Distribuidora Andina",value:"S/ 2,860.40",state:"En tránsito",tone:"blue"},
    {name:"OC-00126",detail:"5 productos · 08 sep",category:"Frutas del Valle",value:"S/ 680.00",state:"Recibida"},
    {name:"OC-00125",detail:"3 productos · 07 sep",category:"Bebidas del Perú",value:"S/ 920.00",state:"Recibida"},
  ]}/>;
}
