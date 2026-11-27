"use client";
import {Dialog} from "@/design-system/dialog";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,FormField,FormSection,Input,ModalBody,ModalFooter,ModalForm,ModalHeader,Select,Skeleton} from "@/design-system";
import {useSession} from "@/providers";
import {PaymentMethodPicker,collectionShift,getPOSCollectionAccess,listSalesPaymentMethods} from "../../pos";
import type {Reservation} from "../domain/types";
import type {DepositChargeDraft} from "../infrastructure/reservations-api";

/**
 * «Cobrar garantía» (maqueta aprobada 2026-11-26): el importe de la reserva entra al turno de
 * caja con su medio de pago, con las mismas tarjetas de medio que Cobros. Se descuenta de la
 * cuenta cuando el cliente llega.
 */
export function ReservationDepositDialog({reservation,busy,formatMoney,formatDate,close,save}:{reservation:Reservation;busy:boolean;formatMoney:(value:number)=>string;formatDate:(value:string)=>string;close:()=>void;save:(draft:DepositChargeDraft)=>void}){
  const{organization,location,user}=useSession();
  const access=useQuery({queryKey:["pos-cash-access",organization?.id,location?.id,user?.id],queryFn:getPOSCollectionAccess,meta:{live:["cash"]}});
  const methods=useQuery({queryKey:["payment-methods","sales"],queryFn:listSalesPaymentMethods,staleTime:60000});
  const[method,setMethod]=useState("");
  const[reference,setReference]=useState("");
  const[shiftId,setShiftId]=useState("");
  const[error,setError]=useState("");
  const options=methods.data?.items??[];
  const selected=options.find(option=>option.code===(method||options[0]?.code));
  const canChoose=Boolean(access.data?.canChooseShift);
  const shifts=canChoose?(access.data?.shifts??[]):[];
  const shift=collectionShift(access.data,shiftId);
  const details=[formatDate(reservation.startsAt),`${reservation.guests} ${reservation.guests===1?"persona":"personas"}`,reservation.tableName].filter(Boolean).join(" · ");

  function submit(event:React.FormEvent){
    event.preventDefault();
    if(!selected)return;
    if(shifts.length>1&&!shiftId){setError("Selecciona la caja que recibe la garantía.");return;}
    if(reference.length>120){setError("La referencia no puede superar 120 caracteres.");return;}
    save({method:selected.code,reference:selected.affectsCash?"":reference,...(canChoose&&shift?{shiftId:shift.id}:{})});
  }

  let body:React.ReactNode;
  if(access.isError)body=<ErrorState title="No pudimos validar tu caja" text={access.error.message} onRetry={()=>void access.refetch()} retrying={access.isFetching}/>;
  else if(methods.isError)body=<ErrorState title="No pudimos cargar los medios de pago" text={methods.error.message} onRetry={()=>void methods.refetch()} retrying={methods.isFetching}/>;
  else if(access.isLoading||methods.isLoading)body=<ModalBody><div className="reservation-deposit-loading" aria-label="Cargando medios de pago" aria-busy="true"><Skeleton height="var(--size-64)"/><Skeleton height="var(--size-72)"/></div></ModalBody>;
  else if(!shift&&!shifts.length)body=<EmptyState icon="register" title="Necesitas un turno de caja" text="Abre un turno o únete a una caja para cobrar la garantía."/>;
  else if(!options.length)body=<EmptyState icon="wallet" title="Sin medios de pago para ventas" text="Activa al menos un medio para ventas en Configuración › Medios de pago."/>;

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal reservation-deposit-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="reservation-deposit-title" aria-busy={busy}>
      <ModalHeader title="Cobrar garantía" titleId="reservation-deposit-title" onClose={close} busy={busy} closeLabel="Cerrar"/>
      {body??<ModalForm onSubmit={submit} noValidate inert={busy}>
        <ModalBody><div className="reservation-deposit-body">
          <div className="reservation-deposit-summary">
            <div><b>{reservation.customerName}</b><small>{details}</small></div>
            <strong>{formatMoney(Number(reservation.depositAmount))}</strong>
          </div>
          {shifts.length>1&&<FormField label="Caja para este cobro"><Select value={shiftId} disabled={busy} onChange={event=>{setShiftId(event.target.value);setError("")}} required><option value="">Selecciona una caja abierta</option>{shifts.map(option=><option key={option.id} value={option.id}>{option.cashRegisterName} · {option.openedByName}</option>)}</Select></FormField>}
          <FormSection icon="payment" tone="brand" title="Medio de pago" bare>
            <PaymentMethodPicker methods={options} value={selected?.code??""} name="reservation-deposit-method" disabled={busy} onChange={code=>{setMethod(code);setError("")}}/>
          </FormSection>
          {selected&&!selected.affectsCash&&<FormField label="Número de operación" error={error&&reference.length>120?error:undefined}>
            <Input icon="invoice" maxLength={120} value={reference} onChange={event=>{setReference(event.target.value);setError("")}} placeholder="Ej. 356723774"/>
          </FormField>}
          {error&&!(reference.length>120)&&<small className="field-error" role="alert">{error}</small>}
          {shift&&<p className="reservation-deposit-note">Caja: <b>{shift.cashRegisterName}</b> · se descuenta de la cuenta cuando lleguen.</p>}
        </div></ModalBody>
        <ModalFooter>
          <Button type="button" action="cancel" onClick={close} disabled={busy}>Cancelar</Button>
          <Button type="submit" icon="wallet" disabled={busy||!selected} loading={busy}>{busy?"Cobrando…":"Cobrar garantía"}</Button>
        </ModalFooter>
      </ModalForm>}
    </Dialog>
  </div>;
}
