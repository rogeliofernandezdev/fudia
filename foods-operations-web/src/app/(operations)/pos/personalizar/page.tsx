"use client";
import { ActionLink } from "@/components/ui/controls";


import { Input, Textarea, Label } from "@/components/ui/controls";


import { useState } from "react";
import Link from "next/link";
import { Icon } from "@/components/icon";

const groups = [
  { id: "cocción", title: "Término de cocción", required: true, single: true, options: ["Término medio", "Tres cuartos", "Bien cocido"] },
  { id: "guarnición", title: "Guarnición", required: true, single: true, options: ["Arroz blanco", "Papas fritas", "Ensalada fresca"] },
  { id: "retirar", title: "Retirar ingredientes", required: false, single: false, options: ["Sin cebolla", "Sin tomate", "Sin culantro"] },
];

export default function CustomizePage() {
  const [selected, setSelected] = useState<Record<string, string[]>>({ cocción: ["Tres cuartos"], guarnición: ["Arroz blanco"] });
  const [note, setNote] = useState("");

  const toggle = (groupId: string, option: string, single: boolean) => {
    setSelected(prev => {
      const current = prev[groupId] ?? [];
      if (single) return { ...prev, [groupId]: [option] };
      return { ...prev, [groupId]: current.includes(option) ? current.filter(x => x !== option) : [...current, option] };
    });
  };

  const selectedCount = Object.values(selected).reduce((sum, list) => sum + list.length, 0);

  return <div className="flow-page"><header className="flow-header"><Link href="/pos"><Icon name="chevron" size={18}/>Volver al pedido</Link><div><span>PERSONALIZAR PRODUCTO</span><h1>Lomo saltado</h1><p>Selecciona opciones e indicaciones para cocina.</p></div></header>
    <div className="customize-layout">
      <section className="customize-product"><div className="custom-photo lomo"/><div><span>FONDOS</span><h2>Lomo saltado</h2><p>Trozos de lomo, cebolla, tomate, papas fritas y arroz.</p><strong>S/ 28.00</strong></div></section>
      <section className="option-panel">
        {groups.map(group => <fieldset key={group.id}><legend>{group.title}{group.required && <b>Obligatorio</b>}</legend><div className="option-grid">{group.options.map(option => {
          const checked = (selected[group.id] ?? []).includes(option);
          return <Label key={option}><Input type={group.single ? "radio" : "checkbox"} name={group.title} checked={checked} onChange={() => toggle(group.id, option, group.single)}/><span>{option}</span></Label>;
        })}</div></fieldset>)}
        <Label className="instruction-field"><span>Indicaciones para cocina</span><Textarea placeholder="Ej. Servir la salsa aparte..." value={note} onChange={e => setNote(e.target.value)}/></Label>
        <footer><div><span>Total{selectedCount > 0 ? ` · ${selectedCount} opción${selectedCount > 1 ? "es" : ""}` : ""}</span><strong>S/ 28.00</strong></div><ActionLink tone="primary" href="/pos"><Icon name="check" size={18}/>Guardar cambios</ActionLink></footer>
      </section>
    </div></div>;
}
