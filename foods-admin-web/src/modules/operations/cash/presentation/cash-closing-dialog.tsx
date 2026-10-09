"use client";
import {Button,Icon,RemoteModalSkeleton} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import {useCashReport} from "../application/use-cash-report";
import type {CashShift,CloseCashShiftDraft} from "../domain/types";
import {CloseCashShiftDialog} from "./cash-dialogs";
import "./cash-report.css";

export function CashClosingDialog({shift,busy,currency,formatMoney,close,save}:{shift:CashShift;busy:boolean;currency:string;formatMoney:(value:number)=>string;close:()=>void;save:(draft:CloseCashShiftDraft)=>Promise<void>}){
 const blind=shift.blindClose&&!shift.expectedVisible;
 const report=useCashReport(shift.id,!blind);
 if(!blind&&report.isLoading)return <RemoteModalSkeleton className="cash-close-modal" label="Cargando resumen para cerrar caja" rows={6} close={close}/>;
 if(!blind&&(report.isError||report.data?.shift.status==="closed"))return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={close} className="crud-modal cash-close-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="cash-close-error-title"><header><div><h2 id="cash-close-error-title">{report.isError?"No pudimos cargar el resumen":"Este turno ya está cerrado"}</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><div className="cash-report-error" role="alert"><p>{report.isError?report.error.message:"Consulta su informe desde Turnos. No vuelvas a registrar el cierre."}</p>{report.isError&&<Button icon="refresh" onClick={()=>void report.refetch()}>Reintentar</Button>}</div></Dialog></div>;
 if(!blind&&!report.data)return null;
 return <CloseCashShiftDialog shift={report.data?.shift??shift} report={blind?undefined:report.data} busy={busy} currency={currency} formatMoney={formatMoney} close={close} save={save}/>;
}
