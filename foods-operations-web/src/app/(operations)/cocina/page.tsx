"use client";

import { Button, Label, Select } from "@/components/ui/controls";


import { useEffect, useState } from "react";
import { Icon } from "@/components/icon";

type TicketState = "pending" | "cooking" | "ready";

type Ticket = {
  id: string;
  table: string;
  channel: "Mesa" | "Delivery" | "Mostrador" | "Recojo";
  state: TicketState;
  startedMin: number;
  targetMin: number;
  lines: { name: string; qty: number; note?: string }[];
};

const initialTickets: Ticket[] = [
  { id: "#0128", table: "Mesa 04", channel: "Mesa", state: "pending", startedMin: 18, targetMin: 22, lines: [{ name: "Lomo saltado", qty: 1, note: "Sin cebolla · Término 3/4" }, { name: "Chicha morada", qty: 2, note: "Sin hielo" }] },
  { id: "#0131", table: "Mesa 08", channel: "Mesa", state: "pending", startedMin: 11, targetMin: 20, lines: [{ name: "Ají de gallina", qty: 2 }, { name: "Arroz con pollo", qty: 1 }] },
  { id: "#0133", table: "Delivery", channel: "Delivery", state: "pending", startedMin: 6, targetMin: 18, lines: [{ name: "Ceviche clásico", qty: 1, note: "Sin ají" }, { name: "Inca Kola", qty: 2 }] },
  { id: "#0134", table: "Mesa 02", channel: "Mesa", state: "pending", startedMin: 3, targetMin: 18, lines: [{ name: "Papa huancaína", qty: 2 }, { name: "Lomo saltado", qty: 1, note: "Bien cocido" }] },
  { id: "#0130", table: "Mesa 09", channel: "Mesa", state: "cooking", startedMin: 9, targetMin: 20, lines: [{ name: "Arroz con mariscos", qty: 1, note: "Sin culantro" }] },
  { id: "#0132", table: "Recojo #31", channel: "Recojo", state: "cooking", startedMin: 4, targetMin: 16, lines: [{ name: "Ceviche mixto", qty: 2 }] },
  { id: "#0129", table: "Mostrador", channel: "Mostrador", state: "ready", startedMin: 16, targetMin: 15, lines: [{ name: "Papa a la huancaína", qty: 2 }] },
  { id: "#0127", table: "Mesa 01", channel: "Mesa", state: "ready", startedMin: 21, targetMin: 15, lines: [{ name: "Chicha morada", qty: 3 }] },
];

const channelIcon = { Mesa: "tables", Delivery: "bike", Mostrador: "store", Recojo: "store" } as const;

const columns: { id: TicketState; label: string; mobileLabel: string; icon: "clock" | "cookingPot" | "check"; tone: "operational" | "primary" | "neutral" }[] = [
  { id: "pending", label: "Por preparar", mobileLabel: "Pendientes", icon: "clock", tone: "operational" },
  { id: "cooking", label: "En preparación", mobileLabel: "Preparando", icon: "cookingPot", tone: "primary" },
  { id: "ready", label: "Listos para entregar", mobileLabel: "Listos", icon: "check", tone: "neutral" },
];

