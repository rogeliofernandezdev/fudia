"use client";
import "./cash.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {CashMovementType,CashShift} from "../domain/types";
import {closeCashShift,createCashMovement,getCashShift,getCurrentCashShift,listCashShifts,openCashShift} from "../infrastructure/cash-api";
import {CashMovementDialog,CashShiftDetailDialog,CloseCashShiftDialog,OpenCashShiftDialog} from "./cash-dialogs";

type CashTab="current"|"history";

export function CashPage(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const settings=useSettings();
  const canManage=can("cash.manage");
  const[tab,setTab]=useState<CashTab>("current");
  const[q,setQ]=useState("");
  const[status,setStatus]=useState("");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[openDialog,setOpenDialog]=useState(false);
  const[movementType,setMovementType]=useState<CashMovementType|null>(null);
  const[closeDialog,setCloseDialog]=useState(false);
  const[detailId,setDetailId]=useState<string|null>(null);

  const current=useQuery({
    queryKey:["cash-shift","current"],
    queryFn:getCurrentCashShift,
    refetchInterval:30000,
  });
  const history=useQuery({
    queryKey:["cash-shifts",q,status,page,size],
    queryFn:()=>listCashShifts({q,status,page,pageSize:size}),
    enabled:tab==="history",
  });
  const detail=useQuery({
    queryKey:["cash-shift",detailId],
    queryFn:()=>getCashShift(detailId!),
    enabled:Boolean(detailId),
  });

  function refreshCash(){
    void qc.invalidateQueries({queryKey:["cash-shift"]});
    void qc.invalidateQueries({queryKey:["cash-shifts"]});
  }

  const openShift=useMutation({
    mutationFn:openCashShift,
    onSuccess:shift=>{
      setOpenDialog(false);
      refreshCash();
      notify({tone:"success",title:"Turno abierto",message:`${shift.code} quedó listo para operar.`});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo abrir el turno",message:error.message}),
  });

  const movement=useMutation({
    mutationFn:({shiftId,draft}:{shiftId:string;draft:Parameters<typeof createCashMovement>[1]})=>createCashMovement(shiftId,draft),
    onSuccess:(_,variables)=>{
      setMovementType(null);
      refreshCash();
      notify({
        tone:"success",
        title:variables.draft.movementType==="income"?"Ingreso registrado":"Egreso registrado",
        message:"El saldo esperado de caja quedó actualizado.",
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo registrar el movimiento",message:error.message}),
  });

  const closeShift=useMutation({
    mutationFn:({shiftId,draft}:{shiftId:string;draft:Parameters<typeof closeCashShift>[1]})=>closeCashShift(shiftId,draft),
    onSuccess:shift=>{
      setCloseDialog(false);
      refreshCash();
      notify({
        tone:"success",
        title:"Turno cerrado",
        message:Number(shift.varianceAmount??0)===0?"El arqueo quedó cuadrado.":"El cierre quedó guardado con su diferencia de caja.",
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo cerrar el turno",message:error.message}),
  });

  const shift=current.data?.shift??null;
  const historyItems=history.data?.items??[];

  function money(value:number){
    if(!Number.isFinite(value))return "—";
    const sign=value<0?"-":"";
    const formatted=formatRegionalNumber(Math.abs(value),location?.country,{
      minimumFractionDigits:settings.currencyDecimals,
      maximumFractionDigits:settings.currencyDecimals,
    });
    return settings.currencyPosition==="before"
      ?`${sign}${settings.currencySymbol} ${formatted}`
      :`${sign}${formatted} ${settings.currencySymbol}`;
  }

  function dateTime(value:string){
    return formatRegionalDateTime(value,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"});
  }

  function changeTab(next:CashTab){
    setTab(next);
    setQ("");
    setStatus("");
    setPage(1);
    if(next==="history")void qc.invalidateQueries({queryKey:["cash-shifts"]});
  }

  const headerAction=undefined;

  return <div className="cash-page">
    <PageHeader
      eyebrow="OPERACIÓN"
      title="Caja y turnos"
      description="Abre tu turno, registra movimientos de efectivo y cierra con arqueo."
      action={headerAction}
    />

    <section className="panel standardized-management cash-panel">
      <nav className="cash-tabs" aria-label="Caja y turnos">
        <button type="button" className={tab==="current"?"active":""} aria-pressed={tab==="current"} onClick={()=>changeTab("current")}><Icon name="sales" size={17}/><span>Caja</span>{shift&&<b>Abierta</b>}</button>
        <button type="button" className={tab==="history"?"active":""} aria-pressed={tab==="history"} onClick={()=>changeTab("history")}><Icon name="clock" size={17}/><span>Turnos</span></button>
      </nav>

      {tab==="current"
        ?<CurrentCash
          shift={shift}
          loading={current.isLoading}
          error={current.isError?current.error.message:null}
          canManage={canManage}
          money={money}
          dateTime={dateTime}
          retry={()=>current.refetch()}
          open={()=>setOpenDialog(true)}
          addMovement={setMovementType}
          close={()=>setCloseDialog(true)}
        />
        :<section className="cash-history">
          <div className="cash-history-toolbar">
            <label className="cash-history-search"><Icon name="search" size={17}/><Input value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar por turno o cajero..."/></label>
            <Select aria-label="Filtrar turnos" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
              <option value="">Todos los turnos</option>
              <option value="open">Abiertos</option>
              <option value="closed">Cerrados</option>
            </Select>
            <p><Icon name="lock" size={14}/>Los turnos cerrados conservan su arqueo y movimientos.</p>
          </div>

          {history.isLoading?<CashHistorySkeleton/>
          :history.isError?<CashError title="No pudimos cargar los turnos" message={history.error.message} retry={()=>history.refetch()}/>
          :!historyItems.length?<div className="cash-history-empty"><span><Icon name="clock" size={21}/></span><b>{q||status?"Sin coincidencias":"Aún no hay turnos registrados"}</b><p>{q||status?"Ajusta la búsqueda o el filtro.":"Los cierres de caja aparecerán aquí para consulta y control."}</p></div>
          :<div className="table-wrap hover-scroll cash-history-table"><table>
            <thead><tr><th>TURNO</th><th>CAJERO</th><th>APERTURA</th><th>CIERRE</th><th>ESPERADO</th><th>CONTADO</th><th>DIFERENCIA</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
            <tbody>{historyItems.map((item,index)=>{
              const variance=Number(item.varianceAmount??0);
              return <tr className={index%2?"alternate":""} key={item.id}>
                <td><span className="row-icon"><Icon name="sales" size={17}/></span><b>{item.code}</b><small>{item.movementCount} {item.movementCount===1?"movimiento":"movimientos"}</small></td>
                <td><b>{item.openedByName}</b></td>
                <td>{dateTime(item.openedAt)}</td>
                <td>{item.closedAt?dateTime(item.closedAt):"—"}</td>
                <td><b>{money(Number(item.closingExpectedAmount??item.expectedAmount))}</b></td>
                <td>{item.closingCountedAmount!==null?money(Number(item.closingCountedAmount)):"—"}</td>
                <td><b className={item.status==="open"?"cash-neutral":variance===0?"cash-balanced":variance>0?"cash-positive":"cash-negative"}>{item.status==="open"?"—":`${variance>0?"+":""}${money(variance)}`}</b></td>
                <td><Status tone={item.status==="open"?"green":"gray"}>{item.status==="open"?"Abierto":"Cerrado"}</Status></td>
                <td><div className="table-actions"><RowActionButton action="view" onClick={()=>setDetailId(item.id)}/></div></td>
              </tr>;
            })}</tbody>
          </table></div>}

          {!history.isLoading&&!history.isError&&<Pagination page={page} size={size} total={history.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
        </section>
      }
    </section>

    {openDialog&&<OpenCashShiftDialog busy={openShift.isPending} close={()=>setOpenDialog(false)} save={draft=>openShift.mutate(draft)}/>}
    {movementType&&shift&&<CashMovementDialog type={movementType} busy={movement.isPending} close={()=>setMovementType(null)} save={draft=>movement.mutate({shiftId:shift.id,draft})}/>}
    {closeDialog&&shift&&<CloseCashShiftDialog shift={shift} busy={closeShift.isPending} formatMoney={money} close={()=>setCloseDialog(false)} save={draft=>closeShift.mutate({shiftId:shift.id,draft})}/>}
    {detailId&&(detail.isLoading
      ?<RemoteModalSkeleton className="cash-detail-modal" label="Cargando turno de caja" rows={6} close={()=>setDetailId(null)}/>
      :detail.isError
        ?<CashDetailError message={detail.error.message} close={()=>setDetailId(null)}/>
        :detail.data&&<CashShiftDetailDialog shift={detail.data} formatMoney={money} formatDateTime={dateTime} close={()=>setDetailId(null)}/>)}
  </div>;
}

function CurrentCash({shift,loading,error,canManage,money,dateTime,retry,open,addMovement,close}:{shift:CashShift|null;loading:boolean;error:string|null;canManage:boolean;money:(value:number)=>string;dateTime:(value:string)=>string;retry:()=>void;open:()=>void;addMovement:(type:CashMovementType)=>void;close:()=>void}){
  if(loading)return <div className="cash-current-loading"><i/><div>{Array.from({length:4},(_,index)=><span key={index}/>)}</div><i/></div>;
  if(error)return <CashError title="No pudimos cargar tu caja" message={error} retry={retry}/>;
  if(!shift)return <div className="cash-no-shift">
    <span><Icon name="sales" size={24}/></span>
    <b>No tienes un turno de caja abierto</b>
    <p>Abre un turno con el fondo inicial antes de registrar ingresos o egresos.</p>
    {canManage&&<Button icon="plus" onClick={open}>Abrir turno</Button>}
  </div>;

  const movements=shift.movements??[];
  return <div className="cash-current">
    <header className="cash-current-head">
      <div className="cash-shift-identity"><span><Icon name="sales" size={19}/></span><div><small>TURNO ACTUAL</small><h2>{shift.code}</h2><p>Abierto por {shift.openedByName} · {dateTime(shift.openedAt)}</p></div></div>
      <div className="cash-current-head-actions">
        <Status tone="green">Abierto</Status>
        {canManage&&<><Button kind="secondary" icon="plus" onClick={()=>addMovement("income")}>Ingreso</Button><Button kind="secondary" icon="minus" onClick={()=>addMovement("expense")}>Egreso</Button><Button kind="danger" icon="lock" onClick={close}>Cerrar turno</Button></>}
      </div>
    </header>

    <section className="cash-balance-strip" aria-label="Resumen del turno">
      <div><small>FONDO INICIAL</small><b>{money(Number(shift.openingAmount))}</b></div>
      <div className="income"><small>INGRESOS</small><b>+{money(Number(shift.incomeAmount))}</b></div>
      <div className="expense"><small>EGRESOS</small><b>-{money(Number(shift.expenseAmount))}</b></div>
      <div className="expected"><small>SALDO ESPERADO</small><strong>{money(Number(shift.expectedAmount))}</strong></div>
    </section>

    {shift.openingNote&&<div className="cash-opening-note"><Icon name="edit" size={13}/><span>{shift.openingNote}</span></div>}

    <section className="cash-movements">
      <header><div><small>MOVIMIENTOS DEL TURNO</small><h3>Ingresos y egresos manuales</h3></div><b>{movements.length}</b></header>
      {movements.length?<div className="table-wrap hover-scroll"><table><thead><tr><th>HORA</th><th>TIPO</th><th>MOTIVO</th><th>USUARIO</th><th>MONTO</th></tr></thead><tbody>{movements.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}>
        <td>{dateTime(item.createdAt)}</td>
        <td><span className={"cash-movement-kind "+item.movementType}><Icon name={item.movementType==="income"?"plus":"minus"} size={12}/>{item.movementType==="income"?"Ingreso":"Egreso"}</span></td>
        <td><b>{item.reason}</b>{item.note&&<small>{item.note}</small>}</td>
        <td>{item.createdByName}</td>
        <td><b className={item.movementType}>{item.movementType==="income"?"+":"-"}{money(Number(item.amount))}</b></td>
      </tr>)}</tbody></table></div>:<div className="cash-movements-empty"><span><Icon name="receipt" size={18}/></span><b>Sin movimientos manuales</b><p>El fondo inicial ya forma parte del saldo esperado. Registra aquí solo ingresos o egresos adicionales.</p></div>}
    </section>
  </div>;
}

function CashHistorySkeleton(){return <div className="cash-history-loading">{Array.from({length:6},(_,index)=><i key={index}/>)}</div>}

function CashError({title,message,retry}:{title:string;message:string;retry:()=>void}){return <div className="cash-state-error"><span><Icon name="alert" size={22}/></span><b>{title}</b><p>{message}</p><Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button></div>}

function CashDetailError({message,close}:{message:string;close:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-detail-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>TURNO DE CAJA</small><h2>No pudimos cargar el detalle</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><div className="cash-state-error"><span><Icon name="alert" size={22}/></span><b>Detalle no disponible</b><p>{message}</p></div></section></div>}
