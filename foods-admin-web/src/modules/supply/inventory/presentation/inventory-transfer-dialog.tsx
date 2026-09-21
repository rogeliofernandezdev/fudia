"use client";
import {useMemo,useState} from "react";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import type {InventoryProductOption,InventoryTransferDraft,LocationOption} from "../domain/types";

export function InventoryTransferDialog({items,locations,currentLocationId,busy,close,save}:{items:InventoryProductOption[];locations:LocationOption[];currentLocationId:string;busy:boolean;close:()=>void;save:(draft:InventoryTransferDraft)=>void}){
 const destinations=locations.filter(l=>l.id!==currentLocationId);
 const[first]=items;
 const[draft,setDraft]=useState<InventoryTransferDraft>({toLocationId:destinations[0]?.id??"",notes:"",items:first?[{inventoryItemId:first.id,quantity:""}]:[]});
 const selectedIds=useMemo(()=>new Set(draft.items.map(i=>i.inventoryItemId)),[draft.items]);
 function add(){const next=items.find(i=>!selectedIds.has(i.id));if(next)setDraft({...draft,items:[...draft.items,{inventoryItemId:next.id,quantity:""}]})}
 function submit(){
  const lines=draft.items.filter(i=>Number(i.quantity)>0);
  if(!draft.toLocationId||!lines.length)return;
  for(const line of lines){const item=items.find(i=>i.id===line.inventoryItemId);if(item&&Number(line.quantity)>Number(item.quantity)+0.000001)return}
  save({...draft,items:lines});
 }
 return <div className="modal-backdrop modal-overlay-in"><section className="crud-modal modal-panel-in" role="dialog" aria-modal="true">
  <header><span className="modal-title-icon"><Icon name="truck" size={18}/></span><div><small>TRANSFERENCIA ENTRE LOCALES</small><h2>Mover stock</h2></div><button onClick={close} aria-label="Cerrar"><Icon name="close"/></button></header>
  <div className="form-grid"><label className="span-2">Local destino<Select value={draft.toLocationId} onChange={e=>setDraft({...draft,toLocationId:e.target.value})}><option value="">Selecciona destino</option>{destinations.map(l=><option key={l.id} value={l.id}>{l.name}</option>)}</Select></label></div>
  <div className="table-wrap"><table><thead><tr><th>ARTÍCULO</th><th>DISPONIBLE</th><th>TRANSFERIR</th><th></th></tr></thead><tbody>{draft.items.map((line,index)=>{const item=items.find(x=>x.id===line.inventoryItemId);return <tr key={index}><td><Select value={line.inventoryItemId} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,inventoryItemId:e.target.value}:x)})}>{items.map(i=><option key={i.id} value={i.id} disabled={selectedIds.has(i.id)&&i.id!==line.inventoryItemId}>{i.name} · {i.unit}</option>)}</Select></td><td>{item?.quantity??"0"} {item?.unit}</td><td><Input type="number" min="0.001" max={item?.quantity} step="0.001" value={line.quantity} onChange={e=>setDraft({...draft,items:draft.items.map((x,i)=>i===index?{...x,quantity:e.target.value}:x)})}/></td><td><Button kind="ghost" onClick={()=>setDraft({...draft,items:draft.items.filter((_,i)=>i!==index)})}>Quitar</Button></td></tr>})}</tbody></table></div>
  <Button kind="secondary" icon="plus" onClick={add} disabled={draft.items.length>=items.length}>Agregar artículo</Button>
  <label>Nota<Textarea rows={2} maxLength={500} value={draft.notes} onChange={e=>setDraft({...draft,notes:e.target.value})} placeholder="Motivo o referencia de traslado"/></label>
  <footer><Button kind="ghost" onClick={close}>Cancelar</Button><Button disabled={busy||!draft.toLocationId} onClick={submit}>{busy?"Transfiriendo…":"Confirmar transferencia"}</Button></footer>
 </section></div>
}
