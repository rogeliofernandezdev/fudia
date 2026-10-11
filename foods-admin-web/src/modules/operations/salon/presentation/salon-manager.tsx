"use client";

import {Dialog} from "@/design-system/dialog";
import "../../styles/salon-comanda.css";
import "../../styles/salon.css";
import {useState,useCallback} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,Icon,PageHeader,SearchField,Skeleton,Status} from "@/design-system";
import {ComandaView} from "../../orders/presentation/comanda-view";
import type {Draft,Order} from "../domain/types";
import {createSalonOrder,getSalonFloor,getSalonOrder,updateSalonOrder,updateSalonOrderStatus} from "../infrastructure/salon-api";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {useMoney} from "@/providers";

import {canManageOrderService} from "../../orders/domain/order-actions";
import {addOrderConsumption} from "../../orders/infrastructure/service-api";
import {OrderDetailContent,OrderDetailSkeleton,orderDetailTitleId} from "../../orders/presentation/order-detail-dialog";
import {isEditableOrder,parseIsoDate,statusMeta} from "../../orders/presentation/order-meta";

/* ── helpers ── */
function occupiedFor(iso:string){
  const d=parseIsoDate(iso);if(!d)return"—";
  const m=Math.max(0,Math.floor((Date.now()-d.getTime())/60000));
  if(m<60)return`${m} min`;return`${Math.floor(m/60)}h ${m%60}min`;
}
const emptyDraft=(tableId=""):Draft=>({channel:"salon",customerName:"",customerPhone:"",address:"",reference:"",tableId,notes:"",deliveryFee:"0",lines:[]});
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
  const{can,user}=useSession();
  const money=useMoney();
  const canManage=can("orders.manage");

  const[q,setQ]=useState("");
  const[detailId,setDetailId]=useState<string|null>(null);
  const[draft,setDraft]=useState<Draft|null>(null);
  const[editingOrderId,setEditingOrderId]=useState<string|null>(null);
  const[cancelTarget,setCancelTarget]=useState<Order|null>(null);
  const[zone,setZone]=useState("");
  const[additionKey,setAdditionKey]=useState<string|null>(null);

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

  const invalidate=useCallback(()=>Promise.all(["salon-floor","orders","order","pos-orders","pos-order","dashboard","kitchen-tickets","product-availability","products","sales"].map(key=>qc.invalidateQueries({queryKey:[key]}))),[qc]);

  const advance=useMutation({
    mutationFn:(v:{id:string;status:string})=>updateSalonOrderStatus(v.id,v.status),
    onSuccess:(order)=>{const refresh=invalidate();if(order.status!=="entregado")notify({tone:"success",title:"Pedido actualizado",message:"El estado fue actualizado correctamente."});return refresh},
    onError:e=>notify({tone:"danger",title:"No se pudo actualizar el pedido",message:e.message}),
  });
  const create=useMutation({
    mutationFn:({draft,sendToKitchen}:{draft:Draft;sendToKitchen:boolean})=>createSalonOrder(draft,sendToKitchen),
    onSuccess:(o,variables)=>{
      setDraft(null);setEditingOrderId(null);setDetailId(null);invalidate();
      notify({
        tone:"success",
        title:variables.sendToKitchen?"Comanda enviada":"Comanda guardada",
        message:variables.sendToKitchen?`Pedido ${o.code} registrado y enviado a su área de atención.`:`Pedido ${o.code} guardado, sin enviar a cocina.`,
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
    mutationFn:({id,v}:{id:string;v:Draft})=>additionKey?addOrderConsumption(id,v,additionKey):updateSalonOrder(id,v),
    onSuccess:o=>{
      setDraft(null);
      setEditingOrderId(null);
      setAdditionKey(null);
      setDetailId(null);
      invalidate();
      void qc.invalidateQueries({queryKey:["salon-floor"]});
      void qc.invalidateQueries({queryKey:["order",o.id]});
      notify({tone:"success",title:"Comanda actualizada",message:`Pedido ${o.code} guardado correctamente.`});
    },
    onError:e=>{
      void qc.invalidateQueries({queryKey:["salon-floor"]});
      void qc.invalidateQueries({queryKey:["order",editingOrderId]});
      void qc.invalidateQueries({queryKey:["order-catalog","salon"]});
      void qc.invalidateQueries({queryKey:["order-catalog","combos"]});
      notify({tone:"danger",title:"No se pudo guardar la comanda",message:e.message});
    },
  });
  const cancel=useMutation({
    mutationFn:(o:Order)=>updateSalonOrderStatus(o.id,"cancelado"),
    onSuccess:()=>{setCancelTarget(null);invalidate();setDetailId(null);notify({tone:"success",title:"Pedido cancelado",message:"La mesa quedó libre."})},
    onError:e=>notify({tone:"danger",title:"No se pudo cancelar el pedido",message:e.message}),
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
  const closeDraft=()=>{setDraft(null);setEditingOrderId(null);setAdditionKey(null)};
  const openNewDraft=(tableId:string)=>{setEditingOrderId(null);setAdditionKey(null);setDraft(emptyDraft(tableId))};
  const addProducts=(o:Order)=>{setAdditionKey(crypto.randomUUID());setEditingOrderId(o.id);setDraft({...emptyDraft(o.tableId),customerName:o.customerName});setDetailId(null)};
  const editOrder=(o:Order)=>{
    if(!canManage||!canManageOrderService(o,user?.id))return;
    if(!isEditableOrder(o))return;
    setEditingOrderId(o.id);
    setDraft(draftFromOrder(o));
    setDetailId(null);
  };

  const zoneTabs=[{value:"",label:"Todas"},...zones.map(z=>({value:z,label:z}))];
  const loading=detail.isLoading;
  const closeDetail=()=>setDetailId(null);
  const countSkeleton=<Skeleton width="var(--size-24)" height="var(--size-18)"/>;

  return(
    <>
      <PageHeader title="Salón" description="Consulta el estado de las mesas y gestiona las comandas activas desde un solo lugar."/>

      <section className="salon-overview-stats" aria-label="Resumen del salón">
        <div className="salon-overview-stat">
          <span className="salon-overview-stat-icon"><Icon name="grid" size={18}/></span>
          <span><small>Mesas</small><b>{floor.isLoading?countSkeleton:tables.length}</b></span>
        </div>
        <div className="salon-overview-stat free">
          <span className="salon-overview-stat-icon"><Icon name="check" size={18}/></span>
          <span><small>Libres</small><b>{floor.isLoading?countSkeleton:freeCount}</b></span>
        </div>
        <div className="salon-overview-stat occupied">
          <span className="salon-overview-stat-icon"><Icon name="receipt" size={18}/></span>
          <span><small>Ocupadas</small><b>{floor.isLoading?countSkeleton:occupiedCount}</b></span>
        </div>
      </section>

      <section className="salon-controls" aria-label="Controles del salón">
        <div className="salon-toolbar">
          <SearchField className="salon-search" label="Buscar mesas" value={q} onChange={e=>setQ(e.target.value)} placeholder="Buscar mesa, zona, pedido o familia…"/>
          <Button kind="secondary" icon="refresh" className="salon-refresh" onClick={()=>floor.refetch()} aria-label="Actualizar salón" disabled={floor.isFetching}>
            {floor.isFetching?"Actualizando…":"Actualizar"}
          </Button>
        </div>
        {zones.length>1&&<div className="cmd-chips salon-zones" role="group" aria-label="Filtrar por zona">{zoneTabs.map(item=><button type="button" key={item.value} className="cmd-chip" aria-pressed={zone===item.value} onClick={()=>setZone(item.value)}>{zone===item.value&&<span className="cmd-chip-check" aria-hidden="true"><Icon name="check" size={14}/></span>}{item.label}</button>)}</div>}
      </section>

      {/* Floor plan */}
      {floor.isLoading?(
        <div className="salon-grid" role="status" aria-busy="true" aria-label="Cargando mesas">
          {Array.from({length:8},(_,i)=><SalonTableSkeleton key={i}/>)}
        </div>
      ):floor.isError?(
        <ErrorState title="No pudimos cargar el salón" text={floor.error.message} onRetry={()=>floor.refetch()} retrying={floor.isFetching}/>
      ):!tables.length?(
        <EmptyState icon="utensils" title="Aún no hay mesas configuradas" text={<>Registra las mesas del restaurante en <strong>Mesas y zonas</strong> para empezar a tomar pedidos.</>}/>
      ):(
        <div className="salon-grid">
          {visible.map(t=>{
            const o=t.order;
            const meta=o?statusMeta[o.status]??{label:o.status,tone:"gray" as const}:null;
            const tableState=o?"occupied":"free";
            return(
              <button
                key={t.id}
                type="button"
                className={`salon-table ${tableState}`}
                onClick={()=>o?setDetailId(o.id):canManage&&openNewDraft(t.id)}
                disabled={!o&&!canManage}
                aria-label={`Mesa ${t.name}${o?`, ocupada, pedido ${o.code}`:", libre"}`}
              >
                <div className="salon-table-inner">
                  <div className="salon-table-top">
                    {o&&meta?<Status tone={meta.tone}>{meta.label}</Status>:<Status tone="green">Libre</Status>}
                    {o&&<span className="salon-table-elapsed"><Icon name="clock" size={14}/>{occupiedFor(o.createdAt)}</span>}
                  </div>

                  <div className="salon-table-name-row">
                    <span className="salon-table-icon">
                      <Icon name="utensils" size={18}/>
                    </span>
                    <strong className="salon-table-name">{t.name}</strong>
                    <small className="salon-table-seats">{t.seats} personas{zones.length>1&&t.zone?` · ${t.zone}`:""}</small>
                    {o?.customerName&&<span className="salon-table-family">{o.customerName}</span>}
                    {o&&<span className="salon-table-staff">Mozo: {o.waiterName||"Sin asignar"}</span>}
                    {Boolean(o?.collectedByNames?.length)&&<span className="salon-table-staff">Cobrado por: {o!.collectedByNames!.join(" · ")}</span>}
                  </div>

                  {o?(
                    <div className="salon-table-order">
                      <b className="salon-table-total">{money(o.total)}</b>
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
          {!visible.length&&<EmptyState className="salon-grid-empty" icon="search" title="Sin coincidencias" text="Ajusta la búsqueda o el filtro de zona."/>}
        </div>
      )}

      {/* Nueva / edición de comanda */}
      {draft&&(
        <ComandaView
          initial={draft}
          mode={additionKey?"append":editingOrderId?"edit":"create"}
          allTables={tables}
          busy={editingOrderId?update.isPending:create.isPending}
          close={closeDraft}
          save={async(v,sendToKitchen)=>{try{if(editingOrderId)await update.mutateAsync({id:editingOrderId,v});else await create.mutateAsync({draft:v,sendToKitchen})}catch{/* El error mantiene la comanda abierta para corregir o reintentar. */}}}
          notify={notify}
        />
      )}

      {/* Detalle del pedido */}
      {detailId&&(
        <div className="modal-backdrop modal-overlay-in">
          <Dialog onResponseClose={closeDetail} className="crud-modal order-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby={orderDetailTitleId} aria-busy={loading}>
            {loading?<OrderDetailSkeleton variant="table" close={closeDetail}/>:(
              <OrderDetailContent
                variant="table"
                order={detail.data}
                error={detail.error?.message}
                retry={()=>detail.refetch()}
                retrying={detail.isFetching}
                canManage={canManage}
                busy={advance.isPending}
                close={closeDetail}
                advance={st=>advance.mutate({id:detailId,status:st})}
                edit={editOrder}
                add={addProducts}
                cancel={o=>setCancelTarget(o)}
              />
            )}
          </Dialog>
        </div>
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

/** Tarjeta de mesa en carga: estado, tiempo, icono, nombre, personas y acción. */
function SalonTableSkeleton(){
  return(
    <div className="salon-table salon-table-skeleton" aria-hidden="true">
      <div className="salon-table-inner">
        <div className="salon-table-top">
          <Skeleton width="var(--size-72)" height="var(--size-20)" radius="var(--radius-pill)"/>
          <Skeleton width="var(--size-48)" height="var(--size-20)" radius="var(--radius-tile)"/>
        </div>
        <div className="salon-table-name-row">
          <Skeleton width="var(--size-36)" height="var(--size-36)" radius="var(--radius-pill)"/>
          <Skeleton width="var(--size-92)" height="var(--size-20)"/>
          <Skeleton width="var(--size-72)" height="var(--size-12)"/>
        </div>
        <Skeleton className="salon-table-skeleton-action" height="var(--size-40)" radius="var(--radius-card)"/>
      </div>
    </div>
  );
}
