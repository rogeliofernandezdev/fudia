"use client";
import {useQuery} from "@tanstack/react-query";
import {DataTable} from "@/design-system/data-table";
import {EmptyState,ErrorState,Icon,TableSkeleton,Tag} from "@/design-system";
import {useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import type {LocationFailedDelivery} from "../domain/types";
import {getLocationFailedDeliveries} from "../infrastructure/delivery-api";

const columns=["Pedido","Cliente","Central","Comida","Reembolso","Central asume","Fudia asume","Aporte Fudia"];

/**
 * Delivery › Entregas fallidas: incidencias que registró la central (el
 * motorizado no ubicó al cliente). La central reembolsa al motorizado el 100 %
 * de la comida y la pérdida se reparte según la validación y los límites.
 */
export function FailedDeliveries(){
  const{location}=useSession();
  const failed=useQuery({queryKey:["delivery-failed"],queryFn:getLocationFailedDeliveries});
  const items=failed.data?.items??[];
  const date=(value:string)=>formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone});
  const money=(item:LocationFailedDelivery,amount:string)=>`${item.currencySymbol} ${amount}`.trim();
  const share=(item:LocationFailedDelivery)=><Tag icon={item.fudiaShareEligible?"circleCheck":"circleMinus"}>{item.fudiaShareEligible?"Con aporte Fudia":"Sin aporte Fudia"}</Tag>;
  const notes=(item:LocationFailedDelivery)=>item.validationNotes.length>0&&<ul className="delivery-failed-notes" aria-label="Validación de la incidencia">{item.validationNotes.map(note=><li key={note}>{note}</li>)}</ul>;
  return <section className="panel management standardized-management delivery-failed" aria-label="Entregas fallidas">
    {failed.isLoading?<TableSkeleton columns={columns} rows={3} label="Cargando entregas fallidas"/>:
      failed.isError?<ErrorState title="No pudimos cargar las entregas fallidas" text={failed.error.message} retrying={failed.isFetching} onRetry={()=>void failed.refetch()}/>:
      !items.length?<EmptyState icon="circleCheck" title="Sin entregas fallidas" text="Aquí verás los deliveries que la central no pudo entregar en los últimos 90 días."/>:
      <DataTable label="Entregas fallidas" containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>
        {items.map((item,index)=><tr className={index%2?"alternate":""} key={`${item.orderCode}-${item.registeredAt}`}>
          <td><div className="delivery-board-order"><b>{item.orderCode}</b><small>{date(item.registeredAt)}</small></div></td>
          <td><div className="delivery-board-customer"><b>{item.customerName||"Sin nombre"}</b><small>{item.address||"Dirección por confirmar"}</small></div></td>
          <td><div className="delivery-board-owner"><b>{item.partnerName}</b>{item.courierName&&<small><Icon name="motorbike" size={14}/>{item.courierName}</small>}</div></td>
          <td className="tabular">{money(item,item.foodAmount)}</td>
          <td className="tabular">{money(item,item.courierReimbursement)}</td>
          <td className="tabular">{money(item,item.partnerShare)}</td>
          <td className="tabular">{money(item,item.fudiaShare)}</td>
          <td><div className="delivery-failed-share">{share(item)}{notes(item)}</div></td>
        </tr>)}
      </tbody></DataTable>}
  </section>;
}
