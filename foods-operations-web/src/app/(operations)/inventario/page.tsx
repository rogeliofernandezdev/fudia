"use client";

import { Button, Input, Table } from "@/components/ui/controls";


import { useState } from "react";
import { Icon } from "@/components/icon";

type Supply = {
  id: string;
  name: string;
  unit: string;
  stock: number;
  min: number;
  cost: string;
  category: "Carnes" | "Verduras" | "Bebidas" | "Secos" | "Lácteos";
};

const initialSupplies: Supply[] = [
  { id: "S-001", name: "Lomo de res", unit: "kg", stock: 8, min: 5, cost: "S/ 32.00", category: "Carnes" },
  { id: "S-002", name: "Pescado fresco", unit: "kg", stock: 3, min: 6, cost: "S/ 45.00", category: "Carnes" },
  { id: "S-003", name: "Pollo entero", unit: "kg", stock: 12, min: 8, cost: "S/ 14.00", category: "Carnes" },
  { id: "S-004", name: "Cebolla roja", unit: "kg", stock: 15, min: 10, cost: "S/ 3.50", category: "Verduras" },
  { id: "S-005", name: "Ají amarillo", unit: "kg", stock: 2, min: 4, cost: "S/ 12.00", category: "Verduras" },
  { id: "S-006", name: "Papa amarilla", unit: "kg", stock: 20, min: 15, cost: "S/ 4.00", category: "Verduras" },
  { id: "S-007", name: "Arroz", unit: "kg", stock: 25, min: 20, cost: "S/ 5.50", category: "Secos" },
  { id: "S-008", name: "Inca Kola 1.5L", unit: "unid", stock: 18, min: 12, cost: "S/ 7.00", category: "Bebidas" },
  { id: "S-009", name: "Chicha morada", unit: "L", stock: 6, min: 10, cost: "S/ 8.00", category: "Bebidas" },
  { id: "S-010", name: "Leche evaporada", unit: "unid", stock: 24, min: 12, cost: "S/ 4.50", category: "Lácteos" },
];

const categories = ["Todos", "Carnes", "Verduras", "Bebidas", "Secos", "Lácteos"] as const;

export default function InventoryPage() {
  const [supplies] = useState(initialSupplies);
  const [category, setCategory] = useState<string>("Todos");
  const [search, setSearch] = useState("");

  const visible = supplies.filter(s => (category === "Todos" || s.category === category) && s.name.toLowerCase().includes(search.toLowerCase()));
  const lowStock = supplies.filter(s => s.stock <= s.min);
  const totalValue = supplies.reduce((sum, s) => sum + parseFloat(s.cost.replace("S/ ", "")) * s.stock, 0);

  return <div className="inv-page">
    <header className="inv-header">
      <div>
        <span className="inv-eyebrow">ABASTECIMIENTO</span>
        <h1>Inventario</h1>
        <p>{supplies.length} insumos · {lowStock.length} con stock bajo</p>
      </div>
      <Button tone="primary" className="inv-new"><Icon name="plus" size={18}/>Nuevo insumo</Button>
    </header>

    <div className="inv-stats">
      <div className="inv-stat"><b>{supplies.length}</b><small>Total insumos</small></div>
      <div className="inv-stat warn"><b>{lowStock.length}</b><small>Stock bajo</small></div>
      <div className="inv-stat"><b>S/ {totalValue.toFixed(2)}</b><small>Valor en stock</small></div>
    </div>

    {lowStock.length > 0 && <div className="inv-alert">
      <Icon name="bell" size={18}/>
      <div><strong>{lowStock.length} insumos necesitan reposición</strong><span>{lowStock.map(s => s.name).join(", ")}</span></div>
    </div>}

    <div className="inv-filters">
      <div className="inv-search"><Icon name="search" size={16}/><Input placeholder="Buscar insumo..." value={search} onChange={e => setSearch(e.target.value)}/></div>
      <div className="inv-cats">
        {categories.map(cat => <Button key={cat} className={category === cat ? "active" : ""} onClick={() => setCategory(cat)}>{cat}</Button>)}
      </div>
    </div>

    <Table caption="Existencias del local">
      <thead><tr>
        <th scope="col">Insumo</th><th scope="col">Categoría</th><th scope="col">Stock</th><th scope="col">Mínimo</th><th scope="col">Costo</th><th scope="col">Estado</th>
      </tr></thead><tbody>
      {visible.map(s => {
        const low = s.stock <= s.min;
        return <tr key={s.id}>
          <td><strong>{s.name}</strong><small>{s.id}</small></td>
          <td>{s.category}</td>
          <td><b>{s.stock} {s.unit}</b></td>
          <td>{s.min} {s.unit}</td>
          <td>{s.cost}</td>
          <td><span className={"inv-status " + (low ? "low" : "ok")}>{low ? "Reponer" : "OK"}</span></td>
        </tr>;
      })}
      {visible.length === 0 && <tr><td colSpan={6}><div className="inv-empty"><Icon name="search" size={24}/><b>Sin resultados</b><span>No se encontraron insumos.</span></div></td></tr>}
      </tbody>
    </Table>
  </div>;
}
