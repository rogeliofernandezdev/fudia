"use client";
import {useForm} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {paymentResolver,refundResolver} from "../domain/pos-schema";
import type {Payment,PaymentDraft,POSOrderSummary,RefundDraft} from "../domain/types";

export function PaymentDialog({order,shiftName,busy,formatMoney,close,save}:{order:POSOrderSummary;shiftName:string;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;save:(draft:PaymentDraft)=>void}){
  const{register,handleSubmit,watch,formState:{errors}}=useForm<PaymentDraft>({
    defaultValues:{method:"cash",amount:Number(order.remainingAmount).toFixed(2),reference:""},
    resolver:paymentResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const method=watch("method");
  return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal pos-payment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-payment-title" aria-busy={busy}>
    <div className="modal-accent"/>
    <header><span className="modal-title-icon"><Icon name="sales" size={18}/></span><div><small>COBRO</small><h2 id="pos-payment-title">{order.code}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="pos-payment-body">
        <section className="pos-payment-summary">
          <div><small>TOTAL</small><b>{formatMoney(Number(order.total))}</b></div>
          <div><small>PAGADO</small><b>{formatMoney(Number(order.paidAmount))}</b></div>
          <div className="remaining"><small>SALDO</small><strong>{formatMoney(Number(order.remainingAmount))}</strong></div>
        </section>
        <div className="pos-shift-note"><Icon name="sales" size={14}/><span>Turno activo: <b>{shiftName}</b></span></div>
        <label>Método de pago
          <Select {...register("method")}>
            <option value="cash">Efectivo</option>
            <option value="card">Tarjeta</option>
            <option value="transfer">Transferencia</option>
            <option value="other">Otro</option>
          </Select>
        </label>
        <label>Monto
          <Input type="number" min="0.01" max={order.remainingAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
          {errors.amount?.message&&<small className="wizard-field-error">{errors.amount.message}</small>}
        </label>
        {method!=="cash"&&<label>Referencia opcional
          <Input maxLength={120} {...register("reference")} placeholder={method==="card"?"Ej. voucher o últimos 4 dígitos":method==="transfer"?"Ej. código de operación":"Referencia del cobro"}/>
          {errors.reference?.message&&<small className="wizard-field-error">{errors.reference.message}</small>}
        </label>}
        {method==="cash"&&<div className="pos-cash-impact"><Icon name="plus" size={14}/><span>Este cobro incrementará automáticamente el efectivo esperado del turno.</span></div>}
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" icon="check" disabled={busy}>{busy?"Cobrando…":"Registrar cobro"}</Button></footer>
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
    <header><span className="modal-title-icon"><Icon name="refresh" size={18}/></span><div><small>DEVOLUCIÓN</small><h2 id="pos-refund-title">{payment.orderCode}</h2></div><button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button></header>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="pos-payment-body">
        <div className="pos-refund-source"><small>PAGO DISPONIBLE PARA DEVOLVER</small><strong>{formatMoney(max)}</strong><span>{payment.method==="cash"?"Efectivo":payment.method==="card"?"Tarjeta":payment.method==="transfer"?"Transferencia":"Otro"} · {payment.cashRegisterName}</span></div>
        <label>Monto a devolver
          <Input autoFocus type="number" min="0.01" max={payment.netAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
          {errors.amount?.message&&<small className="wizard-field-error">{errors.amount.message}</small>}
        </label>
        <label>Motivo
          <Input maxLength={120} {...register("reason")} aria-invalid={Boolean(errors.reason)} placeholder="Ej. Cobro duplicado"/>
          {errors.reason?.message&&<small className="wizard-field-error">{errors.reason.message}</small>}
        </label>
        <label>Observación opcional
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Detalle adicional de la devolución"/>
        </label>
        {payment.method==="cash"&&<div className="pos-refund-warning"><Icon name="alert" size={14}/><span>La devolución saldrá del efectivo esperado del turno que estás operando ahora.</span></div>}
      </div>
      <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" kind="danger" icon="refresh" disabled={busy}>{busy?"Devolviendo…":"Confirmar devolución"}</Button></footer>
    </form>
    {busy&&<div className="modal-busy" role="status"><i/><span>Procesando devolución…</span></div>}
  </section></div>;
}
