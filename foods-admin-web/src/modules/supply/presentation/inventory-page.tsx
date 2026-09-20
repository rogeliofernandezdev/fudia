import {ManagementPage} from "@/shared/management";

export function InventoryPage(){
  return <ManagementPage eyebrow="ABASTECIMIENTO" title="Inventario" description="Controla existencias, lotes, movimientos, conteos y mermas por local." action="Registrar movimiento" columns={["INSUMO","ALMACÉN","EXISTENCIA"]} rows={[
    {name:"Lomo fino",detail:"INS-001 · kg",category:"Almacén principal",value:"2.4 kg",state:"Crítico",tone:"orange"},
    {name:"Papa amarilla",detail:"INS-018 · kg",category:"Almacén principal",value:"18.6 kg",state:"Saludable"},
    {name:"Limón sutil",detail:"INS-024 · kg",category:"Cocina",value:"5.8 kg",state:"Bajo",tone:"orange"},
    {name:"Arroz extra",detail:"INS-031 · kg",category:"Almacén principal",value:"42 kg",state:"Saludable"},
    {name:"Vino tinto",detail:"INS-048 · botella",category:"Bar",value:"16 und",state:"Saludable"},
  ]}/>;
}
