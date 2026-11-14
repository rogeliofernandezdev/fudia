"use client";
import {useQuery} from "@tanstack/react-query";
import {DataTable} from "@/design-system/data-table";
import {EmptyState,ErrorState,Icon,InlineNotice,TableSkeleton} from "@/design-system";
import {useSession} from "@/providers";
import {DeliveryPartnersAdmin} from "@/modules/platform/presentation/delivery-partners-admin";
import {formatPartnerPhone} from "../domain/board";
import {getDeliveryProviders} from "../infrastructure/delivery-api";

const columns=["Central","Teléfono"];

/**
 * Delivery › Proveedores. Plataforma (solo Fudia) administra aquí las centrales;
 * el restaurante ve en solo lectura las que cubren el distrito de su local.
 */
export function DeliveryProviders(){
  const{user}=useSession();
  if(user?.platformAdmin)return <DeliveryPartnersAdmin/>;
  return <LocationProviders/>;
}

function LocationProviders(){
  const providers=useQuery({queryKey:["delivery-providers"],queryFn:getDeliveryProviders});
  const data=providers.data;
  const items=data?.items??[];
  return <section className="panel management standardized-management delivery-providers" aria-label="Proveedores de delivery">
    {providers.isLoading?<TableSkeleton columns={columns} rows={3} label="Cargando proveedores de delivery"/>:
      providers.isError?<ErrorState title="No pudimos cargar los proveedores" text={providers.error.message} retrying={providers.isFetching} onRetry={()=>void providers.refetch()}/>:
      data&&<>
        <p className="delivery-providers-intro">{data.enabled
          ?`Cuando cocina inicia la preparación, el pedido se ofrece a la vez a estas centrales${data.districtName?` de ${data.districtName}`:""}; la primera que acepta lo recoge. Fudia las administra.`
          :"Delivery Fudia no está activo para tu empresa: repartes con tu delivery propio. Fudia lo activa desde su plataforma."}</p>
        {data.enabled&&!data.districtName&&<InlineNotice tone="warning" title="Falta el distrito del local">Sin distrito ninguna central recibe los pedidos. Complétalo en Locales.</InlineNotice>}
        {!items.length?<EmptyState icon="motorbike" title="Sin centrales para tu local" text={data.enabled?"Ninguna central activa cubre el distrito del local. Los pedidos quedan para tu delivery propio.":"Las centrales aparecen aquí cuando Fudia active Delivery Fudia para tu empresa."}/>:
          <DataTable label="Proveedores de delivery" containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>
            {items.map((item,index)=><tr className={index%2?"alternate":""} key={`${item.name}-${item.whatsappPhone}`}>
              <td><span className={`row-icon r${index%3}`}><Icon name="motorbike" size={18}/></span><b>{item.name}</b></td>
              <td>{item.whatsappPhone?<a href={`tel:+${item.whatsappPhone}`}>{formatPartnerPhone(item.whatsappPhone)}</a>:"—"}</td>
            </tr>)}
          </tbody></DataTable>}
      </>}
  </section>;
}
