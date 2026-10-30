"use client";
/* eslint-disable @next/next/no-img-element -- Imágenes de presentación provenientes del catálogo del restaurante. */
import "../../styles/salon-comanda.css";
import {useState,useEffect,useRef,useCallback} from "react";
import {Button,ConfirmDialog,Dialog,FormField,Icon,Input,NumberStepper,Select,Textarea,ModalHeader,ModalFooter} from "@/design-system";
import {useMoney,useSession} from "@/providers";
import {useQuery} from "@tanstack/react-query";
import {listSelectedProductAvailability} from "../../salon/infrastructure/catalog-api";
import {useFeedback} from "@/providers/feedback-provider";
import {CatalogProduct,ComboConfigurator,ComboSelection,ConfiguredCombo,ComandaCatalog} from "../../salon/presentation/comanda-catalog";
import type {Draft,FloorTable,LineDraft} from "../../salon/domain/types";
import type {IconName} from "@/design-system/icons";

const tableTitle=(name:string)=>/^mesa\b/i.test(name.trim())?name.trim():`Mesa ${name.trim()}`;
const uniqueKey=()=>typeof crypto!=="undefined"&&typeof crypto.randomUUID==="function"?crypto.randomUUID():`${Date.now()}-${Math.random()}`;
const sameDraft=(a:Draft,b:Draft)=>JSON.stringify([a.lines,a.notes,a.customerName,a.tableId])===JSON.stringify([b.lines,b.notes,b.customerName,b.tableId]);

/**
 * Comanda a pantalla completa (Salón, Pedidos y pedido manual). El título es el dato que el mozo confirma
 * (mesa o cliente). Agregar desde la carta no abre el resumen ni el teclado en móvil: el contador de la
 * barra inferior confirma la acción. Cerrar con cambios pide confirmar el descarte.
 */
