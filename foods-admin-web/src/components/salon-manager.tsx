"use client";

import "../app/orders.css";
import "../app/salon.css";
import {useState,useCallback} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,PageHeader,Select,Status} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {ComandaCatalog} from "@/components/comanda-catalog";
import {apiFetch} from "@/shared/api/client";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {useSettings} from "@/providers/settings-context";

/* ── types ── */
type OrderItem={id:string;productId:string;name:string;qty:string;unitPrice:string;note:string};
type Order={id:string;code:string;channel:string;status:string;customerId:string;customerName:string;customerPhone:string;address:string;reference:string;tableId:string;tableName:string;notes:string;subtotal:string;deliveryFee:string;total:string;createdAt:string;updatedAt:string;itemCount?:number;items?:OrderItem[]};
type FloorTable={id:string;name:string;zone:string;seats:number;order:Order|null};
type Product={id:string;name:string;price:string;categoryId:string|null;imageUrl:string|null};
type LineDraft={productId:string;name:string;qty:number;unitPrice:number;note:string};
type Draft={channel:string;customerName:string;customerPhone:string;address:string;reference:string;tableId:string;notes:string;deliveryFee:string;lines:LineDraft[]};

/* ── helpers ── */
const statusMeta:Record<string,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={
  nuevo:{label:"Nuevo",tone:"blue"},confirmado:{label:"Confirmado",tone:"blue"},
  preparando:{label:"Preparando",tone:"orange"},listo:{label:"Listo",tone:"green"},
  en_camino:{label:"En camino",tone:"orange"},entregado:{label:"Entregado",tone:"gray"},
  cancelado:{label:"Cancelado",tone:"gray"},
};
function nextAction(o:Order):{status:string;label:string}|null{
  if(o.status==="nuevo")return{status:"confirmado",label:"Confirmar"};
  if(o.status==="confirmado")return{status:"preparando",label:"Iniciar"};
  if(o.status==="preparando")return{status:"listo",label:"Marcar listo"};
  if(o.status==="listo")return{status:"entregado",label:"Entregar"};
  if(o.status==="en_camino")return{status:"entregado",label:"Entregar"};
  return null;
}
function parseIsoDate(iso:string):Date|null{
  if(!iso)return null;
  const normalized=iso.replace(/([+-]\d{2})$/,"$1:00");
  const d=new Date(normalized);
  if(!isNaN(d.getTime()))return d;
  const d2=new Date(iso);
  return isNaN(d2.getTime())?null:d2;
}
function occupiedFor(iso:string){
  const d=parseIsoDate(iso);if(!d)return"—";
  const m=Math.max(0,Math.floor((Date.now()-d.getTime())/60000));
  if(m<60)return`${m} min`;return`${Math.floor(m/60)}h ${m%60}min`;
}
function timeAgo(iso:string){
  const d=parseIsoDate(iso);if(!d)return"—";
  const m=Math.floor((Date.now()-d.getTime())/60000);
  if(m<1)return"Justo ahora";if(m<60)return`Hace ${m} min`;
  const h=Math.floor(m/60);if(h<24)return`Hace ${h} h`;
  return new Intl.DateTimeFormat("es-PE",{dateStyle:"medium",timeStyle:"short"}).format(d);
}
const money=(v:string|number)=>Number(v).toFixed(2);
const emptyDraft=(tableId=""):Draft=>({channel:"salon",customerName:"",customerPhone:"",address:"",reference:"",tableId,notes:"",deliveryFee:"0",lines:[]});

