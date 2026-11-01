"use client";
import {useState} from "react";
import {useFieldArray,useForm,useWatch} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import "./tax-regimes.css";
import {Button,EmptyState,ErrorState,FormField,Icon,Input,ModalBody,ModalFooter,ModalForm,ModalHeader,RowActionButton,Select,Status,TableSkeleton} from "@/design-system";
import {ConfirmDialog} from "@/design-system/confirm-dialog";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {Profile,TaxRegime,TaxRegimeDraft} from "../domain/types";
import {draftFromRegime,rateToPercent,regimeStatus,shiftDate,totalPercent,type TaxRegimeStatus} from "../domain/tax-regimes";
import {taxRegimeResolver} from "../domain/organization-schemas";
import {deleteTaxRegime,listTaxRegimes,saveTaxRegime} from "../infrastructure/organizations-api";

const regimeColumns=["Régimen","Vigencia","Impuestos","Total","Estado","Acciones"];
const statusView:Record<TaxRegimeStatus,{tone:"green"|"blue"|"gray";label:string}>={current:{tone:"green",label:"Vigente"},scheduled:{tone:"blue",label:"Programado"},ended:{tone:"gray",label:"Finalizado"}};
const blankRegime:TaxRegimeDraft={name:"",validFrom:"",validUntil:"",components:[{name:"",percent:""}]};

export function useTaxRegimes(profileId:string|undefined,enabled:boolean){
 const{organization}=useSession();
 return useQuery({queryKey:["tax-regimes",organization?.id,profileId],queryFn:()=>listTaxRegimes(profileId!),enabled:enabled&&Boolean(profileId)});
}

