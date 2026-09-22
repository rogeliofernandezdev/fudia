"use client";
import "../../styles/table-qr.css";
import {useState,useEffect,useRef} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import QRCode from "qrcode";
import {Icon} from "@/design-system/icons";
import {Button,PageHeader,Pagination,RowActionButton,Status} from "@/design-system/page-header";
import {ConfirmDialog} from "@/design-system/confirm-dialog";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import type {RowDraft,Table,ZoneDraft} from "../domain/types";
import {createTables,deactivateTableOrZone,listActiveZones,listTables,listZones,saveTable as persistTable,saveZone as persistZone} from "../infrastructure/tables-api";

const emptyRow:RowDraft={name:"",seats:"2",zone:"",active:true,isNew:true};
const emptyZone:ZoneDraft={name:"",sortOrder:0,active:true};


function TableSkeleton(){return <div className="table-skeleton"><div className="sk-head"><i/><i/><i/><i/><i/><i/></div>{Array.from({length:5}).map((_,i)=><div className="sk-row" key={i}><div className="sk-name"><span/><b/></div><i/><i/><i/><i/><i/></div>)}</div>}

async function printAllQrs(tables:Table[],restaurantName:string){
 const origin=typeof window!=="undefined"?window.location.origin:"";
 const printable=tables.filter(t=>t.qrEnabled&&t.qrToken&&t.active);
 if(!printable.length){alert("No hay mesas con QR activo para imprimir.");return}
 // Generar los QRs en canvases temporales
 const cards=await Promise.all(printable.map(t=>new Promise<string>((resolve)=>{
   const canvas=document.createElement("canvas");
   QRCode.toCanvas(canvas,`${origin}/mesa/${t.qrToken}`,{width:200,margin:1,color:{dark:"#0f1c2e",light:"#ffffff"}},(err)=>{if(err){resolve("");return}
     resolve(`<div class="qr-print-card"><div class="qr-print-card-brand"><img src="${origin}/assets/images/logo.png" alt="fudIA" width="32" height="32"/><b>${restaurantName}</b></div><img src="${canvas.toDataURL("image/png")}" class="qr-print-card-img"/><p class="qr-print-card-hint">Escanea para ver la carta y hacer tu pedido</p></div>`);
   });
 })));
 const html=`<!DOCTYPE html><html><head><meta charset="utf-8"><title>QRs de Mesas - ${restaurantName}</title>
 <style>
   *{box-sizing:border-box;margin:0;padding:0}
   body{font-family:Manrope,Arial,sans-serif;background:#fff;padding:0}
   .grid{display:grid;grid-template-columns:repeat(3,1fr);gap:14px;padding:20mm}
   @media print{.no-print{display:none}body{padding:0}.grid{gap:10px;padding:10mm}}
   .qr-print-card{border:2px solid #e4e7ec;border-radius:18px;padding:20px;text-align:center;display:flex;flex-direction:column;align-items:center;gap:14px;page-break-inside:avoid}
   .qr-print-card-brand{display:flex;align-items:center;gap:8px}
   .qr-print-card-brand img{border-radius:8px}
   .qr-print-card-brand b{font-size:16px;font-weight:800;color:#3946b8;letter-spacing:-.03em}
   .qr-print-card-img{width:200px;height:200px;border-radius:8px}
   .qr-print-card-hint{font-size:9px;color:#98a2b3;line-height:1.4}
   .actions{position:fixed;top:16px;right:16px;display:flex;gap:10px;z-index:100}
   .actions button{padding:10px 18px;border:1px solid #e4e7ec;border-radius:10px;background:#fff;font-size:12px;font-weight:700;cursor:pointer}
   .actions button.primary{background:#63dcae;color:#fff;border-color:transparent}
 </style></head><body>
 <div class="no-print actions"><button class="primary" onclick="window.print()">Imprimir</button><button onclick="window.close()">Cerrar</button></div>
 <div class="grid">${cards.join("")}</div>
 </body></html>`;
 const w=window.open("","_blank","width=900,height=700");
 if(w){w.document.write(html);w.document.close();}
}

