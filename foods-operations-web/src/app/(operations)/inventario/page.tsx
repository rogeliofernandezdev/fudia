"use client";

import {useEffect,useMemo,useState} from "react";
import {Button,Input,Table} from "@/components/ui/controls";
import {Icon} from "@/components/icon";
import {operationsFetch} from "@/lib/operations-api";

type InventoryItem={
  inventoryItemId:string;
  sku:string;
  name:string;
  kind:"product"|"ingredient";
  unit:string;
  quantity:string;
  minimumStock:string;
  reorderPoint:string;
  optimalStock:string;
  averageUnitCost:string;
  stockValue:string;
  status:"ok"|"low"|"out";
};
type InventoryResponse={items:InventoryItem[];total:number;page:number;pageSize:number};

export default function InventoryPage(){
  const[items,setItems]=useState<InventoryItem[]>([]);
  const[search,setSearch]=useState("");
  const[loading,setLoading]=useState(true);
  const[error,setError]=useState("");

  useEffect(()=>{
    let active=true;
    void operationsFetch<InventoryResponse>("inventory?page=1&pageSize=100").then(data=>{
      if(active)setItems(data.items);
    }).catch(e=>{
      if(active)setError(e instanceof Error?e.message:"No se pudo cargar Inventario.");
    }).finally(()=>{if(active)setLoading(false)});
    return()=>{active=false};
  },[]);

  const visible=useMemo(()=>{
    const q=search.trim().toLowerCase();
    return q?items.filter(i=>i.name.toLowerCase().includes(q)||i.sku.toLowerCase().includes(q)):items;
  },[items,search]);
  const low=items.filter(i=>i.status!=="ok");
  const totalValue=items.reduce((sum,i)=>sum+Number(i.stockValue||0),0);

  if(loading)return <div className="inv-page"><div className="inv-empty"><b>Cargando inventario…</b><span>Consultando existencias del local activo.</span></div></div>;

  return <div className="inv-page">
    <header className="inv-header">
      <div><span className="inv-eyebrow">ABASTECIMIENTO</span><h1>Inventario</h1><p>{items.length} artículos · {low.length} requieren atención</p></div>
      <Button className="inv-new" onClick={()=>window.location.reload()}><Icon name="refresh" size={18}/>Actualizar</Button>
    </header>
    {error&&<div className="inv-alert"><Icon name="bell" size={18}/><div><strong>No se pudo cargar Inventario</strong><span>{error}</span></div></div>}
    <div className="inv-stats">
      <div className="inv-stat"><b>{items.length}</b><small>Total artículos</small></div>
      <div className="inv-stat warn"><b>{low.length}</b><small>Stock bajo / agotado</small></div>
      <div className="inv-stat"><b>{totalValue.toFixed(2)}</b><small>Valor de stock</small></div>
    </div>
    {low.length>0&&<div className="inv-alert"><Icon name="bell" size={18}/><div><strong>{low.length} artículos necesitan atención</strong><span>{low.slice(0,8).map(i=>i.name).join(", ")}{low.length>8?"…":""}</span></div></div>}
    <div className="inv-filters"><div className="inv-search"><Icon name="search" size={16}/><Input placeholder="Buscar artículo..." value={search} onChange={e=>setSearch(e.target.value)}/></div></div>
    <Table caption="Existencias reales del local">
      <thead><tr><th>Artículo</th><th>Tipo</th><th>Stock</th><th>Mínimo / Reorden</th><th>Costo prom.</th><th>Valor</th><th>Estado</th></tr></thead>
      <tbody>{visible.map(i=><tr key={i.inventoryItemId}>
        <td><strong>{i.name}</strong><small>{i.sku}</small></td>
        <td>{i.kind==="ingredient"?"Insumo":"Producto"}</td>
        <td><b>{Number(i.quantity).toFixed(3)} {i.unit}</b></td>
        <td>{Number(i.minimumStock).toFixed(3)} / {Number(i.reorderPoint).toFixed(3)}</td>
        <td>{Number(i.averageUnitCost).toFixed(4)}</td>
        <td>{Number(i.stockValue).toFixed(2)}</td>
        <td><span className={"inv-status "+(i.status==="ok"?"ok":"low")}>{i.status==="ok"?"OK":i.status==="out"?"Sin stock":"Reponer"}</span></td>
      </tr>)}
      {!visible.length&&<tr><td colSpan={7}><div className="inv-empty"><Icon name="search" size={24}/><b>Sin resultados</b><span>No se encontraron artículos.</span></div></td></tr>}
      </tbody>
    </Table>
  </div>;
}
