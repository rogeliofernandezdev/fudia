"use client";

import { Button, Input } from "@/components/ui/controls";


import { useState } from "react";
import { Icon } from "@/components/icon";

type Customer = {
  id: string;
  name: string;
  phone: string;
  email: string;
  orders: number;
  totalSpent: string;
  lastVisit: string;
  channel: "Salón" | "Delivery" | "Recojo" | "WhatsApp";
  tag?: "VIP" | "Frecuente" | "Nuevo";
};

const initialCustomers: Customer[] = [
  { id: "C-001", name: "Juan Pérez", phone: "+51 987 654 321", email: "juan.perez@email.com", orders: 47, totalSpent: "S/ 2,840", lastVisit: "Hace 2 días", channel: "Salón", tag: "VIP" },
  { id: "C-002", name: "Andrea Ruiz", phone: "+51 912 345 678", email: "andrea.ruiz@email.com", orders: 28, totalSpent: "S/ 1,920", lastVisit: "Hace 5 días", channel: "Delivery", tag: "Frecuente" },
  { id: "C-003", name: "Carlos Díaz", phone: "+51 998 887 766", email: "carlos.diaz@email.com", orders: 15, totalSpent: "S/ 890", lastVisit: "Hace 1 semana", channel: "Recojo", tag: "Frecuente" },
  { id: "C-004", name: "María Torres", phone: "+51 955 443 322", email: "maria.torres@email.com", orders: 3, totalSpent: "S/ 180", lastVisit: "Hace 3 días", channel: "WhatsApp", tag: "Nuevo" },
  { id: "C-005", name: "Pedro Salas", phone: "+51 977 665 544", email: "pedro.salas@email.com", orders: 32, totalSpent: "S/ 2,100", lastVisit: "Ayer", channel: "Delivery", tag: "VIP" },
  { id: "C-006", name: "Lucía Ortiz", phone: "+51 988 776 655", email: "lucia.ortiz@email.com", orders: 8, totalSpent: "S/ 520", lastVisit: "Hace 4 días", channel: "Salón", tag: "Nuevo" },
];

const tagTone: Record<string, string> = { VIP: "violet", Frecuente: "blue", Nuevo: "green" };

export default function CustomersPage() {
  const [customers] = useState(initialCustomers);
  const [search, setSearch] = useState("");
  const [tagFilter, setTagFilter] = useState<string>("Todos");

  const visible = customers.filter(c => (tagFilter === "Todos" || c.tag === tagFilter) && (c.name.toLowerCase().includes(search.toLowerCase()) || c.phone.includes(search)));
  const totalCustomers = customers.length;
  const vipCount = customers.filter(c => c.tag === "VIP").length;
  const totalRevenue = customers.reduce((s, c) => s + parseFloat(c.totalSpent.replace("S/ ", "").replace(",", "")), 0);

  return <div className="cust-page">
    <header className="cust-header">
      <div>
        <span className="cust-eyebrow">RELACIÓN</span>
        <h1>Clientes</h1>
        <p>{totalCustomers} clientes · {vipCount} VIP · S/ {totalRevenue.toLocaleString()} en ventas históricas</p>
      </div>
      <Button tone="primary" className="cust-new"><Icon name="plus" size={18}/>Nuevo cliente</Button>
    </header>

    <div className="cust-filters">
      <div className="cust-search"><Icon name="search" size={16}/><Input placeholder="Buscar por nombre o teléfono..." value={search} onChange={e => setSearch(e.target.value)}/></div>
      <div className="cust-tags">
        <Button className={tagFilter === "Todos" ? "active" : ""} onClick={() => setTagFilter("Todos")}>Todos</Button>
        <Button className={tagFilter === "VIP" ? "active" : ""} onClick={() => setTagFilter("VIP")}>VIP</Button>
        <Button className={tagFilter === "Frecuente" ? "active" : ""} onClick={() => setTagFilter("Frecuente")}>Frecuentes</Button>
        <Button className={tagFilter === "Nuevo" ? "active" : ""} onClick={() => setTagFilter("Nuevo")}>Nuevos</Button>
      </div>
    </div>

    <div className="cust-list">
      {visible.map(c => <article className="cust-card" key={c.id}>
        <div className="cust-avatar">{c.name.split(" ").map(w => w[0]).join("").slice(0, 2)}</div>
        <div className="cust-info">
          <div className="cust-top">
            <strong>{c.name}</strong>
            {c.tag && <span className={"cust-tag " + tagTone[c.tag]}>{c.tag}</span>}
          </div>
          <div className="cust-meta">
            <span><Icon name="whatsapp" size={12}/>{c.phone}</span>
            <span><Icon name="orders" size={12}/>{c.orders} pedidos</span>
            <span><Icon name="cash" size={12}/>{c.totalSpent}</span>
            <small>{c.lastVisit}</small>
          </div>
        </div>
        <Button className="cust-action"><Icon name="chevron" size={16}/></Button>
      </article>)}
      {visible.length === 0 && <div className="cust-empty"><Icon name="user" size={28}/><b>Sin clientes</b><span>No se encontraron clientes.</span></div>}
    </div>
  </div>;
}