export function TablesManager(){
 const client=useQueryClient();const{notify}=useFeedback();
 const{organization,can}=useSession();
 const canManageTables=can("tables.manage");
 const canManageZones=can("menu.manage");
 const[tab,setTab]=useState<"tables"|"zones">("tables");
 const[search,setSearch]=useState("");const[statusFilter,setStatusFilter]=useState("");
 const[page,setPage]=useState(1);const[pageSize,setPageSize]=useState(10);
 const[zonePage,setZonePage]=useState(1);const[zonePageSize,setZonePageSize]=useState(10);
 const[confirm,setConfirm]=useState<{kind:"tables"|"zones";id:string;name:string}|null>(null);
 const[adding,setAdding]=useState(false);
 const[newRows,setNewRows]=useState<RowDraft[]>([]);
 const[tableDraft,setTableDraft]=useState<RowDraft|null>(null);
 const[zoneDraft,setZoneDraft]=useState<ZoneDraft|null>(null);
 const[qrTable,setQrTable]=useState<Table|null>(null);
 const[printQr,setPrintQr]=useState(false);

 const tables=useQuery({queryKey:["tables",search,statusFilter,page,pageSize],queryFn:()=>listTables({q:search,status:statusFilter,page,pageSize})});
 const zones=useQuery({queryKey:["zones",zonePage,zonePageSize],queryFn:()=>listZones(zonePage,zonePageSize)});
 const zoneOptions=useQuery({queryKey:["zone-options"],queryFn:listActiveZones});
 const saveBatch=useMutation({mutationFn:(items:RowDraft[])=>createTables(items),onSuccess:()=>{setAdding(false);setNewRows([]);notify({tone:"success",title:"Mesas registradas",message:"Las mesas ya están disponibles en el POS con su QR generado."});void client.invalidateQueries({queryKey:["tables"]})},onError:e=>notify({tone:"danger",title:"No se pudieron registrar",message:e.message})});
 const saveTable=useMutation({mutationFn:(draft:RowDraft)=>persistTable(draft),onSuccess:()=>{setTableDraft(null);notify({tone:"success",title:"Mesa actualizada",message:"Nombre, capacidad, zona y configuración quedaron guardados."});void client.invalidateQueries({queryKey:["tables"]})},onError:e=>notify({tone:"danger",title:"No se pudo actualizar la mesa",message:e.message})});
 const deactivate=useMutation({mutationFn:(vars:{kind:"tables"|"zones";id:string})=>deactivateTableOrZone(vars.kind,vars.id),onSuccess:()=>{setConfirm(null);notify({tone:"success",title:"Desactivado",message:"El registro dejó de estar disponible."});void client.invalidateQueries({queryKey:["tables"]});void client.invalidateQueries({queryKey:["zones"]})},onError:e=>notify({tone:"danger",title:"No se pudo desactivar",message:e.message})});
 const saveZone=useMutation({mutationFn:(d:ZoneDraft)=>persistZone(d),onSuccess:(_,d)=>{setZoneDraft(null);notify({tone:"success",title:d.id?"Zona actualizada":"Zona registrada",message:"La zona ya está disponible para asignar a las mesas."});void client.invalidateQueries({queryKey:["zones"]})},onError:e=>notify({tone:"danger",title:"No se pudo guardar la zona",message:e.message})});

 const data=tables.data;
 const zoneList=zones.data?.items??[];const activeZoneOptions=zoneOptions.data?.items??[];
 const validNewRows=newRows.filter(r=>r.name.trim()!=="");

 function addRow(){setNewRows([...newRows,{...emptyRow}])}
 function removeRow(idx:number){setNewRows(newRows.filter((_,i)=>i!==idx))}
 function updateRow(idx:number,patch:Partial<RowDraft>){setNewRows(newRows.map((r,i)=>i===idx?{...r,...patch}:r))}
 function saveAll(){if(!validNewRows.length){notify({tone:"danger",title:"Sin mesas para guardar",message:"Agrega al menos una mesa con nombre."});return}saveBatch.mutate(validNewRows)}
 function cancelAdd(){setAdding(false);setNewRows([])}

 return <><PageHeader eyebrow="OPERACIÓN" title="Mesas y zonas" description="Registra las mesas y zonas del local. Cada mesa genera un QR para vincular el proceso de atención."/>
 <div className="catalog-tabs-row"><div className="catalog-tabs"><button className={tab==="tables"?"active":""} onClick={()=>setTab("tables")}><Icon name="grid" size={16}/>Mesas<b>{tables.data?.total??0}</b></button><button className={tab==="zones"?"active":""} onClick={()=>setTab("zones")}><Icon name="store" size={16}/>Zonas<b>{zones.data?.total??0}</b></button></div>{tab==="tables"&&!adding&&<div className="qr-batch-actions"><Button icon="qr" kind="secondary" onClick={()=>setPrintQr(true)}>Imprimir QRs</Button>{canManageTables&&<Button icon="plus" onClick={()=>{setAdding(true);addRow()}}>Nueva mesa</Button>}</div>}{tab==="zones"&&canManageZones&&<Button icon="plus" onClick={()=>setZoneDraft(emptyZone)}>Nueva zona</Button>}</div>

 {tab==="tables"?<section className="panel management catalog-panel"><div className="toolbar"><label><Icon name="search" size={18}/><input value={search} onChange={e=>{setSearch(e.target.value);setPage(1)}} placeholder="Buscar por nombre..."/></label><select aria-label="Filtrar por estado" value={statusFilter} onChange={e=>{setStatusFilter(e.target.value);setPage(1)}}><option value="">Todos los estados</option><option value="active">Activas</option><option value="inactive">Inactivas</option></select></div>
 {tables.isLoading?<TableSkeleton/>:tables.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las mesas</b><p>{tables.error.message}</p><Button onClick={()=>tables.refetch()}>Reintentar</Button></div>:!adding&&!data?.items.length?<div className="catalog-state"><span><Icon name="grid"/></span><b>No encontramos mesas</b><p>{search||statusFilter?"Ajusta los filtros para ver otros resultados.":"Crea la primera mesa para comenzar a operar."}</p>{canManageTables&&<Button icon="plus" onClick={()=>{setAdding(true);addRow()}}>Nueva mesa</Button>}</div>:<>
 <div className="table-wrap hover-scroll"><table><thead><tr><th>MESA</th><th>ZONA</th><th>ASIENTOS</th><th>ESTADO</th><th>QR</th><th>ACCIONES</th></tr></thead><tbody>
 {adding&&newRows.map((r,idx)=><tr className="editing-row" key={`new-${idx}`}><td><input className="ds-input" autoFocus={idx===0} value={r.name} onChange={e=>updateRow(idx,{name:e.target.value})} placeholder="Ej. Mesa 1"/></td><td><select className="ds-select" value={r.zone} onChange={e=>updateRow(idx,{zone:e.target.value})}><option value="">Sin zona</option>{activeZoneOptions.map(z=><option key={z.id} value={z.name}>{z.name}</option>)}</select></td><td><input className="ds-input seats-input" type="number" min="1" inputMode="numeric" value={r.seats} onChange={e=>updateRow(idx,{seats:e.target.value})}/></td><td>—</td><td><i style={{color:"var(--text-tertiary)"}}>Se genera al guardar</i></td><td><div className="table-actions"><RowActionButton action="remove" onClick={()=>newRows.length>1?removeRow(idx):cancelAdd()}/></div></td></tr>)}
 {data?.items.map((t,i)=><tr className={i%2?"alternate":""} key={t.id}><td><span className={`row-icon r${i%3}`}><Icon name="grid" size={18}/></span><b>{t.name}</b></td><td>{t.zone||"Sin zona"}</td><td><b>{t.seats}</b></td><td><Status tone={t.active?"green":"gray"}>{t.active?"Activa":"Inactiva"}</Status></td><td>{t.qrEnabled&&t.qrToken?<button className="qr-cell-btn" onClick={()=>setQrTable(t)} title="Escanear QR de la mesa"><Icon name="qr" size={18}/><span>Escanear</span></button>:<span className="table-muted">—</span>}</td><td><div className="table-actions">{canManageTables&&<RowActionButton action="edit" label={`Editar ${t.name}`} onClick={()=>setTableDraft({id:t.id,name:t.name,seats:String(t.seats),zone:t.zone,active:t.active,qrEnabled:t.qrEnabled})}/>} {canManageTables&&t.active&&<RowActionButton action="deactivate" onClick={()=>setConfirm({kind:"tables",id:t.id,name:t.name})}/>}</div></td></tr>)}
 </tbody></table></div>
 {adding&&<div className="batch-actions"><div><Button icon="plus" kind="ghost" onClick={addRow}>Nueva fila</Button></div><div><button type="button" className="button ghost" onClick={cancelAdd}>Cancelar</button><button type="button" className="button primary" disabled={saveBatch.isPending||!validNewRows.length} onClick={saveAll}>{saveBatch.isPending?"Guardando...":`Guardar ${validNewRows.length||""} ${validNewRows.length===1?"mesa":"mesas"}`}</button></div></div>}
 {!adding&&<Pagination page={page} size={pageSize} total={data?.total??0} onPage={setPage} onSize={value=>{setPageSize(value);setPage(1)}}/>}
 </>}</section>

 :<section className="panel management catalog-panel"><div className="toolbar"><div className="toolbar-title"><b>Zonas del local</b><small>Agrupa las mesas por zona (salón, terraza, barra, etc.).</small></div></div>
 {zones.isLoading?<TableSkeleton/>:zones.isError?<div className="catalog-state error"><span><Icon name="alert"/></span><b>No pudimos cargar las zonas</b><p>{zones.error.message}</p><Button onClick={()=>zones.refetch()}>Reintentar</Button></div>:!zoneList.length?<div className="catalog-state"><span><Icon name="menu"/></span><b>No encontramos zonas</b><p>Crea la primera zona para organizar las mesas.</p><Button icon="plus" onClick={()=>setZoneDraft(emptyZone)}>Nueva zona</Button></div>:<>
 <div className="table-wrap hover-scroll"><table><thead><tr><th>ZONA</th><th>ORDEN</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{zoneList.map((z,i)=><tr className={i%2?"alternate":""} key={z.id}><td><span className={`row-icon r${i%3}`}><Icon name="store" size={18}/></span><b>{z.name}</b></td><td>{z.sortOrder}</td><td><Status tone={z.active?"green":"gray"}>{z.active?"Activa":"Inactiva"}</Status></td><td><div className="table-actions"><RowActionButton action="edit" onClick={()=>setZoneDraft({id:z.id,name:z.name,sortOrder:z.sortOrder,active:z.active})}/>{z.active&&<RowActionButton action="deactivate" onClick={()=>setConfirm({kind:"zones",id:z.id,name:z.name})}/>}</div></td></tr>)}</tbody></table></div><Pagination page={zonePage} size={zonePageSize} total={zones.data?.total??0} onPage={setZonePage} onSize={value=>{setZonePageSize(value);setZonePage(1)}}/>
 </>}</section>}

 {tableDraft&&<TableDialog draft={tableDraft} zones={activeZoneOptions.map(zone=>zone.name)} busy={saveTable.isPending} close={()=>setTableDraft(null)} save={draft=>saveTable.mutate(draft)}/>} 
 {zoneDraft&&<ZoneDialog draft={zoneDraft} busy={saveZone.isPending} close={()=>setZoneDraft(null)} save={d=>saveZone.mutate(d)}/>}
 {qrTable&&<QrDialog table={qrTable} restaurantName={organization?.name??"Restaurante"} close={()=>setQrTable(null)}/>}
 {printQr&&<PrintQrDialog tables={data?.items??[]} restaurantName={organization?.name??"Restaurante"} close={()=>setPrintQr(false)}/>}
 <ConfirmDialog open={Boolean(confirm)} title={`Desactivar ${confirm?.kind==="tables"?"mesa":"zona"}`} description={`"${confirm?.name??""}" dejará de estar disponible para nuevas operaciones.`} confirmLabel="Desactivar" pending={deactivate.isPending} onCancel={()=>setConfirm(null)} onConfirm={()=>confirm&&deactivate.mutate({kind:confirm.kind,id:confirm.id})}/>
 </>;
}

