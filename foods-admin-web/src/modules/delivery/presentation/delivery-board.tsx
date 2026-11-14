"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {DataTable} from "@/design-system/data-table";
import {ConfirmDialog,EmptyState,ErrorState,Icon,IconButton,Select,Status,TableSkeleton} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {orderStatus} from "@/modules/operations/orders/presentation/order-meta";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {deliveryIssue,deliveryMode,filterBoard,formatPartnerPhone,pickupAlertLabels,waitMinutes,type BoardFilter} from "../domain/board";
import {canRetryDispatch,canUseOwnDelivery,dispatchStatusMeta,isDispatchStatus} from "../domain/dispatch-status";
import type {DeliveryBoardOrder} from "../domain/types";
import {getDeliveryBoard,retryDeliveryDispatch,switchToOwnDelivery} from "../infrastructure/delivery-api";

const POLL_MS=10_000;
const columns=["Pedido","Cliente","Estado","Reparto","Responsable","Recogida","Total","Acciones"];
export const deliveryBoardKey=["delivery-board"];

/**
 * Delivery › Pedidos: tablero en vivo de los deliveries del local, propios y de
 * Delivery Fudia. Caja reintenta las centrales o pasa el pedido a su delivery
 * propio desde aquí; el detalle del pedido solo resume el despacho.
 */
export function DeliveryBoard(){
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const client=useQueryClient();
  const canManage=can("orders.manage");
  const[filter,setFilter]=useState<BoardFilter>("active");
  const[ownTarget,setOwnTarget]=useState<DeliveryBoardOrder|null>(null);
  const board=useQuery({queryKey:deliveryBoardKey,queryFn:getDeliveryBoard,refetchInterval:POLL_MS});
  const refresh=()=>{for(const key of [deliveryBoardKey[0],"order","orders","staff-notices"])void client.invalidateQueries({queryKey:[key]});};
  const retry=useMutation({
    mutationFn:(order:DeliveryBoardOrder)=>retryDeliveryDispatch(order.id),
    onSuccess:()=>{refresh();notify({tone:"success",title:"Pedido ofrecido de nuevo",message:"Las centrales con cobertura lo reciben ahora en su panel de reparto."});},
    onError:e=>notify({tone:"danger",title:"No se pudo reintentar",message:e.message}),
  });
  const own=useMutation({
    mutationFn:(order:DeliveryBoardOrder)=>switchToOwnDelivery(order.id),
    onSuccess:()=>{setOwnTarget(null);refresh();notify({tone:"success",title:"Delivery propio",message:"Ninguna central recogerá este pedido; despáchalo con tu repartidor."});},
    onError:e=>{setOwnTarget(null);notify({tone:"danger",title:"No se pudo cambiar",message:e.message});},
  });
  const all=board.data?.items??[];
  const items=filterBoard(all,filter);
  const busy=retry.isPending||own.isPending;
  const time=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{timeStyle:"short"});
  const money=(order:DeliveryBoardOrder)=>`${order.currencySymbol} ${order.total}`.trim();

  const orderState=(order:DeliveryBoardOrder)=>{const meta=orderStatus(order.status);return <Status tone={meta.tone} solid>{meta.label}</Status>;};
  const dispatch=(order:DeliveryBoardOrder)=>{
    if(!isDispatchStatus(order.dispatchStatus))return <span className="delivery-board-muted">{deliveryMode(order)}</span>;
    const meta=dispatchStatusMeta[order.dispatchStatus];
    return <Status tone={meta.tone} icon={meta.icon}>{meta.label}</Status>;
  };
  const owner=(order:DeliveryBoardOrder)=>{
    const phone=formatPartnerPhone(order.partnerPhone);
    if(!order.partnerName&&!order.courierName)return <span className="delivery-board-muted">—</span>;
    return <div className="delivery-board-owner">
      {order.partnerName&&<b>{order.partnerName}</b>}
      {phone&&<a href={`tel:+${order.partnerPhone}`}>{phone}</a>}
      {order.courierName&&<small><Icon name="motorbike" size={14}/>{order.courierName}</small>}
    </div>;
  };
  const pickup=(order:DeliveryBoardOrder)=>{
    const issue=deliveryIssue(order);
    if(issue)return <Status tone={order.failedDelivery?"red":"orange"} icon="alert">{issue}</Status>;
    if(order.pickupAlert){const meta=pickupAlertLabels[order.pickupAlert];return <Status tone={meta.tone} icon="clock">{`${meta.label} · ${waitMinutes(order.pickupWaitSeconds)}`}</Status>;}
    if(order.pickedUpAt)return <span className="delivery-board-muted">Recogido {time(order.pickedUpAt)}</span>;
    return <span className="delivery-board-muted">—</span>;
  };
  const actions=(order:DeliveryBoardOrder)=>{
    const status=order.dispatchStatus;
    if(!canManage||!isDispatchStatus(status))return null;
    return <>
      {canRetryDispatch(status)&&<IconButton icon="refresh" label={`Reintentar centrales para ${order.code}`} disabled={busy} onClick={()=>retry.mutate(order)}/>}
      {canUseOwnDelivery(status)&&<IconButton icon="store" label={`Usar delivery propio en ${order.code}`} disabled={busy} onClick={()=>setOwnTarget(order)}/>}
    </>;
  };

  return <section className="panel management standardized-management delivery-board" aria-label="Pedidos de delivery">
    <div className="toolbar">
      <Select chip aria-label="Filtrar pedidos de delivery" value={filter} onChange={e=>setFilter(e.target.value as BoardFilter)}>
        <option value="active">En curso</option>
        <option value="today">Todos de hoy</option>
      </Select>
    </div>
    {board.isLoading?<TableSkeleton columns={columns} rows={4} label="Cargando pedidos de delivery"/>:
      board.isError?<ErrorState title="No pudimos cargar los pedidos de delivery" text={board.error.message} retrying={board.isFetching} onRetry={()=>void board.refetch()}/>:
      !items.length?<EmptyState icon="truck" title={filter==="active"&&all.length?"Sin deliveries en curso":"Aún no hay deliveries hoy"} text={filter==="active"&&all.length?"Los deliveries entregados o cerrados siguen en «Todos de hoy».":"Los pedidos de delivery del local aparecen aquí al instante."}/>:
      <DataTable label="Pedidos de delivery" containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>
        {items.map((order,index)=><tr className={index%2?"alternate":""} key={order.id}>
          <td><div className="delivery-board-order"><b>{order.code}</b><small>{time(order.createdAt)}</small></div></td>
          <td><div className="delivery-board-customer"><b>{order.customerName||"Sin nombre"}</b><small>{order.address||"Dirección por confirmar"}</small></div></td>
          <td>{orderState(order)}</td>
          <td>{dispatch(order)}</td>
          <td>{owner(order)}</td>
          <td>{pickup(order)}</td>
          <td className="tabular">{money(order)}</td>
          <td><div className="table-actions">{actions(order)}</div></td>
        </tr>)}
      </tbody></DataTable>}
    <ConfirmDialog open={Boolean(ownTarget)} tone="danger" title="Usar delivery propio" subject={ownTarget?{label:"Pedido",name:ownTarget.code}:undefined}
      description={ownTarget?.dispatchStatus==="accepted"?"Ninguna central recogerá este pedido y la que lo aceptó recibirá el aviso. Despáchalo con tu repartidor.":"Ninguna central recogerá este pedido. Despáchalo con tu repartidor."}
      confirmLabel="Usar propio" pending={own.isPending} onCancel={()=>setOwnTarget(null)} onConfirm={()=>ownTarget&&own.mutate(ownTarget)}/>
  </section>;
}
