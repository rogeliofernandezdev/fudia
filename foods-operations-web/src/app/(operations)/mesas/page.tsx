"use client";
import { ActionLink, Button, Input, Label } from "@/components/ui/controls";


import { useState } from "react";
import { Icon } from "@/components/icon";

type Tone = "free" | "occupied" | "cooking" | "reserved" | "pay";

type TableInfo = {
  n: string;
  seats: number;
  tone: Tone;
  state: string;
  meta: string;
  waiter?: string;
  order?: string;
  total?: string;
  time?: string;
  lines?: { qty: number; name: string; price: string; note?: string }[];
};

const tables: TableInfo[] = [
  { n: "01", seats: 2, tone: "free", state: "Libre", meta: "Disponible" },
  { n: "02", seats: 4, tone: "occupied", state: "Ocupada", meta: "S/ 68.00", waiter: "Marco Díaz", order: "#0122", total: "S/ 68.00", lines: [{ qty: 2, name: "Arroz con pollo", price: "S/ 44.00" }, { qty: 2, name: "Chicha morada", price: "S/ 16.00" }] },
  { n: "03", seats: 4, tone: "free", state: "Libre", meta: "Disponible" },
  { n: "04", seats: 4, tone: "cooking", state: "En cocina", meta: "18 min", waiter: "Lucía Ramos", order: "#0128", total: "S/ 44.00", time: "18 min", lines: [{ qty: 1, name: "Lomo saltado", price: "S/ 28.00", note: "Sin cebolla · Término 3/4" }, { qty: 2, name: "Chicha morada", price: "S/ 16.00", note: "Sin hielo" }] },
  { n: "05", seats: 6, tone: "reserved", state: "Reservada", meta: "8:00 p. m.", waiter: "—", order: "—", total: "—", time: "8:00 p. m." },
  { n: "06", seats: 2, tone: "free", state: "Libre", meta: "Disponible" },
  { n: "07", seats: 4, tone: "pay", state: "Por cobrar", meta: "S/ 124.50", waiter: "Ana Torres", order: "#0124", total: "S/ 124.50", time: "32 min", lines: [{ qty: 3, name: "Ají de gallina", price: "S/ 72.00" }, { qty: 2, name: "Inca Kola", price: "S/ 12.00" }, { qty: 1, name: "Suspiro limeño", price: "S/ 12.00" }] },
  { n: "08", seats: 4, tone: "occupied", state: "Ocupada", meta: "S/ 42.00", waiter: "Marco Díaz", order: "#0131", total: "S/ 42.00", lines: [{ qty: 1, name: "Ají de gallina", price: "S/ 24.00" }, { qty: 1, name: "Arroz con pollo", price: "S/ 22.00" }] },
  { n: "09", seats: 6, tone: "free", state: "Libre", meta: "Disponible" },
  { n: "10", seats: 2, tone: "occupied", state: "Ocupada", meta: "S/ 79.90", waiter: "Lucía Ramos", order: "#0135", total: "S/ 79.90", lines: [{ qty: 1, name: "Ceviche clásico", price: "S/ 30.00" }, { qty: 1, name: "Lomo saltado", price: "S/ 28.00" }] },
  { n: "11", seats: 4, tone: "pay", state: "Por cobrar", meta: "S/ 86.00", waiter: "Ana Torres", order: "#0126", total: "S/ 86.00", time: "18 min", lines: [{ qty: 2, name: "Ceviche mixto", price: "S/ 62.00" }, { qty: 2, name: "Inca Kola", price: "S/ 12.00" }] },
  { n: "12", seats: 4, tone: "free", state: "Libre", meta: "Disponible" },
];

const filters = [
  { id: "all", label: "Todas", match: () => true },
  { id: "free", label: "Libres", match: (t: TableInfo) => t.tone === "free" },
  { id: "occupied", label: "Ocupadas", match: (t: TableInfo) => t.tone === "occupied" || t.tone === "cooking" || t.tone === "pay" },
  { id: "attention", label: "Atención", match: (t: TableInfo) => t.tone === "cooking" || t.tone === "pay" || t.tone === "reserved" },
] as const;

const stateText: Record<Tone, string> = { free: "MESA LIBRE", occupied: "MESA OCUPADA", cooking: "EN COCINA", reserved: "RESERVADA", pay: "POR COBRAR" };

