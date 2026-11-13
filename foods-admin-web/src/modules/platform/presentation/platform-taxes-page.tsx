"use client";
import "./platform-taxes.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,Icon,InlineNotice,Input,ModalFooter,ModalHeader,PageHeader,RowActionButton,Status,TableSkeleton} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {useFeedback} from "@/providers";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {fullDate,lastDay,percentToRate,ratePercentLabel,ratePeriodLabel,ratePeriodState,rateToPercent,shiftDate,type RatePeriodState,type SpecialTaxRate} from "@/shared/fiscal/special-tax";
import {deleteSpecialTaxRate,getPlatformGlobalCatalogs,listSpecialTaxRegimes,saveSpecialTaxRate,type SpecialTaxRegimesView} from "../infrastructure/platform-api";

// Plataforma › Impuestos: the reduced IGV that the law sets for a country
// (Peru: MYPE restaurants and hotels in the SUNAT padrón). Platform
// administrators keep its rate per period; each company only says in its
// fiscal profile whether it is enrolled.
type Regime=SpecialTaxRegimesView["items"][number];
// «Hasta» is inclusive on screen; the API stores an exclusive end.
type RateDraft={id?:string;validFrom:string;validUntil:string;percent:string;legalBasis:string};

const queryKey=["platform-special-tax-regimes"];
const columns=["Periodo","IGV","Base legal","Estado","Acciones"];
const periodView:Record<RatePeriodState,{tone:"green"|"blue"|"gray";label:string}>={current:{tone:"green",label:"Vigente"},scheduled:{tone:"blue",label:"Programado"},ended:{tone:"gray",label:"Finalizado"}};

const dayIn=(country:string)=>(iso:string)=>formatRegionalCalendarDate(iso,country,fullDate);
const percentIn=(country:string)=>(rate:string)=>ratePercentLabel(rate,v=>formatRegionalNumber(v,country,{maximumFractionDigits:4}));

function expiryNotice(regime:Regime){
 const day=dayIn(regime.country);
 const{status}=regime;
 if(status.state==="expired")return {title:"No hay tasa para hoy",text:"Las empresas acogidas facturan con el IGV general. Registra el periodo vigente."};
 if(status.state==="expiring"&&status.coverageEnd)return {title:`La última tasa registrada termina el ${day(lastDay(status.coverageEnd))}`,text:`Registra el periodo siguiente; desde el ${day(status.coverageEnd)} las empresas acogidas facturarán con el IGV general.`};
 return null;
}

function draftError(draft:RateDraft){
 if(!draft.validFrom)return "Indica desde cuándo rige.";
 if(draft.validUntil&&draft.validUntil<draft.validFrom)return "«Hasta» debe ser igual o posterior a «Desde».";
 if(!/^\d{1,3}([.]\d{1,4})?$/.test(draft.percent.trim())||Number(draft.percent)>100)return "Usa un porcentaje entre 0 y 100 con hasta 4 decimales.";
 if(!draft.legalBasis.trim())return "Indica la ley que fija la tasa.";
 return "";
}

