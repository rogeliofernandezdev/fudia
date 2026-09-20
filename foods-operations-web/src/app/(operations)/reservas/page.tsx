"use client";

import { Button } from "@/components/ui/controls";


import { useState } from "react";
import { Icon } from "@/components/icon";

type Reservation = {
  id: string;
  client: string;
  phone: string;
  date: string;
  time: string;
  guests: number;
  table: string;
  state: "Pendiente" | "Confirmada" | "Sentada" | "Cancelada";
  note?: string;
};

const initialReservations: Reservation[] = [
  { id: "R-001", client: "Familia García", phone: "+51 987 123 456", date: "Hoy", time: "12:30", guests: 4, table: "Mesa 06", state: "Confirmada", note: "Ventana" },
  { id: "R-002", client: "Carlos Mendoza", phone: "+51 945 789 123", date: "Hoy", time: "13:00", guests: 2, table: "Mesa 03", state: "Pendiente" },
  { id: "R-003", client: "Cumpleaños - Lucía", phone: "+51 988 456 789", date: "Hoy", time: "19:30", guests: 8, table: "Mesa 10", state: "Confirmada", note: "Pastel incluido" },
  { id: "R-004", client: "Empresa Tech SAC", phone: "+51 955 321 654", date: "Hoy", time: "20:00", guests: 12, table: "Salón privado", state: "Confirmada", note: "Cuenta corporativa" },
  { id: "R-005", client: "Ana Quiroz", phone: "+51 977 654 321", date: "Mañana", time: "13:30", guests: 3, table: "Mesa 05", state: "Pendiente" },
  { id: "R-006", client: "Pareja Romero", phone: "+51 966 111 222", date: "Mañana", time: "20:30", guests: 2, table: "Mesa 02", state: "Confirmada", note: "Aniversario" },
];

const stateTone: Record<Reservation["state"], string> = {
  Pendiente: "violet",
  Confirmada: "blue",
  Sentada: "green",
  Cancelada: "muted",
};

const stateAction: Record<Reservation["state"], string | null> = {
  Pendiente: "Confirmar",
  Confirmada: "Sentar",
  Sentada: null,
  Cancelada: null,
};

const nextState: Record<Reservation["state"], Reservation["state"] | null> = {
  Pendiente: "Confirmada",
  Confirmada: "Sentada",
  Sentada: null,
  Cancelada: null,
};

export default function ReservationsPage() {
  const [reservations, setReservations] = useState(initialReservations);
  const [filter, setFilter] = useState<"Hoy" | "Mañana" | "Todas">("Hoy");

  const visible = filter === "Todas" ? reservations : reservations.filter(r => r.date === filter);
  const todayCount = reservations.filter(r => r.date === "Hoy").length;
  const pendingCount = reservations.filter(r => r.state === "Pendiente").length;
  const totalGuests = reservations.filter(r => r.date === "Hoy").reduce((s, r) => s + r.guests, 0);

  const advance = (id: string) => {
    setReservations(prev => prev.map(r => {
      if (r.id !== id) return r;
      const next = nextState[r.state];
      return next ? { ...r, state: next } : r;
    }));
  };

  return <div className="res-page">
    <header className="res-header">
      <div>
        <span className="res-eyebrow">SALÓN</span>
        <h1>Reservas</h1>
        <p>{todayCount} reservas hoy · {totalGuests} comensales · {pendingCount} pendientes</p>
      </div>
      <Button tone="primary" className="res-new"><Icon name="plus" size={18}/>Nueva reserva</Button>
    </header>

    <div className="res-filters">
      <Button className={filter === "Hoy" ? "active" : ""} onClick={() => setFilter("Hoy")}>Hoy</Button>
      <Button className={filter === "Mañana" ? "active" : ""} onClick={() => setFilter("Mañana")}>Mañana</Button>
      <Button className={filter === "Todas" ? "active" : ""} onClick={() => setFilter("Todas")}>Todas</Button>
    </div>

    <div className="res-list">
      {visible.map(r => <article className="res-card" key={r.id}>
        <div className="res-card-time">
          <strong>{r.time}</strong>
          <small>{r.date}</small>
        </div>
        <div className="res-card-info">
          <div className="res-card-top">
            <strong>{r.client}</strong>
            <span className={"res-state " + stateTone[r.state]}>{r.state}</span>
          </div>
          <div className="res-card-meta">
            <span><Icon name="user" size={12}/>{r.guests} personas</span>
            <span><Icon name="tables" size={12}/>{r.table}</span>
            <span><Icon name="whatsapp" size={12}/>{r.phone}</span>
          </div>
          {r.note && <div className="res-card-note"><Icon name="bell" size={12}/>{r.note}</div>}
        </div>
        <div className="res-card-action">
          {stateAction[r.state] && <Button tone="operational" className="res-advance" onClick={() => advance(r.id)}>{stateAction[r.state]}<Icon name="chevron" size={14}/></Button>}
          {r.state === "Sentada" && <span className="res-seated"><Icon name="check" size={14}/>Sentada</span>}
          {r.state === "Cancelada" && <span className="res-cancelled">Cancelada</span>}
        </div>
      </article>)}
      {visible.length === 0 && <div className="res-empty"><Icon name="tables" size={28}/><b>Sin reservas</b><span>No hay reservas para este día.</span></div>}
    </div>
  </div>;
}
