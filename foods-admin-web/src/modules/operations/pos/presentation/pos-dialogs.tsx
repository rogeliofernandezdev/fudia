"use client";
import {Dialog} from "@/design-system/dialog";
import {useForm,useWatch} from "react-hook-form";
import {Button,FormField,FormSection,Icon,Input,Textarea,ModalHeader,ModalForm,ModalBody,ModalFooter} from "@/design-system";
import {cashChange,paymentResolverFor,refundResolver} from "../domain/pos-schema";
import type {Payment,PaymentDraft,PaymentMethodOption,POSOrderSummary,RefundDraft} from "../domain/types";

/** El icono sigue a «Impacta caja»; el nombre y el orden vienen del catálogo. */
export function paymentMethodIcon(affectsCash:boolean){return affectsCash?"cash" as const:"wallet" as const}

export function PaymentForm({order,methods,shiftName,busy,formatMoney,close,save}:{order:POSOrderSummary;methods:PaymentMethodOption[];shiftName:string;busy:boolean;formatMoney:(value:number)=>string;close:()=>void;save:(draft:PaymentDraft)=>void}){
  const{control,register,handleSubmit,formState:{errors}}=useForm<PaymentDraft>({
    defaultValues:{method:methods[0]?.code??"",amount:Number(order.remainingAmount).toFixed(2),reference:"",receivedAmount:""},
    resolver:paymentResolverFor(methods,Number(order.remainingAmount)),
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const method=useWatch({control,name:"method"});
  const amount=useWatch({control,name:"amount"});
  const received=useWatch({control,name:"receivedAmount"});
  const selected=methods.find(option=>option.code===method);
  const change=cashChange(amount,received);
  return <ModalForm onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <ModalBody className="pos-payment-body">
        <section className="pos-payment-overview">
          <div className="pos-payment-balance"><span><Icon name="wallet" size={18}/></span><div><small>Saldo a cobrar</small><strong>{formatMoney(Number(order.remainingAmount))}</strong></div></div>
          <dl><div><dt>Total del pedido</dt><dd>{formatMoney(Number(order.total))}</dd></div><div><dt>Pagado</dt><dd>{formatMoney(Number(order.paidAmount))}</dd></div></dl>
        </section>
        <div className="pos-shift-note"><Icon name="register" size={14}/><span>Turno activo: <b>{shiftName}</b></span></div>
        <FormSection icon="payment" tone="brand" title="Medio de pago" bare>
        <fieldset className="pos-methods">
          <legend className="sr-only">Medio de pago</legend>
          {methods.map(option=><label className={"pos-method"+(method===option.code?" active":"")} key={option.code}>
            <input type="radio" value={option.code} {...register("method")}/>
            <i aria-hidden="true"/>
            <span className="pos-method-icon"><Icon name={paymentMethodIcon(option.affectsCash)} size={18}/></span>
            <span>{option.name}</span>
          </label>)}
          {errors.method?.message&&<small className="field-error">{errors.method.message}</small>}
        </fieldset>
        </FormSection>
        <FormSection icon="cash" title="Monto del cobro" bare>
        <div className="pos-payment-fields">
        <FormField label="Monto" error={errors.amount?.message}>
          <Input type="number" min="0.01" max={order.remainingAmount} step="0.01" inputMode="decimal" {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
        </FormField>
        {selected?.affectsCash&&<>
        <FormField label="Efectivo recibido" error={errors.receivedAmount?.message}>
          <Input type="number" min={amount||"0.01"} step="0.01" inputMode="decimal" {...register("receivedAmount")} aria-invalid={Boolean(errors.receivedAmount)} placeholder="0.00"/>
        </FormField>
        <div className="pos-cash-change" role="status" aria-live="polite"><span>Vuelto</span><strong>{change===null?"—":formatMoney(change)}</strong></div>
        </>}
        {selected&&!selected.affectsCash&&<FormField label="Referencia" optional error={errors.reference?.message}>
          <Input icon="invoice" maxLength={120} {...register("reference")} placeholder="Ej. voucher o número de operación"/>
        </FormField>}
        </div>
        </FormSection>
        {selected?.affectsCash&&<div className="pos-cash-impact"><Icon name="cash" size={14}/><span>Este cobro incrementará automáticamente el efectivo esperado del turno.</span></div>}
      </ModalBody>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy} loading={busy}>{busy?"Registrando…":"Registrar cobro"}</Button></ModalFooter>
    </ModalForm>;
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

    <ModalHeader title={payment.orderCode} titleId="pos-refund-title" icon="undo" subtitle="Devolución" onClose={close} busy={busy} closeLabel="Cerrar"/>
    <ModalForm onSubmit={handleSubmit(save)} noValidate inert={busy}>
      <ModalBody className="pos-payment-body">
        <div className="pos-refund-source"><span className={"pos-payment-method"+(payment.affectsCash?" cash":"")}><Icon name={paymentMethodIcon(payment.affectsCash)} size={16}/></span><div><small>Pago disponible para devolver</small><strong>{formatMoney(max)}</strong><span>{payment.methodName} · {payment.cashRegisterName}</span></div></div>
        <FormSection icon="undo" tone="peach" title="Datos de la devolución" bare>
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
        </FormSection>
        {payment.affectsCash&&<div className="pos-refund-warning"><Icon name="alert" size={14}/><span>La devolución saldrá del efectivo esperado del turno que estás operando ahora.</span></div>}
      </ModalBody>
      <ModalFooter><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" kind="danger" icon="undo" disabled={busy} loading={busy}>{busy?"Devolviendo…":"Confirmar devolución"}</Button></ModalFooter>
    </ModalForm>
  </Dialog></div>;
}