/* ═══════════════════════════════════════════════════
   SalonManager — vista operativa del mozo
═══════════════════════════════════════════════════ */
export function SalonManager(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can}=useSession();
  const settings=useSettings();
  const canManage=can("orders.manage");

  const[q,setQ]=useState("");
  const[detailId,setDetailId]=useState<string|null>(null);
  const[draft,setDraft]=useState<Draft|null>(null);
  const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
  const[zone,setZone]=useState("");

  const floor=useQuery({
    queryKey:["salon-floor"],
    queryFn:()=>apiFetch<{items:FloorTable[]}>("orders/floor"),
    refetchInterval:30000,
  });
  const detail=useQuery({
    queryKey:["order",detailId],
    queryFn:()=>apiFetch<Order>(`orders/${detailId}`),
    enabled:Boolean(detailId),
  });

  const invalidate=useCallback(()=>{
    void qc.invalidateQueries({queryKey:["salon-floor"]});
    void qc.invalidateQueries({queryKey:["order",detailId]});
  },[qc,detailId]);

  const advance=useMutation({
    mutationFn:(v:{id:string;status:string})=>apiFetch<Order>(`orders/${v.id}/status`,{method:"PATCH",body:JSON.stringify({status:v.status})}),
    onSuccess:()=>{invalidate();notify({tone:"success",title:"Pedido actualizado",message:"El estado fue actualizado correctamente."})},
    onError:e=>notify({tone:"danger",title:"Error",message:e.message}),
  });
  const create=useMutation({
    mutationFn:(v:Draft)=>apiFetch<Order>("orders",{method:"POST",body:JSON.stringify({channel:v.channel,customerName:v.customerName,customerPhone:v.customerPhone,address:v.address,reference:v.reference,tableId:v.tableId,notes:v.notes,deliveryFee:Number(v.deliveryFee)||0,items:v.lines.map(l=>({productId:l.productId,name:l.name,qty:l.qty,unitPrice:l.unitPrice,note:l.note}))})}),
    onSuccess:o=>{setDraft(null);invalidate();setDetailId(o.id);notify({tone:"success",title:"Mesa abierta",message:`Pedido ${o.code} registrado.`})},
    onError:e=>notify({tone:"danger",title:"Error",message:e.message}),
  });
  const cancel=useMutation({
    mutationFn:(o:Order)=>apiFetch<Order>(`orders/${o.id}/status`,{method:"PATCH",body:JSON.stringify({status:"cancelado"})}),
    onSuccess:()=>{setCancelTarget(null);invalidate();notify({tone:"success",title:"Pedido cancelado",message:"La mesa quedó libre."})},
    onError:e=>notify({tone:"danger",title:"Error",message:e.message}),
  });

  const tables=floor.data?.items??[];
  const zones=[...new Set(tables.map(t=>t.zone).filter(Boolean))];
  const visible=tables.filter(t=>{
    const matchZone=!zone||t.zone===zone;
    const needle=q.trim().toLowerCase();
    const matchQ=!needle||t.name.toLowerCase().includes(needle)||t.zone.toLowerCase().includes(needle)||(t.order?.code??"").toLowerCase().includes(needle)||(t.order?.customerName??"").toLowerCase().includes(needle);
    return matchZone&&matchQ;
  });
  const occupiedCount=tables.filter(t=>t.order).length;
  const freeCount=tables.length-occupiedCount;

  return(
    <>
      <PageHeader
        eyebrow="OPERACIÓN"
        title="Salón"
        description="Plano de mesas en vivo: toca una libre para abrir comanda o una ocupada para ver su pedido."
      />

      {/* Toolbar */}
      <div className="salon-toolbar">
        <label className="salon-search">
          <Icon name="search" size={18}/>
          <input value={q} onChange={e=>setQ(e.target.value)} placeholder="Buscar mesa, zona o pedido…"/>
          {q&&<button type="button" onClick={()=>setQ("")} aria-label="Limpiar búsqueda"><Icon name="close" size={16}/></button>}
        </label>
        {zones.length>1&&(
          <div className="salon-zones">
            <button className={"salon-zone"+(zone===""?" active":"")} onClick={()=>setZone("")}>Todas</button>
            {zones.map(z=>(
              <button key={z} className={"salon-zone"+(zone===z?" active":"")} onClick={()=>setZone(z)}>{z}</button>
            ))}
          </div>
        )}
        <button className="salon-refresh" onClick={()=>floor.refetch()} aria-label="Actualizar salón" disabled={floor.isFetching}>
          <Icon name="refresh" size={17}/>
        </button>
      </div>

      {/* Floor plan */}
      {floor.isLoading?(
        <div className="salon-skeleton">
          {Array.from({length:8}).map((_,i)=><div key={i} className="salon-skeleton-card"/>)}
        </div>
      ):floor.isError?(
        <div className="catalog-state error">
          <span><Icon name="alert" size={22}/></span>
          <b>No pudimos cargar el salón</b>
          <p>{floor.error.message}</p>
          <Button kind="ghost" onClick={()=>floor.refetch()}>Reintentar</Button>
        </div>
      ):!tables.length?(
        <div className="catalog-state">
          <span><Icon name="utensils" size={22}/></span>
          <b>Sin mesas configuradas</b>
          <p>Registra las mesas del restaurante en <strong>Mesas y zonas</strong> para empezar a tomar pedidos.</p>
        </div>
      ):(
        <>
          <div className="salon-grid">
            {visible.map(t=>{
              const o=t.order;
              const meta=o?statusMeta[o.status]??{label:o.status,tone:"gray" as const}:null;
              const tableState=o?"occupied":"free";
              return(
                <button
                  key={t.id}
                  className={`salon-table ${tableState}`}
                  onClick={()=>o?setDetailId(o.id):canManage&&setDraft(emptyDraft(t.id))}
                  disabled={!o&&!canManage}
                  aria-label={`Mesa ${t.name}${o?`, ocupada, pedido ${o.code}`:", libre"}`}
                >
                  <div className="salon-table-inner">
                    {/* Top row — estado + tiempo */}
                    <div className="salon-table-top">
                      {o&&meta
                        ?<span className={`salon-table-state st-${meta.tone}`}>{meta.label}</span>
                        :<span className="salon-table-state st-free">Libre</span>}
                      {o&&<span className="salon-table-elapsed"><Icon name="clock" size={10}/>{occupiedFor(o.createdAt)}</span>}
                    </div>

                    {/* Table identifier */}
                    <div className="salon-table-name-row">
                      <span className="salon-table-icon">
                        <Icon name="utensils" size={17}/>
                      </span>
                      <strong className="salon-table-name">{t.name}</strong>
                      <small className="salon-table-seats">{t.seats} personas{zones.length>1&&t.zone?` · ${t.zone}`:""}</small>
                      {o?.customerName&&<span className="salon-table-family">{o.customerName}</span>}
                    </div>

                    {/* Order info or CTA */}
                    {o?(
                      <div className="salon-table-order">
                        <b className="salon-table-total">{settings.currencySymbol} {money(o.total)}</b>
                        <Icon name="eye" size={20}/>
                      </div>
                    ):(
                      <span className="salon-table-cta">
                        <Icon name="plus" size={14}/>
                        Abrir mesa
                      </span>
                    )}
                  </div>
                </button>
              );
            })}
            {!visible.length&&(
              <div className="salon-empty">
                <Icon name="search" size={22}/>
                <b>Sin coincidencias</b>
                <p>Ajusta la búsqueda o el filtro de zona.</p>
              </div>
            )}
          </div>

          {/* Legend */}
          <footer className="salon-legend">
            <div className="salon-legend-pills">
              <span className="salon-legend-pill"><i className="free"/>Libres <b>{freeCount}</b></span>
              <span className="salon-legend-pill"><i className="occupied"/>Ocupadas <b>{occupiedCount}</b></span>
            </div>
            {occupiedCount>0&&<span className="salon-legend-note">{occupiedCount} pedido{occupiedCount!==1?"s":""} activo{occupiedCount!==1?"s":""}</span>}
          </footer>
        </>
      )}

      {/* Comanda nueva */}
      {draft&&(
        <ComandaView
          initial={draft}
          allTables={tables}
          busy={create.isPending}
          currencySymbol={settings.currencySymbol}
          close={()=>setDraft(null)}
          save={v=>create.mutate(v)}
          notify={notify}
        />
      )}

      {/* Detalle del pedido */}
      {detailId&&(
        <OrderDetail
          loading={detail.isLoading}
          order={detail.data}
          error={detail.error?.message}
          currencySymbol={settings.currencySymbol}
          canManage={canManage}
          busy={advance.isPending}
          close={()=>setDetailId(null)}
          advance={st=>advance.mutate({id:detailId,status:st})}
          cancel={o=>setCancelTarget(o)}
        />
      )}

      <ConfirmDialog
        open={Boolean(cancelTarget)}
        title="Cancelar pedido"
        description={`El pedido ${cancelTarget?.code??""} quedará cancelado y la mesa quedará libre.`}
        tone="danger"
        confirmLabel="Cancelar pedido"
        pending={cancel.isPending}
        onCancel={()=>setCancelTarget(null)}
        onConfirm={()=>cancelTarget&&cancel.mutate(cancelTarget)}
      />
    </>
  );
}