function TableDialog({draft,zones,busy,close,save}:{draft:RowDraft;zones:string[];busy:boolean;close:()=>void;save:(draft:RowDraft)=>void}){
 const[value,setValue]=useState(draft);
 const zoneOptions=value.zone&&!zones.includes(value.zone)?[value.zone,...zones]:zones;
 const valid=value.name.trim().length>0&&Number(value.seats)>0;
 return <div className="modal-backdrop modal-overlay-in" role="presentation"><section className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="table-edit-title" aria-busy={busy}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="grid" size={18}/></span><div><small>EDITAR MESA</small><h2 id="table-edit-title">Configuración de la mesa</h2></div><button onClick={close} disabled={busy} aria-label="Cerrar"><Icon name="close"/></button></header>
 <form onSubmit={event=>{event.preventDefault();if(valid)save(value)}}>
  <div className="form-grid">
   <label className="span-2">Nombre de la mesa<input className="ds-input" required maxLength={80} value={value.name} onChange={event=>setValue({...value,name:event.target.value})} placeholder="Ej. Mesa 1"/></label>
   <label>Asientos<input className="ds-input" required type="number" min="1" max="99" inputMode="numeric" value={value.seats} onChange={event=>setValue({...value,seats:event.target.value})}/></label>
   <label>Zona<select className="ds-select" value={value.zone} onChange={event=>setValue({...value,zone:event.target.value})}><option value="">Sin zona</option>{zoneOptions.map(zone=><option value={zone} key={zone}>{zone}</option>)}</select></label>
   <label className="switch-row compact"><input type="checkbox" checked={value.active} onChange={event=>setValue({...value,active:event.target.checked})}/><span/><b>Mesa activa</b></label>
   <label className="switch-row compact"><input type="checkbox" checked={value.qrEnabled??true} onChange={event=>setValue({...value,qrEnabled:event.target.checked})}/><span/><b>QR habilitado</b></label>
  </div>
  <div className="table-edit-note"><Icon name="alert" size={14}/><span>Desactivar una mesa con pedidos abiertos será rechazado por seguridad.</span></div>
  <footer><button type="button" className="button ghost" disabled={busy} onClick={close}>Cancelar</button><button type="submit" className="button primary" disabled={busy||!valid}>{busy?"Guardando…":"Guardar"}</button></footer>
 </form>
 </section></div>;
}

