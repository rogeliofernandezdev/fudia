"use client";

import "../app/orders.css";
import "../app/salon.css";
import "../app/salon-comanda.css";
import {useState,useCallback} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Input,PageHeader,Select,Status,Textarea} from "@/design-system";
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
    <div className="salon-comanda-shell" role="dialog" aria-modal="true" aria-label="Nueva comanda">
      <header className="salon-comanda-header">
        <div className="salon-comanda-header-main">
          <button type="button" className="salon-comanda-icon-button" aria-label="Volver al salón" onClick={close}>
            <Icon name="chevronLeft" size={20}/>
          </button>
          <div className="salon-comanda-heading">
            <h2>Nueva comanda</h2>
          </div>
        </div>
        <div className="salon-comanda-header-actions">
          <span className="salon-comanda-channel"><i/>Salón</span>
          <button type="button" className="salon-comanda-icon-button" aria-label="Cerrar comanda" onClick={close}>
            <Icon name="close" size={18}/>
          </button>
        </div>
      </header>

      <div className="salon-comanda-main">
        <section className="salon-comanda-catalog" aria-label="Carta del restaurante">
          <div className="salon-comanda-context">
            <span className="salon-comanda-context-icon"><Icon name="utensils" size={18}/></span>
            <div className="salon-comanda-context-copy">
              <span>Mesa</span>
              <strong>{tableName||"Pendiente de seleccionar"}</strong>
            </div>
            <div className="salon-comanda-context-customer">
              <Input
                autoFocus
                value={v.customerName}
                onChange={e=>setV({...v,customerName:e.target.value})}
                placeholder="Cliente / familia (opcional)"
                aria-label="Nombre del cliente o familia"
              />
            </div>
            <div className="salon-comanda-context-stat">
              <small>Personas</small>
              <span className="salon-comanda-context-stat-value">
                <Icon name="users" size={14}/>
                <b>{selTable?.seats??"—"}</b>
              </span>
            </div>
          </div>
          <ComandaCatalog qtyByProduct={qtyByProduct} onPick={tap} onRemove={untap} currencySymbol={currencySymbol} variant="salon"/>
        </section>

        <aside className={"salon-comanda-summary"+(ticketOpen?" open":"")} aria-label="Resumen de la comanda">
          <i className="salon-comanda-summary-grip" aria-hidden="true"/>
          <header className="salon-comanda-summary-head">
            <div className="salon-comanda-summary-title">
              <span>Pedido actual</span>
              <h3>Resumen de la comanda</h3>
            </div>
            <span className="salon-comanda-count-badge">{count} ítem{count===1?"":"s"}</span>
            <button type="button" className="salon-comanda-sheet-close" onClick={()=>setTicketOpen(false)} aria-label="Volver a la carta">
              <Icon name="close" size={16}/>
            </button>
          </header>

          {!v.tableId&&(
            <div className="salon-comanda-table-card">
              <span className="salon-comanda-table-icon"><Icon name="utensils" size={17}/></span>
              <Select className="salon-comanda-table-select" value={v.tableId} onChange={e=>setV({...v,tableId:e.target.value})} aria-label="Mesa">
                <option value="">Selecciona una mesa</option>
                {freeTables.map(t=><option key={t.id} value={t.id}>{t.zone?t.zone+" · ":""}{t.name}</option>)}
              </Select>
            </div>
          )}

          <div className="salon-comanda-summary-scroll">
            {!v.lines.length?(
              <div className="salon-comanda-empty">
                <span className="salon-comanda-empty-icon"><Icon name="receipt" size={21}/></span>
                <strong>La comanda está vacía</strong>
                <p>Selecciona platos de la carta. Aquí aparecerán cantidades, notas y el total del pedido.</p>
              </div>
            ):(
              <>
                <div className="salon-comanda-lines">
                  {v.lines.map((l,i)=>(
                    <article className="salon-comanda-line" key={l.productId||i}>
                      <div className="salon-comanda-line-main">
                        <span className="salon-comanda-line-qty">{l.qty}×</span>
                        <div className="salon-comanda-line-copy">
                          <strong>{l.name}</strong>
                          <small>{currencySymbol} {money(l.unitPrice)} c/u</small>
                        </div>
                        <em className="salon-comanda-line-total">{currencySymbol} {money(l.qty*l.unitPrice)}</em>
                      </div>
                      <div className="salon-comanda-line-controls">
                        <div className="salon-comanda-stepper">
                          <button type="button" onClick={()=>step(i,-1)} aria-label={"Quitar un "+l.name}><Icon name="minus" size={12}/></button>
                          <b>{l.qty}</b>
                          <button type="button" onClick={()=>step(i,1)} aria-label={"Agregar un "+l.name}><Icon name="plus" size={12}/></button>
                        </div>
                        <Input className="salon-comanda-line-note" value={l.note} onChange={e=>patchLine(i,{note:e.target.value})} placeholder="Nota del plato, ej. sin cebolla" aria-label={"Nota para "+l.name}/>
                      </div>
                    </article>
                  ))}
                </div>
                <div className="salon-comanda-general-notes">
                  <label htmlFor="salon-comanda-notes">Notas generales</label>
                  <Textarea id="salon-comanda-notes" rows={3} value={v.notes} onChange={e=>setV({...v,notes:e.target.value})} placeholder="Indicaciones para cocina o atención"/>
                </div>
              </>
            )}
          </div>

          <footer className="salon-comanda-summary-foot">
            <div className="salon-comanda-total">
              <div className="salon-comanda-total-copy">
                <span>Subtotal</span>
              </div>
              <strong>{currencySymbol} {money(subtotal)}</strong>
            </div>
            <Button icon="receipt" className="salon-comanda-submit" onClick={submit} disabled={busy||!v.lines.length}>
              {busy?"Registrando…":"Registrar comanda"}
            </Button>
          </footer>
        </aside>
      </div>

      <button type="button" className={"salon-comanda-summary-overlay"+(ticketOpen?" open":"")} onClick={()=>setTicketOpen(false)} aria-label="Cerrar resumen"/>

      <div className="salon-comanda-mobile-bar">
        <button type="button" className="salon-comanda-mobile-summary" onClick={()=>setTicketOpen(true)} aria-expanded={ticketOpen}>
          <Icon name="receipt" size={17}/>
          <span>
            <small>Comanda · {count} ítem{count===1?"":"s"}</small>
            <b>{currencySymbol} {money(subtotal)}</b>
          </span>
          <Icon name="chevron" size={15}/>
        </button>
        {v.lines.length>0&&(
          <button type="button" className="salon-comanda-mobile-submit" onClick={submit} disabled={busy}>
            {busy?"…":<><Icon name="receipt" size={14}/><span>Registrar</span></>}
          </button>
        )}
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
  const itemCount=order?(order.items??[]).reduce((sum,it)=>sum+Number(it.qty||0),0):0;
  return(
    <div className="modal-backdrop modal-overlay-in">
      <section className="crud-modal order-detail salon-order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="salon-order-detail-title">
        <div className="salon-order-detail-accent"/>
        <header className="salon-order-detail-head">
          <div className="salon-order-detail-identity">
            <span className="salon-order-detail-icon"><Icon name="utensils" size={20}/></span>
            <div className="salon-order-detail-heading">
              <small>MESA ACTIVA</small>
              <h2 id="salon-order-detail-title">{order?.tableName||"Mesa"}</h2>
              {order?.customerName&&<p>{order.customerName}</p>}
            </div>
          </div>
          <div className="salon-order-detail-status">
            {order&&meta&&<Status tone={meta.tone}>{meta.label}</Status>}
          </div>
          <button type="button" className="salon-order-detail-close" aria-label="Cerrar detalle" onClick={close}><Icon name="close" size={17}/></button>
        </header>

        {loading?<Loading/>:error?(
          <div className="order-detail-body"><div className="catalog-state error"><span><Icon name="alert" size={22}/></span><b>Error al cargar el pedido</b><p>{error}</p></div></div>
        ):order&&meta&&(
          <>
            <div className="order-detail-body salon-order-detail-body">
              <section className="salon-order-detail-meta" aria-label="Datos del pedido">
                <div>
                  <span className="salon-order-detail-meta-icon"><Icon name="receipt" size={15}/></span>
                  <span><small>PEDIDO</small><b>{order.code}</b></span>
                </div>
                <div>
                  <span className="salon-order-detail-meta-icon"><Icon name="clock" size={15}/></span>
                  <span><small>ABIERTO</small><b>{timeAgo(order.createdAt)}</b></span>
                </div>
                <div>
                  <span className="salon-order-detail-meta-icon"><Icon name="utensils" size={15}/></span>
                  <span><small>CONSUMO</small><b>{itemCount} ítem{itemCount===1?"":"s"}</b></span>
                </div>
              </section>

              <section className="salon-order-detail-consumption">
                <header className="salon-order-detail-section-head">
                  <div>
                    <small>DETALLE</small>
                    <h3>Productos del pedido</h3>
                  </div>
                  <span>{(order.items??[]).length} línea{(order.items??[]).length===1?"":"s"}</span>
                </header>

                <div className="order-detail-items salon-order-detail-items">
                  {(order.items??[]).map(it=>(
                    <div className="order-detail-line salon-order-detail-line" key={it.id}>
                      <b className="salon-order-detail-qty">{Number(it.qty)}×</b>
                      <div className="order-detail-line-info">
                        <span>{it.name}</span>
                        <small>{currencySymbol} {money(it.unitPrice)} c/u</small>
                        {it.note&&<em>{it.note}</em>}
                      </div>
                      <strong className="salon-order-detail-line-total">{currencySymbol} {money(Number(it.qty)*Number(it.unitPrice))}</strong>
                    </div>
                  ))}
                  {!(order.items??[]).length&&(
                    <div className="salon-order-detail-empty">
                      <Icon name="receipt" size={20}/>
                      <span>Sin ítems cargados aún.</span>
                    </div>
                  )}
                </div>

                {order.notes&&(
                  <div className="salon-order-detail-notes">
                    <span><Icon name="edit" size={15}/></span>
                    <div><b>Notas generales</b><p>{order.notes}</p></div>
                  </div>
                )}
              </section>

              <section className="salon-order-detail-totals">
                <div className="salon-order-detail-subtotal">
                  <span>Subtotal</span>
                  <b>{currencySymbol} {money(order.subtotal)}</b>
                </div>
                {Number(order.deliveryFee)>0&&(
                  <div className="salon-order-detail-subtotal">
                    <span>Delivery</span>
                    <b>{currencySymbol} {money(order.deliveryFee)}</b>
                  </div>
                )}
                <div className="salon-order-detail-grand">
                  <span>Total del pedido</span>
                  <strong>{currencySymbol} {money(order.total)}</strong>
                </div>
              </section>
            </div>

            {canManage&&(
              <footer className="order-detail-actions salon-order-detail-actions">
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
