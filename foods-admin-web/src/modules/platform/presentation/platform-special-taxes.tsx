"use client";
import "./platform-special-taxes.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,FormField,InlineNotice,Input,RowActionButton,Status} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {useFeedback} from "@/providers";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {fullDate,lastDay,percentToRate,ratePercentLabel,ratePeriodState,rateToPercent,shiftDate,type RatePeriodState,type SpecialTaxRate} from "@/shared/fiscal/special-tax";
import {deleteSpecialTaxRate,listSpecialTaxRegimes,saveSpecialTaxRate,type SpecialTaxRegimesView} from "../infrastructure/platform-api";

// The legal special tax of a country (MYPE restaurante in Peru) belongs to the
// country record: platform administrators keep its dated rates and every
// company enrolled in it invoices with them.
export type SpecialTaxRegimeItem=SpecialTaxRegimesView["items"][number];
// «Hasta» is inclusive on screen; the API stores an exclusive end.
type RateDraft={id?:string;validFrom:string;validUntil:string;percent:string;legalBasis:string};

const queryKey=["platform-special-tax-regimes"];
const columns=["Vigencia","Tasa","Base legal","Estado","Acciones"];
const periodView:Record<RatePeriodState,{tone:"green"|"blue"|"gray";label:string}>={current:{tone:"green",label:"Vigente"},scheduled:{tone:"blue",label:"Programada"},ended:{tone:"gray",label:"Finalizada"}};

export function useSpecialTaxRegimes(){return useQuery({queryKey,queryFn:listSpecialTaxRegimes})}

const dayIn=(country:string)=>(iso:string)=>formatRegionalCalendarDate(iso,country,fullDate);
const percentIn=(country:string)=>(rate:string)=>ratePercentLabel(rate,v=>formatRegionalNumber(v,country,{maximumFractionDigits:4}));

function expiryNotice(regime:SpecialTaxRegimeItem){
 const day=dayIn(regime.country);
 const{status}=regime;
 if(status.state==="expired")return {title:"Sin vigencia para hoy",text:"Las empresas acogidas facturan con su tasa general. Registra la nueva vigencia."};
 if(status.state==="expiring"&&status.coverageEnd)return {title:`La última vigencia termina el ${day(lastDay(status.coverageEnd))}`,text:`Registra la nueva vigencia; desde el ${day(status.coverageEnd)} las empresas acogidas facturarán con su tasa general.`};
 return null;
}

// Table cell of the country row: today's rate and, when it is running out,
// a warning state.
export function SpecialTaxCell({regime}:{regime?:SpecialTaxRegimeItem}){
 if(!regime)return <>—</>;
 const{status}=regime;
 const percent=percentIn(regime.country),day=dayIn(regime.country);
 const label=regime.name.split(" (")[0];
 if(status.state==="expired")return <><b>{label}</b><Status tone="orange">Sin vigencia</Status></>;
 return <><b>{label}: {percent(status.rate!)}</b>{status.rateValidTo&&<small>hasta {day(lastDay(status.rateValidTo))}</small>}{status.state==="expiring"&&<Status tone="orange">Por vencer</Status>}</>;
}

export function SpecialTaxNotices({regimes,countryName}:{regimes:SpecialTaxRegimeItem[];countryName:(code:string)=>string}){
 return <>{regimes.map(regime=>{const notice=expiryNotice(regime);return notice&&<InlineNotice key={regime.code} className="special-tax-page-notice" tone="warning" title={`${countryName(regime.country)} · ${regime.name}`}>{notice.title}. {notice.text}</InlineNotice>})}</>;
}

function draftError(draft:RateDraft){
 if(!draft.validFrom)return "Indica desde cuándo rige.";
 if(draft.validUntil&&draft.validUntil<draft.validFrom)return "«Hasta» debe ser igual o posterior a «Desde».";
 if(!/^\d{1,3}([.]\d{1,4})?$/.test(draft.percent.trim())||Number(draft.percent)>100)return "Usa un porcentaje entre 0 y 100 con hasta 4 decimales.";
 if(!draft.legalBasis.trim())return "Indica la ley o norma que fija la tasa.";
 return "";
}

