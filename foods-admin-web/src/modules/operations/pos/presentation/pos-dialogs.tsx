"use client";
import {Dialog} from "@/design-system/dialog";
import {useState} from "react";
import {useForm,useWatch} from "react-hook-form";
import {Button,FormDisclosure,FormField,FormSection,Icon,IconTile,Input,MoneyInput,Select,Textarea,ModalHeader,ModalForm,ModalBody,ModalFooter,type IconTileTone} from "@/design-system";
import {calculateTip,canCollectTip,tipInputValue,type TipPolicy,type TipReceipt} from "../../tips";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {CheckoutFiscalSection,blankFiscalDraft,fiscalRequest,validateFiscalDraft,type CheckoutOptions,type FiscalDraft,type FiscalErrors} from "@/modules/einvoicing";
import {cashChange,cashSuggestions,paymentResolverFor,refundResolver} from "../domain/pos-schema";
import type {POSCollectionShift,Payment,PaymentDraft,PaymentMethodOption,POSOrder,POSOrderSummary,RefundDraft} from "../domain/types";

/** El icono sigue a «Impacta caja»; el nombre y el orden vienen del catálogo. */
export function paymentMethodIcon(affectsCash:boolean){return affectsCash?"cash" as const:"wallet" as const}

const digitalTones:IconTileTone[]=["peach","digital","sky","teal","rose"];
/**
 * Color de identidad de cada medio, sin depender de su nombre: el efectivo es verde y los
 * demás reciben un tono distinto según su orden en el catálogo del API.
 */
export function paymentMethodTones(methods:PaymentMethodOption[]){
  let digital=0;
  return Object.fromEntries(methods.map(method=>[method.code,method.affectsCash?"brand":digitalTones[digital++%digitalTones.length]])) as Record<string,IconTileTone>;
}

