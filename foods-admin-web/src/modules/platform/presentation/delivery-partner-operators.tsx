"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Dialog} from "@/design-system/dialog";
import {Button,ConfirmDialog,EmptyState,ErrorState,FormField,Icon,IconButton,Input,ModalBody,ModalFooter,ModalHeader,RowActionButton,Skeleton,Status,Tag} from "@/design-system";
import {useFeedback} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {partnerPanelLoginPath} from "@/shared/routing/page-routes";
import type {DeliveryOperator,DeliveryOperatorCredentials,DeliveryOperatorDraft,DeliveryPartner} from "../domain/types";
import {createDeliveryOperator,listDeliveryOperators,resetDeliveryOperatorPassword,updateDeliveryOperator} from "../infrastructure/platform-api";

const blank:DeliveryOperatorDraft={fullName:"",email:""};
const validEmail=(value:string)=>/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(value.trim());

/** Acceso recién creado: la contraseña temporal se muestra una sola vez. */
function CredentialsDialog({credentials,onClose}:{credentials:DeliveryOperatorCredentials;onClose:()=>void}){
  const {notify}=useFeedback();
  const url=typeof window==="undefined"?partnerPanelLoginPath:`${window.location.origin}${partnerPanelLoginPath}`;
  const message=`Panel de reparto de Fudia\n${url}\nCorreo: ${credentials.operator.email}\nContraseña temporal: ${credentials.temporaryPassword}\nAl ingresar crearás tu propia contraseña.`;
  const copy=useMutation({
    mutationFn:async()=>{await navigator.clipboard.writeText(message)},
    onSuccess:()=>notify({tone:"success",title:"Acceso copiado",message:`Envíalo solo a ${credentials.operator.fullName}.`}),
    onError:()=>notify({tone:"danger",title:"No se pudo copiar",message:"Revisa el permiso de portapapeles de tu navegador y vuelve a intentarlo."}),
  });
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={onClose} className="crud-modal compact platform-operator-credentials modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="operator-credentials-title">
    <ModalHeader title="Contraseña temporal" titleId="operator-credentials-title" onClose={onClose} closeLabel="Cerrar"/>
    <ModalBody><div className="operator-credentials">
      <dl>
        <div><dt>Operador</dt><dd>{credentials.operator.fullName}</dd></div>
        <div><dt>Correo</dt><dd>{credentials.operator.email}</dd></div>
        <div><dt>Ingresa en</dt><dd>{url}</dd></div>
      </dl>
      <p className="operator-credentials-password" aria-label="Contraseña temporal">{credentials.temporaryPassword}</p>
      <p className="operator-credentials-note"><Icon name="info" size={16}/>Se muestra una sola vez. Al ingresar, creará su propia contraseña.</p>
    </div></ModalBody>
    <ModalFooter><Button type="button" kind="secondary" onClick={onClose}>Listo</Button><Button type="button" appearance="strong" icon="link" disabled={copy.isPending} loading={copy.isPending} onClick={()=>copy.mutate()}>{copy.isPending?"Copiando…":"Copiar acceso"}</Button></ModalFooter>
  </Dialog></div>;
}

/**
 * Operadores de una central de Delivery Fudia (pestaña «Operadores» al editar
 * la central): cada persona entra al panel con su correo. Plataforma la da de
 * alta con una contraseña temporal, la restablece o le quita el acceso.
 */
