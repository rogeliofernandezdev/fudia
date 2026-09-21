"use client";
import "./cash.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {CashMovementType,CashRegister,CashShift} from "../domain/types";
import {closeCashShift,createCashMovement,createCashRegister,getCashShift,listCashRegisters,listCashShifts,openCashShift} from "../infrastructure/cash-api";
import {CashMovementDialog,CashRegisterDialog,CashShiftDetailDialog,CloseCashShiftDialog,OpenCashShiftDialog} from "./cash-dialogs";

type CashTab="registers"|"history";

export function CashPage(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{can,location}=useSession();
  const settings=useSettings();
  const canManage=can("cash.manage");
  const[tab,setTab]=useState<CashTab>("registers");
  const[q,setQ]=useState("");
  const[status,setStatus]=useState("");
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[registerDialog,setRegisterDialog]=useState(false);
  const[shiftTarget,setShiftTarget]=useState<CashRegister|null>(null);
  const[movementTarget,setMovementTarget]=useState<{shift:CashShift;type:CashMovementType}|null>(null);
  const[closeTarget,setCloseTarget]=useState<CashShift|null>(null);
  const[detailId,setDetailId]=useState<string|null>(null);

  const registers=useQuery({
    queryKey:["cash-registers"],
    queryFn:()=>listCashRegisters(),
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
    void qc.invalidateQueries({queryKey:["cash-registers"]});
    void qc.invalidateQueries({queryKey:["cash-shifts"]});
    void qc.invalidateQueries({queryKey:["cash-shift"]});
  }

  const createRegister=useMutation({
    mutationFn:createCashRegister,
    onSuccess:item=>{
      setRegisterDialog(false);
      refreshCash();
      notify({tone:"success",title:"Caja registrada",message:`${item.name} ya puede recibir turnos.`});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo registrar la caja",message:error.message}),
  });

  const startShift=useMutation({
    mutationFn:({cashRegisterId,draft}:{cashRegisterId:string;draft:Parameters<typeof openCashShift>[1]})=>openCashShift(cashRegisterId,draft),
    onSuccess:shift=>{
      setShiftTarget(null);
      refreshCash();
      notify({tone:"success",title:"Turno iniciado",message:`${shift.cashRegisterName} quedó operativa con el turno ${shift.code}.`});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo iniciar el turno",message:error.message}),
  });

  const movement=useMutation({
    mutationFn:({shiftId,draft}:{shiftId:string;draft:Parameters<typeof createCashMovement>[1]})=>createCashMovement(shiftId,draft),
    onSuccess:(_,variables)=>{
      setMovementTarget(null);
      refreshCash();
      notify({
        tone:"success",
        title:variables.draft.movementType==="income"?"Ingreso registrado":"Egreso registrado",
        message:"El saldo esperado del turno quedó actualizado.",
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo registrar el movimiento",message:error.message}),
  });

  const closeShift=useMutation({
    mutationFn:({shiftId,draft}:{shiftId:string;draft:Parameters<typeof closeCashShift>[1]})=>closeCashShift(shiftId,draft),
    onSuccess:shift=>{
      setCloseTarget(null);
      refreshCash();
      notify({
        tone:"success",
        title:"Turno cerrado",
        message:Number(shift.varianceAmount??0)===0?"El arqueo quedó cuadrado.":"El cierre quedó guardado con su diferencia de caja.",
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo cerrar el turno",message:error.message}),
  });

  const registerItems=registers.data?.items??[];
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

  const headerAction=tab==="registers"&&canManage&&registerItems.length>0
    ?<Button icon="plus" onClick={()=>setRegisterDialog(true)}>Registrar caja</Button>
    :undefined;

  return <div className="cash-page">
    <PageHeader
      eyebrow="OPERACIÓN"
      title="Caja y turnos"
      description="Administra las cajas del local y abre un turno sobre cada caja cuando vaya a operar."
      action={headerAction}
    />

    <section className="panel standardized-management cash-panel">
      <nav className="cash-tabs" aria-label="Caja y turnos">
        <button type="button" className={tab==="registers"?"active":""} aria-pressed={tab==="registers"} onClick={()=>changeTab("registers")}><Icon name="sales" size={17}/><span>Cajas</span>{registerItems.length>0&&<b>{registerItems.length}</b>}</button>
        <button type="button" className={tab==="history"?"active":""} aria-pressed={tab==="history"} onClick={()=>changeTab("history")}><Icon name="clock" size={17}/><span>Turnos</span></button>
      </nav>

      {tab==="registers"
        ?<CashRegisters
          items={registerItems}
          loading={registers.isLoading}
          error={registers.isError?registers.error.message:null}
          canManage={canManage}
          money={money}
          dateTime={dateTime}
          retry={()=>registers.refetch()}
          create={()=>setRegisterDialog(true)}
          start={setShiftTarget}
          view={shift=>setDetailId(shift.id)}
          movement={(shift,type)=>setMovementTarget({shift,type})}
          close={setCloseTarget}
        />
        :<section className="cash-history">
          <div className="cash-history-toolbar">
            <label className="cash-history-search"><Icon name="search" size={17}/><Input value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar por turno, caja o cajero..."/></label>
            <Select aria-label="Filtrar turnos" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}>
              <option value="">Todos los turnos</option>
              <option value="open">Abiertos</option>
              <option value="closed">Cerrados</option>
            </Select>
            <p><Icon name="lock" size={14}/>Los turnos cerrados conservan su caja, arqueo y movimientos.</p>
          </div>

          {history.isLoading?<CashHistorySkeleton/>
          :history.isError?<CashError title="No pudimos cargar los turnos" message={history.error.message} retry={()=>history.refetch()}/>
          :!historyItems.length?<div className="cash-history-empty"><span><Icon name="clock" size={21}/></span><b>{q||status?"Sin coincidencias":"Aún no hay turnos registrados"}</b><p>{q||status?"Ajusta la búsqueda o el filtro.":"Cuando una caja cierre su primer turno aparecerá aquí para consulta."}</p></div>
          :<div className="table-wrap hover-scroll cash-history-table"><table>
            <thead><tr><th>TURNO</th><th>CAJA</th><th>CAJERO</th><th>APERTURA</th><th>CIERRE</th><th>ESPERADO</th><th>CONTADO</th><th>DIFERENCIA</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
            <tbody>{historyItems.map((item,index)=>{
              const variance=Number(item.varianceAmount??0);
              return <tr className={index%2?"alternate":""} key={item.id}>
                <td><span className="row-icon"><Icon name="clock" size={17}/></span><b>{item.code}</b><small>{item.movementCount} {item.movementCount===1?"movimiento":"movimientos"}</small></td>
                <td><b>{item.cashRegisterName}</b></td>
                <td>{item.openedByName}</td>
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

    {registerDialog&&<CashRegisterDialog busy={createRegister.isPending} close={()=>setRegisterDialog(false)} save={draft=>createRegister.mutate(draft)}/>}
    {shiftTarget&&<OpenCashShiftDialog cashRegister={shiftTarget} busy={startShift.isPending} close={()=>setShiftTarget(null)} save={draft=>startShift.mutate({cashRegisterId:shiftTarget.id,draft})}/>}
    {movementTarget&&<CashMovementDialog type={movementTarget.type} busy={movement.isPending} close={()=>setMovementTarget(null)} save={draft=>movement.mutate({shiftId:movementTarget.shift.id,draft})}/>}
    {closeTarget&&<CloseCashShiftDialog shift={closeTarget} busy={closeShift.isPending} formatMoney={money} close={()=>setCloseTarget(null)} save={draft=>closeShift.mutate({shiftId:closeTarget.id,draft})}/>}
    {detailId&&(detail.isLoading
      ?<RemoteModalSkeleton className="cash-detail-modal" label="Cargando turno de caja" rows={6} close={()=>setDetailId(null)}/>
      :detail.isError
        ?<CashDetailError message={detail.error.message} close={()=>setDetailId(null)}/>
        :detail.data&&<CashShiftDetailDialog shift={detail.data} formatMoney={money} formatDateTime={dateTime} close={()=>setDetailId(null)}/>)}
  </div>;
}

function CashRegisters({items,loading,error,canManage,money,dateTime,retry,create,start,view,movement,close}:{items:CashRegister[];loading:boolean;error:string|null;canManage:boolean;money:(value:number)=>string;dateTime:(value:string)=>string;retry:()=>void;create:()=>void;start:(item:CashRegister)=>void;view:(shift:CashShift)=>void;movement:(shift:CashShift,type:CashMovementType)=>void;close:(shift:CashShift)=>void}){
  if(loading)return <CashRegistersSkeleton/>;
  if(error)return <CashError title="No pudimos cargar las cajas" message={error} retry={retry}/>;
  if(!items.length)return <div className="cash-no-registers">
    <span><Icon name="sales" size={24}/></span>
    <b>Aún no hay cajas registradas</b>
    <p>Primero registra una caja del local. Después podrás iniciar turnos y realizar el arqueo sobre esa caja.</p>
    {canManage&&<Button icon="plus" onClick={create}>Registrar caja</Button>}
  </div>;

  return <div className="cash-registers-grid">{items.map(item=>{
    const shift=item.openShift;
    return <article className="cash-register-card" key={item.id}>
      <header>
        <div className="cash-register-name"><span><Icon name="sales" size={18}/></span><div><small>{item.code}</small><h2>{item.name}</h2></div></div>
        <Status tone={shift?"green":"gray"}>{shift?"Turno abierto":"Sin turno"}</Status>
      </header>

      {shift?<>
        <section className="cash-register-shift">
          <div><small>TURNO</small><b>{shift.code}</b></div>
          <div><small>CAJERO</small><b>{shift.openedByName}</b></div>
          <div><small>APERTURA</small><b>{dateTime(shift.openedAt)}</b></div>
          <div className="expected"><small>SALDO ESPERADO</small><strong>{money(Number(shift.expectedAmount))}</strong></div>
        </section>
        <footer>
          <RowActionButton action="view" label="Ver turno" onClick={()=>view(shift)}/>
          {canManage&&<><Button kind="secondary" icon="plus" onClick={()=>movement(shift,"income")}>Ingreso</Button><Button kind="secondary" icon="minus" onClick={()=>movement(shift,"expense")}>Egreso</Button><Button kind="danger" icon="lock" onClick={()=>close(shift)}>Cerrar turno</Button></>}
        </footer>
      </>:<div className="cash-register-empty">
        <div><small>ESTADO</small><b>Disponible para iniciar un turno</b><p>El turno registrará fondo inicial, movimientos y arqueo de {item.name}.</p></div>
        {canManage&&item.active&&<Button icon="clock" onClick={()=>start(item)}>Iniciar turno</Button>}
      </div>}
    </article>;
  })}</div>;
}

function CashRegistersSkeleton(){return <div className="cash-registers-grid cash-registers-loading">{Array.from({length:4},(_,index)=><i key={index}/>)}</div>}

function CashHistorySkeleton(){return <div className="cash-history-loading">{Array.from({length:6},(_,index)=><i key={index}/>)}</div>}

function CashError({title,message,retry}:{title:string;message:string;retry:()=>void}){return <div className="cash-state-error"><span><Icon name="alert" size={22}/></span><b>{title}</b><p>{message}</p><Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button></div>}

function CashDetailError({message,close}:{message:string;close:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-detail-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>TURNO DE CAJA</small><h2>No pudimos cargar el detalle</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><div className="cash-state-error"><span><Icon name="alert" size={22}/></span><b>Detalle no disponible</b><p>{message}</p></div></section></div>}