function QrDialog({table,restaurantName,close}:{table:Table;restaurantName:string;close:()=>void}){
 const canvasRef=useRef<HTMLCanvasElement>(null);
 const qrUrl=typeof window!=="undefined"?`${window.location.origin}/mesa/${table.qrToken}`:"";

 useEffect(()=>{
   if(canvasRef.current&&qrUrl){
     QRCode.toCanvas(canvasRef.current,qrUrl,{width:240,margin:2,color:{dark:"#1a2151",light:"#ffffff"}},(err)=>{
       if(err){console.error(err);return}
       const canvas=canvasRef.current;if(!canvas)return;
       const ctx=canvas.getContext("2d");if(!ctx)return;
       const logo=new Image();logo.crossOrigin="anonymous";
       logo.onload=()=>{
         const size=52;const x=(canvas.width-size)/2;const y=(canvas.height-size)/2;
         ctx.fillStyle="#fff";ctx.fillRect(x-5,y-5,size+10,size+10);
         ctx.save();ctx.beginPath();ctx.roundRect(x,y,size,size,10);ctx.clip();ctx.drawImage(logo,x,y,size,size);ctx.restore();
       };logo.src="/assets/images/logo.png";
     });
   }
 },[qrUrl]);

 function download(){
   if(!canvasRef.current)return;
   const qr=canvasRef.current.toDataURL("image/png");
   const c=document.createElement("canvas");
   c.width=340;c.height=440;
   const ctx=c.getContext("2d");if(!ctx)return;
   ctx.fillStyle="#fff";ctx.fillRect(0,0,340,440);
   ctx.strokeStyle="#e4e7ec";ctx.lineWidth=2;ctx.roundRect(16,16,308,408,18);ctx.stroke();
   const logo=new Image();logo.crossOrigin="anonymous";
   logo.onload=()=>{
     ctx.save();ctx.beginPath();ctx.roundRect(130,40,80,80,12);ctx.clip();ctx.drawImage(logo,130,40,80,80);ctx.restore();
     ctx.fillStyle="#3946b8";ctx.font="800 20px Manrope,Arial";ctx.textAlign="center";ctx.fillText(restaurantName,170,148);
     const qrImg=new Image();qrImg.onload=()=>{
       ctx.drawImage(qrImg,50,170,240,240);
       ctx.fillStyle="#98a2b3";ctx.font="10px Manrope,Arial";
       ctx.fillText("Escanea para ver la carta y hacer tu pedido",170,430);
       const link=document.createElement("a");
       link.download=`qr-mesa-${table.name.replace(/\s+/g,"-").toLowerCase()}.png`;
       link.href=c.toDataURL("image/png");link.click();
     };qrImg.src=qr;
   };logo.src="/assets/images/logo.png";
 }

 function share(){
   if(!canvasRef.current)return;
   canvasRef.current.toBlob(async(blob)=>{
     if(!blob)return;
     const file=new File([blob],`qr-mesa-${table.name.replace(/\s+/g,"-").toLowerCase()}.png`,{type:"image/png"});
     if(navigator.canShare&&navigator.canShare({files:[file]})){
       try{await navigator.share({title:`QR Mesa ${table.name}`,text:`Escanea para acceder a la mesa ${table.name}`,files:[file]})}catch(_){}
     }else if(navigator.share){
       navigator.share({title:`QR Mesa ${table.name}`,text:`Escanea para acceder a la mesa ${table.name}`,url:qrUrl}).catch(()=>{});
     }else{
       navigator.clipboard.writeText(qrUrl);
     }
   },"image/png");
 }

 return <div className="modal-backdrop modal-overlay-in" role="presentation"><section className="crud-modal compact modal-panel-in qr-modal" role="dialog" aria-modal="true" aria-labelledby="qr-title">
 <header><span className="modal-title-icon"><Icon name="qr" size={18}/></span><div><small>CÓDIGO QR DE LA MESA</small><h2 id="qr-title">{table.name}</h2></div><button onClick={close} aria-label="Cerrar"><Icon name="close"/></button></header>
 <div className="qr-dialog-body">
   <div className="qr-hero">
     <div className="qr-hero-glow"/>
     <div className="qr-brand"><img src="/assets/images/logo.png" alt="fudIA" width={24} height={24}/><b>{restaurantName}</b></div>
     <div className="qr-canvas-wrap"><canvas ref={canvasRef}/></div>
     <div className="qr-hero-label"><Icon name="qr" size={14}/><span>Escanea para acceder</span></div>
   </div>
   <div className="qr-info">
     <p className="qr-hint">El cliente escanea este código desde su celular para ver la carta digital, llamar al mozo y hacer pedidos desde la mesa.</p>
     <div className="qr-url"><Icon name="share" size={14}/><span>{qrUrl}</span></div>
   </div>
 </div>
 <footer className="qr-footer">
   <button type="button" className="button ghost" onClick={share}><Icon name="share" size={16}/>Compartir</button>
   <button type="button" className="button primary" onClick={download}><Icon name="download" size={16}/>Descargar</button>
 </footer>
 </section></div>;
}

