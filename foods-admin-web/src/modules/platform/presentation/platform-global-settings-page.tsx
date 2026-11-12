"use client";
import {DataTable} from "@/design-system/data-table";

import "./platform-global-settings.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,Icon,PageHeader,RowActionButton,Status,TableSkeleton} from "@/design-system";
import {useFeedback} from "@/providers";
import type {PlatformWhatsAppChannel,PlatformWhatsAppChannelDraft} from "../domain/types";
import {getPlatformGlobalCatalogs,listPlatformWhatsAppChannels,savePlatformWhatsAppChannel} from "../infrastructure/platform-api";
import {WhatsAppChannelEditor,connectionLabel} from "./whatsapp-channel-editor";
import {ConciergeUsageSettingsPanel} from "./platform-concierge-usage";

const blank:PlatformWhatsAppChannelDraft={countryCode:"PE",phoneNumber:"",displayName:"",active:false,provider:"META_CLOUD_API"};
const columns=["País","Moneda","Nombre","Número","Proveedor","Conexión","Estado","Acciones"];
const toDraft=(item:PlatformWhatsAppChannel):PlatformWhatsAppChannelDraft=>({countryCode:item.countryCode,phoneNumber:item.phoneNumber,displayName:item.displayName,active:item.active,provider:item.provider??"META_CLOUD_API",revision:item.revision});

