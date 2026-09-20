"use client";

import { Button } from "@/components/ui/controls";


import { useState } from "react";
import { Icon } from "@/components/icon";

type Channel = "whatsapp" | "delivery" | "recojo";

type OrderLine = { name: string; qty: number; price: string; note?: string };

type Order = {
  id: string;
  client: string;
  phone: string;
  channel: Channel;
  state: "Nuevo" | "Confirmado" | "Preparando" | "Listo" | "En camino" | "Entregado";
  total: string;
  subtotal: string;
  delivery: string;
  time: string;
  address?: string;
  lines: OrderLine[];
};

const channelInfo: Record<Channel, { label: string; icon: "whatsapp" | "bike" | "store" }> = {
  whatsapp: { label: "WhatsApp", icon: "whatsapp" },
  delivery: { label: "Delivery", icon: "bike" },
  recojo: { label: "Recojo", icon: "store" },
};

const initialOrders: Order[] = [
  { id: "#W-018", client: "Juan Pérez", phone: "+51 987 654 321", channel: "whatsapp", state: "Nuevo", total: "S/ 49.90", subtotal: "S/ 44.90", delivery: "S/ 5.00", time: "Hace 2 min", lines: [{ name: "Lomo saltado", qty: 1, price: "S/ 24.00", note: "Sin cebolla" }, { name: "Chicha morada", qty: 1, price: "S/ 8.00" }, { name: "Suspiro limeño", qty: 1, price: "S/ 12.90" }] },
  { id: "#D-042", client: "Andrea Ruiz", phone: "+51 912 345 678", channel: "delivery", state: "Preparando", total: "S/ 76.00", subtotal: "S/ 68.00", delivery: "S/ 8.00", time: "Hace 8 min", address: "Av. Larco 845, Miraflores", lines: [{ name: "Ceviche clásico", qty: 1, price: "S/ 32.00" }, { name: "Ají de gallina", qty: 1, price: "S/ 22.00", note: "Sin pan" }, { name: "Inca Kola", qty: 2, price: "S/ 7.00" }] },
  { id: "#R-031", client: "Carlos Díaz", phone: "+51 998 887 766", channel: "recojo", state: "Listo", total: "S/ 38.50", subtotal: "S/ 38.50", delivery: "S/ 0.00", time: "Hace 16 min", lines: [{ name: "Arroz con pollo", qty: 1, price: "S/ 24.00" }, { name: "Papa a la huancaína", qty: 1, price: "S/ 14.50" }] },
  { id: "#W-017", client: "María Torres", phone: "+51 955 443 322", channel: "whatsapp", state: "En camino", total: "S/ 92.00", subtotal: "S/ 82.00", delivery: "S/ 10.00", time: "Hace 24 min", address: "Jr. de la Unión 500, Centro", lines: [{ name: "Arroz con mariscos", qty: 2, price: "S/ 35.00" }, { name: "Chicha morada", qty: 2, price: "S/ 6.00" }] },
  { id: "#D-041", client: "Pedro Salas", phone: "+51 977 665 544", channel: "delivery", state: "Confirmado", total: "S/ 54.00", subtotal: "S/ 48.00", delivery: "S/ 6.00", time: "Hace 12 min", address: "Calle Las Begonias 450, San Isidro", lines: [{ name: "Lomo saltado", qty: 1, price: "S/ 24.00" }, { name: "Ceviche mixto", qty: 1, price: "S/ 24.00" }] },
  { id: "#R-030", client: "Lucía Ortiz", phone: "+51 988 776 655", channel: "recojo", state: "Preparando", total: "S/ 29.90", subtotal: "S/ 29.90", delivery: "S/ 0.00", time: "Hace 6 min", lines: [{ name: "Ají de gallina", qty: 1, price: "S/ 22.00" }, { name: "Suspiro limeño", qty: 1, price: "S/ 7.90" }] },
];

const nextState: Record<Order["state"], Order["state"] | null> = {
  Nuevo: "Confirmado",
  Confirmado: "Preparando",
  Preparando: "Listo",
  Listo: "Entregado",
  "En camino": "Entregado",
  Entregado: null,
};

const actionLabel: Record<Order["state"], string> = {
  Nuevo: "Confirmar",
  Confirmado: "Iniciar",
  Preparando: "Marcar listo",
  Listo: "Entregar",
  "En camino": "Entregar",
  Entregado: "Entregado",
};

const stateTone: Record<Order["state"], string> = {
  Nuevo: "violet",
  Confirmado: "blue",
  Preparando: "blue",
  Listo: "green",
  "En camino": "violet",
  Entregado: "muted",
};

const channels: { id: Channel | "all"; label: string; icon: "whatsapp" | "bike" | "store" | "orders" }[] = [
  { id: "all", label: "Todos", icon: "orders" },
  { id: "whatsapp", label: "WhatsApp", icon: "whatsapp" },
  { id: "delivery", label: "Delivery", icon: "bike" },
  { id: "recojo", label: "Recojo", icon: "store" },
];