// «Tasa especial» tab of the country form: the dated rates, edited in place
// so no dialog opens on top of the country dialog.
export function SpecialTaxTab({regime,today}:{regime:SpecialTaxRegimeItem;today:string}){
 const{notify}=useFeedback();
 const client=useQueryClient();
 const[draft,setDraft]=useState<RateDraft|null>(null);
 const[error,setError]=useState("");
 const[remove,setRemove]=useState<SpecialTaxRate|null>(null);
 const day=dayIn(regime.country),percent=percentIn(regime.country);
 const save=useMutation({
  mutationFn:(d:RateDraft)=>saveSpecialTaxRate(regime.code,{validFrom:d.validFrom,validTo:d.validUntil?shiftDate(d.validUntil,1):null,rate:percentToRate(d.percent),legalBasis:d.legalBasis.trim()},d.id),
  onSuccess:()=>{setDraft(null);void client.invalidateQueries({queryKey});notify({tone:"success",title:"Vigencia guardada",message:"Las empresas acogidas facturarán con esta tasa en esas fechas."})},
  onError:e=>notify({tone:"danger",title:"No se pudo guardar",message:e.message}),
 });
 const removal=useMutation({
  mutationFn:(rate:SpecialTaxRate)=>deleteSpecialTaxRate(rate.id),
  onSuccess:()=>{setRemove(null);void client.invalidateQueries({queryKey});notify({tone:"success",title:"Vigencia eliminada",message:"Los comprobantes emitidos conservan la tasa con la que salieron."})},
  onError:e=>{setRemove(null);notify({tone:"danger",title:"No se pudo eliminar",message:e.message})},
 });
 const last=regime.rates.at(-1);
 // A new period starts where the registered calendar ends and lasts a year.
 const next=()=>{const from=last?.validTo&&last.validTo>today?last.validTo:today;setError("");setDraft({validFrom:from,validUntil:shiftDate(`${Number(from.slice(0,4))+1}${from.slice(4)}`,-1),percent:"",legalBasis:last?.legalBasis??""})};
 const edit=(rate:SpecialTaxRate)=>{setError("");setDraft({id:rate.id,validFrom:rate.validFrom,validUntil:rate.validTo?lastDay(rate.validTo):"",percent:rateToPercent(rate.rate),legalBasis:rate.legalBasis})};
 const submit=()=>{if(!draft)return;const message=draftError(draft);setError(message);if(!message)save.mutate(draft)};
 const busy=save.isPending;
 const notice=expiryNotice(regime);
 const enrolled=regime.enrolledProfiles===1?"1 empresa acogida":`${regime.enrolledProfiles} empresas acogidas`;

 if(draft)return <form className="special-tax-tab" onSubmit={event=>{event.preventDefault();submit()}} noValidate inert={busy}>
  <h3 className="modal-section-title">{draft.id?"Editar vigencia":"Nueva vigencia"}</h3>
  <div className="form-grid">
   {error&&<p className="field-error span-2" role="alert">{error}</p>}
   <FormField label="Desde"><Input type="date" value={draft.validFrom} onChange={e=>setDraft({...draft,validFrom:e.target.value})}/></FormField>
   <FormField label="Hasta" optional><Input type="date" value={draft.validUntil} onChange={e=>setDraft({...draft,validUntil:e.target.value})}/></FormField>
   <FormField className="span-2" label="Porcentaje %" help="Tasa total del comprobante (IGV + IPM)."><Input inputMode="decimal" value={draft.percent} onChange={e=>setDraft({...draft,percent:e.target.value})} placeholder="10.5"/></FormField>
   <FormField className="span-2" label="Base legal"><Input maxLength={120} value={draft.legalBasis} onChange={e=>setDraft({...draft,legalBasis:e.target.value})} placeholder="Ley 32387"/></FormField>
  </div>
  <div className="special-tax-actions"><Button action="cancel" onClick={()=>setDraft(null)} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar vigencia"}</Button></div>
 </form>;

 return <div className="special-tax-tab">
  <div className="special-tax-summary"><div><b>{regime.name}</b><small>{enrolled}. Un cambio aplica a todas; los comprobantes emitidos no cambian.</small></div><Button kind="secondary" icon="plus" onClick={next}>Nueva vigencia</Button></div>
  {notice&&<InlineNotice tone="warning" title={notice.title}>{notice.text}</InlineNotice>}
  {!regime.rates.length?<EmptyState icon="percent" title="Sin vigencias" text="Registra la tasa y las fechas que fija la ley."/>:
  <DataTable label={`Vigencias de ${regime.name}`} containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{[...regime.rates].reverse().map((rate,i)=>{const state=periodView[ratePeriodState(rate,today)];return <tr className={i%2?"alternate":""} key={rate.id}>
   <td><b>{day(rate.validFrom)}</b><small>{rate.validTo?`hasta ${day(lastDay(rate.validTo))}`:"sin fecha de fin"}</small></td>
   <td><b>{percent(rate.rate)}</b></td>
   <td>{rate.legalBasis}</td>
   <td><Status tone={state.tone}>{state.label}</Status></td>
   <td><div className="table-actions"><RowActionButton action="edit" label={`Editar vigencia desde ${day(rate.validFrom)}`} onClick={()=>edit(rate)}/><RowActionButton action="remove" label={`Eliminar vigencia desde ${day(rate.validFrom)}`} onClick={()=>setRemove(rate)}/></div></td>
  </tr>})}</tbody></DataTable>}
  <ConfirmDialog open={Boolean(remove)} title="Eliminar vigencia" description="En esas fechas las empresas acogidas facturarán con su tasa general." confirmLabel="Eliminar" pending={removal.isPending} onCancel={()=>setRemove(null)} onConfirm={()=>remove&&removal.mutate(remove)}/>
 </div>;
}
