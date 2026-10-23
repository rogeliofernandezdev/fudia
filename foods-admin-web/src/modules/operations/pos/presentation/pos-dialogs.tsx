"use client";
import {Dialog} from "@/design-system/dialog";
import {useForm,useWatch} from "react-hook-form";
import {Button,FormField,Icon,Input,Textarea,ModalHeader,ModalFooter} from "@/design-system";
import {paymentResolver,refundResolver} from "../domain/pos-schema";
import type {Payment,PaymentDraft,PaymentMethodOption,POSOrderSummary,RefundDraft} from "../domain/types";

/** El icono sigue a «Impacta caja»; el nombre y el orden vienen del catálogo. */
export function paymentMethodIcon(affectsCash:boolean){return affectsCash?"cash" as const:"wallet" as const}

export function PaymentDialog({order,methods,shiftName,busy,formatMoney,close,save}:{order:POSOrderSummary;methods:PaymentMethodOption[];shiftName:string;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;save:(draft:PaymentDraft)=>void}){
  const{control,register,handleSubmit,formState:{errors}}=useForm<PaymentDraft>({
    defaultValues:{method:methods[0]?.code??"",amount:Number(order.remainingAmount).toFixed(2),reference:""},
    resolver:paymentResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const method=useWatch({control,name:"method"});
  const selected=methods.find(option=>option.code===method);
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal pos-payment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-payment-title" aria-busy={busy}>

    <ModalHeader title={order.tableName||order.customerName||"Registrar cobro"} titleId="pos-payment-title" icon="cash" subtitle="COBRO" onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="pos-payment-body">
        <section className="pos-payment-overview">
          <div className="pos-payment-balance"><span><Icon name="wallet" size={18}/></span><div><small>SALDO A COBRAR</small><strong>{formatMoney(Number(order.remainingAmount))}</strong></div></div>
          <dl><div><dt>Total del pedido</dt><dd>{formatMoney(Number(order.total))}</dd></div><div><dt>Pagado</dt><dd>{formatMoney(Number(order.paidAmount))}</dd></div></dl>
        </section>
        <div className="pos-shift-note"><Icon name="register" size={14}/><span>Turno activo: <b>{shiftName}</b></span></div>
        <fieldset className="pos-methods">
          <legend>Método de pago</legend>
          {methods.map(option=><label className={"pos-method"+(method===option.code?" active":"")} key={option.code}>
            <input type="radio" value={option.code} {...register("method")}/>
            <i aria-hidden="true"/>
            <span className="pos-method-icon"><Icon name={paymentMethodIcon(option.affectsCash)} size={18}/></span>
            <span>{option.name}</span>
          </label>)}
          {errors.method?.message&&<small className="field-error">{errors.method.message}</small>}
        </fieldset>
        <div className="pos-payment-fields">
        <FormField label="Monto" error={errors.amount?.message}>
          <Input type="number" min="0.01" max={order.remainingAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
        </FormField>
        {selected&&!selected.affectsCash&&<FormField label="Referencia" optional error={errors.reference?.message}>
          <Input maxLength={120} {...register("reference")} placeholder="Ej. voucher o número de operación"/>
        </FormField>}
        </div>
        {selected?.affectsCash&&<div className="pos-cash-impact"><Icon name="cash" size={14}/><span>Este cobro incrementará automáticamente el efectivo esperado del turno.</span></div>}
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Registrando…":"Registrar cobro"}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}

export function RefundDialog({payment,busy,formatMoney,close,save}:{payment:Payment;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;save:(draft:RefundDraft)=>void}){
  const max=Number(payment.netAmount);
  const{register,handleSubmit,formState:{errors}}=useForm<RefundDraft>({
    defaultValues:{amount:max.toFixed(2),reason:"",note:""},
    resolver:refundResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal pos-payment-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="pos-refund-title" aria-busy={busy}>

    <ModalHeader title={payment.orderCode} titleId="pos-refund-title" icon="undo" subtitle="DEVOLUCIÓN" onClose={close} busy={busy} closeLabel="Cerrar"/>
    <form onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <div className="pos-payment-body">
        <div className="pos-refund-source"><span className={"pos-payment-method"+(payment.affectsCash?" cash":"")}><Icon name={paymentMethodIcon(payment.affectsCash)} size={16}/></span><div><small>PAGO DISPONIBLE PARA DEVOLVER</small><strong>{formatMoney(max)}</strong><span>{payment.methodName} · {payment.cashRegisterName}</span></div></div>
        <div className="pos-payment-fields refund-fields">
        <FormField label="Monto a devolver" error={errors.amount?.message}>
          <Input autoFocus type="number" min="0.01" max={payment.netAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
        </FormField>
        <FormField label="Motivo" error={errors.reason?.message}>
          <Input maxLength={120} {...register("reason")} aria-invalid={Boolean(errors.reason)} placeholder="Ej. Cobro duplicado"/>
        </FormField>
        <FormField label="Observación" optional className="pos-refund-note">
          <Textarea rows={3} maxLength={240} {...register("note")} placeholder="Detalle adicional de la devolución"/>
        </FormField>
        </div>
        {payment.affectsCash&&<div className="pos-refund-warning"><Icon name="alert" size={14}/><span>La devolución saldrá del efectivo esperado del turno que estás operando ahora.</span></div>}
      </div>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" kind="danger" icon="undo" disabled={busy} loading={busy}>{busy?"Devolviendo…":"Confirmar devolución"}</Button></ModalFooter>
    </form>
  </Dialog></div>;
}