export default function OrdersPage() {
  const [orders, setOrders] = useState(initialOrders);
  const [channel, setChannel] = useState<Channel | "all">("all");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const countFor = (ch: Channel) => orders.filter(o => o.channel === ch).length;
  const visible = channel === "all" ? orders : orders.filter(o => o.channel === channel);
  const selected = orders.find(o => o.id === selectedId) ?? null;

  const advance = (id: string) => {
    setOrders(prev => prev.map(o => {
      if (o.id !== id) return o;
      const next = nextState[o.state];
      return next ? { ...o, state: next, time: "Justo ahora" } : o;
    }));
  };

  const advanceSelected = () => { if (selectedId) advance(selectedId); };

  return <div className="orders-page">
    <header className="orders-header">
      <div>
        <span className="orders-eyebrow">VENTA OMNICANAL</span>
        <h1>Pedidos</h1>
        <p>WhatsApp, delivery y recojo en una sola bandeja.</p>
      </div>
      <Button tone="primary" className="orders-new"><Icon name="plus" size={18}/>Nuevo pedido</Button>
    </header>

    <div className="orders-channels">
      {channels.map(ch => {
        const count = ch.id === "all" ? orders.length : countFor(ch.id as Channel);
        return <Button key={ch.id} className={"orders-ch" + (channel === ch.id ? " active" : "")} onClick={() => setChannel(ch.id)}>
          <Icon name={ch.icon} size={18}/>
          <span>{ch.label}</span>
          <b>{count}</b>
        </Button>;
      })}
    </div>

    <div className="orders-list">
      {visible.map(order => {
        const ch = channelInfo[order.channel];
        const tone = stateTone[order.state];
        const canAdvance = nextState[order.state] !== null;
        return <article className={"order-card" + (order.id === selectedId ? " selected" : "")} key={order.id} onClick={() => setSelectedId(order.id)}>
          <div className="order-card-left">
            <div className={"order-channel-icon " + order.channel}><Icon name={ch.icon} size={18}/></div>
            <div className="order-card-info">
              <div className="order-card-top">
                <strong>{order.client}</strong>
                <span className="order-id">{order.id}</span>
              </div>
              <div className="order-card-meta">
                <span className="order-channel-tag"><Icon name={ch.icon} size={12}/>{ch.label}</span>
                <span className={"order-state-tag os-" + tone}>{order.state}</span>
                <small>{order.time}</small>
              </div>
            </div>
          </div>
          <div className="order-card-right">
            <b className="order-total">{order.total}</b>
            {canAdvance
              ? <Button tone="operational" className="order-advance" onClick={e => { e.stopPropagation(); advance(order.id); }}>{actionLabel[order.state]}<Icon name="chevron" size={14}/></Button>
              : <span className="order-done"><Icon name="check" size={14}/>Entregado</span>}
          </div>
        </article>;
      })}
      {visible.length === 0 && <div className="orders-empty"><Icon name="orders" size={28}/><b>Sin pedidos</b><span>No hay pedidos en este canal.</span></div>}
    </div>

    {selected && <aside className="order-detail-overlay" onClick={() => setSelectedId(null)}>
      <div className="order-detail" onClick={e => e.stopPropagation()}>
        <header className="order-detail-head">
          <div className="order-detail-head-left">
            <div className={"order-channel-icon " + selected.channel}><Icon name={channelInfo[selected.channel].icon} size={20}/></div>
            <div>
              <h2>{selected.client}</h2>
              <small>{selected.id} · {channelInfo[selected.channel].label}</small>
            </div>
          </div>
          <Button className="order-detail-close" onClick={() => setSelectedId(null)} aria-label="Cerrar"><Icon name="close" size={18}/></Button>
        </header>

        <div className="order-detail-status">
          <span className={"order-state-tag os-" + stateTone[selected.state]}>{selected.state}</span>
          <small>{selected.time}</small>
        </div>

        <div className="order-detail-contact">
          <div><Icon name="whatsapp" size={15}/><span>{selected.phone}</span></div>
          {selected.address && <div><Icon name="bike" size={15}/><span>{selected.address}</span></div>}
        </div>

        <div className="order-detail-items">
          <h3>Detalle del pedido</h3>
          {selected.lines.map(line => <div className="order-detail-line" key={line.name}>
            <b>{line.qty}×</b>
            <div className="order-detail-line-info">
              <span>{line.name}</span>
              {line.note && <em>{line.note}</em>}
            </div>
            <strong>{line.price}</strong>
          </div>)}
        </div>

        <div className="order-detail-totals">
          <div><span>Subtotal</span><b>{selected.subtotal}</b></div>
          <div><span>Delivery</span><b>{selected.delivery}</b></div>
          <div className="order-detail-grand"><span>Total</span><strong>{selected.total}</strong></div>
        </div>

        <footer className="order-detail-actions">
          {nextState[selected.state] ? <Button tone="operational" className="order-detail-primary" onClick={advanceSelected}>{actionLabel[selected.state]}<Icon name="chevron" size={16}/></Button> : <span className="order-detail-done"><Icon name="check" size={16}/>Pedido entregado</span>}
          <Button className="order-detail-secondary"><Icon name="whatsapp" size={16}/>Contactar</Button>
        </footer>
      </div>
    </aside>}
  </div>;
}
