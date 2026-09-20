"use client";
import { ActionLink, Button, Input, Label, Select } from "@/components/ui/controls";


import { useMemo, useState } from "react";
import { Icon } from "@/components/icon";
import Link from "next/link";

type Product = { name: string; price: number; category: string; visual: string; badge?: string };
type OrderItem = { name: string; price: number; qty: number; note: string };

const products: Product[] = [
  { name: "Lomo saltado", price: 28, category: "Fondos", visual: "lomo", badge: "Más pedido" },
  { name: "Ají de gallina", price: 24, category: "Fondos", visual: "aji" },
  { name: "Ceviche clásico", price: 30, category: "Entradas", visual: "ceviche", badge: "Popular" },
  { name: "Arroz con pollo", price: 22, category: "Fondos", visual: "arroz" },
  { name: "Papa a la huancaína", price: 14, category: "Entradas", visual: "papa" },
  { name: "Chicha morada", price: 8, category: "Bebidas", visual: "chicha" },
  { name: "Inca Kola", price: 6, category: "Bebidas", visual: "inka" },
  { name: "Suspiro limeño", price: 12, category: "Postres", visual: "suspiro" },
];

const categories = ["Todos", "Entradas", "Fondos", "Bebidas", "Postres"];

type OrderTab = { id: string; label: string; number: string; meta: string };

const orderTabs: OrderTab[] = [
  { id: "mesa04", label: "Mesa 04", number: "#0128", meta: "Mesa 04 · 2 personas" },
  { id: "mostrador", label: "Mostrador", number: "#0129", meta: "Mostrador · Para llevar" },
];

const initialOrders: Record<string, OrderItem[]> = {
  mesa04: [
    { name: "Lomo saltado", price: 28, qty: 1, note: "Sin cebolla · Término 3/4" },
    { name: "Chicha morada", price: 8, qty: 2, note: "Sin hielo" },
  ],
  mostrador: [{ name: "Ceviche clásico", price: 30, qty: 1, note: "" }],
};

const currency = (value: number) => `S/ ${value.toFixed(2)}`;