export function DeliveryPartnerOperators({partner}:{partner:DeliveryPartner}){
  const {notify}=useFeedback();
  const client=useQueryClient();
  const queryKey=["platform-delivery-operators",partner.id];
  const operators=useQuery({queryKey,queryFn:()=>listDeliveryOperators(partner.id)});
  const [draft,setDraft]=useState<DeliveryOperatorDraft|null>(null);
  const [touched,setTouched]=useState(false);
  const [credentials,setCredentials]=useState<DeliveryOperatorCredentials|null>(null);
  const [resetTarget,setResetTarget]=useState<DeliveryOperator|null>(null);
  const [toggleTarget,setToggleTarget]=useState<DeliveryOperator|null>(null);
  const refresh=()=>void client.invalidateQueries({queryKey});
  const create=useMutation({
    mutationFn:(value:DeliveryOperatorDraft)=>createDeliveryOperator(partner.id,value),
    onSuccess:result=>{setDraft(null);setTouched(false);refresh();setCredentials(result)},
    onError:e=>notify({tone:"danger",title:"No se pudo agregar",message:e.message}),
  });
  const reset=useMutation({
    mutationFn:(operator:DeliveryOperator)=>resetDeliveryOperatorPassword(partner.id,operator.id),
    onSuccess:result=>{setResetTarget(null);refresh();setCredentials(result)},
    onError:e=>{setResetTarget(null);notify({tone:"danger",title:"No se pudo restablecer",message:e.message})},
  });
  const toggle=useMutation({
    mutationFn:(operator:DeliveryOperator)=>updateDeliveryOperator(partner.id,operator,!operator.active),
    onSuccess:(_,operator)=>{
      setToggleTarget(null);
      refresh();
      notify({tone:"success",title:operator.active?"Acceso desactivado":"Acceso activado",message:operator.active?`${operator.fullName} ya no puede entrar al panel; sus sesiones se cerraron.`:`${operator.fullName} puede volver a entrar al panel.`});
    },
    onError:e=>{setToggleTarget(null);notify({tone:"danger",title:"No se pudo actualizar",message:e.message})},
  });
  const items=operators.data?.items??[];
  const nameError=touched&&draft&&draft.fullName.trim().length<2?"Escribe su nombre (2 a 80 caracteres).":"";
  const emailError=touched&&draft&&!validEmail(draft.email)?"Escribe un correo válido.":"";
  const submit=()=>{
    setTouched(true);
    if(!draft||draft.fullName.trim().length<2||!validEmail(draft.email))return;
    create.mutate({fullName:draft.fullName.trim(),email:draft.email.trim()});
  };
  const lastLogin=(operator:DeliveryOperator)=>operator.lastLoginAt?`Último ingreso: ${formatRegionalDateTime(operator.lastLoginAt,{},{dateStyle:"short",timeStyle:"short"})}`:"Aún no ingresa";

  return <div className="partner-operators">
    <div className="partner-operators-head">
      <p>Cada persona entra al panel de reparto con su correo y una contraseña temporal que luego cambia.</p>
      {!draft&&<Button kind="secondary" appearance="strong" icon="plus" onClick={()=>{setDraft({...blank});setTouched(false)}}>Agregar operador</Button>}
    </div>
    {draft&&<form className="partner-operator-form" onSubmit={event=>{event.preventDefault();submit()}} aria-busy={create.isPending} noValidate>
      <FormField label="Nombre" error={nameError||undefined}><Input value={draft.fullName} maxLength={80} autoFocus autoComplete="off" disabled={create.isPending} placeholder="Lucía Ramos" onChange={event=>setDraft({...draft,fullName:event.target.value})}/></FormField>
      <FormField label="Correo electrónico" error={emailError||undefined}><Input icon="mail" type="email" inputMode="email" value={draft.email} maxLength={180} autoComplete="off" disabled={create.isPending} placeholder="lucia@central.pe" onChange={event=>setDraft({...draft,email:event.target.value})}/></FormField>
      <div className="partner-operator-form-actions"><Button type="button" action="cancel" disabled={create.isPending} onClick={()=>{setDraft(null);setTouched(false)}}>Cancelar</Button><Button type="submit" appearance="strong" icon="userCheck" disabled={create.isPending} loading={create.isPending}>{create.isPending?"Agregando…":"Agregar"}</Button></div>
    </form>}
    {operators.isLoading?<div className="partner-operators-loading" aria-busy="true"><Skeleton height="var(--size-64)" radius="var(--radius-tile)"/><Skeleton height="var(--size-64)" radius="var(--radius-tile)"/><span className="sr-only">Cargando operadores</span></div>:
      operators.isError?<ErrorState title="No pudimos cargar los operadores" text={operators.error.message} retrying={operators.isFetching} onRetry={()=>void operators.refetch()}/>:
      !items.length?(draft?null:<EmptyState icon="users" title="Sin operadores" text={`Agrega a la primera persona de ${partner.name} para que entre al panel de reparto.`}/>):
      <ul className="partner-operators-list" aria-label={`Operadores de ${partner.name}`}>{items.map(operator=><li key={operator.id} className={operator.active?"":"is-inactive"}>
        <span className="row-icon r0"><Icon name="user" size={18}/></span>
        <div className="partner-operator-copy"><b>{operator.fullName}</b><span>{operator.email}</span><small>{lastLogin(operator)}</small></div>
        <div className="partner-operator-state">{operator.active&&operator.mustChangePassword&&<Tag icon="key">Clave temporal</Tag>}<Status active={operator.active}>{operator.active?"Activo":"Inactivo"}</Status></div>
        <div className="table-actions"><IconButton icon="key" label={`Nueva contraseña temporal para ${operator.fullName}`} disabled={!operator.active} onClick={()=>setResetTarget(operator)}/><RowActionButton action={operator.active?"deactivate":"activate"} stateLabel={`Acceso de ${operator.fullName}`} onClick={()=>setToggleTarget(operator)}/></div>
      </li>)}</ul>}
    {credentials&&<CredentialsDialog credentials={credentials} onClose={()=>setCredentials(null)}/>}
    <ConfirmDialog open={Boolean(resetTarget)} tone="danger" title="Nueva contraseña temporal" subject={resetTarget?{label:"Operador",name:resetTarget.fullName}:undefined} description="Su contraseña actual dejará de funcionar y se cerrarán sus sesiones abiertas. Deberá crear una nueva al ingresar." confirmLabel="Generar contraseña" pending={reset.isPending} onCancel={()=>setResetTarget(null)} onConfirm={()=>resetTarget&&reset.mutate(resetTarget)}/>
    <ConfirmDialog open={Boolean(toggleTarget)} tone={toggleTarget?.active?"danger":"success"} title={toggleTarget?.active?"Desactivar acceso":"Activar acceso"} subject={toggleTarget?{label:"Operador",name:toggleTarget.fullName}:undefined} description={toggleTarget?.active?"No podrá entrar al panel de reparto y se cerrarán sus sesiones abiertas.":"Podrá volver a entrar al panel con su contraseña."} confirmLabel={toggleTarget?.active?"Desactivar":"Activar"} pending={toggle.isPending} onCancel={()=>setToggleTarget(null)} onConfirm={()=>toggleTarget&&toggle.mutate(toggleTarget)}/>
  </div>;
}