export default function KitchenPage() {
  const [tickets, setTickets] = useState(initialTickets);
  const [tick, setTick] = useState(0);
  const [activeLane, setActiveLane] = useState<TicketState>("pending");

  useEffect(() => {
    const id = setInterval(() => setTick(t => t + 1), 10000);
    return () => clearInterval(id);
  }, []);

  const advance = (id: string) => {
    setTickets(prev => prev.map(t => {
      if (t.id !== id) return t;
      if (t.state === "pending") return { ...t, state: "cooking" as const, startedMin: 0 };
      if (t.state === "cooking") return { ...t, state: "ready" as const };
      return t;
    }));
  };

  const dismiss = (id: string) => setTickets(prev => prev.filter(t => t.id !== id));

  const getElapsed = (t: Ticket) => t.state === "ready" ? t.startedMin : Math.min(t.targetMin, t.startedMin + Math.round(tick / 6));
  const getProgress = (t: Ticket) => Math.min(100, Math.round(((t.startedMin + (t.state === "cooking" ? tick : 0) / 60) / t.targetMin) * 100));
  const getTone = (t: Ticket) => {
    if (t.state === "ready") return "ready";
    const late = (t.startedMin + (t.state === "cooking" ? tick : 0) / 60) >= t.targetMin;
    if (late) return "late";
    if (getProgress(t) >= 70) return "warning";
    return "fresh";
  };

  const pendingCount = tickets.filter(ticket => ticket.state === "pending").length;
  const cookingCount = tickets.filter(ticket => ticket.state === "cooking").length;
  const readyCount = tickets.filter(ticket => ticket.state === "ready").length;
  const attentionCount = tickets.filter(ticket => getTone(ticket) === "late" || getTone(ticket) === "warning").length;

  return <div className="kds-page">
    <header className="kds-header">
      <div>
        <span className="kds-eyebrow">OPERACIÓN EN COCINA</span>
        <h1>Gestión de comandas</h1>
        <p>Prioriza, prepara y entrega pedidos en tiempo real.</p>
      </div>
      <div className="kds-live"><i/>En vivo</div>
    </header>

    <section className="kds-overview" aria-label="Resumen de comandas">
      <div><span className="pending"><Icon name="clock" size={18}/></span><p><small>POR PREPARAR</small><strong>{pendingCount}</strong></p></div>
      <div><span className="cooking"><Icon name="cookingPot" size={18}/></span><p><small>EN PREPARACIÓN</small><strong>{cookingCount}</strong></p></div>
      <div><span className="ready"><Icon name="check" size={18}/></span><p><small>LISTOS</small><strong>{readyCount}</strong></p></div>
      <div><span className="attention"><Icon name="clock" size={18}/></span><p><small>REQUIEREN ATENCIÓN</small><strong>{attentionCount}</strong></p></div>
    </section>

    <div className="kds-controls"><div><span>COLA DE PRODUCCIÓN</span><h2>Comandas por estado</h2></div><Label>Estación<Select aria-label="Estación de cocina" defaultValue="all"><option value="all">Todas las estaciones</option><option value="hot">Cocina caliente</option><option value="cold">Cocina fría</option><option value="bar">Barra</option></Select></Label></div>

    <nav className="kds-mobile-tabs" aria-label="Estados de comandas">{columns.map(column => {
      const count = tickets.filter(ticket => ticket.state === column.id).length;
      return <Button key={column.id} aria-pressed={activeLane === column.id} className={activeLane === column.id ? "active" : ""} onClick={() => setActiveLane(column.id)}><Icon name={column.icon} size={16}/><span className="kds-tab-full">{column.label}</span><span className="kds-tab-short">{column.mobileLabel}</span><b>{count}</b></Button>;
    })}</nav>

    <div className="kds-board">
      {columns.map(col => {
        const colTickets = tickets.filter(t => t.state === col.id).sort((left, right) => getProgress(right) - getProgress(left));
        return <section data-mobile-active={activeLane === col.id} className={"kds-col kds-col-" + col.id} key={col.id}>
          <header className="kds-col-head">
            <span className="kds-col-icon"><Icon name={col.icon} size={18}/></span>
            <h2>{col.label}</h2>
            <b>{colTickets.length}</b>
          </header>
          <div className="kds-col-list">
            {colTickets.map(ticket => {
              const tone = getTone(ticket);
              const elapsed = getElapsed(ticket);
              const progress = getProgress(ticket);
              return <article className={"kds-card " + tone} key={ticket.id}>
                <div className="kds-card-head">
                  <div className="kds-card-who">
                    <strong>{ticket.table}</strong>
                    <small><Icon name={channelIcon[ticket.channel]} size={11}/>{ticket.channel} · {ticket.id}</small>
                  </div>
                  <div className={"kds-time " + tone}>
                    <b>{elapsed}</b><span>min</span>
                  </div>
                </div>
                <div className="kds-card-bar"><i style={{ width: `${progress}%` }}/></div>
                <div className="kds-card-items">
                  {ticket.lines.map(line => <div className="kds-item" key={line.name}>
                    <b>{line.qty}×</b>
                    <div>
                      <span>{line.name}</span>
                      {line.note && <em>{line.note}</em>}
                    </div>
                  </div>)}
                </div>
                <footer className="kds-card-footer">
                  <Button tone={col.tone} className="kds-btn wide" onClick={() => col.id === "ready" ? dismiss(ticket.id) : advance(ticket.id)}>
                    {col.id === "pending" && <><Icon name="cookingPot" size={16}/>Iniciar</>}
                    {col.id === "cooking" && <><Icon name="check" size={16}/>Marcar listo</>}
                    {col.id === "ready" && <><Icon name="check" size={16}/>Entregado</>}
                  </Button>
                </footer>
              </article>;
            })}
            {colTickets.length === 0 && <div className="kds-col-empty">Sin tickets</div>}
          </div>
        </section>;
      })}
    </div>
  </div>;
}