// Dated tax regimes of one fiscal profile. The profile selector only appears
// when the company operates in more than one country.
export function TaxRegimesPanel({profiles,profileId,setProfileId,query,canManage,edit}:{profiles:Profile[];profileId:string|undefined;setProfileId:(id:string)=>void;query:ReturnType<typeof useTaxRegimes>;canManage:boolean;edit:(draft:TaxRegimeDraft)=>void}){
 const{location}=useSession();
 const{notify}=useFeedback();
 const client=useQueryClient();
 const[remove,setRemove]=useState<TaxRegime|null>(null);
 const removal=useMutation({
  mutationFn:(regime:TaxRegime)=>deleteTaxRegime(profileId!,regime.id),
  onSuccess:()=>{setRemove(null);void client.invalidateQueries({queryKey:["tax-regimes"]});notify({tone:"success",title:"Régimen eliminado",message:"Los comprobantes emitidos conservan los impuestos que se aplicaron."})},
  onError:error=>{setRemove(null);notify({tone:"danger",title:"No se pudo eliminar",message:error.message})},
 });
 const country=location?.country;
 const day=(value:string)=>formatRegionalCalendarDate(value,country,{dateStyle:"medium"});
 const percent=(rate:string)=>`${formatRegionalNumber(Number(rateToPercent(rate)),country,{maximumFractionDigits:4})} %`;
 const view=query.data;
 return <section className="panel management catalog-panel">
  {profiles.length>1&&<div className="toolbar"><Select chip aria-label="País del régimen" value={profileId??""} onChange={event=>setProfileId(event.target.value)}>{profiles.map(profile=><option key={profile.id} value={profile.id}>{profile.countryName}</option>)}</Select></div>}
  {!profileId?<EmptyState icon="receipt" title="Sin perfiles fiscales activos" text="Crea un perfil fiscal para registrar sus impuestos."/>:query.isLoading?<TableSkeleton columns={regimeColumns} label="Cargando regímenes de impuesto"/>:query.isError?<ErrorState title="No pudimos cargar los regímenes" text={query.error.message} onRetry={()=>void query.refetch()} retrying={query.isFetching}/>:!view?.items.length?<EmptyState icon="percent" title="Sin regímenes de impuesto" text={`Se aplica la tasa general: ${view?.general.name??""} ${view?percent(view.general.rate):""}.`} action={canManage?<Button icon="plus" onClick={()=>edit(blankRegime)}>Nuevo régimen</Button>:undefined}/>:
  <DataTable label="Regímenes de impuesto" containerClassName="hover-scroll"><thead><tr>{regimeColumns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{view.items.map((regime,i)=>{const status=statusView[regimeStatus(regime,view.effective.date)];return <tr className={i%2?"alternate":""} key={regime.id}>
   <td><span className={`row-icon r${i%3}`}><Icon name="percent" size={18}/></span><b>{regime.name}</b></td>
   <td>{day(regime.validFrom)}<small>{regime.validTo?`hasta ${day(shiftDate(regime.validTo,-1))}`:"sin fecha de fin"}</small></td>
   <td>{regime.components.map(component=>`${component.name} ${percent(component.rate)}`).join(" · ")}</td>
   <td><b>{percent(regime.totalRate)}</b></td>
   <td><Status tone={status.tone}>{status.label}</Status></td>
   <td><div className="table-actions">{canManage&&<><RowActionButton action="edit" onClick={()=>edit(draftFromRegime(regime))}/><RowActionButton action="remove" label={`Eliminar ${regime.name}`} onClick={()=>setRemove(regime)}/></>}</div></td>
  </tr>})}</tbody></DataTable>}
  <ConfirmDialog open={Boolean(remove)} title="Eliminar régimen" description={`«${remove?.name??""}» dejará de aplicarse en sus fechas; regirá la tasa general.`} confirmLabel="Eliminar" pending={removal.isPending} onCancel={()=>setRemove(null)} onConfirm={()=>remove&&removal.mutate(remove)}/>
 </section>;
}

export function TaxRegimeDialog({value:initial,profileId,countryName,close}:{value:TaxRegimeDraft;profileId:string;countryName:string;close:()=>void}){
 const{location}=useSession();
 const{notify}=useFeedback();
 const client=useQueryClient();
 const{register,control,handleSubmit,formState:{errors}}=useForm<TaxRegimeDraft>({defaultValues:initial,resolver:taxRegimeResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const{fields,append,remove}=useFieldArray({control,name:"components"});
 const components=useWatch({control,name:"components"})??[];
 const save=useMutation({
  mutationFn:(draft:TaxRegimeDraft)=>saveTaxRegime(profileId,draft),
  onSuccess:()=>{void client.invalidateQueries({queryKey:["tax-regimes"]});close();notify({tone:"success",title:"Régimen guardado",message:`Los impuestos de ${countryName} quedaron actualizados.`})},
  onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message}),
 });
 const busy=save.isPending;
 const total=`${formatRegionalNumber(totalPercent(components),location?.country,{maximumFractionDigits:4})} %`;
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={()=>{if(!busy)close()}} className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="tax-regime-title">
  <ModalHeader title="Régimen de impuesto" titleId="tax-regime-title" icon="percent" subtitle={initial.id?"Editar régimen":"Nuevo régimen"} onClose={close} busy={busy} closeLabel="Cerrar"/>
  <ModalForm onSubmit={handleSubmit(draft=>save.mutate(draft))} noValidate inert={busy}>
   <ModalBody><div className="form-sections-flow">
    <section className="form-section"><div className="form-grid">
     <FormField className="span-2" label="Nombre" error={errors.name?.message}><Input autoFocus maxLength={60} {...register("name")} placeholder="Régimen MYPE restaurantes 2026"/></FormField>
     <FormField label="Desde" error={errors.validFrom?.message}><Input type="date" {...register("validFrom")}/></FormField>
     <FormField label="Hasta" optional error={errors.validUntil?.message}><Input type="date" {...register("validUntil")}/></FormField>
    </div></section>
    <section className="form-section tax-regime-components" aria-labelledby="tax-regime-components-title">
     <h3 id="tax-regime-components-title" className="modal-section-title">Impuestos</h3>
     {fields.map((field,index)=>{const lineErrors=errors.components?.[index];return <div className="tax-regime-component" key={field.id}>
      <FormField label="Impuesto" icon={false} error={lineErrors?.name?.message}><Input maxLength={30} {...register(`components.${index}.name`)} placeholder={index?"IPM":"IGV"}/></FormField>
      <FormField label="Porcentaje %" icon={false} error={lineErrors?.percent?.message}><Input inputMode="decimal" {...register(`components.${index}.percent`)} placeholder="0"/></FormField>
      <RowActionButton action="remove" label="Quitar impuesto" onClick={()=>remove(index)} disabled={busy||fields.length===1}/>
     </div>})}
     {errors.components?.root?.message||errors.components?.message?<p className="field-error" role="alert">{errors.components?.root?.message??errors.components?.message}</p>:null}
     {fields.length<5&&<Button kind="secondary" icon="plus" onClick={()=>append({name:"",percent:""})} disabled={busy}>Agregar impuesto</Button>}
    </section>
   </div></ModalBody>
   <ModalFooter note={<>Total <b>{total}</b></>}><Button action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
  </ModalForm>
 </Dialog></div>;
}