/* ═══════════════════════════════════════════════════
   ComandaView — toma de pedido
═══════════════════════════════════════════════════ */
function ComandaView({initial,allTables,busy,currencySymbol,close,save,notify}:{initial:Draft;allTables:FloorTable[];busy:boolean;currencySymbol:string;close:()=>void;save:(v:Draft)=>void;notify:(n:{tone:"danger"|"success";title:string;message:string})=>void}){
  const[v,setV]=useState(initial);
  const[ticketOpen,setTicketOpen]=useState(false);
  const freeTables=allTables.filter(t=>!t.order||t.id===v.tableId);
  const selTable=allTables.find(t=>t.id===v.tableId);
  const tableName=selTable?.name??"";
  const tableLabel=selTable?(selTable.zone?`${selTable.zone} · ${selTable.name}`:selTable.name):"";
  const qtyByProduct=Object.fromEntries(v.lines.map(l=>[l.productId,l.qty]));
  const patchLine=(i:number,p:Partial<LineDraft>)=>setV({...v,lines:v.lines.map((l,n)=>n===i?{...l,...p}:l)});
  const tap=(p:Product)=>setV(prev=>{const i=prev.lines.findIndex(l=>l.productId===p.id);if(i>=0)return{...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+1}:l)};return{...prev,lines:[...prev.lines,{productId:p.id,name:p.name,qty:1,unitPrice:Number(p.price)||0,note:""}]}});
  const untap=(p:Product)=>setV(prev=>({...prev,lines:prev.lines.map(l=>l.productId===p.id?{...l,qty:l.qty-1}:l).filter(l=>l.qty>0)}));
  const step=(i:number,d:number)=>setV(prev=>({...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+d}:l).filter(l=>l.qty>0)}));
  const subtotal=v.lines.reduce((a,l)=>a+l.qty*l.unitPrice,0);
  const count=v.lines.reduce((a,l)=>a+l.qty,0);
  const submit=()=>{
    if(!v.lines.length){notify({tone:"danger",title:"Comanda vacía",message:"Agrega al menos un producto."});return}
    if(!v.tableId){notify({tone:"danger",title:"Falta mesa",message:"Selecciona la mesa del pedido."});return}
    save(v);
  };
  return(
    <div className="comanda" role="dialog" aria-modal="true">
      <header className="comanda-head">
        <button className="comanda-back" aria-label="Volver" onClick={close}><Icon name="chevronLeft" size={20}/></button>
        <div className="comanda-title">
          <small>Nueva comanda</small>
          <h2>{tableName||"Elige la mesa"}</h2>
        </div>
        <span className="comanda-channel-tag"><i/>Salón</span>
        <button className="comanda-close" aria-label="Cerrar" onClick={close}><Icon name="close" size={18}/></button>
      </header>
      <div className="comanda-body">
        <ComandaCatalog qtyByProduct={qtyByProduct} onPick={tap} onRemove={untap} currencySymbol={currencySymbol}/>
        {/* Ticket — recibo */}
        <aside className={"comanda-ticket"+(ticketOpen?" open":"")}>
          <i className="comanda-ticket-grip" aria-hidden="true"/>
          <div className="comanda-receipt-head">
            <div className="comanda-ticket-headrow">
              <small className="comanda-eyebrow">Comanda</small>
              <button type="button" className="comanda-ticket-close" onClick={()=>setTicketOpen(false)} aria-label="Volver a la carta"><Icon name="close" size={16}/></button>
            </div>
            {v.tableId?(
              <div className="comanda-receipt-meta"><span>Mesa</span><i className="comanda-receipt-leader" aria-hidden="true"/><b>{tableLabel}</b></div>
            ):(
              <Select value={v.tableId} onChange={e=>setV({...v,tableId:e.target.value})} aria-label="Mesa">
                <option value="">Selecciona una mesa</option>
                {freeTables.map(t=><option key={t.id} value={t.id}>{t.zone?`${t.zone} · `:""}{t.name}</option>)}
              </Select>
            )}
          </div>
          <div className="comanda-receipt-lines">
            {!v.lines.length&&<p className="comanda-empty"><Icon name="utensils" size={20}/>Toca los platos para armar la comanda</p>}
            {v.lines.map((l,i)=>(
              <div className="comanda-receipt-line" key={l.productId||i}>
                <div className="comanda-receipt-row">
                  <b>{l.name}</b>
                  <span className="comanda-receipt-leader" aria-hidden="true"/>
                  <em>{currencySymbol} {money(l.qty*l.unitPrice)}</em>
                </div>
                <div className="comanda-receipt-controls">
                  <div className="comanda-receipt-stepper">
                    <button type="button" onClick={()=>step(i,-1)} aria-label="Quitar uno"><Icon name="minus" size={12}/></button>
                    <b>{l.qty}</b>
                    <button type="button" onClick={()=>step(i,1)} aria-label="Agregar uno"><Icon name="plus" size={12}/></button>
                  </div>
                  <input value={l.note} onChange={e=>patchLine(i,{note:e.target.value})} placeholder="Nota (ej. sin cebolla)" aria-label="Nota"/>
                </div>
              </div>
            ))}
            {v.lines.length>0&&<input className="comanda-notes" value={v.notes} onChange={e=>setV({...v,notes:e.target.value})} placeholder="Notas del pedido (cocina o entrega)" aria-label="Notas del pedido"/>}
          </div>
          <footer className="comanda-receipt-foot">
            <div className="comanda-receipt-total"><span>Subtotal</span><span className="comanda-receipt-leader" aria-hidden="true"/><b>{currencySymbol} {money(subtotal)}</b></div>
          </footer>
          <i className="comanda-receipt-zigzag" aria-hidden="true"/>
        </aside>
      </div>
      <footer className="comanda-foot">
        <div className="comanda-foot-info"><Icon name="receipt" size={14}/><span><b>{count}</b> ítem{count===1?"":"s"} en la comanda</span></div>
        <button type="button" className="comanda-foot-finalize" onClick={submit} disabled={busy||!v.lines.length}>{busy?"Registrando…":<>FINALIZAR <b>{currencySymbol} {money(subtotal)}</b></>}</button>
      </footer>
      {/* Barra móvil */}
      <div className={"comanda-bar"+(v.lines.length?" ready":"")}>
        <button type="button" className="comanda-bar-info" onClick={()=>setTicketOpen(!ticketOpen)} aria-expanded={ticketOpen}>
          <b>{count}</b> ítem{count===1?"":"s"} · {currencySymbol} {money(subtotal)}<Icon name="chevron" size={15}/>
        </button>
        {v.lines.length>0&&<button type="button" className="comanda-bar-finish" onClick={submit} disabled={busy}>{busy?"…":"FINALIZAR"}</button>}
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════
   OrderDetail — detalle del pedido activo en mesa
═══════════════════════════════════════════════════ */
function OrderDetail({loading,order,error,currencySymbol,canManage,busy,close,advance,cancel}:{loading:boolean;order?:Order;error?:string;currencySymbol:string;canManage:boolean;busy:boolean;close:()=>void;advance:(st:string)=>void;cancel:(o:Order)=>void}){
  const meta=order?statusMeta[order.status]??{label:order.status,tone:"gray" as const}:null;
  const action=order?nextAction(order):null;
  return(
    <div className="modal-backdrop modal-overlay-in">
      <section className="crud-modal order-detail modal-panel-in" role="dialog" aria-modal="true">
        <div className="modal-accent"/>
        <header>
          <span className="modal-title-icon"><Icon name="utensils"/></span>
          <div><h2>Mesa {order?.tableName??order?.code??"—"}</h2><small>PEDIDO ACTIVO</small></div>
          <button aria-label="Cerrar" onClick={close}><Icon name="close"/></button>
        </header>
        {loading?<Loading/>:error?(
          <div className="order-detail-body"><div className="catalog-state error"><span><Icon name="alert" size={22}/></span><b>Error al cargar el pedido</b><p>{error}</p></div></div>
        ):order&&meta&&(
          <>
            <div className="order-detail-body">
              <section className="order-detail-summary">
                <div><small>MESA</small><b>{order.tableName||"—"}</b></div>
                <div><small>CÓDIGO</small><b>{order.code}</b></div>
                <div><small>TIEMPO</small><b>{timeAgo(order.createdAt)}</b></div>
                <Status tone={meta.tone}>{meta.label}</Status>
              </section>
              <section className="customer-detail-section">
                <header><div><small>CONSUMO</small><h3>Productos del pedido</h3></div></header>
                <div className="order-detail-items">
                  {(order.items??[]).map(it=>(
                    <div className="order-detail-line" key={it.id}>
                      <b>{Number(it.qty)}×</b>
                      <div className="order-detail-line-info">
                        <span>{it.name}</span>
                        {it.note&&<em>{it.note}</em>}
                      </div>
                      <strong>{currencySymbol} {money(Number(it.qty)*Number(it.unitPrice))}</strong>
                    </div>
                  ))}
                  {!(order.items??[]).length&&<p style={{color:"var(--ink-400)",fontSize:"12px",textAlign:"center",padding:"16px"}}>Sin ítems cargados aún.</p>}
                </div>
                {order.notes&&<div className="customer-detail-notes"><b>Notas</b><p>{order.notes}</p></div>}
              </section>
              <section className="order-detail-totals">
                <div><span>Subtotal</span><b>{currencySymbol} {money(order.subtotal)}</b></div>
                {Number(order.deliveryFee)>0&&<div><span>Delivery</span><b>{currencySymbol} {money(order.deliveryFee)}</b></div>}
                <div className="order-detail-grand"><span>Total</span><strong>{currencySymbol} {money(order.total)}</strong></div>
              </section>
            </div>
            {canManage&&(
              <footer className="order-detail-actions">
                {action&&<Button className="order-detail-primary" disabled={busy} onClick={()=>advance(action.status)}>{action.label}<Icon name="chevron" size={15}/></Button>}
                {order.status==="entregado"&&<span className="order-detail-done"><Icon name="check" size={15}/>Mesa entregada</span>}
                {!["entregado","cancelado"].includes(order.status)&&<Button kind="ghost" className="order-detail-cancel" disabled={busy} onClick={()=>cancel(order)}>Cancelar pedido</Button>}
              </footer>
            )}
          </>
        )}
      </section>
    </div>
  );
}

function Loading(){return <div className="customers-loading" aria-label="Cargando"><i/><i/><i/><i/></div>}
