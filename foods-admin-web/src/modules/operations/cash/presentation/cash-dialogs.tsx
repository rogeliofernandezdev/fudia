"use client";
import {Dialog} from "@/design-system/dialog";
import {cashShiftAttribution,cashShiftTeamName} from "../domain/shift-attribution";
import {useRef,useState} from "react";
import {CashReportContent} from "./cash-report-content";
import {useForm,useWatch} from "react-hook-form";
import {Button,FormField,FormSection,Icon,Input,Status,Tabs,Textarea,ModalHeader,ModalFooter,ModalItemList,ModalWorkspace,DetailHero,DetailContact,DetailFacts,DetailTimeline} from "@/design-system";
import {cashMovementResolver,cashRegisterResolver,closeCashShiftResolver,openCashShiftResolver} from "../domain/cash-schema";
import type {CashMovementDraft,CashMovementType,CashRegister,CashRegisterDraft,CashShift,CashShiftReport,CloseCashShiftDraft,OpenCashShiftDraft} from "../domain/types";

export function CashRegisterDialog({initial,busy,close,save}:{initial?:CashRegister|null;busy:boolean;close:()=>void;save:(draft:CashRegisterDraft)=>void}){
  const editing=Boolean(initial);
  const{register,handleSubmit,formState:{errors}}=useForm<CashRegisterDraft>({
    defaultValues:{name:initial?.name??"",blindClose:initial?.blindClose??false},
    resolver:cashRegisterResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-register-title" aria-busy={busy}>

    <ModalHeader title={editing?"Editar caja":"Registrar caja"} titleId="cash-register-title" icon="sales" subtitle="Cajas" onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        <p className="cash-dialog-intro">{editing?"Actualiza el nombre operativo de la caja. Su historial y turnos se conservarán.":"Crea una caja del local. Luego podrás iniciar y cerrar turnos sobre esta caja sin perder su historial."}</p>
        <FormSection icon="register" title="Datos de la caja" bare>
        <FormField label="Nombre de la caja" icon="register" error={errors.name?.message}>
          <Input autoFocus maxLength={80} {...register("name")} placeholder="Ej. Caja principal"/>
        </FormField>
        </FormSection>
        <FormSection icon="lock" tone="digital" title="Control del cierre" bare>
        <label className="switch-row cash-blind-switch">
          <input type="checkbox" {...register("blindClose")}/>
          <span/>
          <b>Cierre ciego<small>El cajero contará el efectivo sin ver el saldo esperado. Supervisores con permiso sí podrán consultarlo.</small></b>
        </label>
        </FormSection>
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Guardando…":"Guardar"}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}

export function OpenCashShiftDialog({cashRegister,busy,close,save}:{cashRegister:CashRegister;busy:boolean;close:()=>void;save:(draft:OpenCashShiftDraft)=>void}){
  const{register,handleSubmit,formState:{errors}}=useForm<OpenCashShiftDraft>({
    defaultValues:{openingAmount:"0",note:""},
    resolver:openCashShiftResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-open-title" aria-busy={busy}>

    <ModalHeader title="Iniciar turno" titleId="cash-open-title" icon="clock" subtitle={cashRegister.name} onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="cash-dialog-body">
        <p className="cash-dialog-intro">Estás iniciando un turno en <b>{cashRegister.name}</b>. Registra el efectivo inicial que quedará como base del arqueo.</p>
        <FormSection icon="cash" tone="brand" title="Fondo del turno" bare>
        <FormField label="Fondo inicial" error={errors.openingAmount?.message}>
          <Input autoFocus type="number" min="0" step="0.01" inputMode="decimal" {...register("openingAmount")} placeholder="0.00"/>
        </FormField>
        <FormField label="Observación" optional error={errors.note?.message}>
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Ej. Fondo recibido del turno anterior"/>
        </FormField>
        </FormSection>
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Iniciando…":"Iniciar turno"}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}

export function CashMovementDialog({type,busy,close,save}:{type:CashMovementType;busy:boolean;close:()=>void;save:(draft:CashMovementDraft)=>void}){
  const isIncome=type==="income";
  const{register,handleSubmit,formState:{errors}}=useForm<CashMovementDraft>({
    defaultValues:{movementType:type,amount:"",reason:"",note:""},
    resolver:cashMovementResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-movement-title" aria-busy={busy}>

    <ModalHeader title={isIncome?"Registrar ingreso":"Registrar egreso"} titleId="cash-movement-title" icon={isIncome?"plus":"minus"} subtitle="Movimiento de caja" onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <input type="hidden" {...register("movementType")}/>
      <div className="cash-dialog-body">
        <p className="cash-dialog-intro">{isIncome?"Registra efectivo que entra a la caja fuera de un cobro de venta.":"Registra efectivo que sale de caja y deja el motivo trazable."}</p>
        <FormSection icon={isIncome?"plus":"minus"} tone={isIncome?"brand":"peach"} title={isIncome?"Datos del ingreso":"Datos del egreso"} bare>
        <FormField label="Monto" error={errors.amount?.message}>
          <Input autoFocus type="number" min="0.01" step="0.01" inputMode="decimal" {...register("amount")} placeholder="0.00"/>
        </FormField>
        <FormField label="Motivo" error={errors.reason?.message}>
          <Input maxLength={120} {...register("reason")} placeholder={isIncome?"Ej. Reposición de fondo":"Ej. Compra menor o retiro"}/>
        </FormField>
        <FormField label="Observación" optional error={errors.note?.message}>
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Referencia o detalle adicional"/>
        </FormField>
        </FormSection>
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Guardando…":isIncome?"Registrar ingreso":"Registrar egreso"}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}

export function CloseCashShiftDialog({shift,report,busy,currency,formatMoney,close,save}:{shift:CashShift;report?:CashShiftReport;busy:boolean;currency:string;formatMoney:(value:number)=>string;close:()=>void;save:(draft:CloseCashShiftDraft)=>void|Promise<void>}){
  const blind=shift.blindClose&&!shift.expectedVisible;
  const expected=Number(shift.expectedAmount||0);
  const denominations=denominationsForCurrency(currency);
  const[byDenomination,setByDenomination]=useState(false);
  const sending=useRef(false);
  const{control,register,handleSubmit,setValue,formState:{errors}}=useForm<CloseCashShiftDraft>({
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

  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-modal cash-close-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-close-title" aria-busy={busy}>

    <ModalHeader title="Cerrar turno" titleId="cash-close-title" icon="lock" subtitle={shift.cashRegisterName} onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={event=>event.preventDefault()} onKeyDown={event=>{if(event.key==="Enter"&&(event.target as HTMLElement).tagName==="INPUT")event.preventDefault();}} noValidate inert={busy}>
      <div className="cash-dialog-body">
        {report&&<CashReportContent report={report}/>}
        {blind
          ?<div className="cash-blind-close-note"><Icon name="lock" size={16}/><span><b>Cierre ciego activo</b><small>Cuenta el efectivo sin consultar el saldo esperado. La diferencia se calculará al confirmar.</small></span></div>
          :!report&&<section className="cash-close-summary">
            <div><small>Fondo inicial</small><b>{formatMoney(Number(shift.openingAmount))}</b></div>
            <div><small>Ingresos</small><b>{formatMoney(Number(shift.incomeAmount))}</b></div>
            <div><small>Egresos</small><b>{formatMoney(Number(shift.expenseAmount))}</b></div>
            <div className="expected"><small>Efectivo esperado</small><strong>{formatMoney(expected)}</strong></div>
          </section>}

        <FormSection icon="cash" tone="brand" title="Conteo de efectivo" bare>
        <Tabs className="cash-count-mode" label="Modo de conteo" value={byDenomination?"denominations":"total"} onChange={value=>setByDenomination(value==="denominations")} items={[{value:"total",label:"Monto total",icon:"cash"},{value:"denominations",label:"Por denominaciones",icon:"wallet"}]}/>

        {!byDenomination?<FormField label="Efectivo contado" error={errors.countedAmount?.message}>
          <Input autoFocus type="number" min="0" step="0.01" inputMode="decimal" {...register("countedAmount")} placeholder="0.00"/>
        </FormField>:<section className="cash-denomination-count">
          <header><div><small>Conteo</small><b>Billetes y monedas</b></div><strong>{formatMoney(denominationTotal)}</strong></header>
          <div>{denominations.map((value,index)=><label key={value}>
            <span>{formatMoney(value)}</span>
            <input type="hidden" {...register(`counts.${index}.denomination`)}/>
            <Input type="number" min="0" step="1" inputMode="numeric" {...register(`counts.${index}.quantity`)} placeholder="0"/>
            <b>{formatMoney(value*(Number(countLines[index]?.quantity)||0))}</b>
          </label>)}</div>
          {errors.countedAmount?.message&&<small className="field-error" role="alert">{errors.countedAmount.message}</small>}
        </section>}

        {!blind&&Number.isFinite(counted)&&counted>=0&&<div className={"cash-close-difference "+differenceTone}><span><small>Diferencia</small><b>{difference>0?"+":""}{formatMoney(difference)}</b></span><em>{differenceTone==="balanced"?"Caja cuadrada":differenceTone==="positive"?"Sobrante":"Faltante"}</em></div>}

        </FormSection>
        <FormSection icon="note" tone="neutral" title="Observación" bare>
        <FormField label="Observación" optional error={errors.note?.message}>
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Explica cualquier diferencia si corresponde"/>
        </FormField>
        </FormSection>
        <p className="cash-close-warning"><Icon name="alert" size={14}/>Al cerrar el turno ya no se podrán registrar nuevos movimientos.</p>
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="button" icon="lock" disabled={busy} onClick={()=>{if(busy||sending.current)return;sending.current=true;if(byDenomination)setValue("countedAmount",denominationTotal.toFixed(2));else setValue("counts",denominations.map(value=>({denomination:String(value),quantity:""})));void handleSubmit(async draft=>{try{await save({...draft,counts:byDenomination?draft.counts:[]});}finally{sending.current=false;}},()=>{sending.current=false;})().catch(()=>{sending.current=false;});}} loading={busy}>{busy?"Cerrando…":"Cerrar turno"}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}

export function CashShiftDetailDialog({shift,formatMoney,formatDateTime,formatBusinessDate,close,viewReport}:{shift:CashShift;formatMoney:(value:number)=>string;formatDateTime:(value:string)=>string;formatBusinessDate:(value:string)=>string;close:()=>void;viewReport?:()=>void}){
  const closed=shift.status==="closed";
  const variance=Number(shift.varianceAmount??0);
  const movements=shift.movements??[];
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-detail-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-detail-title">
    <ModalHeader title="Detalle del turno" titleId="cash-detail-title" icon="sales" onClose={close} closeLabel="Cerrar"/>
    <ModalWorkspace summary={<section className="modal-summary" aria-labelledby="cash-detail-summary-title">
      <div className="modal-summary-heading"><h3 id="cash-detail-summary-title" className="modal-section-title">Resumen de caja</h3></div>
      <div className="modal-summary-lines">
        <div className="modal-summary-line"><span>Fondo inicial</span><b>{formatMoney(Number(shift.openingAmount))}</b></div>
        <div className="modal-summary-line"><span>Ingresos</span><b>{formatMoney(Number(shift.incomeAmount))}</b></div>
        <div className="modal-summary-line"><span>Egresos</span><b>{formatMoney(Number(shift.expenseAmount))}</b></div>
        {closed&&<div className="modal-summary-line"><span>Contado</span><b>{formatMoney(Number(shift.closingCountedAmount??0))}</b></div>}
        {closed&&<div className={"modal-summary-line cash-detail-variance "+(variance===0?"balanced":variance>0?"positive":"negative")}><span>Diferencia</span><b>{variance>0?"+":""}{formatMoney(variance)}</b></div>}
      </div>
      <div className="modal-summary-total"><span>Esperado</span><strong>{formatMoney(Number(shift.closingExpectedAmount??shift.expectedAmount))}</strong></div>
    </section>}>
    <div className="modal-body cash-detail-body" tabIndex={0} aria-label="Datos y movimientos del turno">
      <DetailHero title={shift.cashRegisterName} status={<Status tone={closed?"gray":"green"}>{closed?"Cerrado":"Abierto"}</Status>} meta={<><Icon name="calendar" size={14}/>Día operativo {formatBusinessDate(shift.businessDate)}</>}/>
      <DetailContact icon="user" name={shift.openedByName||"Sin responsable registrado"} detail={<>{closed?"Abrió el turno":"Cajero"} · <time dateTime={shift.openedAt}>{formatDateTime(shift.openedAt)}</time></>}/>
      <DetailFacts label="Datos del turno" items={[
        {label:"Apertura",value:<time dateTime={shift.openedAt}>{formatDateTime(shift.openedAt)}</time>},
        ...(closed
          ?[{label:"Cerrado por",value:<>{cashShiftAttribution(shift).name}{shift.closedAt&&<small>Cierre: <time dateTime={shift.closedAt}>{formatDateTime(shift.closedAt)}</time></small>}</>}]
          :[{label:"Equipo actual",value:cashShiftTeamName(shift)}]),
      ]}/>
      {(shift.openingNote||shift.closingNote)&&<section className="modal-detail-notes cash-detail-notes"><Icon name="edit" size={16}/><div><h3>Notas del turno</h3>{shift.openingNote&&<p><b>Apertura:</b> {shift.openingNote}</p>}{shift.closingNote&&<p><b>Cierre:</b> {shift.closingNote}</p>}</div></section>}
      <section className="cash-detail-section" aria-labelledby="cash-detail-movements-title">
        <header className="cash-detail-section-head modal-section-rule"><h3 id="cash-detail-movements-title" className="modal-section-title">Movimientos</h3><span>{shift.movementCount} {shift.movementCount===1?"movimiento":"movimientos"}</span></header>
        {movements.length?<DetailTimeline label="Movimientos del turno" steps={movements.map(item=>({
          key:item.id,
          icon:item.movementType==="income"?"plus":"minus",
          title:item.reason,
          time:<time dateTime={item.createdAt}>{formatDateTime(item.createdAt)}</time>,
          description:[item.movementType==="income"?"Ingreso":"Egreso",cashMovementSourceLabel(item.sourceType),item.createdByName,item.note].filter(Boolean).join(" · "),
          aside:<span className={"cash-detail-movement-amount "+item.movementType}>{item.movementType==="income"?"+":"-"}{formatMoney(Number(item.amount))}</span>,
        }))}/>:<p className="cash-detail-empty">Este turno no registró movimientos.</p>}
      </section>
    </div></ModalWorkspace>
    {viewReport&&(closed||shift.expectedVisible)&&<ModalFooter><Button kind="secondary" icon="eye" onClick={viewReport}>Previsualizar informe</Button></ModalFooter>}
  </Dialog></div>;
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