export function ComandaView({initial,mode,allTables,busy,close,save,channelLabel="Salón",onBack,onResponseClose}:{initial:Draft;mode:"create"|"edit"|"append";allTables:FloorTable[];busy:boolean;close:(draft:Draft)=>void;save:(v:Draft,sendToKitchen:boolean)=>void|Promise<void>;notify:(n:{tone:"danger"|"success";title:string;message:string})=>void;channelLabel?:string;onBack?:(draft:Draft)=>void;onResponseClose?:()=>void}){
  const money=useMoney();
  const {location}=useSession();
  const adding=mode==="append";
  const editing=mode!=="create";
  const salon=initial.channel==="salon";
  const[v,setV]=useState(initial);
  const[sheetOpen,setSheetOpen]=useState(false);
  const[validationError,setValidationError]=useState("");
  const[comboEditor,setComboEditor]=useState<{comboId:string;lineKey?:string;initialSelections:ComboSelection[]}|null>(null);
  const[confirmDiscard,setConfirmDiscard]=useState(false);
  const[openNotes,setOpenNotes]=useState<string[]>([]);
  const[announcement,setAnnouncement]=useState("");
  const[pendingAction,setPendingAction]=useState<"draft"|"send"|null>(null);
  const submitting=useRef(false);
  const locked=()=>busy||submitting.current;
  const dirty=!sameDraft(v,initial);
  const freeTables=allTables.filter(t=>!t.order||t.id===v.tableId);
  const selTable=allTables.find(t=>t.id===v.tableId);
  const qtyByProduct=v.lines.reduce<Record<string,number>>((acc,line)=>{acc[line.productId]=(acc[line.productId]??0)+line.qty;return acc},{});
  // Tope por existencias: lo que la carta informa como disponible más lo ya reservado por este pedido guardado.
  // Productos sin control (remaining null) no tienen tope; el backend valida de nuevo al guardar.
  const[availability,setAvailability]=useState<Record<string,CatalogProduct>>({});
  const availabilityRef=useRef<Record<string,CatalogProduct>>({});
  const reserved=initial.lines.reduce<Record<string,number>>((acc,line)=>{if(line.itemType==="product"&&line.sourceItemId)acc[line.productId]=(acc[line.productId]??0)+line.qty;return acc},{});
  const recordAvailability=useCallback((items:CatalogProduct[])=>{
    const next={...availabilityRef.current};let changed=false;
    for(const item of items){
      const previous=next[item.id];
      if(!previous||previous.remaining!==item.remaining||previous.availabilityStatus!==item.availabilityStatus){next[item.id]=item;changed=true}
    }
    if(changed){availabilityRef.current=next;setAvailability(next)}
  },[]);
  const blockedFor=(productId:string)=>["sold_out","unavailable"].includes(availabilityRef.current[productId]?.availabilityStatus??"");
  const quantityLimit=(productId:string,remaining?:number|null)=>typeof remaining==="number"&&Number.isFinite(remaining)?Math.max(0,Math.floor(remaining+(reserved[productId]??0))):undefined;
  const limitFor=(productId:string,remaining?:number|null)=>quantityLimit(productId,remaining===undefined?availabilityRef.current[productId]?.remaining:remaining);
  const selectedIds=Array.from(new Set(v.lines.filter(line=>line.itemType==="product").map(line=>line.productId))).sort();
  const selectedAvailability=useQuery({
    queryKey:["order-selected-availability",location?.id,selectedIds],
    queryFn:()=>listSelectedProductAvailability(selectedIds),
    enabled:selectedIds.length>0,
    refetchInterval:10000,
    refetchIntervalInBackground:false,
    refetchOnWindowFocus:"always",
    staleTime:0,
  });
  useEffect(()=>{if(selectedAvailability.data)recordAvailability(selectedAvailability.data.items)},[selectedAvailability.data,selectedAvailability.dataUpdatedAt,recordAvailability]);
  // Una categoría con caché anterior no debe reemplazar el estado fresco de lo seleccionado.
  const recordCatalogAvailability=(items:CatalogProduct[])=>recordAvailability(items.filter(item=>!selectedAvailability.data?.items.some(selected=>selected.id===item.id)));
  const {notify:feedback}=useFeedback();
  // Superar las existencias abre el aviso de validación (tono info: no cierra la comanda) y no cambia la cantidad.
  const warnStock=(name:string,limit:number)=>{
    setAnnouncement(`${name}: solo hay ${limit} disponibles.`);
    feedback({tone:"info",title:"Cantidad no disponible",message:`La cantidad excede las existencias: solo quedan ${limit} ${limit===1?"unidad":"unidades"} de ${name}.`});
  };
  const count=v.lines.reduce((a,l)=>a+l.qty,0);
  const subtotal=v.lines.reduce((a,l)=>a+l.qty*l.unitPrice,0);
  const deliveryFee=v.channel==="delivery"?Number(v.deliveryFee)||0:0;
  const total=subtotal+deliveryFee;
  const countLabel=`${count} ${count===1?"producto":"productos"}`;

  const patchLine=(i:number,p:Partial<LineDraft>)=>{if(!locked())setV(prev=>({...prev,lines:prev.lines.map((l,n)=>n===i?{...l,...p}:l)}))};
  const tap=(p:CatalogProduct)=>{
    if(locked()||selectedAvailability.isError)return;
    if(!availabilityRef.current[p.id])recordAvailability([p]);
    const current=availabilityRef.current[p.id]??p;
    if(["sold_out","unavailable"].includes(current.availabilityStatus??""))return;
    const limit=limitFor(p.id,current.remaining);
    if(limit!==undefined&&(qtyByProduct[p.id]??0)>=limit){warnStock(p.name,limit);return}
    const existing=v.lines.find(l=>l.itemType==="product"&&l.productId===p.id);
    setV(prev=>{
      // Revalidar sobre el borrador más reciente impide exceder el tope con clics batched.
      const selected=prev.lines.reduce((sum,line)=>sum+(line.productId===p.id?line.qty:0),0);
      if(limit!==undefined&&selected+1>limit)return prev;
      const i=prev.lines.findIndex(l=>l.itemType==="product"&&l.productId===p.id);
      if(i>=0)return{...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+1}:l)};
      return{...prev,lines:[...prev.lines,{lineKey:p.id,itemType:"product",productId:p.id,name:p.name,qty:1,unitPrice:Number(p.price)||0,note:"",selections:[],imageUrl:p.imageUrl}]};
    });
    setValidationError("");
    setAnnouncement(`${p.name} agregado. Cantidad: ${(existing?.qty??0)+1}.`);
  };
  const setQty=(i:number,qty:number)=>{
    if(locked()||!Number.isInteger(qty)||qty<1)return;
    setV(prev=>{
      const line=prev.lines[i];if(!line)return prev;
      const limit=line.itemType==="product"?limitFor(line.productId):undefined;
      const selected=prev.lines.reduce((sum,item)=>sum+(item.productId===line.productId?item.qty:0),0);
      if(qty>line.qty&&((line.itemType==="product"&&(!availabilityRef.current[line.productId]||blockedFor(line.productId)))||(limit!==undefined&&selected-line.qty+qty>limit)))return prev;
      return {...prev,lines:prev.lines.map((item,index)=>index===i?{...item,qty}:item)};
    });
  };
  const removeLine=(i:number)=>{if(locked())return;const removed=v.lines[i];setV(prev=>({...prev,lines:prev.lines.filter((_,n)=>n!==i)}));if(removed)setAnnouncement(`${removed.name} quitado de la comanda.`)};
  const toggleNote=(key:string)=>setOpenNotes(prev=>prev.includes(key)?prev.filter(item=>item!==key):[...prev,key]);
  const configureCombo=(comboId:string)=>{if(!locked())setComboEditor({comboId,initialSelections:[]})};
  const editCombo=(line:LineDraft)=>{if(!locked())setComboEditor({comboId:line.productId,lineKey:line.lineKey,initialSelections:line.selections})};
  const applyCombo=(combo:ConfiguredCombo)=>{
    if(locked())return;
    const editingLineKey=comboEditor?.lineKey;
    const lineKey=editingLineKey??`combo-${combo.productId}-${uniqueKey()}`;
    setV(prev=>{
      if(editingLineKey)return{...prev,lines:prev.lines.map(line=>line.lineKey===editingLineKey?{...line,name:combo.name,unitPrice:combo.unitPrice,selections:combo.selections,repriceCombo:true,imageUrl:combo.imageUrl}:line)};
      return{...prev,lines:[...prev.lines,{lineKey,itemType:"combo",productId:combo.productId,name:combo.name,qty:1,unitPrice:combo.unitPrice,note:"",selections:combo.selections,imageUrl:combo.imageUrl}]};
    });
    setComboEditor(null);
    setAnnouncement(`${combo.name} ${editingLineKey?"actualizado":"agregado"}.`);
  };

  useEffect(()=>{
    if(!dirty)return;
    const prevent=(event:BeforeUnloadEvent)=>{event.preventDefault();event.returnValue=""};
    window.addEventListener("beforeunload",prevent);return()=>window.removeEventListener("beforeunload",prevent);
  },[dirty]);

  const requestClose=()=>{if(locked())return;if(dirty){setConfirmDiscard(true);return}close(v)};
  const editingOnly=editing&&!adding;
  const submitIcon=editingOnly?"check":"send";
  const busyLabel=editingOnly?"Guardando…":"Enviando…";
  const savingDraft=pendingAction==="draft";
  const actionsBusy=busy||pendingAction!==null;
  const submitLabel=adding?"Agregar y enviar a cocina":editing?"Guardar":"Enviar a cocina";
  const submit=async(sendToKitchen:boolean)=>{
    if(locked())return;
    if(!v.lines.length){setValidationError("Agrega al menos un producto.");return}
    if(salon&&!v.tableId){setValidationError("Selecciona la mesa del pedido.");setSheetOpen(true);return}
    setValidationError("");
    submitting.current=true;
    setPendingAction(salon&&!editing&&!sendToKitchen?"draft":"send");
    try{await save(salon?v:{...v,tableId:""},salon?(adding||sendToKitchen):true)}finally{submitting.current=false;setPendingAction(null)}
  };

  const verb=adding?"Agregar a ":editing?"Editar ":"";
  const title=salon
    ?(selTable?`${verb}${tableTitle(selTable.name)}`:adding?"Agregar productos":editing?"Editar comanda":"Nueva comanda")
    :`${verb}${channelLabel}${v.customerName?` · ${v.customerName.trim()}`:""}`;
  const context:{icon:IconName;text:string}[]=(salon
    ?(selTable?[{icon:"mapPin",text:selTable.zone},{icon:"users",text:selTable.seats?`${selTable.seats} ${selTable.seats===1?"persona":"personas"}`:""},{icon:"user",text:v.customerName.trim()}]:[{icon:"tables",text:"Selecciona la mesa en el resumen"}])
    :[{icon:"phone",text:v.customerPhone},{icon:"mapPin",text:v.channel==="delivery"?v.address:""},{icon:"note",text:v.channel==="delivery"?v.reference:""}]
  ).filter((item):item is {icon:IconName;text:string}=>Boolean(item.text?.trim()));

  return(
    <Dialog onResponseClose={onResponseClose??(()=>close(v))} as="div" className="cmd-shell" role="dialog" aria-modal="true" aria-busy={busy} aria-labelledby="cmd-title">
      <ModalHeader title={title} titleId="cmd-title" icon="dish" onClose={requestClose} busy={busy} busyOverlay={false} closeLabel={salon?"Cerrar comanda y volver al salón":"Cerrar comanda"} details={<>{onBack&&<Button kind="ghost" icon="chevronLeft" disabled={busy} onClick={()=>{if(!locked())onBack(v)}}>Editar datos del pedido</Button>}{context.length?<ul className="cmd-context">{context.map(item=><li key={item.icon+item.text}><Icon name={item.icon} size={14}/><span>{item.text}</span></li>)}</ul>:<span>{channelLabel}</span>}</>}/>

      <div className="cmd-body">
        <section className="cmd-pane" aria-label="Carta del restaurante">
          <ComandaCatalog qtyByProduct={qtyByProduct} reservedByProduct={reserved} availabilityByProduct={availability} onAvailability={recordCatalogAvailability} onPick={tap} onConfigureCombo={configureCombo} disabled={busy||selectedAvailability.isError}/>
        </section>

        <aside className={"cmd-summary"+(sheetOpen?" open":"")} aria-label="Resumen de la comanda">
          <header className="cmd-summary-head">
            <i className="cmd-grip" aria-hidden="true"/>
            <h3>Comanda</h3>
            <span className="cmd-summary-count">{countLabel}</span>
            <button type="button" className="cmd-sheet-close" onClick={()=>setSheetOpen(false)} aria-label="Volver a la carta" title="Volver a la carta"><Icon name="chevron" size={24}/></button>
          </header>

          <div className="cmd-summary-scroll">
            {salon&&(!v.tableId||!editing)&&<div className="cmd-fields">
              {!v.tableId&&<FormField label="Mesa" error={validationError==="Selecciona la mesa del pedido."?validationError:undefined}>
                <Select disabled={busy} value={v.tableId} onChange={e=>{setV({...v,tableId:e.target.value});setValidationError("")}}>
                  <option value="">Selecciona una mesa</option>
                  {freeTables.map(t=><option key={t.id} value={t.id}>{t.zone?t.zone+" · ":""}{t.name}</option>)}
                </Select>
              </FormField>}
              {!editing&&<FormField label="Cliente o familia" optional>
                <Input disabled={busy} value={v.customerName} maxLength={120} autoComplete="off" onChange={e=>setV({...v,customerName:e.target.value})}/>
              </FormField>}
            </div>}

            {!v.lines.length?(
              <div className="cmd-empty">
                <span aria-hidden="true"><Icon name="dish" size={24}/></span>
                <strong>Aún no hay productos</strong>
                <p>Toca un plato de la carta para agregarlo. Los menús y combos piden sus opciones.</p>
              </div>
            ):(
              <ul className="cmd-lines">
                {v.lines.map((l,i)=>{
                  const noteOpen=Boolean(l.note)||openNotes.includes(l.lineKey);
                  const limit=l.itemType==="product"?quantityLimit(l.productId,availability[l.productId]?.remaining):undefined;
                  const blocked=l.itemType==="product"&&(!availability[l.productId]||selectedAvailability.isError||["sold_out","unavailable"].includes(availability[l.productId]?.availabilityStatus??""));
                  const lineMax=blocked?l.qty:limit===undefined?undefined:Math.max(l.qty,limit-((qtyByProduct[l.productId]??0)-l.qty));
                  return(
                    <li className={"cmd-line"+(l.itemType==="combo"?" combo":"")} key={l.lineKey}>
                      <span className="cmd-line-visual" aria-hidden="true"><Icon name={l.itemType==="combo"?"combo":"dish"} size={24}/>{l.imageUrl&&<img className="cmd-item-thumb" src={l.imageUrl} alt="" loading="lazy" onError={event=>{event.currentTarget.hidden=true}}/>}</span>
                      <div className="cmd-line-head">
                        <strong>{l.name}</strong>
                        <em>{money(l.qty*l.unitPrice)}</em>
                        {l.qty>1&&<small className="cmd-line-unit">{money(l.unitPrice)} c/u</small>}
                        {blocked?<small className="cmd-line-limit">{selectedAvailability.isError?"No pudimos actualizar la disponibilidad":!availability[l.productId]?"Verificando disponibilidad…":"No disponible para agregar"}</small>:lineMax!==undefined&&l.qty>=lineMax&&<small className="cmd-line-limit">Máximo disponible: {limit}</small>}
                      </div>
                      <NumberStepper appearance="soft" value={l.qty} min={1} max={lineMax} label={l.name} disabled={busy} onChange={qty=>{if(lineMax!==undefined&&limit!==undefined&&qty>lineMax){warnStock(l.name,limit);return}setQty(i,qty)}} onRemove={()=>removeLine(i)}/>
                      {l.itemType==="combo"&&<ul className="cmd-line-selections" aria-label="Elecciones del menú">
                        {l.selections.map(sel=><li key={sel.groupId+sel.productId}><b>{sel.groupName}</b> {sel.name}{sel.surcharge>0?` (+ ${money(sel.surcharge)})`:""}</li>)}
                      </ul>}
                      {(l.itemType==="combo"||!noteOpen)&&<div className="cmd-line-controls">
                        {l.itemType==="combo"&&<button type="button" className="cmd-text-button" disabled={busy} onClick={()=>editCombo(l)}><Icon name="edit" size={16}/>Cambiar</button>}
                        {!noteOpen&&<button type="button" className="cmd-text-button" disabled={busy} onClick={()=>toggleNote(l.lineKey)} aria-label={`Agregar nota a ${l.name}`}><Icon name="note" size={16}/>Nota</button>}
                      </div>}
                      {noteOpen&&<Input disabled={busy} maxLength={240} className="cmd-line-note" value={l.note} autoFocus={!l.note} onChange={e=>patchLine(i,{note:e.target.value})} placeholder="Ej. sin ají, término medio" aria-label={"Nota para "+l.name}/>}
                    </li>
                  );
                })}
              </ul>
            )}

            {v.lines.length>0&&(v.notes||openNotes.includes("general")
              ?<FormField label="Nota para cocina" optional className="cmd-general-note"><Textarea disabled={busy} maxLength={500} rows={2} value={v.notes} onChange={e=>setV({...v,notes:e.target.value})} placeholder="Indicaciones para cocina o atención"/></FormField>
              :<button type="button" className="cmd-text-button cmd-add-general" disabled={busy} onClick={()=>toggleNote("general")}><Icon name="note" size={16}/>Agregar nota para cocina</button>)}
          </div>

          <ModalFooter className="cmd-summary-foot">
            {selectedAvailability.isError&&<p className="cmd-error" role="alert">No pudimos actualizar la disponibilidad. <button type="button" className="cmd-text-button" onClick={()=>void selectedAvailability.refetch()}>Reintentar</button></p>}
            {validationError&&<p className="cmd-error" role="alert">{validationError}</p>}
            {v.channel==="delivery"&&<dl className="cmd-charges"><div><dt>Subtotal</dt><dd>{money(subtotal)}</dd></div><div><dt>Envío</dt><dd>{money(deliveryFee)}</dd></div></dl>}
            <div className="cmd-total"><span>{v.channel==="delivery"?"Total":"Subtotal"}</span><strong>{money(total)}</strong></div>
            <div className="cmd-actions">
              {salon&&!editing&&<Button icon="orders" kind="secondary" className="salon-comanda-draft" onClick={()=>submit(false)} disabled={actionsBusy||!v.lines.length} loading={savingDraft}>{savingDraft?"Guardando…":"Guardar comanda"}</Button>}
              <Button icon={submitIcon} className="salon-comanda-submit" onClick={()=>submit(!editing)} disabled={actionsBusy||!v.lines.length} loading={actionsBusy&&!savingDraft}>{actionsBusy&&!savingDraft?busyLabel:submitLabel}</Button>
            </div>
          </ModalFooter>
        </aside>
      </div>

      <button type="button" tabIndex={-1} className={"cmd-scrim"+(sheetOpen?" open":"")} onClick={()=>setSheetOpen(false)} aria-hidden="true"/>

      <div className="cmd-bar">
        <div className="cmd-bar-copy"><small>{v.channel==="delivery"?"Total":"Subtotal"}</small><b>{money(total)}</b></div>
        <Button icon="shoppingBag" className="cmd-bar-submit" onClick={()=>setSheetOpen(true)} aria-expanded={sheetOpen} aria-label={`Ver comanda: ${countLabel}, ${money(total)}`} disabled={busy}>Ver comanda{count>0&&<span className="cmd-bar-count">{count}</span>}</Button>
      </div>

      <p className="sr-only" aria-live="polite">{announcement}</p>

      {comboEditor&&(
        <ComboConfigurator
          key={comboEditor.lineKey??comboEditor.comboId}
          comboId={comboEditor.comboId}
          initialSelections={comboEditor.initialSelections}
          editing={Boolean(comboEditor.lineKey)}
          onClose={()=>setComboEditor(null)}
          onConfirm={applyCombo}
        />
      )}
      <ConfirmDialog open={confirmDiscard} title="¿Descartar la comanda?" description={salon&&selTable?<>Los productos de <strong>{tableTitle(selTable.name)}</strong> no se guardarán.</>:"Los productos agregados no se guardarán."} confirmLabel="Descartar" onCancel={()=>setConfirmDiscard(false)} onConfirm={()=>{setConfirmDiscard(false);close(v)}}/>
    </Dialog>
  );
}