function ZoneDialog({draft,busy,close,save}:{draft:ZoneDraft;busy:boolean;close:()=>void;save:(d:ZoneDraft)=>void}){
 const[value,setValue]=useState(draft);
 return <div className="modal-backdrop modal-overlay-in" role="presentation"><section className="crud-modal compact modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="zone-title" aria-busy={busy}><div className="modal-accent"/><header><span className="modal-title-icon"><Icon name="store" size={18}/></span><div><small>{value.id?"EDITAR ZONA":"NUEVA ZONA"}</small><h2 id="zone-title">Información de la zona</h2></div><button onClick={close} disabled={busy} aria-label="Cerrar"><Icon name="close"/></button></header><form onSubmit={e=>{e.preventDefault();save(value)}}><div className="form-grid">
 <label className="span-2">Nombre de la zona<input required maxLength={60} value={value.name} onChange={e=>setValue({...value,name:e.target.value})} placeholder="Ej. Terraza"/></label>
 <label className="span-2">Orden<input type="number" min="0" inputMode="numeric" value={value.sortOrder} onChange={e=>setValue({...value,sortOrder:Number(e.target.value)})} placeholder="0"/></label>
 </div><footer><button type="button" className="button ghost" disabled={busy} onClick={close}>Cancelar</button><button type="submit" className="button primary" disabled={busy}>Guardar</button></footer></form></section></div>;
}

