"use client";
import {EmptyState,ErrorState,RemoteModalSkeleton,ModalHeader} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useCashReport} from "../application/use-cash-report";
import type {CashShift,CloseCashShiftDraft} from "../domain/types";
import {CloseCashShiftDialog} from "./cash-dialogs";
import "./cash-report.css";

export function CashClosingDialog({shift,busy,currency,formatMoney,close,save}:{shift:CashShift;busy:boolean;currency:string;formatMoney:(value:number)=>string;close:()=>void;save:(draft:CloseCashShiftDraft)=>Promise<void>}){
 const blind=shift.blindClose&&!shift.expectedVisible;
 const report=useCashReport(shift.id,!blind);
 if(!blind&&report.isLoading)return <RemoteModalSkeleton className="cash-close-modal" label="Cargando resumen para cerrar caja" rows={6} close={close}/>;
 if(!blind&&(report.isError||report.data?.shift.status==="closed"))return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-close-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-close-error-title">

  <ModalHeader title="Cerrar turno" titleId="cash-close-error-title" icon={report.isError?"alert":"lock"} subtitle={shift.cashRegisterName.toUpperCase()} onClose={close} closeLabel="Cerrar"/>
  {report.isError
   ?<ErrorState title="No pudimos cargar el resumen" text={report.error.message} retrying={report.isFetching} onRetry={()=>void report.refetch()}/>
   :<EmptyState icon="lock" title="Este turno ya está cerrado" text="Consulta su informe desde Turnos. No vuelvas a registrar el cierre."/>}
 </Dialog></div>;
 if(!blind&&!report.data)return null;
 return <CloseCashShiftDialog shift={report.data?.shift??shift} report={blind?undefined:report.data} busy={busy} currency={currency} formatMoney={formatMoney} close={close} save={save}/>;
}
