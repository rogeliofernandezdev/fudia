"use client";
import {useState} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,Icon,Input,Status,Textarea} from "@/design-system";
import {cashMovementResolver,cashRegisterResolver,closeCashShiftResolver,openCashShiftResolver} from "../domain/cash-schema";
import type {CashMovementDraft,CashMovementType,CashRegister,CashRegisterDraft,CashShift,CloseCashShiftDraft,OpenCashShiftDraft} from "../domain/types";

export function CashRegisterDialog({initial,busy,close,save}:{initial?:CashRegister|null;busy:boolean;close:()=>void;save:(draft:CashRegisterDraft)=>void}){
  const editing=Boolean(initial);
  const{register,handleSubmit,formState:{errors}}=useForm<CashRegisterDraft>({
    defaultValues:{name:initial?.name??"",blindClose:initial?.blindClose??false},
    resolver:cashRegisterResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-register-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="sales" size={18}/></span><div><small>CAJAS</small><h2 id="cash-register-title">{editing?"Editar caja":"Registrar caja"}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        <p className="cash-dialog-intro">{editing?"Actualiza el nombre operativo de la caja. Su historial y turnos se conservarán.":"Crea una caja del local. Luego podrás iniciar y cerrar turnos sobre esta caja sin perder su historial."}</p>
        <label>Nombre de la caja
          <Input autoFocus maxLength={80} {...register("name")} aria-invalid={Boolean(errors.name)} placeholder="Ej. Caja principal"/>
          {errors.name?.message&&<small className="wizard-field-error">{errors.name.message}</small>}
        </label>
        <label className="switch-row cash-blind-switch">
          <input type="checkbox" {...register("blindClose")}/>
          <span/>
          <b>Cierre ciego<small>El cajero contará el efectivo sin ver el saldo esperado. Supervisores con permiso sí podrán consultarlo.</small></b>
        </label>
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Guardando…":editing?"Guardar cambios":"Registrar caja"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Guardando caja…</span></div>}
  </section></div>;
}

export function OpenCashShiftDialog({cashRegister,busy,close,save}:{cashRegister:CashRegister;busy:boolean;close:()=>void;save:(draft:OpenCashShiftDraft)=>void}){
  const{register,handleSubmit,formState:{errors}}=useForm<OpenCashShiftDraft>({
    defaultValues:{openingAmount:"0",note:""},
    resolver:openCashShiftResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-open-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="clock" size={18}/></span><div><small>{cashRegister.name.toUpperCase()}</small><h2 id="cash-open-title">Iniciar turno</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        <p className="cash-dialog-intro">Estás iniciando un turno en <b>{cashRegister.name}</b>. Registra el efectivo inicial que quedará como base del arqueo.</p>
        <label>Fondo inicial
          <Input autoFocus type="number" min="0" step="0.01" inputMode="decimal" {...register("openingAmount")} aria-invalid={Boolean(errors.openingAmount)} placeholder="0.00"/>
          {errors.openingAmount?.message&&<small className="wizard-field-error">{errors.openingAmount.message}</small>}
        </label>
        <label>Observación opcional
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Ej. Fondo recibido del turno anterior"/>
          {errors.note?.message&&<small className="wizard-field-error">{errors.note.message}</small>}
        </label>
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Iniciando…":"Iniciar turno"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Iniciando turno…</span></div>}
  </section></div>;
}

export function CashMovementDialog({type,busy,close,save}:{type:CashMovementType;busy:boolean;close:()=>void;save:(draft:CashMovementDraft)=>void}){
  const isIncome=type==="income";
  const{register,handleSubmit,formState:{errors}}=useForm<CashMovementDraft>({
    defaultValues:{movementType:type,amount:"",reason:"",note:""},
    resolver:cashMovementResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-movement-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name={isIncome?"plus":"minus"} size={18}/></span><div><small>MOVIMIENTO DE CAJA</small><h2 id="cash-movement-title">{isIncome?"Registrar ingreso":"Registrar egreso"}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <input type="hidden" {...register("movementType")}/>
      <div className="cash-dialog-body">
        <p className="cash-dialog-intro">{isIncome?"Registra efectivo que entra a la caja fuera de un cobro de venta.":"Registra efectivo que sale de caja y deja el motivo trazable."}</p>
        <label>Monto
          <Input autoFocus type="number" min="0.01" step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)} placeholder="0.00"/>
          {errors.amount?.message&&<small className="wizard-field-error">{errors.amount.message}</small>}
        </label>
        <label>Motivo
          <Input maxLength={120} {...register("reason")} aria-invalid={Boolean(errors.reason)} placeholder={isIncome?"Ej. Reposición de fondo":"Ej. Compra menor o retiro"}/>
          {errors.reason?.message&&<small className="wizard-field-error">{errors.reason.message}</small>}
        </label>
        <label>Observación opcional
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Referencia o detalle adicional"/>
          {errors.note?.message&&<small className="wizard-field-error">{errors.note.message}</small>}
        </label>
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" kind={isIncome?"success":"primary"} disabled={busy}>{busy?"Guardando…":isIncome?"Registrar ingreso":"Registrar egreso"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Guardando movimiento…</span></div>}
  </section></div>;
}

export function CloseCashShiftDialog({shift,busy,currency,formatMoney,close,save}:{shift:CashShift;busy:boolean;currency:string;formatMoney:(value:number)=>string;close:()=>void;save:(draft:CloseCashShiftDraft)=>void}){
  const blind=shift.blindClose&&!shift.expectedVisible;
  const expected=Number(shift.expectedAmount||0);
  const denominations=denominationsForCurrency(currency);
  const[byDenomination,setByDenomination]=useState(false);
  const{control,register,handleSubmit,formState:{errors}}=useForm<CloseCashShiftDraft>({
    defaultValues:{
      countedAmount:shift.expectedVisible&&Number.isFinite(expected)?expected.toFixed(2):"",
      note:"",
      counts:denominations.map(value=>({denomination:String(value),quantity:""})),
    },
    resolver:closeCashShiftResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const countedValue=useWatch({control,name:"countedAmount"})??"";
  const countLines=useWatch({control,name:"counts"})??[];
  const denominationTotal=countLines.reduce((sum,line)=>sum+(Number(line.denomination)||0)*(Number(line.quantity)||0),0);
  const counted=byDenomination?denominationTotal:Number(countedValue);
  const difference=Number.isFinite(counted)?counted-expected:0;
  const differenceTone=Math.abs(difference)<.005?"balanced":difference>0?"positive":"negative";

  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-modal cash-close-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-close-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="lock" size={18}/></span><div><small>{shift.cashRegisterName.toUpperCase()}</small><h2 id="cash-close-title">Cerrar turno</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        {blind
          ?<div className="cash-blind-close-note"><Icon name="lock" size={15}/><span><b>Cierre ciego activo</b><small>Cuenta el efectivo sin consultar el saldo esperado. La diferencia se calculará al confirmar.</small></span></div>
          :<section className="cash-close-summary">
            <div><small>FONDO INICIAL</small><b>{formatMoney(Number(shift.openingAmount))}</b></div>
            <div><small>INGRESOS</small><b>{formatMoney(Number(shift.incomeAmount))}</b></div>
            <div><small>EGRESOS</small><b>{formatMoney(Number(shift.expenseAmount))}</b></div>
            <div className="expected"><small>EFECTIVO ESPERADO</small><strong>{formatMoney(expected)}</strong></div>
          </section>}

        <div className="cash-count-mode">
          <button type="button" className={!byDenomination?"active":""} onClick={()=>setByDenomination(false)}>Monto total</button>
          <button type="button" className={byDenomination?"active":""} onClick={()=>setByDenomination(true)}>Por denominaciones</button>
        </div>

        {!byDenomination?<label>Efectivo contado
          <Input autoFocus type="number" min="0" step="0.01" inputMode="decimal" {...register("countedAmount")} aria-invalid={Boolean(errors.countedAmount)} placeholder="0.00"/>
          {errors.countedAmount?.message&&<small className="wizard-field-error">{errors.countedAmount.message}</small>}
        </label>:<section className="cash-denomination-count">
          <header><div><small>CONTEO</small><b>Billetes y monedas</b></div><strong>{formatMoney(denominationTotal)}</strong></header>
          <div>{denominations.map((value,index)=><label key={value}>
            <span>{formatMoney(value)}</span>
            <input type="hidden" {...register(`counts.${index}.denomination`)}/>
            <Input type="number" min="0" step="1" inputMode="numeric" {...register(`counts.${index}.quantity`)} placeholder="0"/>
            <b>{formatMoney(value*(Number(countLines[index]?.quantity)||0))}</b>
          </label>)}</div>
          {errors.countedAmount?.message&&<small className="wizard-field-error">{errors.countedAmount.message}</small>}
        </section>}

        {!blind&&Number.isFinite(counted)&&counted>=0&&<div className={"cash-close-difference "+differenceTone}><span><small>DIFERENCIA</small><b>{difference>0?"+":""}{formatMoney(difference)}</b></span><em>{differenceTone==="balanced"?"Caja cuadrada":differenceTone==="positive"?"Sobrante":"Faltante"}</em></div>}

        <label>Observación opcional
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Explica cualquier diferencia si corresponde"/>
          {errors.note?.message&&<small className="wizard-field-error">{errors.note.message}</small>}
        </label>
        <p className="cash-close-warning"><Icon name="alert" size={14}/>Al cerrar el turno ya no se podrán registrar nuevos movimientos.</p>
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Volver</Button><Button type="submit" kind="danger" disabled={busy}>{busy?"Cerrando…":"Cerrar turno"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Cerrando turno…</span></div>}
  </section></div>;
}

export function CashShiftDetailDialog({shift,formatMoney,formatDateTime,formatBusinessDate,close}:{shift:CashShift;formatMoney:(value:number)=>string;formatDateTime:(value:string)=>string;formatBusinessDate:(value:string)=>string;close:()=>void}){
  const variance=Number(shift.varianceAmount??0);
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-detail-title">
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="sales" size={18}/></span><div><small>{shift.cashRegisterName.toUpperCase()}</small><h2 id="cash-detail-title">{shift.code}</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
    <div className="cash-detail-body">
      <section className="cash-detail-head">
        <div><small>CAJA</small><b>{shift.cashRegisterName}</b></div>
        <div><small>DÍA OPERATIVO</small><b>{formatBusinessDate(shift.businessDate)}</b></div>
        <div><small>CAJERO</small><b>{shift.openedByName}</b></div>
        <div><small>APERTURA</small><b>{formatDateTime(shift.openedAt)}</b></div>
        <div><small>CIERRE</small><b>{shift.closedAt?formatDateTime(shift.closedAt):"En curso"}</b></div>
        <Status tone={shift.status==="open"?"green":"gray"}>{shift.status==="open"?"Abierto":"Cerrado"}</Status>
      </section>
      <section className="cash-detail-totals">
        <div><small>Fondo inicial</small><b>{formatMoney(Number(shift.openingAmount))}</b></div>
        <div><small>Ingresos</small><b>{formatMoney(Number(shift.incomeAmount))}</b></div>
        <div><small>Egresos</small><b>{formatMoney(Number(shift.expenseAmount))}</b></div>
        <div><small>Esperado</small><b>{formatMoney(Number(shift.closingExpectedAmount??shift.expectedAmount))}</b></div>
        {shift.status==="closed"&&<><div><small>Contado</small><b>{formatMoney(Number(shift.closingCountedAmount??0))}</b></div><div className={variance===0?"balanced":variance>0?"positive":"negative"}><small>Diferencia</small><b>{variance>0?"+":""}{formatMoney(variance)}</b></div></>}
      </section>
      {(shift.openingNote||shift.closingNote)&&<section className="cash-detail-notes">{shift.openingNote&&<div><small>APERTURA</small><p>{shift.openingNote}</p></div>}{shift.closingNote&&<div><small>CIERRE</small><p>{shift.closingNote}</p></div>}</section>}
      <section className="cash-detail-movements">
        <header><div><small>MOVIMIENTOS</small><h3>{shift.movementCount} {shift.movementCount===1?"movimiento":"movimientos"}</h3></div></header>
        {shift.movements?.length?<div className="table-wrap hover-scroll"><table><thead><tr><th>FECHA</th><th>TIPO</th><th>ORIGEN</th><th>MOTIVO</th><th>USUARIO</th><th>MONTO</th></tr></thead><tbody>{shift.movements.map(item=><tr key={item.id}><td>{formatDateTime(item.createdAt)}</td><td><span className={"cash-movement-kind "+item.movementType}><Icon name={item.movementType==="income"?"plus":"minus"} size={12}/>{item.movementType==="income"?"Ingreso":"Egreso"}</span></td><td><span className="cash-movement-source">{cashMovementSourceLabel(item.sourceType)}</span></td><td><b>{item.reason}</b>{item.note&&<small>{item.note}</small>}</td><td>{item.createdByName}</td><td><b className={item.movementType}>{item.movementType==="income"?"+":"-"}{formatMoney(Number(item.amount))}</b></td></tr>)}</tbody></table></div>:<div className="cash-detail-empty"><Icon name="receipt" size={19}/><span>Este turno no registró movimientos.</span></div>}
      </section>
    </div>
  </section></div>;
}

function cashMovementSourceLabel(source:string){
  if(source==="cash_sale")return"Venta";
  if(source==="cash_refund")return"Devolución";
  if(source==="cash_pull")return"Retiro";
  if(source==="transfer_in")return"Transferencia entrada";
  if(source==="transfer_out")return"Transferencia salida";
  if(source==="deposit")return"Depósito";
  if(source==="adjustment")return"Ajuste";
  return"Manual";
}

function denominationsForCurrency(currency:string){
  if(currency==="PEN")return[200,100,50,20,10,5,2,1,.5,.2,.1];
  if(currency==="USD")return[100,50,20,10,5,1,.25,.1,.05,.01];
  if(currency==="EUR")return[200,100,50,20,10,5,2,1,.5,.2,.1,.05,.02,.01];
  return[100,50,20,10,5,1];
}