function PrintQrDialog({tables,restaurantName,close}:{tables:Table[];restaurantName:string;close:()=>void}){
 const canvasRefs=useRef<Record<string,HTMLCanvasElement|null>>({});
 const origin=typeof window!=="undefined"?window.location.origin:"";
 const printable=tables.filter(t=>t.qrEnabled&&t.qrToken&&t.active);

 useEffect(()=>{
   printable.forEach(t=>{
     const canvas=canvasRefs.current[t.id];
     if(canvas){
       QRCode.toCanvas(canvas,`${origin}/mesa/${t.qrToken}`,{width:200,margin:1,color:{dark:"#1a2151",light:"#ffffff"}},(err)=>{
         if(err){console.error(err);return}
         // Superponer el logo en el centro del QR
         const ctx=canvas.getContext("2d");if(!ctx)return;
         const logo=new Image();logo.crossOrigin="anonymous";
         logo.onload=()=>{
           const size=44;const x=(canvas.width-size)/2;const y=(canvas.height-size)/2;
           ctx.fillStyle="#fff";ctx.fillRect(x-4,y-4,size+8,size+8);
           ctx.save();ctx.beginPath();ctx.roundRect(x,y,size,size,8);ctx.clip();ctx.drawImage(logo,x,y,size,size);ctx.restore();
         };logo.src="/assets/images/logo.png";
       });
     }
   });
 },[printable.length]);

 function doPrint(){
   document.body.classList.add("printing-qrs");
   window.print();
   window.onafterprint=()=>{document.body.classList.remove("printing-qrs");window.onafterprint=null};
 }

 return <div className="modal-backdrop modal-overlay-in" role="presentation"><section className="crud-modal qr-print-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="print-qr-title">
 <header><span className="modal-title-icon"><Icon name="qr" size={18}/></span><div><small>IMPRIMIR QRs</small><h2 id="print-qr-title">Códigos QR de las mesas</h2></div><button onClick={close} aria-label="Cerrar"><Icon name="close"/></button></header>
 <div className="qr-print-preview">
   {printable.length===0?<div className="qr-print-empty"><Icon name="qr" size={32}/><b>No hay mesas con QR activo</b><p>Crea mesas o activa sus QRs para imprimir.</p></div>:
   <div className="qr-print-grid">{printable.map(t=><div key={t.id} className="qr-print-card">
     <div className="qr-print-card-accent"/>
     <div className="qr-print-card-brand"><span className="qr-print-card-logo"><img src="/assets/images/logo.png" alt="fudIA" width={28} height={28}/></span><b>{restaurantName}</b></div>
     <canvas ref={el=>{canvasRefs.current[t.id]=el}}/>
     <div className="qr-print-card-footer"><Icon name="qr" size={12}/><span>Escanea para ver la carta y hacer tu pedido</span></div>
   </div>)}</div>}
 </div>
 <footer className="qr-footer">
   <button type="button" className="button ghost" onClick={close}>Cancelar</button>
   <button type="button" className="button primary" onClick={doPrint} disabled={!printable.length}><Icon name="download" size={16}/>Imprimir {printable.length} QRs</button>
 </footer>
 </section></div>;
}