export default function TablesPage() {
  const [filter, setFilter] = useState("all");
  const [selectedN, setSelectedN] = useState("04");
  const [query, setQuery] = useState("");

  const normalizedQuery = query.trim().toLowerCase();
  const visible = tables.filter(t => filters.find(f => f.id === filter)!.match(t) && (!normalizedQuery || t.n.includes(normalizedQuery) || t.waiter?.toLowerCase().includes(normalizedQuery)));
  const selected = tables.find(t => t.n === selectedN) ?? tables[0];
  const freeCount = tables.filter(t => t.tone === "free").length;
  const occupiedCount = tables.filter(t => t.tone === "occupied" || t.tone === "cooking" || t.tone === "pay").length;
  const attentionCount = tables.filter(t => t.tone === "cooking" || t.tone === "pay" || t.tone === "reserved").length;

  return <div className="floor-workspace">
    <section className="floor-main">
      <header className="floor-header"><div><span>SALÓN PRINCIPAL</span><h1>Mesas y servicio</h1><p>Estado del salón actualizado en tiempo real</p></div><Button tone="primary" className="open-table"><Icon name="plus" size={19}/>Abrir mesa</Button></header>
      <div className="floor-toolbar">
        <div className="floor-tabs">
          <Button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>Todas <b>{tables.length}</b></Button>
          <Button className={filter === "free" ? "active" : ""} onClick={() => setFilter("free")}>Libres <b>{freeCount}</b></Button>
          <Button className={filter === "occupied" ? "active" : ""} onClick={() => setFilter("occupied")}>Ocupadas <b>{occupiedCount}</b></Button>
          <Button className={filter === "attention" ? "active" : ""} onClick={() => setFilter("attention")}>Atención <b>{attentionCount}</b></Button>
        </div>
        <Label className="floor-search"><Icon name="search" size={17}/><Input aria-label="Buscar mesa" placeholder="Mesa o responsable" value={query} onChange={event => setQuery(event.target.value)}/>{query && <Button layout="icon" aria-label="Limpiar búsqueda" onClick={() => setQuery("")}><Icon name="close" size={15}/></Button>}</Label>
      </div>
      <div className="floor-section-title"><div><h2>Distribución del salón</h2><span>12 mesas · 46 asientos</span></div><Button><Icon name="tables" size={17}/>Vista de salón</Button></div>
      <div className="floor-plan">{visible.map(table => <Button layout="card" aria-pressed={table.n === selectedN} className={"floor-table " + table.tone + (table.n === selectedN ? " selected" : "")} key={table.n} onClick={() => setSelectedN(table.n)}>
        <span className="table-top"><small>MESA</small><i className="state-dot"/><em>{table.state}</em></span>
        <strong>{table.n}</strong>
        <span className="seat-row">{Array.from({ length: Math.min(table.seats, 4) }).map((_, i) => <i key={i}/>)}<small>{table.seats} personas</small></span>
        <span className="table-foot"><b>{table.meta}</b><Icon name="chevron" size={16}/></span>
      </Button>)}{visible.length === 0 && <div className="floor-empty"><Icon name="search"/><b>No encontramos mesas</b><small>Prueba con otro número o responsable.</small></div>}</div>
      <footer className="floor-legend"><span><i className="free"/>Libre <b>{freeCount}</b></span><span><i className="occupied"/>Ocupada <b>{tables.filter(t => t.tone === "occupied").length}</b></span><span><i className="cooking"/>En cocina <b>{tables.filter(t => t.tone === "cooking").length}</b></span><span><i className="pay"/>Por cobrar <b>{tables.filter(t => t.tone === "pay").length}</b></span><span><i className="reserved"/>Reservada <b>{tables.filter(t => t.tone === "reserved").length}</b></span></footer>
    </section>
    <aside className="table-detail">
      <header><div><span>{stateText[selected.tone]}</span><h2>Mesa {selected.n}</h2><p>{selected.tone === "free" ? "Disponible para asignar" : selected.tone === "reserved" ? "Reserva programada" : `Servicio iniciado hace ${selected.time ?? "—"}`}</p></div><Button><Icon name="menu" size={19}/></Button></header>
      {selected.tone !== "free" && selected.tone !== "reserved" && <div className="detail-state"><span><i/>{selected.state}</span><b>{selected.time ?? ""}</b></div>}
      <dl>
        <div><dt>Comensales</dt><dd>{selected.seats} personas</dd></div>
        <div><dt>Responsable</dt><dd>{selected.waiter ?? "Por asignar"}</dd></div>
        {selected.order && <div><dt>Pedido</dt><dd>{selected.order}</dd></div>}
      </dl>
      {selected.lines && <div className="detail-order">{selected.lines.map(line => <div key={line.name}><div><span><b>{line.qty}×</b>{line.name}</span><strong>{line.price}</strong></div>{line.note && <small>{line.note}</small>}</div>)}</div>}
      {selected.total && <div className="detail-total"><span>Total actual</span><strong>{selected.total}</strong></div>}
      <div className="detail-actions">
        {selected.tone === "free" && <ActionLink tone="primary" href="/pos" className="detail-primary"><Icon name="plus" size={18}/>Abrir mesa y tomar pedido</ActionLink>}
        {selected.tone === "reserved" && <ActionLink tone="primary" href="/pos" className="detail-primary"><Icon name="check" size={18}/>Asignar mesa y atender</ActionLink>}
        {(selected.tone === "occupied" || selected.tone === "cooking") && <><ActionLink tone="primary" href="/pos" className="detail-primary"><Icon name="pos" size={18}/>Ver comanda<Icon name="chevron" size={17}/></ActionLink><Button><Icon name="printer" size={18}/>Imprimir pre cuenta</Button></>}
        {selected.tone === "pay" && <><ActionLink tone="primary" href="/pos/pago" className="detail-primary"><Icon name="cash" size={18}/>Cobrar {selected.total}<Icon name="chevron" size={17}/></ActionLink><Button><Icon name="printer" size={18}/>Imprimir pre cuenta</Button></>}
      </div>
    </aside>
  </div>;
}