export function PlatformGlobalSettingsPage(){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const catalogs=useQuery({queryKey:["platform-global-catalogs"],queryFn:getPlatformGlobalCatalogs});
 const channels=useQuery({queryKey:["platform-whatsapp-channels"],queryFn:listPlatformWhatsAppChannels});
 const[draft,setDraft]=useState<PlatformWhatsAppChannelDraft|null>(null);
 const[editingId,setEditingId]=useState<string|undefined>();
 const[target,setTarget]=useState<PlatformWhatsAppChannel|null>(null);
 const save=useMutation({
  mutationFn:({value,id}:{value:PlatformWhatsAppChannelDraft;id?:string})=>savePlatformWhatsAppChannel(value,id),
  onSuccess:()=>{
   setDraft(null);setEditingId(undefined);
   void client.invalidateQueries({queryKey:["platform-whatsapp-channels"]});
   void client.invalidateQueries({queryKey:["platform-onboarding-context"]});
   notify({tone:"success",title:"Canal guardado",message:"La disponibilidad del país para onboarding se recalculará según el estado del canal."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const toggle=useMutation({
  mutationFn:(item:PlatformWhatsAppChannel)=>savePlatformWhatsAppChannel({...toDraft(item),active:!item.active},item.id),
  onSuccess:(_,item)=>{
   setTarget(null);
   void client.invalidateQueries({queryKey:["platform-whatsapp-channels"]});
   void client.invalidateQueries({queryKey:["platform-onboarding-context"]});
   notify({tone:"success",title:item.active?"Canal desactivado":"Canal activado",message:item.active?"El país dejó de estar disponible para registrar empresas.":"El país quedó disponible para registrar empresas."});
  },
  onError:e=>notify({tone:"danger",title:"No se pudo actualizar",message:e.message}),
 });
 const edit=(item:PlatformWhatsAppChannel)=>{
  setEditingId(item.id);
  setDraft(toDraft(item));
 };
 const start=()=>{
  const country=catalogs.data?.countryOptions.find(item=>!(channels.data?.items??[]).some(channel=>channel.countryCode===item.code));
  setEditingId(undefined);
  setDraft(country?{...blank,countryCode:country.code,displayName:country.name,phoneNumber:country.callingCode}:{...blank});
 };
 const items=channels.data?.items??[];
 const providers=channels.data?.providerOptions??[];
 const providerName=(item:PlatformWhatsAppChannel)=>providers.find(option=>option.value===item.provider)?.label??"Meta Cloud API";
 const currencyOf=(code:string)=>catalogs.data?.countryOptions.find(country=>country.code===code)?.defaultCurrency??"—";
 const status=(item:PlatformWhatsAppChannel)=><Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status>;
 const actions=(item:PlatformWhatsAppChannel)=><><RowActionButton action="edit" label={`Editar canal ${item.displayName}`} onClick={()=>edit(item)}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado del canal ${item.displayName}`} onClick={()=>setTarget(item)}/></>;

 return <>
  <PageHeader title="Países y WhatsApp" description="Un canal de atención por país: Meta, whatsmeow o chat web de Fudia." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>
  <section className="panel management standardized-management platform-channels-panel">
   {(catalogs.isLoading||channels.isLoading)?<TableSkeleton columns={columns} rows={4} label="Cargando canales de WhatsApp"/>:
    (catalogs.isError||channels.isError)?<ErrorState title="No pudimos cargar la configuración global" text={(catalogs.error??channels.error)?.message} retrying={catalogs.isFetching||channels.isFetching} onRetry={()=>{void catalogs.refetch();void channels.refetch()}}/>:
    !items.length?<EmptyState icon="chat" title="Aún no hay canales" text="Registra el número de WhatsApp de un país para habilitarlo en el registro de empresas." action={<Button icon="plus" onClick={start}>Nuevo canal</Button>}/>:
    <><DataTable label="Canales de WhatsApp por país" containerClassName="hover-scroll"><thead><tr><th>País</th><th>Moneda</th><th>Nombre</th><th>Número</th><th>Proveedor</th><th>Conexión</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>
     {items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}><td><span className={`row-icon r${index%3}`}><Icon name="chat" size={18}/></span><b>{item.countryName}</b><small>{item.countryCode}</small></td><td>{currencyOf(item.countryCode)}</td><td>{item.displayName}</td><td>{item.phoneNumber||"No requiere número"}</td><td>{providerName(item)}</td><td>{item.needsReconciliation?"Requiere verificación":connectionLabel(item.connectionState)}</td><td>{status(item)}</td><td><div className="table-actions">{actions(item)}</div></td></tr>)}
    </tbody></DataTable>
    <div className="management-cards">{items.map(item=><article key={item.id}><header><span className="row-icon r0"><Icon name="chat"/></span><div><b>{item.countryName}</b><small>{item.countryCode}</small></div>{status(item)}</header><dl><div><dt>Nombre</dt><dd>{item.displayName}</dd></div><div><dt>Número</dt><dd>{item.phoneNumber||"No requiere número"}</dd></div><div><dt>Moneda</dt><dd>{currencyOf(item.countryCode)}</dd></div><div><dt>Proveedor</dt><dd>{providerName(item)}</dd></div><div><dt>Conexión</dt><dd>{item.needsReconciliation?"Requiere verificación":connectionLabel(item.connectionState)}</dd></div></dl><footer>{actions(item)}</footer></article>)}</div></>}
  </section>
  <ConciergeUsageSettingsPanel/>

  {draft&&<WhatsAppChannelEditor initial={draft} existing={items.find(item=>item.id===editingId)} countries={catalogs.data?.countryOptions??[]} usedCountries={items.map(item=>item.countryCode)} providers={providers} pending={save.isPending} onClose={()=>{if(!save.isPending){setDraft(null);setEditingId(undefined)}}} onSave={value=>save.mutate({value,id:editingId})}/>}
  <ConfirmDialog open={Boolean(target)} title={target?.active?"Desactivar canal":"Activar canal"} subject={target?{label:"Canal",name:`${target.displayName} · ${target.phoneNumber}`}:undefined} description={target?.active?"El país dejará de estar disponible para registrar empresas nuevas.":"El país quedará disponible para registrar empresas nuevas."} tone={target?.active?"danger":"success"} confirmLabel={target?.active?"Desactivar":"Activar"} pending={toggle.isPending} onCancel={()=>setTarget(null)} onConfirm={()=>target&&toggle.mutate(target)}/>
 </>;
}
