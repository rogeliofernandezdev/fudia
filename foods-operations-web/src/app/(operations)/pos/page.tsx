"use client";

import {ActionLink,Button,Input,Label,Select} from "@/components/ui/controls";
import {useEffect,useMemo,useState} from "react";
import {Icon} from "@/components/icon";
import {operationsFetch} from "@/lib/operations-api";
import {useRouter} from "next/navigation";

type Product={id:string;name:string;price:string;categoryName:string|null;imageURL:string|null;sku:string};
type Availability={productId:string;status:string};
type OrderItem={productId:string;name:string;price:number;qty:number;note:string};

const currency=(value:number)=>`S/ ${value.toFixed(2)}`;

export default function PosPage(){
  const router=useRouter();
  const[products,setProducts]=useState<Product[]>([]);
  const[availability,setAvailability]=useState<Record<string,string>>({});
  const[items,setItems]=useState<OrderItem[]>([]);
  const[search,setSearch]=useState("");
  const[category,setCategory]=useState("Todos");
  const[loading,setLoading]=useState(true);
  const[saving,setSaving]=useState(false);
  const[error,setError]=useState("");
  const[persistedOrder,setPersistedOrder]=useState<{id:string;code:string}|null>(null);
  const[sentToKitchen,setSentToKitchen]=useState(false);

  useEffect(()=>{
    void Promise.all([
      operationsFetch<{items:Product[]}>("products?status=active&page=1&pageSize=100"),
      operationsFetch<{items:Availability[]}>("product-availability?page=1&pageSize=100"),
    ]).then(([catalog,stock])=>{
      setProducts(catalog.items);
      setAvailability(Object.fromEntries(stock.items.map(item=>[item.productId,item.status])));
    }).catch(e=>setError(e instanceof Error?e.message:"No se pudo cargar el catálogo.")).finally(()=>setLoading(false));
  },[]);

  const categories=useMemo(()=>["Todos",...Array.from(new Set(products.map(p=>p.categoryName||"Sin categoría"))).sort()],[products]);
  const visibleProducts=useMemo(()=>{
    const q=search.trim().toLowerCase();
    return products.filter(p=>{
      const cat=p.categoryName||"Sin categoría";
      return (category==="Todos"||cat===category)&&(!q||p.name.toLowerCase().includes(q)||p.sku.toLowerCase().includes(q));
    });
  },[products,search,category]);
  const subtotal=items.reduce((sum,item)=>sum+item.price*item.qty,0);
  const igv=subtotal-subtotal/1.18;
  const totalQty=items.reduce((sum,item)=>sum+item.qty,0);

  function addItem(product:Product){
    if(persistedOrder)return;
    const status=availability[product.id]??"available";
    if(status==="sold_out"||status==="unavailable")return;
    setItems(prev=>prev.some(i=>i.productId===product.id)?prev.map(i=>i.productId===product.id?{...i,qty:i.qty+1}:i):[...prev,{productId:product.id,name:product.name,price:Number(product.price),qty:1,note:""}]);
  }
  function changeQty(productId:string,delta:number){
    if(persistedOrder)return;
    setItems(prev=>prev.map(i=>i.productId===productId?{...i,qty:i.qty+delta}:i).filter(i=>i.qty>0));
  }
  function clearOrder(){if(!persistedOrder)setItems([]);}

  async function persist(sendKitchen:boolean){
    if(!items.length||saving)return null;
    if(persistedOrder){
      if(sendKitchen&&!sentToKitchen){
        await operationsFetch<void>(`orders/${persistedOrder.id}/status`,{method:"PATCH",body:JSON.stringify({status:"confirmado"})});
        setSentToKitchen(true);
      }
      return persistedOrder;
    }
    setSaving(true);setError("");
    try{
      const order=await operationsFetch<{id:string;code:string}>("orders",{
        method:"POST",
        body:JSON.stringify({
          channel:"mostrador",
          customerName:"",
          customerPhone:"",
          address:"",
          reference:"",
          tableId:"",
          notes:"",
          deliveryFee:0,
          items:items.map(item=>({productId:item.productId,name:item.name,qty:item.qty,unitPrice:item.price,note:item.note,selections:[],reprice:true})),
        }),
      });
      setPersistedOrder(order);
      if(sendKitchen){
        await operationsFetch<void>(`orders/${order.id}/status`,{method:"PATCH",body:JSON.stringify({status:"confirmado"})});
        setSentToKitchen(true);
      }
      return order;
    }catch(e){setError(e instanceof Error?e.message:"No se pudo guardar el pedido.");return null;}
    finally{setSaving(false);}
  }

  async function goToPayment(){
    const order=await persist(true);
    if(order)router.push(`/pos/pago?orderId=${order.id}`);
  }

  if(loading)return <div className="pos-empty"><b>Cargando punto de venta…</b><span>Consultando catálogo y disponibilidad.</span></div>;

  return <div className="pos-workspace">
    <section className="pos-catalog">
      <header className="pos-titlebar">
        <div><span className="pos-eyebrow">PUNTO DE VENTA</span><h1>Nueva venta</h1><p>Mostrador · Pedido transaccional</p></div>
        <Button className="table-selector" disabled><span><Icon name="store" size={18}/></span><div><small>CANAL</small><strong>Mostrador</strong></div></Button>
      </header>
      {error&&<div className="missing-card" role="alert"><span>{error}</span></div>}
      {persistedOrder&&<div className="no-change-note"><Icon name="check" size={18}/><span><b>Pedido {persistedOrder.code} registrado</b><small>{sentToKitchen?"Enviado a cocina.":"Listo para cobrar."}</small></span></div>}
      <div className="pos-tools">
        <Label className="pos-search"><Icon name="search" size={19}/><Input aria-label="Buscar productos" placeholder="Buscar plato, bebida o código..." value={search} onChange={e=>setSearch(e.target.value)}/>{search&&<Button aria-label="Limpiar búsqueda" onClick={()=>setSearch("")}><Icon name="minus" size={16}/></Button>}</Label>
      </div>
      <nav className="category-strip" aria-label="Categorías del menú">{categories.map(cat=><Button className={cat===category?"active":""} key={cat} onClick={()=>setCategory(cat)}><span>{cat}</span><b>{cat==="Todos"?products.length:products.filter(p=>(p.categoryName||"Sin categoría")===cat).length}</b></Button>)}</nav>
      <div className="product-section-title"><div><h2>{category==="Todos"?"Todos los productos":category}</h2><span>{visibleProducts.length} productos</span></div><Select className="product-sort" aria-label="Ordenar productos" defaultValue="name"><option value="name">Ordenar: Nombre</option></Select></div>
      <div className="premium-product-grid">
        {visibleProducts.map(product=>{
          const qty=items.find(i=>i.productId===product.id)?.qty??0;
          const status=availability[product.id]??"available";
          const disabled=status==="sold_out"||status==="unavailable"||Boolean(persistedOrder);
          return <Button layout="card" className="premium-product" key={product.id} onClick={()=>addItem(product)} disabled={disabled}>
            <span className="product-photo" aria-hidden="true"/>{status==="low"&&<span className="product-badge">Últimas unidades</span>}{status==="sold_out"&&<span className="product-badge">Agotado</span>}
            {qty>0&&<span className="product-qty-badge">{qty}</span>}
            <span className="product-copy"><small>{product.categoryName||"Sin categoría"}</small><strong>{product.name}</strong><span><b>{currency(Number(product.price))}</b><i className={qty>0?"in-order":""}><Icon name={qty>0?"check":"plus"} size={18}/></i></span></span>
          </Button>;
        })}
      </div>
      {!visibleProducts.length&&<div className="pos-empty"><Icon name="search" size={22}/><b>Sin resultados</b><span>Prueba con otro nombre o categoría.</span></div>}
    </section>
    <aside className="order-panel">
      <header className="order-heading"><div><span>PEDIDO ACTUAL</span><h2>{persistedOrder?.code||"Mostrador"}</h2><p>{totalQty} productos</p></div><Button aria-label="Vaciar pedido" disabled={Boolean(persistedOrder)} onClick={clearOrder}><Icon name="trash" size={19}/></Button></header>
      <div className={"order-status"+(persistedOrder?" has-sent":"")}><span><i className={persistedOrder?"sent":""}/>{persistedOrder?(sentToKitchen?"En cocina":"Registrado"):"Borrador"}</span><small>{persistedOrder?"El pedido ya existe en el backend.":"Aún no se guardó el pedido."}</small></div>
      <div className="order-lines">
        {items.map(item=><article key={item.productId}><div className="line-copy"><strong>{item.name}</strong><small>{item.note||"Sin indicaciones"}</small></div><div className="line-price"><strong>{currency(item.price*item.qty)}</strong><span><Button aria-label="Reducir" disabled={Boolean(persistedOrder)} onClick={()=>changeQty(item.productId,-1)}><Icon name="minus" size={14}/></Button><b>{item.qty}</b><Button aria-label="Aumentar" disabled={Boolean(persistedOrder)} onClick={()=>changeQty(item.productId,1)}><Icon name="plus" size={14}/></Button></span></div></article>)}
        {!items.length&&<div className="order-empty"><Icon name="pos" size={22}/><b>Pedido vacío</b><span>Toca un producto para agregarlo.</span></div>}
      </div>
      <div className="order-summary"><div><span>Subtotal</span><b>{currency(subtotal-igv)}</b></div><div><span>IGV incluido</span><b>{currency(igv)}</b></div><div className="order-total"><span>Total</span><strong>{currency(subtotal)}</strong></div></div>
      <div className="order-actions">
        <Button className="save-order" disabled={!items.length||saving} onClick={()=>void goToPayment()}>{saving?"Guardando…":"Ir al cobro"}</Button>
        <Button tone="primary" className="send-kitchen" disabled={!items.length||saving||sentToKitchen} onClick={()=>void persist(true)}><span><Icon name="kitchen" size={19}/>{saving?"Guardando…":sentToKitchen?"Enviado a cocina":"Enviar a cocina"}</span>{sentToKitchen?<Icon name="check" size={18}/>:<Icon name="chevron" size={18}/>}</Button>
      </div>
      {persistedOrder&&<ActionLink tone="neutral" href={`/pos/pago?orderId=${persistedOrder.id}`} className="wide">Cobrar {persistedOrder.code}</ActionLink>}
    </aside>
  </div>;
}
