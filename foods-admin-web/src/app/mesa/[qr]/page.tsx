"use client";

import {useState,useEffect} from "react";
import {useParams} from "next/navigation";
import {Icon} from "@/design-system/icons";

type TableInfo={
  name:string;
  seats:number;
  zone:string;
  organizationName:string;
  locationName:string;
};

export default function MesaPage(){
  const params=useParams<{qr:string}>();
  const token=params.qr;
  const[info,setInfo]=useState<TableInfo|null>(null);
  const[loading,setLoading]=useState(true);
  const[error,setError]=useState("");

  useEffect(()=>{
    if(!token)return;
    setLoading(true);
    fetch(`/api/public/tables/${token}`)
      .then(r=>{if(!r.ok)throw new Error("No pudimos cargar la mesa.");return r.json()})
      .then(data=>setInfo(data))
      .catch(e=>setError(e.message))
      .finally(()=>setLoading(false));
  },[token]);

  if(loading)return <main className="mesa-public"><div className="mesa-loading"><i/><p>Cargando información de la mesa…</p></div></main>;
  if(error)return <main className="mesa-public"><div className="mesa-error"><Icon name="alert" size={32}/><b>QR no válido</b><p>{error}</p><p className="mesa-hint">Pide ayuda al personal del restaurante.</p></div></main>;
  if(!info)return <main className="mesa-public"><div className="mesa-error"><b>Mesa no encontrada</b></div></main>;

  return <main className="mesa-public">
    <header className="mesa-header">
      <div className="mesa-brand"><Icon name="store" size={20}/><b>{info.organizationName}</b></div>
      <span className="mesa-location"><Icon name="grid" size={14}/>{info.locationName}</span>
    </header>
    <section className="mesa-card">
      <div className="mesa-accent"/>
      <div className="mesa-body">
        <small>MESA ASIGNADA</small>
        <h1>{info.name}</h1>
        <div className="mesa-meta">
          <span><Icon name="users" size={16}/>{info.seats} asientos</span>
          {info.zone&&<span><Icon name="store" size={16}/>{info.zone}</span>}
        </div>
      </div>
    </section>
    <section className="mesa-actions">
      <button className="mesa-action primary">
        <Icon name="menu" size={22}/>
        <div><b>Ver la carta</b><small>Explora el menú digital</small></div>
      </button>
      <button className="mesa-action">
        <Icon name="bell" size={22}/>
        <div><b>Llamar mozo</b><small>Solicita atención</small></div>
      </button>
      <button className="mesa-action">
        <Icon name="receipt" size={22}/>
        <div><b>Mi cuenta</b><small>Revisa tu pedido</small></div>
      </button>
    </section>
    <footer className="mesa-footer"><Icon name="lock" size={12}/> Conexión segura · {info.organizationName}</footer>
  </main>;
}
