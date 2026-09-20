"use client";

import { Button } from "@/components/ui/controls";


import { Icon } from "@/components/icon";

type StaffMember = {
  id: string;
  name: string;
  role: "Cocina" | "Salón" | "Caja" | "Supervisor";
  phone: string;
  shift: string;
  status: "Activo" | "En turno" | "Descanso" | "Ausente";
  ordersToday: number;
  rating: string;
};

const staff: StaffMember[] = [
  { id: "E-001", name: "Rosa Quispe", role: "Cocina", phone: "+51 987 111 222", shift: "10:00 - 18:00", status: "En turno", ordersToday: 24, rating: "4.9" },
  { id: "E-002", name: "Miguel Flores", role: "Cocina", phone: "+51 945 333 444", shift: "16:00 - 23:00", status: "En turno", ordersToday: 12, rating: "4.7" },
  { id: "E-003", name: "Ana Vargas", role: "Salón", phone: "+51 988 555 666", shift: "11:00 - 19:00", status: "En turno", ordersToday: 18, rating: "4.8" },
  { id: "E-004", name: "Luis Ramírez", role: "Salón", phone: "+51 966 777 888", shift: "17:00 - 23:00", status: "Descanso", ordersToday: 8, rating: "4.5" },
  { id: "E-005", name: "Carmen Soto", role: "Caja", phone: "+51 977 999 000", shift: "10:00 - 18:00", status: "En turno", ordersToday: 47, rating: "4.9" },
  { id: "E-006", name: "Diego Torres", role: "Supervisor", phone: "+51 955 111 333", shift: "10:00 - 22:00", status: "En turno", ordersToday: 0, rating: "5.0" },
  { id: "E-007", name: "Patricia Díaz", role: "Salón", phone: "+51 944 222 555", shift: "12:00 - 20:00", status: "Ausente", ordersToday: 0, rating: "4.3" },
];

const statusTone: Record<StaffMember["status"], string> = {
  "En turno": "green",
  "Activo": "blue",
  "Descanso": "violet",
  "Ausente": "muted",
};

const roleIcon = { Cocina: "kitchen", Salón: "tables", Caja: "cash", Supervisor: "user" } as const;

export default function StaffPage() {
  const onShift = staff.filter(s => s.status === "En turno").length;
  const onBreak = staff.filter(s => s.status === "Descanso").length;
  const absent = staff.filter(s => s.status === "Ausente").length;
  const totalOrders = staff.reduce((s, m) => s + m.ordersToday, 0);

  return <div className="staff-page">
    <header className="staff-header">
      <div>
        <span className="staff-eyebrow">EQUIPO</span>
        <h1>Personal</h1>
        <p>{staff.length} miembros · {onShift} en turno · {onBreak} en descanso · {absent} ausentes</p>
      </div>
      <Button tone="primary" className="staff-new"><Icon name="plus" size={18}/>Nuevo miembro</Button>
    </header>

    <div className="staff-stats">
      <div className="staff-stat"><b>{onShift}</b><small>En turno</small></div>
      <div className="staff-stat"><b>{totalOrders}</b><small>Atenciones hoy</small></div>
      <div className="staff-stat"><b>{staff.length}</b><small>Total equipo</small></div>
    </div>

    <div className="staff-list">
      {staff.map(m => <article className="staff-card" key={m.id}>
        <div className="staff-avatar">{m.name.split(" ").map(w => w[0]).join("").slice(0, 2)}</div>
        <div className="staff-info">
          <div className="staff-top">
            <strong>{m.name}</strong>
            <span className={"staff-status " + statusTone[m.status]}>{m.status}</span>
          </div>
          <div className="staff-meta">
            <span className="staff-role"><Icon name={roleIcon[m.role]} size={12}/>{m.role}</span>
            <span><Icon name="clock" size={12}/>{m.shift}</span>
            <span><Icon name="whatsapp" size={12}/>{m.phone}</span>
          </div>
        </div>
        <div className="staff-metrics">
          <div className="staff-metric"><b>{m.ordersToday}</b><small>Pedidos</small></div>
          <div className="staff-metric"><b>★ {m.rating}</b><small>Rating</small></div>
        </div>
      </article>)}
    </div>
  </div>;
}
