"use client";

import {Dialog} from "@/design-system/dialog";
import "../../styles/salon-comanda.css";
import "../../styles/salon.css";
import {useState,useCallback,useEffect} from "react";
import {formatTableOccupation} from "../domain/occupation-time";
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
const tableOrder=new Intl.Collator("es",{numeric:true,sensitivity:"base"});
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
  const[clock,setClock]=useState(()=>Date.now());
  // Advance timers locally even if a background request is delayed; no page reload.
  useEffect(()=>{
    const timer=window.setInterval(()=>setClock(Date.now()),30000);
    return()=>window.clearInterval(timer);
  },[]);

  const floor=useQuery({
    queryKey:["salon-floor"],
    queryFn:getSalonFloor,
    // React Query refreshes only this cached resource, never the entire page.
    refetchInterval:10000,
    refetchIntervalInBackground:false,
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

  // El API agrupa por zona; sin encabezados de zona eso parece desordenado. Orden natural por número de mesa (01, 02… 10).
  const tables=[...floor.data?.items??[]].sort((a,b)=>tableOrder.compare(a.name,b.name));
  const zones=[...new Set(tables.map(t=>t.zone).filter(Boolean))];
  const visible=tables.filter(t=>{
    const matchZone=!zone||t.zone===zone;
    const needle=q.trim().toLowerCase();
    const matchQ=!needle||t.name.toLowerCase().includes(needle)||t.zone.toLowerCase().includes(needle)||(t.order?.code??"").toLowerCase().includes(needle)||(t.order?.customerName??"").toLowerCase().includes(needle);
    return matchZone&&matchQ;
  });
  const occupiedCount=tables.filter(t=>t.order).length;
  const freeCount=tables.length-occupiedCount;
  const openTotal=tables.reduce((sum,t)=>sum+Number(t.order?.total??0),0);
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

      {/* KPI al estilo de la referencia: etiqueta pequeña arriba (en el tono de su estado) y cifra con icono de línea. */}
      <dl className="salon-overview-stats" aria-label="Resumen del salón">
        <div><dt>Mesas</dt><dd><Icon name="grid" size={20}/>{floor.isLoading?countSkeleton:tables.length}</dd></div>
        <div className="free"><dt>Libres</dt><dd><Icon name="check" size={20}/>{floor.isLoading?countSkeleton:freeCount}</dd></div>
        <div className="occupied"><dt>Ocupadas</dt><dd><Icon name="utensils" size={20}/>{floor.isLoading?countSkeleton:occupiedCount}</dd></div>
        <div><dt>Consumo abierto</dt><dd><Icon name="wallet" size={20}/>{floor.isLoading?countSkeleton:money(openTotal)}</dd></div>
      </dl>

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
      ):floor.isError&&!floor.data?(
        <ErrorState title="No pudimos cargar el salón" text={floor.error.message} onRetry={()=>floor.refetch()} retrying={floor.isFetching}/>
      ):!tables.length?(
        <EmptyState icon="utensils" title="Aún no hay mesas configuradas" text={<>Registra las mesas del restaurante en <strong>Mesas y zonas</strong> para empezar a tomar pedidos.</>}/>
      ):(
        <div className="salon-grid">
          {visible.map(t=>{
            const o=t.order;
            const meta=o?statusMeta[o.status]??{label:o.status,tone:"gray" as const}:null;
            const elapsed=o?formatTableOccupation(t.occupiedSince,clock):null;
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
                {/* Ficha simple (referencia): número y estado, una línea de datos y pie con importe y acción. */}
                <header className="salon-table-head">
                  <strong className="salon-table-name">{t.name.replace(/^mesa\s*/i,"")||t.name}</strong>
                  {o&&meta?<Status tone={meta.tone} emphasis>{meta.label}</Status>:<Status tone="navy" emphasis>Libre</Status>}
                </header>

                <p className="salon-table-meta">
                  <span>{t.seats} {t.seats===1?"persona":"personas"}{zones.length>1&&t.zone?` · ${t.zone}`:""}</span>
                  {o&&<span>Mozo: {o.waiterName||"Sin asignar"}</span>}
                </p>

                <footer className="salon-table-foot">
                  {o?<span className="salon-table-amount"><b className="salon-table-total">{money(o.total)}</b>{elapsed?.known&&<small className={`salon-table-elapsed${elapsed.needsReview?" is-stale":""}`} title={elapsed.description}>Hace {elapsed.label}</small>}</span>
                  :<b className="salon-table-open">Abrir mesa</b>}
                  <span className="salon-table-go" aria-hidden="true"><Icon name={o?"eye":"plus"} size={18}/></span>
                </footer>
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
      <div className="salon-table-head">
        <Skeleton width="var(--size-48)" height="var(--size-24)"/>
        <Skeleton width="var(--size-72)" height="var(--size-20)" radius="var(--radius-pill)"/>
      </div>
      <div className="salon-table-meta">
        <Skeleton width="var(--size-120)" height="var(--size-10)"/>
        <Skeleton width="var(--size-92)" height="var(--size-10)"/>
      </div>
      <div className="salon-table-foot">
        <Skeleton width="var(--size-86)" height="var(--size-16)"/>
        <Skeleton width="var(--size-44)" height="var(--size-44)" radius="var(--radius-pill)"/>
      </div>
    </div>
  );
}
