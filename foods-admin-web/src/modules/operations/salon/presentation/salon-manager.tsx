"use client";

import {Dialog} from "@/design-system/dialog";
import "../../styles/orders.css";
import "../../styles/salon.css";
import {useState,useCallback} from "react";
import Link from "next/link";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Status} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {ComandaView} from "../../orders/presentation/comanda-view";
import type {Draft,Order} from "../domain/types";
import {createSalonOrder,getSalonFloor,getSalonOrder,updateSalonOrder,updateSalonOrderStatus} from "../infrastructure/salon-api";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {useSettings} from "@/providers/settings-context";

import {nextOrderAction} from "../../orders/domain/order-actions";

/* ── helpers ── */
const statusMeta:Record<string,{label:string;tone:"green"|"blue"|"orange"|"gray"}>={
  nuevo:{label:"Nuevo",tone:"blue"},confirmado:{label:"Confirmado",tone:"blue"},
  preparando:{label:"Preparando",tone:"orange"},listo:{label:"Listo para entregar",tone:"green"},
  en_camino:{label:"En camino",tone:"orange"},entregado:{label:"Entregado",tone:"gray"},
  cancelado:{label:"Cancelado",tone:"gray"},
};

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
function timeAgo(iso:string,country?:string,timeZone?:string){
  const d=parseIsoDate(iso);if(!d)return"—";
  const m=Math.floor((Date.now()-d.getTime())/60000);
  if(m<1)return"Justo ahora";if(m<60)return`Hace ${m} min`;
  const h=Math.floor(m/60);if(h<24)return`Hace ${h} h`;
  return formatRegionalDateTime(d.toISOString(),{country,timeZone},{dateStyle:"medium",timeStyle:"short"});
}
const money=(v:string|number)=>Number(v).toFixed(2);
const emptyDraft=(tableId=""):Draft=>({channel:"salon",customerName:"",customerPhone:"",address:"",reference:"",tableId,notes:"",deliveryFee:"0",lines:[]});
const editableOrderStatus=(status:string)=>status==="nuevo"||status==="confirmado";
const draftFromOrder=(o:Order):Draft=>({
  channel:o.channel||"salon",
  customerName:o.customerName||"",
  customerPhone:o.customerPhone||"",
  address:o.address||"",
  reference:o.reference||"",
  tableId:o.tableId||"",
  notes:o.notes||"",
  deliveryFee:o.deliveryFee||"0",
  lines:(o.items??[]).map(it=>({
    lineKey:it.id,
    sourceItemId:it.id,
    itemType:it.itemType==="combo"?"combo":"product",
    productId:it.productId,
    name:it.name,
    qty:Number(it.qty)||1,
    unitPrice:Number(it.unitPrice)||0,
    note:it.note||"",
    selections:(it.selections??[]).map(sel=>({
      groupId:sel.groupId,
      groupName:sel.groupName,
      productId:sel.productId,
      name:sel.name,
      surcharge:Number(sel.surcharge)||0,
    })),
  })),
});

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
  const[editingOrderId,setEditingOrderId]=useState<string|null>(null);
  const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
  const[zone,setZone]=useState("");

  const floor=useQuery({
    queryKey:["salon-floor"],
    queryFn:getSalonFloor,
    refetchInterval:10000,
    refetchIntervalInBackground:true,
  });
  const detail=useQuery({
    queryKey:["order",detailId],
    queryFn:()=>getSalonOrder(detailId!),
    enabled:Boolean(detailId),
    refetchInterval:query=>query.state.data?.completedAt?false:10000,
  });

  const invalidate=useCallback(()=>{
    void qc.invalidateQueries({queryKey:["salon-floor"]});
    for(const key of ["orders","order","pos-orders","pos-order","dashboard"])void qc.invalidateQueries({queryKey:[key]});
  },[qc]);

  const advance=useMutation({
    mutationFn:(v:{id:string;status:string})=>updateSalonOrderStatus(v.id,v.status),
    onSuccess:(order)=>{invalidate();notify({tone:"success",title:order.status==="entregado"?(order.completedAt?"Mesa liberada":"Pedido entregado"):"Pedido actualizado",message:order.status==="entregado"&&!order.completedAt?"La entrega quedó registrada. La mesa conserva su cuenta abierta hasta finalizar el cobro.":"El estado fue actualizado correctamente."})},
    onError:e=>notify({tone:"danger",title:"Error",message:e.message}),
  });
  const create=useMutation({
    mutationFn:({draft,sendToKitchen}:{draft:Draft;sendToKitchen:boolean})=>createSalonOrder(draft,sendToKitchen),
    onSuccess:(o,variables)=>{
      setDraft(null);setEditingOrderId(null);setDetailId(null);invalidate();
      notify({
        tone:"success",
        title:variables.sendToKitchen?"Comanda enviada a cocina":"Borrador guardado",
        message:variables.sendToKitchen?`Pedido ${o.code} registrado y enviado a Cocina.`:`Pedido ${o.code} quedó pendiente de envío a Cocina.`,
      });
    },
    onError:e=>{
      void qc.invalidateQueries({queryKey:["salon-floor"]});
      void qc.invalidateQueries({queryKey:["order-catalog","salon"]});
      void qc.invalidateQueries({queryKey:["order-catalog","combos"]});
      notify({tone:"danger",title:"No se pudo registrar la comanda",message:e.message});
    },
  });
  const update=useMutation({
    mutationFn:({id,v}:{id:string;v:Draft})=>updateSalonOrder(id,v),
    onSuccess:o=>{
      setDraft(null);
      setEditingOrderId(null);
      setDetailId(null);
      void qc.invalidateQueries({queryKey:["salon-floor"]});
      void qc.invalidateQueries({queryKey:["order",o.id]});
      notify({tone:"success",title:"Comanda actualizada",message:`Pedido ${o.code} guardado correctamente.`});
    },
    onError:e=>{
      void qc.invalidateQueries({queryKey:["salon-floor"]});
      void qc.invalidateQueries({queryKey:["order",editingOrderId]});
      void qc.invalidateQueries({queryKey:["order-catalog","salon"]});
      void qc.invalidateQueries({queryKey:["order-catalog","combos"]});
      notify({tone:"danger",title:"No se pudo guardar",message:e.message});
    },
  });
  const cancel=useMutation({
    mutationFn:(o:Order)=>updateSalonOrderStatus(o.id,"cancelado"),
    onSuccess:()=>{setCancelTarget(null);invalidate();setDetailId(null);notify({tone:"success",title:"Pedido cancelado",message:"La mesa quedó libre."})},
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
  const closeDraft=()=>{setDraft(null);setEditingOrderId(null)};
  const openNewDraft=(tableId:string)=>{setEditingOrderId(null);setDraft(emptyDraft(tableId))};
  const editOrder=(o:Order)=>{
    if(!editableOrderStatus(o.status)){
      notify({tone:"danger",title:"Comanda no editable",message:"Solo se puede editar un pedido en estado Nuevo o Confirmado."});
      return;
    }
    if(Number(o.paidAmount??0)>0.00001){
      notify({tone:"danger",title:"Comanda con pagos",message:"Devuelve primero los pagos registrados antes de modificar la comanda."});
      return;
    }
    setEditingOrderId(o.id);
    setDraft(draftFromOrder(o));
    setDetailId(null);
  };

  return(
    <>
      <section className="salon-overview" aria-labelledby="salon-title">
        <div className="salon-overview-copy">
          <span className="salon-overview-eyebrow"><Icon name="store" size={14}/>Operación de salón</span>
          <h1 id="salon-title">Salón</h1>
          <p>Consulta el estado de las mesas y gestiona las comandas activas desde un solo lugar.</p>
        </div>

        <div className="salon-overview-stats" aria-label="Resumen del salón">
          <div className="salon-overview-stat">
            <span className="salon-overview-stat-icon"><Icon name="grid" size={17}/></span>
            <span><small>Mesas</small><b>{tables.length}</b></span>
          </div>
          <div className="salon-overview-stat free">
            <span className="salon-overview-stat-icon"><Icon name="check" size={17}/></span>
            <span><small>Libres</small><b>{freeCount}</b></span>
          </div>
          <div className="salon-overview-stat occupied">
            <span className="salon-overview-stat-icon"><Icon name="receipt" size={17}/></span>
            <span><small>Ocupadas</small><b>{occupiedCount}</b></span>
          </div>
        </div>
      </section>

      <section className="salon-controls" aria-label="Controles del salón">
        <div className="salon-toolbar">
          <label className="salon-search">
            <Icon name="search" size={18}/>
            <input value={q} onChange={e=>setQ(e.target.value)} placeholder="Buscar mesa, zona, pedido o familia…"/>
            {q&&<button type="button" onClick={()=>setQ("")} aria-label="Limpiar búsqueda"><Icon name="close" size={16}/></button>}
          </label>
          <button className="salon-refresh" onClick={()=>floor.refetch()} aria-label="Actualizar salón" disabled={floor.isFetching}>
            <Icon name="refresh" size={17}/>
            <span>{floor.isFetching?"Actualizando…":"Actualizar"}</span>
          </button>
        </div>

        {zones.length>1&&(
          <div className="salon-filter-row">
            <span className="salon-filter-label">Zona</span>
            <div className="salon-zones" role="group" aria-label="Filtrar por zona">
              <button className={"salon-zone"+(zone===""?" active":"")} onClick={()=>setZone("")}>Todas</button>
              {zones.map(z=>(
                <button key={z} className={"salon-zone"+(zone===z?" active":"")} onClick={()=>setZone(z)}>{z}</button>
              ))}
            </div>
          </div>
        )}
      </section>

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
                  onClick={()=>o?setDetailId(o.id):canManage&&openNewDraft(t.id)}
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


        </>
      )}

      {/* Nueva / edición de comanda */}
      {draft&&(
        <ComandaView
          initial={draft}
          mode={editingOrderId?"edit":"create"}
          allTables={tables}
          busy={editingOrderId?update.isPending:create.isPending}
          currencySymbol={settings.currencySymbol}
          close={closeDraft}
          save={(v,sendToKitchen)=>editingOrderId?update.mutate({id:editingOrderId,v}):create.mutate({draft:v,sendToKitchen})}
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
          edit={editOrder}
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
   OrderDetail — detalle del pedido activo en mesa
═══════════════════════════════════════════════════ */
function OrderDetail({loading,order,error,currencySymbol,canManage,busy,close,advance,edit,cancel}:{loading:boolean;order?:Order;error?:string;currencySymbol:string;canManage:boolean;busy:boolean;close:()=>void;advance:(st:string)=>void;edit:(o:Order)=>void;cancel:(o:Order)=>void}){
  const{location}=useSession();
  const meta=order?statusMeta[order.status]??{label:order.status,tone:"gray" as const}:null;
  const action=order?nextOrderAction(order):null;
  const paid=Number(order?.paidAmount??0);
  const editable=Boolean(order&&editableOrderStatus(order.status)&&paid<=0.00001);
  const remaining=Number(order?.remainingAmount??order?.total??0);
  const canCharge=Boolean(order&&(order.status==="listo"||order.status==="entregado")&&!order.completedAt&&remaining>0.00001);
  const hasPayments=paid>0.00001;
  const canCancel=Boolean(order&&(order.status==="nuevo"||order.status==="confirmado")&&!hasPayments);
  const actionIcon=action?.status==="entregado"?"availability":action?.status==="confirmado"?"chefHat":action?.icon;
  const itemCount=order?(order.items??[]).reduce((sum,it)=>sum+Number(it.qty||0),0):0;
  const tableLabel=order?.tableName?(/^mesa\b/i.test(order.tableName)?order.tableName:`Mesa ${order.tableName}`):"Mesa";
  return(
    <div className="modal-backdrop modal-overlay-in">
      <Dialog className="crud-modal order-detail salon-order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="salon-order-detail-title" aria-busy={loading}>
        <div className="salon-order-detail-accent" aria-hidden="true"/>
        {loading?(
          <OrderDetailSkeleton close={close}/>
        ):(
          <>
            <header className="salon-order-detail-head">
              <div className="salon-order-detail-identity">
                <span className="salon-order-detail-icon"><Icon name="utensils" size={20}/></span>
                <div className="salon-order-detail-heading">
                  <h2 id="salon-order-detail-title">{tableLabel}</h2>
                  {order&&<p>{order.customerName&&<><span>{order.customerName}</span><span aria-hidden="true"> · </span></>}<span className="salon-order-detail-opened"><Icon name="clock" size={14}/>{timeAgo(order.createdAt,location?.country,location?.timezone)}</span></p>}
                </div>
              </div>
              <div className="salon-order-detail-status">
                {order&&meta&&<Status tone={meta.tone}>{meta.label}</Status>}
              </div>
              <button type="button" className="salon-order-detail-close" aria-label="Cerrar detalle" onClick={close}><Icon name="close" size={17}/></button>
            </header>

            {error?(
              <div className="order-detail-body"><div className="catalog-state error"><span><Icon name="alert" size={22}/></span><b>Error al cargar el pedido</b><p>{error}</p></div></div>
            ):order&&meta&&(
              <>
                <div className="order-detail-body salon-order-detail-body">
                  <div className="salon-order-detail-content">
                  <section className="salon-order-detail-consumption" aria-labelledby="salon-order-products-title">
                    <header className="salon-order-detail-section-head">
                      <div>
                        <h3 id="salon-order-products-title">Productos del pedido</h3>
                      </div>
                      <span>{itemCount} unidad{itemCount===1?"":"es"}</span>
                    </header>

                    <div className="order-detail-items salon-order-detail-items">
                      {(order.items??[]).map(it=>(
                        <div className="order-detail-line salon-order-detail-line" key={it.id}>
                          <b className="salon-order-detail-qty">{Number(it.qty)}×</b>
                          <div className="order-detail-line-info">
                            <span>{it.name}</span>
                            <small>{currencySymbol} {money(it.unitPrice)} c/u</small>
                            {it.itemType==="combo"&&(it.selections??[]).length>0&&(
                              <div className="salon-order-detail-selections">
                                {(it.selections??[]).map(sel=><small key={sel.groupId+sel.productId}><b>{sel.groupName}:</b> {sel.name}{Number(sel.surcharge)>0?` (+${currencySymbol} ${money(sel.surcharge)})`:""}</small>)}
                              </div>
                            )}
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
                <section className={`salon-order-detail-totals${remaining<=0.00001?" is-paid":""}`} aria-labelledby="salon-order-account-title">
                  <h3 id="salon-order-account-title" className="sr-only">Resumen de cuenta</h3>
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
                  <div className={`salon-order-detail-subtotal salon-order-detail-paid${hasPayments?" has-payment":""}`}>
                    <span>Pagado</span>
                    <b>{currencySymbol} {money(order.paidAmount??0)}</b>
                  </div>
                  <div className="salon-order-detail-grand">
                    <span>{remaining>0.00001?"Saldo pendiente":"Total pagado"}</span>
                    <strong>{currencySymbol} {money(remaining>0.00001?remaining:order.total)}</strong>
                  </div>
                </section>
                  </div>
                </div>

                {canManage&&(
                  <footer className="order-detail-actions salon-order-detail-actions">
                    {order.status==="entregado"&&!canCharge&&<p className="salon-order-detail-notice is-success" role="status"><Icon name="circleCheck" size={18}/><span>{order.completedAt?"Mesa liberada":"Pedido entregado"}</span></p>}
                    {(order.status==="nuevo"||order.status==="confirmado")&&hasPayments&&<p className="salon-order-detail-notice" role="note"><Icon name="info" size={18}/><span>Devuelve los pagos en POS antes de cancelar.</span></p>}
                    <div className="salon-order-detail-buttons" role="group" aria-label="Acciones de la mesa">
                      {(editable||canCancel||(action&&canCharge))&&<div className="salon-order-detail-secondary-actions">
                        {action&&canCharge&&<Button icon={actionIcon} kind="secondary" className="salon-order-detail-deliver" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
                        {editable&&<Button icon="edit" kind="secondary" className="salon-order-detail-edit" disabled={busy} onClick={()=>edit(order)}>Editar comanda</Button>}
                        {canCancel&&<Button icon="cancel" kind="ghost" className="order-detail-cancel" disabled={busy} onClick={()=>cancel(order)}>Cancelar pedido</Button>}
                      </div>}
                      {canCharge?<Link href={`/pos?orderId=${order.id}`} className="button primary salon-order-detail-pay" aria-disabled={busy} aria-busy={busy} tabIndex={busy?-1:undefined} onClick={event=>{if(busy)event.preventDefault();}}><Icon name="payment" size={18}/><span>Cobrar {currencySymbol} {money(remaining)}</span></Link>:action&&<Button icon={actionIcon} className="order-detail-primary" disabled={busy} aria-busy={busy} onClick={()=>advance(action.status)}>{action.label}</Button>}
                    </div>
                  </footer>
                )}
              </>
            )}
          </>
        )}
      </Dialog>
    </div>
  );
}

function OrderDetailSkeleton({close}:{close:()=>void}){
  return <>
    <header className="salon-order-detail-head salon-order-detail-skeleton-head">
      <h2 id="salon-order-detail-title" className="sr-only">Cargando detalle de la mesa</h2>
      <div className="salon-order-detail-identity" aria-hidden="true">
        <span className="salon-order-detail-skeleton-block salon-order-detail-skeleton-icon"/>
        <div className="salon-order-detail-skeleton-copy">
          <span className="salon-order-detail-skeleton-block title"/>
          <span className="salon-order-detail-skeleton-block medium"/>
        </div>
      </div>
      <span className="salon-order-detail-skeleton-block salon-order-detail-skeleton-status" aria-hidden="true"/>
      <button type="button" className="salon-order-detail-close" aria-label="Cerrar detalle" onClick={close}><Icon name="close" size={17}/></button>
    </header>

    <div className="order-detail-body salon-order-detail-body salon-order-detail-skeleton-body" aria-label="Cargando detalle de la mesa">
      <div className="salon-order-detail-content" aria-hidden="true">
      <section className="salon-order-detail-consumption salon-order-detail-skeleton-consumption">
        <div className="salon-order-detail-skeleton-section-head">
          <div>
            <span className="salon-order-detail-skeleton-block heading"/>
          </div>
          <span className="salon-order-detail-skeleton-block tiny"/>
        </div>
        <div className="order-detail-items salon-order-detail-items salon-order-detail-skeleton-items">
          {Array.from({length:3},(_,i)=><div className="salon-order-detail-skeleton-line" key={i}>
            <span className="salon-order-detail-skeleton-block salon-order-detail-skeleton-qty"/>
            <span className="salon-order-detail-skeleton-copy">
              <span className="salon-order-detail-skeleton-block line-title"/>
              <span className="salon-order-detail-skeleton-block medium"/>
            </span>
            <span className="salon-order-detail-skeleton-block price"/>
          </div>)}
        </div>
      </section>
    <section className="salon-order-detail-totals salon-order-detail-skeleton-totals" aria-hidden="true">
      <div className="salon-order-detail-subtotal">
        <span className="salon-order-detail-skeleton-block label"/>
        <span className="salon-order-detail-skeleton-block amount"/>
      </div>
      <div className="salon-order-detail-subtotal">
        <span className="salon-order-detail-skeleton-block label"/>
        <span className="salon-order-detail-skeleton-block amount"/>
      </div>
      <div className="salon-order-detail-grand">
        <span className="salon-order-detail-skeleton-block total-label"/>
        <span className="salon-order-detail-skeleton-block total-amount"/>
      </div>
    </section>
      </div>
    </div>

    <footer className="order-detail-actions salon-order-detail-actions salon-order-detail-skeleton-actions" aria-hidden="true">
      <div className="salon-order-detail-buttons">
        <span className="salon-order-detail-skeleton-block action secondary"/>
        <span className="salon-order-detail-skeleton-block action primary"/>
      </div>
    </footer>
  </>;
}

