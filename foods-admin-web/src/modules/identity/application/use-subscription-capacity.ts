"use client";
import {useSession} from "@/providers/session-context";
import {useOrganizationSubscription} from "./use-organization-subscription";

export function useSubscriptionCapacity(resource:"locations"|"users"){
 const{can}=useSession();
 const subscription=useOrganizationSubscription();
 const canRead=can("subscription.read");
 const data=subscription.data;
 const max=resource==="locations"?data?.plan.maxLocations:data?.plan.maxUsers;
 const plural=resource==="locations"?"locales":"usuarios";
 const singular=resource==="locations"?"local":"usuario";
 const reached=Boolean(data&&max!=null&&data.usage[resource]>=max);
 const blocked=!canRead||!data||subscription.isError||subscription.isFetching||reached;
 const reason=!canRead?`No tienes permiso para consultar el límite de ${plural}.`:subscription.isError?`No pudimos comprobar el límite de ${plural}. Reintenta la consulta.`:!data?`Comprobando el límite de ${plural}…`:reached?`Tu plan incluye ${max} ${max===1?`${singular} activo`:`${plural} activos`}. Ya alcanzaste ese límite.`:"";
 return {subscription,canRead,blocked,reason};
}