export function PlatformTaxesPage(){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const regimes=useQuery({queryKey,queryFn:listSpecialTaxRegimes});
 const catalogs=useQuery({queryKey:["platform-global-catalogs"],queryFn:getPlatformGlobalCatalogs});
 const[editing,setEditing]=useState<{regime:Regime;draft:RateDraft}|null>(null);
 const[removing,setRemoving]=useState<{regime:Regime;rate:SpecialTaxRate}|null>(null);
 const countryName=(code:string)=>catalogs.data?.countryOptions.find(country=>country.code===code)?.name??code;
 const save=useMutation({
  mutationFn:({regime,draft}:{regime:Regime;draft:RateDraft})=>saveSpecialTaxRate(regime.code,{validFrom:draft.validFrom,validTo:draft.validUntil?shiftDate(draft.validUntil,1):null,rate:percentToRate(draft.percent),legalBasis:draft.legalBasis.trim()},draft.id),
  onSuccess:()=>{setEditing(null);void client.invalidateQueries({queryKey});notify({tone:"success",title:"Periodo guardado",message:"Las empresas acogidas facturarán con esta tasa en esas fechas."})},
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const removal=useMutation({
  mutationFn:(rate:SpecialTaxRate)=>deleteSpecialTaxRate(rate.id),
  onSuccess:()=>{setRemoving(null);void client.invalidateQueries({queryKey});notify({tone:"success",title:"Periodo eliminado",message:"Los comprobantes emitidos conservan la tasa con la que salieron."})},
  onError:e=>{setRemoving(null);notify({tone:"danger",title:"No se pudo eliminar",message:e.message})},
 });
 const today=regimes.data?.today??"";
 // A new period starts where the registered calendar ends and lasts a year.
 const create=(regime:Regime)=>{
  const last=regime.rates.at(-1);
  const from=last?.validTo&&last.validTo>today?last.validTo:today;
  setEditing({regime,draft:{validFrom:from,validUntil:shiftDate(`${Number(from.slice(0,4))+1}${from.slice(4)}`,-1),percent:"",legalBasis:last?.legalBasis??""}});
 };
 const edit=(regime:Regime,rate:SpecialTaxRate)=>setEditing({regime,draft:{id:rate.id,validFrom:rate.validFrom,validUntil:rate.validTo?lastDay(rate.validTo):"",percent:rateToPercent(rate.rate),legalBasis:rate.legalBasis}});
 const items=regimes.data?.items??[];
 // Every period of every regime in one standard table; the regime column only
 // appears when more than one country has a reduced IGV.
 const rows=items.flatMap(regime=>regime.rates.map(rate=>({regime,rate})));
 const several=items.length>1;
 const tableColumns=several?["Periodo","Impuesto",...columns.slice(1)]:columns;
 const state=(rate:SpecialTaxRate)=>{const view=periodView[ratePeriodState(rate,today)];return <Status tone={view.tone}>{view.label}</Status>};
 const period=(regime:Regime,rate:SpecialTaxRate)=>ratePeriodLabel(rate,dayIn(regime.country));

 return <>
  <PageHeader title="Impuestos" description="IGV reducido que fija la ley por periodo. Cada empresa lo activa en su perfil fiscal." action={items.length?<Button icon="plus" onClick={()=>create(items[0])}>Nuevo periodo</Button>:undefined}/>
  {items.map(regime=>{const notice=expiryNotice(regime);return notice&&<InlineNotice key={regime.code} className="tax-expiry-notice" tone="warning" title={several?`${countryName(regime.country)}: ${notice.title.charAt(0).toLowerCase()}${notice.title.slice(1)}`:notice.title}>{notice.text}</InlineNotice>})}
  <section className="panel management standardized-management">
   {regimes.isLoading?<TableSkeleton columns={columns} rows={3} label="Cargando impuestos"/>:
    regimes.isError?<ErrorState title="No pudimos cargar los impuestos" text={regimes.error.message} retrying={regimes.isFetching} onRetry={()=>void regimes.refetch()}/>:
    !items.length?<EmptyState icon="tax" title="Sin IGV reducido" text="Ningún país tiene un IGV reducido fijado por ley."/>:
    !rows.length?<EmptyState icon="tax" title="Aún no hay periodos" text="Registra la tasa y las fechas que fija la ley." action={<Button icon="plus" onClick={()=>create(items[0])}>Nuevo periodo</Button>}/>:
    <DataTable label="Periodos del IGV reducido" containerClassName="hover-scroll"><thead><tr>{tableColumns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>
     {rows.map(({regime,rate},index)=><tr className={index%2?"alternate":""} key={rate.id}>
      <td><span className={`row-icon r${index%3}`}><Icon name="calendar" size={18}/></span><b>{period(regime,rate)}</b></td>
      {several&&<td>{regime.name}<small>{countryName(regime.country)}</small></td>}
      <td><b>{percentIn(regime.country)(rate.rate)}</b></td>
      <td>{rate.legalBasis}</td>
      <td>{state(rate)}</td>
      <td><div className="table-actions"><RowActionButton action="edit" label={`Editar periodo ${period(regime,rate)}`} onClick={()=>edit(regime,rate)}/><RowActionButton action="remove" label={`Eliminar periodo ${period(regime,rate)}`} onClick={()=>setRemoving({regime,rate})}/></div></td>
     </tr>)}
    </tbody></DataTable>}
  </section>
  {editing&&<RateDialog key={editing.draft.id??"new"} regime={editing.regime} initial={editing.draft} busy={save.isPending} close={()=>setEditing(null)} save={draft=>save.mutate({regime:editing.regime,draft})}/>}
  <ConfirmDialog open={Boolean(removing)} title="Eliminar periodo" subject={removing?{label:"Periodo",name:period(removing.regime,removing.rate)}:undefined} description="En esas fechas las empresas acogidas facturarán con el IGV general." confirmLabel="Eliminar" pending={removal.isPending} onCancel={()=>setRemoving(null)} onConfirm={()=>removing&&removal.mutate(removing.rate)}/>
 </>;
}

// The regime is read-only context, like the currency of a WhatsApp channel;
// the footer says how many companies the change reaches.
function RateDialog({regime,initial,busy,close,save}:{regime:Regime;initial:RateDraft;busy:boolean;close:()=>void;save:(draft:RateDraft)=>void}){
 const[draft,setDraft]=useState(initial);
 const[error,setError]=useState("");
 const submit=()=>{const message=draftError(draft);setError(message);if(!message)save(draft)};
 const reach=regime.enrolledProfiles===1?"Se aplica a 1 empresa acogida.":`Se aplica a ${regime.enrolledProfiles} empresas acogidas.`;
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal compact tax-rate-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="tax-rate-title" aria-busy={busy}>
  <ModalHeader title={draft.id?"Editar periodo":"Nuevo periodo"} titleId="tax-rate-title" onClose={close} busy={busy} closeLabel="Cerrar"/>
  <form onSubmit={event=>{event.preventDefault();submit()}} noValidate>
   <div className="form-grid">
    {error&&<p className="field-error span-2" role="alert">{error}</p>}
    <FormField className="span-2" label="Impuesto"><Input value={regime.name} readOnly tabIndex={-1}/></FormField>
    <FormField label="Desde"><Input type="date" value={draft.validFrom} onChange={e=>setDraft({...draft,validFrom:e.target.value})}/></FormField>
    <FormField label="Hasta" optional><Input type="date" value={draft.validUntil} onChange={e=>setDraft({...draft,validUntil:e.target.value})}/></FormField>
    <FormField className="span-2" label="IGV %" help="Total que sale en el comprobante (IGV + IPM)."><Input inputMode="decimal" value={draft.percent} onChange={e=>setDraft({...draft,percent:e.target.value})} placeholder="10.5"/></FormField>
    <FormField className="span-2" label="Base legal"><Input maxLength={120} value={draft.legalBasis} onChange={e=>setDraft({...draft,legalBasis:e.target.value})} placeholder="Ley 32387"/></FormField>
   </div>
   <ModalFooter note={reach}><Button action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
  </form>
 </Dialog></div>;
}
