"use client";
import "./cash.css";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {formatRegionalCalendarDate,formatRegionalDateTime,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {CashMovementType,CashRegister,CashRegisterDraft,CashShift} from "../domain/types";
import {closeCashShift,createCashMovement,createCashRegister,getCashShift,listCashRegisters,listCashShifts,openCashShift,setCashRegisterActive,updateCashRegister} from "../infrastructure/cash-api";
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
  const[registerEditTarget,setRegisterEditTarget]=useState<CashRegister|null>(null);
  const[registerStatusTarget,setRegisterStatusTarget]=useState<CashRegister|null>(null);
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

  function openNewRegister(){
    setRegisterEditTarget(null);
    setRegisterDialog(true);
  }

  function openEditRegister(item:CashRegister){
    setRegisterEditTarget(item);
    setRegisterDialog(true);
  }

  const saveRegister=useMutation({
    mutationFn:({target,draft}:{target:CashRegister|null;draft:CashRegisterDraft})=>target?updateCashRegister(target.id,draft):createCashRegister(draft),
    onSuccess:(item,variables)=>{
      setRegisterDialog(false);
      setRegisterEditTarget(null);
      refreshCash();
      notify({
        tone:"success",
        title:variables.target?"Caja actualizada":"Caja registrada",
        message:variables.target?item.name+" quedó actualizada.":item.name+" ya puede recibir turnos.",
      });
    },
    onError:error=>notify({tone:"danger",title:"No se pudo guardar la caja",message:error.message}),
  });

  const toggleRegister=useMutation({
    mutationFn:(item:CashRegister)=>setCashRegisterActive(item.id,!item.active),
    onSuccess:(_,item)=>{
      setRegisterStatusTarget(null);
      refreshCash();
      notify({tone:"success",title:item.active?"Caja desactivada":"Caja activada",message:item.name+" quedó "+(item.active?"fuera":"disponible")+" para nuevos turnos."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo cambiar el estado de la caja",message:error.message}),
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

  const closeShiftMutation=useMutation({
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

  function businessDate(value:string){
    return formatRegionalCalendarDate(value,location?.country,{dateStyle:"medium"});
  }

  function changeTab(next:CashTab){
    setTab(next);
    setQ("");
    setStatus("");
    setPage(1);
    if(next==="history")void qc.invalidateQueries({queryKey:["cash-shifts"]});
  }

  const headerAction=tab==="registers"&&canManage&&registerItems.length>0
    ?<Button icon="plus" onClick={openNewRegister}>Registrar caja</Button>
    :undefined;

  return <div className="cash-page">
    <PageHeader
      eyebrow="OPERACIÓN"
      title="Caja y turnos"
      description="Controla las cajas del local, sus turnos y el efectivo esperado en tiempo real."
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
          businessDate={businessDate}
          retry={()=>registers.refetch()}
          create={openNewRegister}
          edit={openEditRegister}
          toggleStatus={setRegisterStatusTarget}
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
            <p><Icon name="lock" size={14}/>Los turnos cerrados conservan su arqueo y trazabilidad.</p>
          </div>

          {history.isLoading?<CashHistorySkeleton/>
          :history.isError?<CashError title="No pudimos cargar los turnos" message={history.error.message} retry={()=>history.refetch()}/>
          :!historyItems.length?<div className="cash-history-empty"><span><Icon name="clock" size={21}/></span><b>{q||status?"Sin coincidencias":"Aún no hay turnos registrados"}</b><p>{q||status?"Ajusta la búsqueda o el filtro.":"Cuando una caja cierre su primer turno aparecerá aquí para consulta."}</p></div>
          :<div className="table-wrap hover-scroll cash-history-table"><table>
            <thead><tr><th>TURNO</th><th>CAJA</th><th>DÍA OPERATIVO</th><th>CAJERO</th><th>APERTURA</th><th>CIERRE</th><th>DIFERENCIA</th><th>ESTADO</th><th>ACCIONES</th></tr></thead>
            <tbody>{historyItems.map((item,index)=>{
              const variance=Number(item.varianceAmount??0);
              return <tr className={index%2?"alternate":""} key={item.id}>
                <td><span className="row-icon"><Icon name="clock" size={17}/></span><b>{item.code}</b><small>{item.movementCount} {item.movementCount===1?"movimiento":"movimientos"}</small></td>
                <td><b>{item.cashRegisterName}</b></td>
                <td>{businessDate(item.businessDate)}</td>
                <td>{item.openedByName}</td>
                <td>{dateTime(item.openedAt)}</td>
                <td>{item.closedAt?dateTime(item.closedAt):"—"}</td>
                <td><b className={item.status==="open"?"cash-neutral":variance===0?"cash-balanced":variance>0?"cash-positive":"cash-negative"}>{item.status==="open"?"—":(variance>0?"+":"")+money(variance)}</b></td>
                <td><Status tone={item.status==="open"?"green":"gray"}>{item.status==="open"?"Abierto":"Cerrado"}</Status></td>
                <td><div className="table-actions"><RowActionButton action="view" onClick={()=>setDetailId(item.id)}/></div></td>
              </tr>;
            })}</tbody>
          </table></div>}

          {!history.isLoading&&!history.isError&&<Pagination page={page} size={size} total={history.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
        </section>
      }
    </section>

    {registerDialog&&<CashRegisterDialog initial={registerEditTarget} busy={saveRegister.isPending} close={()=>{setRegisterDialog(false);setRegisterEditTarget(null)}} save={draft=>saveRegister.mutate({target:registerEditTarget,draft})}/>} 
    {shiftTarget&&<OpenCashShiftDialog cashRegister={shiftTarget} busy={startShift.isPending} close={()=>setShiftTarget(null)} save={draft=>startShift.mutate({cashRegisterId:shiftTarget.id,draft})}/>}
    {movementTarget&&<CashMovementDialog type={movementTarget.type} busy={movement.isPending} close={()=>setMovementTarget(null)} save={draft=>movement.mutate({shiftId:movementTarget.shift.id,draft})}/>}
    {closeTarget&&<CloseCashShiftDialog shift={closeTarget} busy={closeShiftMutation.isPending} formatMoney={money} close={()=>setCloseTarget(null)} save={draft=>closeShiftMutation.mutate({shiftId:closeTarget.id,draft})}/>}
    {detailId&&(detail.isLoading
      ?<RemoteModalSkeleton className="cash-detail-modal" label="Cargando turno de caja" rows={6} close={()=>setDetailId(null)}/>
      :detail.isError
        ?<CashDetailError message={detail.error.message} close={()=>setDetailId(null)}/>
        :detail.data&&<CashShiftDetailDialog shift={detail.data} formatMoney={money} formatDateTime={dateTime} close={()=>setDetailId(null)}/>)}

    <ConfirmDialog
      open={Boolean(registerStatusTarget)}
      title={registerStatusTarget?.active?"Desactivar caja":"Activar caja"}
      description={registerStatusTarget?.active
        ?"“"+(registerStatusTarget?.name??"")+"” dejará de admitir nuevos turnos. Su historial se conservará."
        :"“"+(registerStatusTarget?.name??"")+"” volverá a estar disponible para iniciar turnos."}
      tone={registerStatusTarget?.active?"danger":"success"}
      confirmLabel={registerStatusTarget?.active?"Desactivar":"Activar"}
      pending={toggleRegister.isPending}
      onCancel={()=>setRegisterStatusTarget(null)}
      onConfirm={()=>registerStatusTarget&&toggleRegister.mutate(registerStatusTarget)}
    />
  </div>;
}

function CashRegisters({items,loading,error,canManage,money,dateTime,businessDate,retry,create,edit,toggleStatus,start,view,movement,close}:{items:CashRegister[];loading:boolean;error:string|null;canManage:boolean;money:(value:number)=>string;dateTime:(value:string)=>string;businessDate:(value:string)=>string;retry:()=>void;create:()=>void;edit:(item:CashRegister)=>void;toggleStatus:(item:CashRegister)=>void;start:(item:CashRegister)=>void;view:(shift:CashShift)=>void;movement:(shift:CashShift,type:CashMovementType)=>void;close:(shift:CashShift)=>void}){
  if(loading)return <CashRegistersSkeleton/>;
  if(error)return <CashError title="No pudimos cargar las cajas" message={error} retry={retry}/>;
  if(!items.length)return <div className="cash-no-registers">
    <span><Icon name="sales" size={24}/></span>
    <b>Aún no hay cajas registradas</b>
    <p>Registra la primera caja del local. Luego podrás iniciar turnos, controlar el efectivo y cerrar con arqueo.</p>
    {canManage&&<Button icon="plus" onClick={create}>Registrar caja</Button>}
  </div>;

  const active=items.filter(item=>item.active);
  const open=active.filter(item=>item.openShift);
  const available=active.length-open.length;
  const expected=open.reduce((sum,item)=>sum+Number(item.openShift?.expectedAmount??0),0);

  return <div className="cash-registers-workspace">
    <section className="cash-overview" aria-label="Resumen operativo de cajas">
      <div className="cash-overview-copy">
        <span className="cash-overview-icon"><Icon name="sales" size={20}/></span>
        <div><small>ESTADO OPERATIVO</small><b>{open.length?open.length+" "+(open.length===1?"caja operando":"cajas operando"):"Sin turnos abiertos"}</b><p>{available} {available===1?"caja disponible":"cajas disponibles"} para iniciar turno.</p></div>
      </div>
      <div className="cash-overview-metrics">
        <div><small>CAJAS ACTIVAS</small><strong>{active.length}</strong></div>
        <div><small>TURNOS ABIERTOS</small><strong>{open.length}</strong></div>
        <div className="money"><small>EFECTIVO ESPERADO</small><strong>{money(expected)}</strong></div>
      </div>
    </section>

    <div className="cash-registers-heading"><div><small>CAJAS DEL LOCAL</small><h2>Operación por caja</h2></div><span>Actualización automática</span></div>

    <div className="cash-registers-grid">{items.map(item=>{
      const shift=item.openShift;
      const statusLabel=!item.active?"Inactiva":shift?"Turno abierto":"Disponible";
      const statusTone=!item.active?"gray":shift?"green":"blue";
      const cardClass="cash-register-card"+(!item.active?" inactive":shift?" open":"");
      return <article className={cardClass} key={item.id}>
        <header>
          <div className="cash-register-name"><span><Icon name="sales" size={18}/></span><div><small>{item.code}</small><h3>{item.name}</h3></div></div>
          <div className="cash-register-header-actions">
            <Status tone={statusTone}>{statusLabel}</Status>
            {canManage&&<div className="cash-register-admin">
              <RowActionButton action="edit" label="Editar caja" onClick={()=>edit(item)}/>
              <RowActionButton action={item.active?"deactivate":"activate"} label={item.active&&shift?"Cierra el turno antes de desactivar":undefined} disabled={Boolean(item.active&&shift)} onClick={()=>toggleStatus(item)}/>
            </div>}
          </div>
        </header>

        {shift?<>
          <section className="cash-register-live">
            <div className="cash-register-live-copy">
              <small>TURNO EN CURSO</small>
              <b>{shift.code}</b>
              <p><Icon name="users" size={12}/>{shift.openedByName}<span>·</span><Icon name="clock" size={12}/>{dateTime(shift.openedAt)}</p>
              <em>Día operativo {businessDate(shift.businessDate)}</em>
            </div>
            <div className="cash-register-balance"><small>SALDO ESPERADO</small><strong>{money(Number(shift.expectedAmount))}</strong><span>{shift.movementCount} {shift.movementCount===1?"movimiento":"movimientos"}</span></div>
          </section>
          <section className="cash-register-flow">
            <div><small>FONDO INICIAL</small><b>{money(Number(shift.openingAmount))}</b></div>
            <div className="income"><small>INGRESOS</small><b>+{money(Number(shift.incomeAmount))}</b></div>
            <div className="expense"><small>EGRESOS</small><b>-{money(Number(shift.expenseAmount))}</b></div>
          </section>
          <footer>
            <RowActionButton action="view" label="Ver turno" onClick={()=>view(shift)}/>
            {canManage&&<><Button kind="secondary" icon="plus" onClick={()=>movement(shift,"income")}>Ingreso</Button><Button kind="secondary" icon="minus" onClick={()=>movement(shift,"expense")}>Egreso</Button><Button kind="danger" icon="lock" onClick={()=>close(shift)}>Cerrar turno</Button></>}
          </footer>
        </>:<div className="cash-register-idle">
          <span><Icon name={item.active?"clock":"power"} size={20}/></span>
          <div><small>{item.active?"LISTA PARA OPERAR":"FUERA DE OPERACIÓN"}</small><b>{item.active?"Sin turno activo":"Caja desactivada"}</b><p>{item.active?"Inicia un turno para registrar fondo, movimientos y arqueo.":"Actívala cuando vuelva a utilizarse en el local."}</p></div>
          {canManage&&item.active&&<Button icon="clock" onClick={()=>start(item)}>Iniciar turno</Button>}
        </div>}
      </article>;
    })}</div>
  </div>;
}

function CashRegistersSkeleton(){return <div className="cash-registers-workspace"><div className="cash-overview cash-overview-skeleton"/><div className="cash-registers-grid cash-registers-loading">{Array.from({length:4},(_,index)=><i key={index}/>)}</div></div>}

function CashHistorySkeleton(){return <div className="cash-history-loading">{Array.from({length:6},(_,index)=><i key={index}/>)}</div>}

function CashError({title,message,retry}:{title:string;message:string;retry:()=>void}){return <div className="cash-state-error"><span><Icon name="alert" size={22}/></span><b>{title}</b><p>{message}</p><Button kind="secondary" icon="refresh" onClick={retry}>Reintentar</Button></div>}

function CashDetailError({message,close}:{message:string;close:()=>void}){return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal cash-detail-modal modal-panel-in" role="dialog" aria-modal="true"><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="alert"/></span><div><small>TURNO DE CAJA</small><h2>No pudimos cargar el detalle</h2></div><button type="button" aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header><div className="cash-state-error"><span><Icon name="alert" size={22}/></span><b>Detalle no disponible</b><p>{message}</p></div></section></div>}
