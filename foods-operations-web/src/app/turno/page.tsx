"use client";

import {Button,Input,Label,Select} from "@/components/ui/controls";
import {useEffect,useState} from "react";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {Icon} from "@/components/icon";
import {Logo} from "@/components/logo";
import {CashShift,operationsFetch} from "@/lib/operations-api";

type Register={id:string;name:string;active:boolean;openShift:CashShift|null};

export default function ShiftPage(){
  const router=useRouter();
  const[registers,setRegisters]=useState<Register[]>([]);
  const[current,setCurrent]=useState<CashShift|null>(null);
  const[registerId,setRegisterId]=useState("");
  const[opening,setOpening]=useState("0.00");
  const[loading,setLoading]=useState(true);
  const[busy,setBusy]=useState(false);
  const[error,setError]=useState("");

  useEffect(()=>{
    void Promise.all([
      operationsFetch<{items:Register[]}>("cash-registers"),
      operationsFetch<{shift:CashShift|null}>("cash-shifts/current"),
    ]).then(([r,c])=>{
      const available=r.items.filter(item=>item.active&&!item.openShift);
      setRegisters(available);
      setCurrent(c.shift);
      if(available[0])setRegisterId(available[0].id);
    }).catch(e=>setError(e instanceof Error?e.message:"No se pudieron cargar las cajas.")).finally(()=>setLoading(false));
  },[]);

  async function openShift(){
    if(!registerId||busy)return;
    const amount=Number(opening);
    if(!Number.isFinite(amount)||amount<0){setError("Ingresa un fondo inicial válido.");return;}
    setBusy(true);setError("");
    try{
      await operationsFetch<CashShift>("cash-shifts",{method:"POST",body:JSON.stringify({cashRegisterId:registerId,openingAmount:amount,note:"Apertura desde Operaciones"})});
      router.replace("/inicio");
      router.refresh();
    }catch(e){setError(e instanceof Error?e.message:"No se pudo abrir el turno.");}
    finally{setBusy(false);}
  }

  return <main className="setup-page"><header><Logo/><Link href="/login"><Icon name="logout"/> Salir</Link></header>
    <section className="setup-card"><div className="setup-card-accent" aria-hidden="true"/><div className="setup-progress"><i className="done"><Icon name="check"/></i><span/><i className="active">2</i><span/><i>3</i></div>
      <div className="setup-intro"><div className="setup-icon"><Icon name="store" size={23}/></div><div><span className="kicker">PREPARA TU JORNADA</span><h1>{current?"Turno activo":"Abre tu turno"}</h1><p>{current?`Ya estás operando ${current.cashRegisterName} con el turno ${current.code}.`:"Selecciona una caja disponible y registra el fondo inicial."}</p></div></div>
      {error&&<div className="missing-card" role="alert"><span>{error}</span></div>}
      {loading?<div className="pos-empty"><b>Cargando cajas…</b></div>:current?
        <div className="setup-action"><Button tone="primary" className="wide" onClick={()=>router.replace("/inicio")}>Continuar con el turno <Icon name="fingerprint"/></Button></div>:
        <>
          <div className="setup-selection-grid">
            <Label htmlFor="shift-register">Caja<div className="shift-field"><Icon name="cash"/><Select className="shift-control" id="shift-register" value={registerId} onChange={e=>setRegisterId(e.target.value)}><option value="">Selecciona una caja</option>{registers.map(item=><option key={item.id} value={item.id}>{item.name}</option>)}</Select></div></Label>
            <Label htmlFor="opening-balance">Fondo inicial<div className="shift-field shift-money"><span>S/</span><Input className="shift-control" id="opening-balance" inputMode="decimal" value={opening} onChange={e=>setOpening(e.target.value)}/></div></Label>
          </div>
          {!registers.length&&<div className="missing-card"><span>No hay cajas activas libres para abrir un turno.</span></div>}
          <div className="setup-action"><Button tone="primary" className="wide" disabled={!registerId||busy} onClick={()=>void openShift()}>{busy?"Abriendo turno…":"Abrir turno y comenzar"} <Icon name="fingerprint"/></Button><p className="setup-help"><Icon name="clock" size={16}/> Los cobros quedarán vinculados a este turno.</p></div>
        </>}
    </section>
  </main>;
}