export default function PosPage() {
  const [activeTab, setActiveTab] = useState("mesa04");
  const [orders, setOrders] = useState(initialOrders);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("Todos");
  const [sentToKitchen, setSentToKitchen] = useState(false);

  const items = orders[activeTab];
  const tab = orderTabs.find(t => t.id === activeTab) ?? orderTabs[0];

  const subtotal = items.reduce((sum, item) => sum + item.price * item.qty, 0);
  const igv = subtotal - subtotal / 1.18;
  const total = subtotal;
  const totalQty = items.reduce((sum, item) => sum + item.qty, 0);

  const visibleProducts = useMemo(() => {
    const query = search.trim().toLowerCase();
    return products.filter(p => {
      const matchesCategory = category === "Todos" || p.category === category;
      const matchesSearch = !query || p.name.toLowerCase().includes(query) || p.category.toLowerCase().includes(query);
      return matchesCategory && matchesSearch;
    });
  }, [search, category]);

  const qtyInOrder = (name: string) => items.find(i => i.name === name)?.qty ?? 0;

  const addItem = (product: Product) => {
    setOrders(prev => ({
      ...prev,
      [activeTab]: prev[activeTab].some(i => i.name === product.name)
        ? prev[activeTab].map(i => (i.name === product.name ? { ...i, qty: i.qty + 1 } : i))
        : [...prev[activeTab], { name: product.name, price: product.price, qty: 1, note: "" }],
    }));
  };

  const changeQty = (name: string, delta: number) => {
    setOrders(prev => ({
      ...prev,
      [activeTab]: prev[activeTab]
        .map(i => (i.name === name ? { ...i, qty: i.qty + delta } : i))
        .filter(i => i.qty > 0),
    }));
  };

  const clearOrder = () => {
    setOrders(prev => ({ ...prev, [activeTab]: [] }));
    setSentToKitchen(false);
  };

  return <div className="pos-workspace">
    <section className="pos-catalog">
      <header className="pos-titlebar">
        <div><span className="pos-eyebrow">PUNTO DE VENTA</span><h1>Tomar pedido</h1><p>{tab.meta} · Atención en salón</p></div>
        <Button className="table-selector"><span><Icon name="tables" size={18}/></span><div><small>MESA ACTUAL</small><strong>{tab.label}</strong></div><Icon name="chevron" size={16}/></Button>
      </header>
      <div className="order-tabs">
        {orderTabs.map(t => <Button className={t.id === activeTab ? "active" : ""} key={t.id} onClick={() => setActiveTab(t.id)}>
          <span>{t.label}</span><small>{t.number}</small>
        </Button>)}
        <Button tone="primary" className="new-order-tab"><Icon name="plus" size={17}/>Nuevo pedido</Button>
      </div>
      <div className="pos-tools">
        <Label className="pos-search"><Icon name="search" size={19}/><Input aria-label="Buscar productos" placeholder="Buscar plato, bebida o código..." value={search} onChange={e => setSearch(e.target.value)}/>{search && <Button aria-label="Limpiar búsqueda" onClick={() => setSearch("")}><Icon name="minus" size={16}/></Button>}<kbd>⌘ K</kbd></Label>
        <Button className="filter-button"><Icon name="filter" size={18}/><span>Filtros</span></Button>
      </div>
      <nav className="category-strip" aria-label="Categorías del menú">{categories.map(cat => {
        const count = cat === "Todos" ? products.length : products.filter(p => p.category === cat).length;
        return <Button className={cat === category ? "active" : ""} key={cat} onClick={() => setCategory(cat)}><span>{cat}</span><b>{count}</b></Button>;
      })}</nav>
      <div className="product-section-title"><div><h2>{category === "Todos" ? "Todos los productos" : category}</h2><span>{visibleProducts.length} disponibles</span></div><Select className="product-sort" aria-label="Ordenar productos" defaultValue="popular"><option value="popular">Ordenar: Popularidad</option><option value="name">Ordenar: Nombre</option><option value="price-asc">Precio: menor a mayor</option><option value="price-desc">Precio: mayor a menor</option></Select></div>
      <div className="premium-product-grid">
        {visibleProducts.map(product => {
          const qty = qtyInOrder(product.name);
          return <Button layout="card" className="premium-product" key={product.name} onClick={() => addItem(product)}>
            <span className={"product-photo " + product.visual} aria-hidden="true"/>{product.badge && <span className="product-badge">{product.badge}</span>}
            {qty > 0 && <span className="product-qty-badge">{qty}</span>}
            <span className="product-copy"><small>{product.category}</small><strong>{product.name}</strong><span><b>{currency(product.price)}</b><i className={qty > 0 ? "in-order" : ""}><Icon name={qty > 0 ? "check" : "plus"} size={18}/></i></span></span>
          </Button>;
        })}
      </div>
      {visibleProducts.length === 0 && <div className="pos-empty"><Icon name="search" size={22}/><b>Sin resultados</b><span>Prueba con otro nombre o categoría.</span></div>}
    </section>
    <aside className="order-panel">
      <header className="order-heading"><div><span>PEDIDO ACTUAL</span><h2>{tab.label}</h2><p>{totalQty} productos · Mozo Rogelio</p></div><Button aria-label="Vaciar pedido" onClick={clearOrder}><Icon name="trash" size={19}/></Button></header>
      <div className={"order-status" + (sentToKitchen ? " has-sent" : "")}><span><i className={sentToKitchen ? "sent" : ""}/>{sentToKitchen ? "En cocina" : "Borrador"}</span><small>{sentToKitchen ? "Enviado a cocina hace un momento" : "Aún no se envió a cocina"}</small></div>
      <div className="order-lines">
        {items.map(item => <OrderLine key={item.name} name={item.name} note={item.note} price={item.price} qty={item.qty} onChangeQty={delta => changeQty(item.name, delta)}/>)}
        {items.length === 0 && <div className="order-empty"><Icon name="pos" size={22}/><b>Pedido vacío</b><span>Toca un producto para agregarlo.</span></div>}
      </div>
      <Button className="kitchen-note"><Icon name="plus" size={17}/>Agregar nota para cocina</Button>
      <div className="order-summary"><div><span>Subtotal</span><b>{currency(subtotal - igv)}</b></div><div><span>IGV incluido</span><b>{currency(igv)}</b></div><div className="order-total"><span>Total</span><strong>{currency(total)}</strong></div></div>
      <div className="order-actions">
        <ActionLink tone="neutral" href="/pos/pago" className="save-order">Ir al cobro</ActionLink>
        <Button tone="primary" className="send-kitchen" disabled={totalQty === 0 || sentToKitchen} onClick={() => setSentToKitchen(true)}><span><Icon name="kitchen" size={19}/>{sentToKitchen ? "Enviado a cocina" : "Enviar a cocina"}</span>{sentToKitchen ? <Icon name="check" size={18}/> : <Icon name="chevron" size={18}/>}</Button>
      </div>
    </aside>
  </div>;
}

function OrderLine({ name, note, price, qty, onChangeQty }: { name: string; note: string; price: number; qty: number; onChangeQty: (delta: number) => void }) {
  return <article>
    <div className="line-copy"><strong>{name}</strong><small>{note || "Sin indicaciones"}</small><Link href="/pos/personalizar">Editar indicaciones</Link></div>
    <div className="line-price"><strong>{currency(price)}</strong><span><Button aria-label="Reducir" onClick={() => onChangeQty(-1)}><Icon name="minus" size={14}/></Button><b>{qty}</b><Button aria-label="Aumentar" onClick={() => onChangeQty(1)}><Icon name="plus" size={14}/></Button></span></div>
  </article>;
}
