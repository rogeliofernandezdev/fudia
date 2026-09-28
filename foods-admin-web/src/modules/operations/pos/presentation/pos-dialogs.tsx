"use client";
import {useForm,useWatch} from "react-hook-form";
import {Button,Icon,Input,Textarea} from "@/design-system";
import {paymentMethodMeta} from "./pos-meta";
import {paymentResolver,refundResolver} from "../domain/pos-schema";
import type {Payment,PaymentDraft,POSOrderSummary,RefundDraft} from "../domain/types";

export function PaymentDialog({order,shiftName,busy,formatMoney,close,save}:{order:POSOrderSummary;shiftName:string;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;save:(draft:PaymentDraft)=>void}){
  const{control,register,handleSubmit,formState:{errors}}=useForm<PaymentDraft>({
    defaultValues:{method:"cash",amount:Number(order.remainingAmount).toFixed(2),reference:""},
    resolver:paymentResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const method=useWatch({control,name:"method"});
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal pos-payment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-payment-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="cash" size={18}/></span><div><small>COBRO</small><h2 id="pos-payment-title">{order.code}{order.tableName?` · ${order.tableName}`:order.customerName?` · ${order.customerName}`:""}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="pos-payment-body">
        <section className="pos-payment-summary">
          <div><small>TOTAL</small><b>{formatMoney(Number(order.total))}</b></div>
          <div><small>PAGADO</small><b>{formatMoney(Number(order.paidAmount))}</b></div>
          <div className="remaining"><small>SALDO</small><strong>{formatMoney(Number(order.remainingAmount))}</strong></div>
        </section>
        <div className="pos-shift-note"><Icon name="register" size={14}/><span>Turno activo: <b>{shiftName}</b></span></div>
        <fieldset className="pos-methods">
          <legend>Método de pago</legend>
          {(["cash","card","transfer","other"] as const).map(value=><label className={"pos-method"+(method===value?" active":"")} key={value}>
            <input type="radio" value={value} {...register("method")}/>
            <span className="pos-method-icon"><Icon name={paymentMethodMeta[value].icon} size={18}/></span>
            <span>{paymentMethodMeta[value].label}</span>
          </label>)}
        </fieldset>
        <label>Monto
          <Input type="number" min="0.01" max={order.remainingAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
          {errors.amount?.message&&<small className="field-error">{errors.amount.message}</small>}
        </label>
        {method!=="cash"&&<label>Referencia opcional
          <Input maxLength={120} {...register("reference")} placeholder={method==="card"?"Ej. voucher o últimos 4 dígitos":method==="transfer"?"Ej. código de operación":"Referencia del cobro"}/>
          {errors.reference?.message&&<small className="field-error">{errors.reference.message}</small>}
        </label>}
        {method==="cash"&&<div className="pos-cash-impact"><Icon name="cash" size={14}/><span>Este cobro incrementará automáticamente el efectivo esperado del turno.</span></div>}
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Cobrando…":"Registrar cobro"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Registrando cobro…</span></div>}
  </section></div>;
}

export function RefundDialog({payment,busy,formatMoney,close,save}:{payment:Payment;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;save:(draft:RefundDraft)=>void}){
  const max=Number(payment.netAmount);
  const{register,handleSubmit,formState:{errors}}=useForm<RefundDraft>({
    defaultValues:{amount:max.toFixed(2),reason:"",note:""},
    resolver:refundResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal pos-payment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-refund-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="undo" size={18}/></span><div><small>DEVOLUCIÓN</small><h2 id="pos-refund-title">{payment.orderCode}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="pos-payment-body">
        <div className="pos-refund-source"><span className={"pos-payment-method pm-"+payment.method}><Icon name={paymentMethodMeta[payment.method]?.icon??"wallet"} size={16}/></span><div><small>PAGO DISPONIBLE PARA DEVOLVER</small><strong>{formatMoney(max)}</strong><span>{paymentMethodMeta[payment.method]?.label??"Otro"} · {payment.cashRegisterName}</span></div></div>
        <label>Monto a devolver
          <Input autoFocus type="number" min="0.01" max={payment.netAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
          {errors.amount?.message&&<small className="field-error">{errors.amount.message}</small>}
        </label>
        <label>Motivo
          <Input maxLength={120} {...register("reason")} aria-invalid={Boolean(errors.reason)} placeholder="Ej. Cobro duplicado"/>
          {errors.reason?.message&&<small className="field-error">{errors.reason.message}</small>}
        </label>
        <label>Observación opcional
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Detalle adicional de la devolución"/>
        </label>
        {payment.method==="cash"&&<div className="pos-refund-warning"><Icon name="alert" size={14}/><span>La devolución saldrá del efectivo esperado del turno que estás operando ahora.</span></div>}
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" kind="danger" icon="undo" disabled={busy}>{busy?"Devolviendo…":"Confirmar devolución"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Procesando devolución…</span></div>}
  </section></div>;
}
