"use client";
import "./context-switcher.css";
import {useState,useEffect,useRef} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Icon} from "@/design-system/icons";
import {useSession} from "@/providers/session-context";
import {useFeedback} from "@/providers/feedback-provider";
import {listOrganizations,listOrgLocations,listAvailableLocations,switchContext} from "../infrastructure/context-api";

export function ContextSwitcher(){
  const{user,organization,location}=useSession();
  const queryClient=useQueryClient();
  const{notify}=useFeedback();
  const[open,setOpen]=useState(false);
  const[orgSel,setOrgSel]=useState("");
  const[locSel,setLocSel]=useState("");
  const ref=useRef<HTMLDivElement>(null);

  const orgId=orgSel||organization?.id||"";
  const locId=locSel||(orgSel&&orgSel!==organization?.id?"":location?.id||"");

  const orgs=useQuery({queryKey:["organizations"],queryFn:listOrganizations,enabled:!!user?.platformAdmin});
  const orgLocations=useQuery({queryKey:["org-locations",orgId],queryFn:()=>listOrgLocations(orgId),enabled:!!orgId&&!!user?.platformAdmin});
  const myLocations=useQuery({queryKey:["available-locations"],queryFn:listAvailableLocations,enabled:!user?.platformAdmin});

  const switchMut=useMutation({
    mutationFn:switchContext,
    onSuccess:(data)=>{setOpen(false);notify({tone:"success",title:"Cambio aplicado",message:`Ahora operas en ${data.organization.name} · ${data.location.name}.`});queryClient.clear();queryClient.invalidateQueries({queryKey:["session-context"]});window.location.reload()},
    onError:(e:Error)=>notify({tone:"danger",title:"No se pudo cambiar",message:e.message})
  });

  useEffect(()=>{
    if(!open)return;
    const close=(e:MouseEvent)=>{if(!ref.current?.contains(e.target as Node))setOpen(false)};
    document.addEventListener("mousedown",close);
    return()=>document.removeEventListener("mousedown",close);
  },[open]);

  if(!user)return null;

  function doSwitch(){
    if(user?.platformAdmin){
      if(!orgId||!locId)return;
      switchMut.mutate({organizationId:orgId,locationId:locId});
    }else{
      if(!locId||locId===location?.id)return;
      switchMut.mutate({organizationId:organization?.id??"",locationId:locId});
    }
  }

  const locOptions=user?.platformAdmin?(orgLocations.data?.items??[]):myLocations.data?.items??[];
  const hasMultipleLocs=locOptions.length>1;
  const canSwitch=user?.platformAdmin||hasMultipleLocs;

  return <div className="context-switcher" ref={ref}>
    <button className="context-btn" onClick={()=>canSwitch&&setOpen(!open)} disabled={!canSwitch} aria-label={user?.platformAdmin?"Cambiar de empresa":"Cambiar de local"}>
      <Icon name="store" size={18}/>
      <span>
        {user?.platformAdmin&&organization&&<><small>EMPRESA</small><b>{organization.name}</b></>}
        {!user?.platformAdmin&&location&&<><small>LOCAL ACTIVO</small><b>{location.name}</b></>}
      </span>
      {canSwitch&&<Icon name="chevron" size={15}/>}
    </button>
    {open&&<div className="context-popover">
      {user?.platformAdmin&&<>
        <label className="context-field">
          <small>EMPRESA</small>
          <select className="ds-select" value={orgId} onChange={e=>{setOrgSel(e.target.value);setLocSel("")}}>
            {orgs.data?.items.map(o=><option key={o.id} value={o.id}>{o.name}</option>)}
          </select>
        </label>
        <label className="context-field">
          <small>LOCAL</small>
          <select className="ds-select" value={locId} onChange={e=>setLocSel(e.target.value)}>
            <option value="">Selecciona un local</option>
            {orgLocations.data?.items.map(l=><option key={l.id} value={l.id}>{l.name}</option>)}
          </select>
        </label>
      </>}
      {!user?.platformAdmin&&<>
        <label className="context-field">
          <small>CAMBIAR LOCAL</small>
          <select className="ds-select" value={locId} onChange={e=>setLocSel(e.target.value)}>
            {myLocations.data?.items.map(l=><option key={l.id} value={l.id}>{l.name}</option>)}
          </select>
        </label>
      </>}
      <button className="context-apply" onClick={doSwitch} disabled={switchMut.isPending||!locId||(user?.platformAdmin&&!orgId)}>
        {switchMut.isPending?"Cambiando...":user?.platformAdmin?"Cambiar de empresa":"Cambiar de local"}
      </button>
    </div>}
  </div>;
}
