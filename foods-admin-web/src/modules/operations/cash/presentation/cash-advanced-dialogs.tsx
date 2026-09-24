"use client";
import {useMemo,useState} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {cashOperationResolver} from "../domain/cash-schema";
import type {CashOperationDraft,CashRegister,CashShift,CashShiftUser,CashUserOption} from "../domain/types";

export function CashTeamDialog({shift,users,options,loading,busyUserId,close,assign,unassign}:{shift:CashShift;users:CashShiftUser[];options:CashUserOption[];loading:boolean;busyUserId:string|null;close:()=>void;assign:(userId:string)=>void;unassign:(userId:string)=>void}){
  const[selected,setSelected]=useState("");
  const available=useMemo(()=>options.filter(option=>!option.assignedShiftId&&!users.some(user=>user.userId===option.id)),[options,users]);
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-modal cash-team-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-team-title">
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="users" size={18}/></span><div><small>{shift.cashRegisterName.toUpperCase()}</small><h2 id="cash-team-title">Equipo del turno</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
    <div className="cash-team-body">
      <p className="cash-dialog-intro">Las personas asignadas pueden registrar movimientos y cobrar desde POS usando este turno.</p>
      <div className="cash-team-add">
        <Select value={selected} onChange={event=>setSelected(event.target.value)} disabled={loading||!available.length}>
          <option value="">{loading?"Cargando usuarios…":available.length?"Selecciona un usuario":"No hay usuarios disponibles"}</option>
          {available.map(option=><option value={option.id} key={option.id}>{option.name}</option>)}
        </Select>
        <Button icon="plus" disabled={!selected||Boolean(busyUserId)} onClick={()=>{if(selected){assign(selected);setSelected("")}}}>Asignar</Button>
      </div>
      <section className="cash-team-list">
        <header><small>EQUIPO ACTUAL</small><b>{users.length} {users.length===1?"persona":"personas"}</b></header>
        {loading?<div className="cash-team-loading">{Array.from({length:2},(_,index)=><i key={index}/>)}</div>
        :users.length?users.map(user=><div className="cash-team-member" key={user.userId}><span><Icon name="users" size={15}/></span><div><b>{user.name}</b><small>Asignado al turno</small></div><Button kind="ghost" disabled={Boolean(busyUserId)} onClick={()=>unassign(user.userId)}>{busyUserId===user.userId?"Quitando…":"Quitar"}</Button></div>)
        :<div className="cash-team-empty">No hay usuarios asignados.</div>}
      </section>
    </div>
  </section></div>;
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
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-operation-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name={type==="transfer"?"share":"stock"} size={18}/></span><div><small>{shift.cashRegisterName.toUpperCase()}</small><h2 id="cash-operation-title">Operación de caja</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        <label>Operación
          <Select {...register("operationType")}>
            <option value="cash_pull">Retiro de efectivo</option>
            <option value="deposit">Depósito</option>
            <option value="transfer">Transferencia entre cajas</option>
          </Select>
        </label>
        {type==="transfer"&&<label>Caja de destino
          <Select {...register("targetShiftId")} aria-invalid={Boolean(errors.targetShiftId)}>
            <option value="">{targets.length?"Selecciona una caja con turno abierto":"No hay otra caja operando"}</option>
            {targets.map(item=><option value={item.openShift!.id} key={item.id}>{item.name} · {item.openShift!.code}</option>)}
          </Select>
          {errors.targetShiftId?.message&&<small className="wizard-field-error">{errors.targetShiftId.message}</small>}
        </label>}
        <label>Monto
          <Input type="number" min="0.01" step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)} placeholder="0.00"/>
          {errors.amount?.message&&<small className="wizard-field-error">{errors.amount.message}</small>}
        </label>
        <label>Motivo
          <Input maxLength={120} {...register("reason")} aria-invalid={Boolean(errors.reason)} placeholder={type==="deposit"?"Ej. Depósito de seguridad":type==="transfer"?"Ej. Reposición de cambio":"Ej. Retiro preventivo"}/>
          {errors.reason?.message&&<small className="wizard-field-error">{errors.reason.message}</small>}
        </label>
        <label>Observación opcional
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Referencia adicional"/>
        </label>
        <p className="cash-close-warning"><Icon name="alert" size={14}/>{type==="transfer"?"La salida y entrada se registrarán juntas en ambas cajas.":"La operación reducirá el efectivo esperado del turno."}</p>
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||type==="transfer"&&!targets.length}>{busy?"Procesando…":label}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Procesando operación…</span></div>}
  </section></div>;
}
