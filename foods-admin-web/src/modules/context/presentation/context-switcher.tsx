"use client";
import "./context-switcher.css";
import {useState,useEffect,useRef,useId} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Icon} from "@/design-system/icons";
import {useSession} from "@/providers/session-context";
import {useFeedback} from "@/providers/feedback-provider";
import {listOrganizations,listOrgLocations,listAvailableLocations,switchContext} from "../infrastructure/context-api";
import {loadSessionContext} from "@/shared/session/session-api";
import {firstAccessibleRoute} from "@/shell/navigation";
import {ContextSwitcherPanel} from "./context-switcher-panel";

export function ContextSwitcher(){
  const{user,organization,location}=useSession();
  const queryClient=useQueryClient();
  const{notify}=useFeedback();
  const[open,setOpen]=useState(false);
  const[orgSel,setOrgSel]=useState("");
  const[locSel,setLocSel]=useState("");
  const ref=useRef<HTMLDivElement>(null);
  const triggerRef=useRef<HTMLButtonElement>(null);
  const panelRef=useRef<HTMLElement>(null);
  const focusOnOpen=useRef(false);
  const panelId=useId();

  const orgId=orgSel||organization?.id||"";
  const locId=locSel||(orgSel&&orgSel!==organization?.id?"":location?.id||"");

  const orgs=useQuery({queryKey:["organizations"],queryFn:listOrganizations,enabled:open&&!!user?.platformAdmin});
  const orgLocations=useQuery({queryKey:["org-locations",orgId],queryFn:()=>listOrgLocations(orgId),enabled:open&&!!orgId&&!!user?.platformAdmin});
  const myLocations=useQuery({queryKey:["available-locations"],queryFn:listAvailableLocations,enabled:!!user&&!user.platformAdmin});
  const locations=user?.platformAdmin?orgLocations:myLocations;
  const locOptions=locations.data?.items??[];
  const loading=locations.isLoading||Boolean(user?.platformAdmin&&orgs.isLoading);
  const error=user?.platformAdmin?(orgs.error??orgLocations.error):myLocations.error;
  const changed=orgId!==organization?.id||locId!==location?.id;
  const canApply=changed&&!loading&&!error&&Boolean(orgId&&locOptions.some(option=>option.id===locId)&&(user?.platformAdmin?orgs.data?.items.some(option=>option.id===orgId):orgId===organization?.id));

  const switchMut=useMutation({
    mutationFn:switchContext,
    onSuccess:async(data)=>{
      setOpen(false);
      notify({tone:"success",title:"Cambio aplicado",message:`Ahora operas en ${data.organization.name} · ${data.location.name}.`});
      queryClient.clear();
      try{
        const context=await loadSessionContext();
        queryClient.setQueryData(["session-context"],context);
        window.location.assign(firstAccessibleRoute(context));
      }catch{
        window.location.reload();
      }
    },
    onError:(e:Error)=>notify({tone:"danger",title:"No se pudo cambiar",message:e.message})
  });

  useEffect(()=>{
    if(!open)return;
    const close=(e:MouseEvent)=>{if(!switchMut.isPending&&!ref.current?.contains(e.target as Node))setOpen(false)};
    document.addEventListener("mousedown",close);
    return()=>document.removeEventListener("mousedown",close);
  },[open,switchMut.isPending]);

  useEffect(()=>{
    if(open&&!loading&&focusOnOpen.current){
      const target=panelRef.current?.querySelector<HTMLElement>("select:not(:disabled)")??panelRef.current?.querySelector<HTMLElement>("button:not(:disabled)");
      target?.focus();
      focusOnOpen.current=false;
    }
  },[open,loading]);

  if(!user)return null;

  function doSwitch(){
    if(!canApply||switchMut.isPending)return;
    switchMut.mutate({organizationId:orgId,locationId:locId});
  }

  const hasMultipleLocs=locOptions.length>1;
  const canSwitch=user.platformAdmin||hasMultipleLocs||myLocations.isLoading||myLocations.isError;
  const closePanel=()=>{if(switchMut.isPending)return;setOpen(false);triggerRef.current?.focus()};

  return <div className="context-switcher" ref={ref}>
    <button ref={triggerRef} type="button" className="context-btn" data-open={open} onClick={()=>{if(!canSwitch||switchMut.isPending)return;if(open){closePanel();return}setOrgSel("");setLocSel("");focusOnOpen.current=true;setOpen(true)}} disabled={!canSwitch||switchMut.isPending} aria-label={user.platformAdmin?`Cambiar empresa y local: ${organization?.name??"sin empresa seleccionada"}`:`Cambiar local: ${location?.name??"sin local seleccionado"}`} aria-haspopup="dialog" aria-expanded={open} aria-controls={open?panelId:undefined}>
      <Icon name="store" size={18}/>
      <span>
        <small>LOCAL ACTIVO</small><b>{location?.name??"Sin local seleccionado"}</b>
      </span>
      {canSwitch&&<Icon name="chevron" size={15}/>}
    </button>
    {open&&<ContextSwitcherPanel panelRef={panelRef} id={panelId} platformAdmin={user.platformAdmin} organizationId={orgId} locationId={locId} organizations={orgs.data?.items??[]} locations={locOptions} loading={loading} error={error?.message} busy={switchMut.isPending} canApply={canApply} onOrganizationChange={id=>{setOrgSel(id);setLocSel("")}} onLocationChange={setLocSel} onApply={doSwitch} onClose={closePanel} onRetry={()=>{if(user.platformAdmin){void orgs.refetch();if(orgId)void orgLocations.refetch()}else void myLocations.refetch()}}/>}
  </div>;
}