export function PaymentForm({order,methods,shiftName,shiftOptions=[],selectedShiftId="",selectShift,busy,formatMoney,currency,close,save,tipPolicy,registeredTip,fiscalOptions}:{order:POSOrderSummary&Pick<POSOrder,"waiterName">;methods:PaymentMethodOption[];shiftName:string;shiftOptions?:POSCollectionShift[];selectedShiftId?:string;selectShift?:(id:string)=>void;busy:boolean;formatMoney:(value:number)=>string;currency:{symbol:string;position:"before"|"after"};close:()=>void;save:(draft:PaymentDraft)=>void;tipPolicy?:TipPolicy;registeredTip?:TipReceipt;fiscalOptions?:CheckoutOptions}){
  const tipAllowed=canCollectTip(tipPolicy,order.channel,Boolean(registeredTip));
  const tipBase=Math.max(0,Number(order.total)-Number(order.deliveryFee??0));
  const tipPercentage=tipPolicy?.percentage??0;
  const{control,register,setValue,handleSubmit,formState:{errors}}=useForm<PaymentDraft>({
    defaultValues:{method:methods[0]?.code??"",amount:Number(order.remainingAmount).toFixed(tipPolicy?.decimals??2),reference:"",receivedAmount:"",tipEnabled:false,tipKind:tipPercentage>0?"percentage":"fixed",tipValue:""},
    resolver:paymentResolverFor(methods,Number(order.remainingAmount),tipPolicy,tipBase),
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const method=useWatch({control,name:"method"});
  const amount=useWatch({control,name:"amount"});
  const received=useWatch({control,name:"receivedAmount"});
  const enabled=useWatch({control,name:"tipEnabled"});
  const kind=useWatch({control,name:"tipKind"})??"percentage";
  const value=useWatch({control,name:"tipValue"})??"";
  const tip=calculateTip(Boolean(enabled)&&tipAllowed,kind,tipInputValue(kind,tipPolicy,value),tipBase,tipPolicy?.decimals??2);
  const collected=(Number(amount)||0)+tip;
  const selected=methods.find(option=>option.code===method);
  const decimals=tipPolicy?.decimals??2;
  const remaining=Number(order.remainingAmount);
  const change=cashChange(String(collected),received);
  const tendered=Number(received);
  const missing=received?.trim()&&Number.isFinite(tendered)&&tendered>0&&tendered<collected?collected-tendered:0;
  const pending=Number(amount)>0&&Number(amount)<remaining?remaining-Number(amount):0;
  const suggestions=selected?.affectsCash?cashSuggestions(collected,decimals):[];
  const setTender=(next:string)=>setValue("receivedAmount",next,{shouldDirty:true,shouldValidate:Boolean(errors.receivedAmount)});
  const paid=Number(order.paidAmount);
  const[partial,setPartial]=useState(false);
  const showPartial=partial||pending>0||Boolean(errors.amount);
  const[tipOpen,setTipOpen]=useState(false);
  const showTip=tipOpen||Boolean(enabled)||Boolean(errors.tipKind||errors.tipValue);
  const result=missing>0?{tone:" short",label:"Falta",value:formatMoney(missing)}:change!==null?{tone:" ready",label:"Vuelto",value:formatMoney(change)}:{tone:"",label:"Vuelto",value:"—"};
  const adjusted=tip>0||pending>0;
  const facts=[paid>0&&`Pagado ${formatMoney(paid)} de ${formatMoney(Number(order.total))}`,tip>0&&`Incluye propina ${formatMoney(tip)}`,pending>0&&`Queda ${formatMoney(pending)} pendiente`].filter((fact):fact is string=>Boolean(fact));
  const tones=paymentMethodTones(methods);
  // One document per order: it is requested by the payment that settles the account.
  const[fiscal,setFiscal]=useState<FiscalDraft>(()=>({...blankFiscalDraft(fiscalOptions),customerName:order.channel==="salon"?"":order.customerName??""}));
  const[fiscalErrors,setFiscalErrors]=useState<FiscalErrors>({});
  const settles=Number(amount)+0.000001>=Number(order.remainingAmount);
  const offerFiscal=Boolean(fiscalOptions?.enabled)&&settles;
  function submit(draft:PaymentDraft){
    if(shiftOptions.length>1&&!selectedShiftId)return;
    if(offerFiscal){
      const errors=validateFiscalDraft(fiscalOptions,fiscal,Number(order.total));
      setFiscalErrors(errors);
      if(Object.keys(errors).length)return;
    }
    save({...draft,fiscal:offerFiscal?fiscalRequest(fiscalOptions,fiscal):undefined});
  }
  return <ModalForm onSubmit={handleSubmit(submit)} noValidate inert={busy}>
      <ModalBody className="pos-payment-body pos-checkout">
        <aside className="pos-checkout-summary" aria-label="Resumen del cobro">
          <div className="pos-checkout-due">
            <span>{adjusted?"Total a cobrar":"Saldo a cobrar"}</span>
            <strong>{formatMoney(adjusted?collected:remaining)}</strong>
            {facts.map(fact=><small key={fact}>{fact}</small>)}
          </div>
          {selected?.affectsCash&&<div className={"pos-checkout-result"+result.tone} role="status" aria-live="polite"><span>{result.label}</span><strong>{result.value}</strong></div>}
          {(order.waiterName||shiftOptions.length<=1)&&<div className="pos-checkout-meta">
            {order.waiterName&&<p><Icon name="user" size={16}/><span>Atendió: <b>{order.waiterName}</b></span></p>}
            {shiftOptions.length<=1&&<p><Icon name="register" size={16}/><span>Caja: <b>{shiftName}</b></span></p>}
          </div>}
        </aside>
        <div className="pos-checkout-main">
          {shiftOptions.length>1&&<FormField label="Caja para este cobro"><Select value={selectedShiftId} disabled={busy} onChange={event=>selectShift?.(event.target.value)} required><option value="">Selecciona una caja abierta</option>{shiftOptions.map(option=><option key={option.id} value={option.id}>{option.cashRegisterName} · {option.openedByName}</option>)}</Select></FormField>}
          <FormSection icon="payment" tone="brand" title="Medio de pago" bare>
          <fieldset className="pos-methods">
            <legend className="sr-only">Medio de pago</legend>
            {methods.map(option=><label className={"pos-method"+(method===option.code?" active":"")} key={option.code}>
              <input type="radio" value={option.code} {...register("method")}/>
              <IconTile name={paymentMethodIcon(option.affectsCash)} tone={tones[option.code]}/>
              <span className="pos-method-name">{option.name}</span>
              <i className="pos-method-check" aria-hidden="true"><Icon name="check" size={14}/></i>
            </label>)}
            {errors.method?.message&&<small className="field-error">{errors.method.message}</small>}
          </fieldset>
          </FormSection>
          {selected?.affectsCash&&<FormSection icon="cash" title="Efectivo recibido" bare>
            <div className="pos-tender-entry">
              <FormField label="Monto entregado por el cliente" error={errors.receivedAmount?.message}>
                <MoneyInput symbol={currency.symbol} position={currency.position} {...register("receivedAmount")} aria-invalid={Boolean(errors.receivedAmount)} placeholder={(0).toFixed(decimals)}/>
              </FormField>
              {suggestions.length>0&&<div className="pos-tender" role="group" aria-label="Montos rápidos de efectivo">
                {suggestions.map((value,index)=><button key={value} type="button" className={"pos-tender-option"+(tendered===value?" active":"")} aria-pressed={tendered===value} onClick={()=>setTender(value.toFixed(decimals))} disabled={busy}>{index===0?"Exacto":formatMoney(value)}</button>)}
              </div>}
            </div>
          </FormSection>}
          {selected&&!selected.affectsCash&&<FormField label="Referencia" optional error={errors.reference?.message}>
            <Input icon="invoice" maxLength={120} {...register("reference")} placeholder="Ej. voucher o número de operación"/>
          </FormField>}
          <div className="pos-checkout-options">
            {tipAllowed&&<FormDisclosure title="Propina" summary={tip>0?formatMoney(tip):"Opcional"} open={showTip} onToggle={()=>setTipOpen(!showTip)}>
              <div className="pos-payment-fields">
                <label className="pos-tip-toggle"><input type="checkbox" {...register("tipEnabled")}/><span>El cliente acepta dejar propina</span></label>
                {enabled&&<>
                  {tipPercentage>0&&tipPolicy?.allowFixed&&<FormField label="Tipo de propina" error={errors.tipKind?.message}><Select {...register("tipKind")}><option value="percentage">Porcentaje del local</option><option value="fixed">Monto libre</option></Select></FormField>}
                  {kind==="fixed"&&<FormField label="Monto de la propina" error={errors.tipValue?.message}><MoneyInput symbol={currency.symbol} position={currency.position} {...register("tipValue")} aria-invalid={Boolean(errors.tipValue)} placeholder={(0).toFixed(decimals)}/></FormField>}
                  <p className="pos-tip-help">{kind==="percentage"?`${formatRegionalNumber(tipPercentage,tipPolicy?.country,{maximumFractionDigits:2})} % sobre el consumo de ${formatMoney(tipBase)}. `:""}Se registra una sola vez por cuenta.</p>
                </>}
              </div>
            </FormDisclosure>}
            {registeredTip&&<p className="pos-tip-help">Propina ya registrada: {formatMoney(Number(registeredTip.amount))}. No se vuelve a cobrar.</p>}
            <FormDisclosure title="Cobrar solo una parte" summary={pending>0?`Queda ${formatMoney(pending)}`:"Opcional"} open={showPartial} onToggle={()=>setPartial(!showPartial)}>
              <FormField label="Monto de este cobro" help="El saldo restante queda pendiente para otro pago." error={errors.amount?.message}>
                <MoneyInput symbol={currency.symbol} position={currency.position} {...register("amount")} aria-invalid={Boolean(errors.amount)}/>
              </FormField>
            </FormDisclosure>
          </div>
          {offerFiscal&&fiscalOptions&&<CheckoutFiscalSection options={fiscalOptions} value={fiscal} onChange={next=>{setFiscal(next);setFiscalErrors({})}} errors={fiscalErrors} disabled={busy}/>}
        </div>
      </ModalBody>
      <ModalFooter><Button type="button" action="cancel" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy||(shiftOptions.length>1&&!selectedShiftId)} loading={busy}>{busy?"Registrando…":collected>0?`Cobrar ${formatMoney(collected)}`:"Registrar cobro"}</Button></ModalFooter>
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
