"use client";
import {Dialog} from "@/design-system/dialog";
import {useMemo,useState} from "react";
import dynamic from "next/dynamic";
import {useForm,useWatch} from "react-hook-form";
import {Button,EmptyState,ErrorState,FormField,Icon,Input,Select,Skeleton,Textarea,ModalHeader,ModalFooter} from "@/design-system";
import {cashOperationResolver} from "../domain/cash-schema";
import {availableCashCollaborators} from "../domain/collaborators";
import {useSession} from "@/providers";
import type {CashOperationDraft,CashRegister,CashShift,CashShiftUser,CashUserOption} from "../domain/types";

const CashCollaboratorSearch=dynamic(()=>import("./cash-collaborator-search").then(module=>module.CashCollaboratorSearch),{ssr:false,loading:()=><Skeleton height="var(--control-height)"/>});

export function CashTeamDialog({shift,users,options,loading,error,retrying,retry,busyUserId,close,assign,unassign}:{shift:CashShift;users:CashShiftUser[];options:CashUserOption[];loading:boolean;error?:string;retrying?:boolean;retry?:()=>void;busyUserId:string|null;close:()=>void;assign:(userId:string)=>void;unassign:(userId:string)=>void}){
  const[selected,setSelected]=useState("");
  const {user:currentUser}=useSession();
  const collaborators=useMemo(()=>users.filter(user=>user.userId!==shift.openedById),[users,shift.openedById]);
  const available=useMemo(()=>availableCashCollaborators(options,users,currentUser?.id??"",shift.status).filter(option=>option.id!==shift.openedById),[options,users,currentUser?.id,shift.status,shift.openedById]);
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-modal cash-team-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-team-title">

    <ModalHeader title="Equipo del turno" titleId="cash-team-title" icon="users" subtitle={shift.cashRegisterName.toUpperCase()} onClose={close} closeLabel="Cerrar"/>
    <div className="cash-team-body">
      <p className="cash-dialog-intro">Los colaboradores pueden cobrar desde POS mientras este turno siga abierto. La asignación no cambia su rol ni permite administrar la caja.</p>
      {error?<ErrorState title="No pudimos cargar el equipo" text={error} onRetry={retry} retrying={retrying}/>:<>
      <div className="cash-team-add">
        <FormField as="div" label={<label htmlFor="cash-team-user">Agregar al equipo</label>}>
          <CashCollaboratorSearch options={available} value={selected} onChange={setSelected} disabled={loading||!available.length||Boolean(busyUserId)} loading={loading}/>
        </FormField>
        <Button icon="plus" disabled={loading||!available.some(option=>option.id===selected)||Boolean(busyUserId)} onClick={()=>{if(!loading&&!busyUserId&&available.some(option=>option.id===selected)){assign(selected);setSelected("")}}}>Asignar</Button>
      </div>
      <section className="cash-team-list">
        <header><small>Colaboradores</small><b>{collaborators.length} {collaborators.length===1?"persona":"personas"}</b></header>
        {loading?<div role="status" aria-busy="true" aria-label="Cargando equipo del turno">{Array.from({length:2},(_,index)=><div className="cash-team-member" aria-hidden="true" key={index}><Skeleton className="cash-sk-icon"/><span className="cash-sk-lines"><Skeleton className="cash-sk-line"/><Skeleton className="cash-sk-line short"/></span><Skeleton className="cash-sk-action"/></div>)}</div>
        :collaborators.length?collaborators.map(user=><div className="cash-team-member" key={user.userId}><span><Icon name="users" size={16}/></span><div><b>{user.name}</b><small>Asignado al turno</small></div><Button kind="ghost" disabled={Boolean(busyUserId)} onClick={()=>{if(user.userId!==shift.openedById)unassign(user.userId)}} loading={busyUserId===user.userId}>{busyUserId===user.userId?"Quitando…":"Quitar"}</Button></div>)
        :<EmptyState className="cash-state-compact" icon="users" title="Sin colaboradores" text="Agrega personas para apoyar los cobros de este turno."/>}
      </section>
      </>}
    </div>
  </Dialog></div>;
}

export function CashOperationDialog({shift,registers,busy,close,save}:{shift:CashShift;registers:CashRegister[];busy:boolean;close:()=>void;save:(draft:CashOperationDraft)=>void}){
  const targets=registers.filter(item=>item.openShift&&item.openShift.id!==shift.id);
  const{control,register,handleSubmit,formState:{errors}}=useForm<CashOperationDraft>({
    defaultValues:{operationType:"cash_pull",targetShiftId:"",amount:"",reason:"",note:""},
    resolver:cashOperationResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const type=useWatch({control,name:"operationType"});
  const label=type==="deposit"?"Registrar depósito":type==="transfer"?"Transferir efectivo":"Registrar retiro";
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-operation-title" aria-busy={busy}>

    <ModalHeader title="Operación de caja" titleId="cash-operation-title" icon={type==="transfer"?"share":"stock"} subtitle={shift.cashRegisterName.toUpperCase()} onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        <FormField label="Operación">
          <Select {...register("operationType")}>
            <option value="cash_pull">Retiro de efectivo</option>
            <option value="deposit">Depósito</option>
            <option value="transfer">Transferencia entre cajas</option>
          </Select>
        </FormField>
        {type==="transfer"&&<FormField label="Caja de destino" error={errors.targetShiftId?.message}>
          <Select {...register("targetShiftId")}>
            <option value="">{targets.length?"Selecciona una caja con turno abierto":"No hay otra caja operando"}</option>
            {targets.map(item=><option value={item.openShift!.id} key={item.id}>{item.name}</option>)}
          </Select>
        </FormField>}
        <FormField label="Monto" error={errors.amount?.message}>
          <Input type="number" min="0.01" step="0.01" inputMode="decimal" {...register("amount")} placeholder="0.00"/>
        </FormField>
        <FormField label="Motivo" error={errors.reason?.message}>
          <Input maxLength={120} {...register("reason")} placeholder={type==="deposit"?"Ej. Depósito de seguridad":type==="transfer"?"Ej. Reposición de cambio":"Ej. Retiro preventivo"}/>
        </FormField>
        <FormField label="Observación" optional error={errors.note?.message}>
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Referencia adicional"/>
        </FormField>
        <p className="cash-close-warning"><Icon name="alert" size={14}/>{type==="transfer"?"La salida y entrada se registrarán juntas en ambas cajas.":"La operación reducirá el efectivo esperado del turno."}</p>
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||type==="transfer"&&!targets.length} loading={busy}>{busy?"Procesando…":label}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}
