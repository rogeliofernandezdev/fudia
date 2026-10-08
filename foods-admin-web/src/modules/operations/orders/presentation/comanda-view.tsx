"use client";
import "../../styles/salon.css";
import "../../styles/salon-comanda.css";
import {useState,useEffect,useRef} from "react";
import {Button,Dialog,Input,Select,Textarea,Icon} from "@/design-system";
import {CatalogProduct,ComboConfigurator,ComboSelection,ConfiguredCombo,ComandaCatalog} from "../../salon/presentation/comanda-catalog";
import type {Draft,FloorTable,LineDraft} from "../../salon/domain/types";
const money=(value:number)=>value.toFixed(2);

export function ComandaView({initial,mode,allTables,busy,currencySymbol,close,save,notify,channelLabel="Salón",onBack}:{initial:Draft;mode:"create"|"edit"|"append";allTables:FloorTable[];busy:boolean;currencySymbol:string;close:(draft:Draft)=>void;save:(v:Draft,sendToKitchen:boolean)=>void|Promise<void>;notify:(n:{tone:"danger"|"success";title:string;message:string})=>void;channelLabel?:string;onBack?:(draft:Draft)=>void}){
  const adding=mode==="append";
  const editing=mode!=="create";
  const salon=initial.channel==="salon";
  const[v,setV]=useState(initial);
  const[ticketOpen,setTicketOpen]=useState(false);
  const[focusNoteKey,setFocusNoteKey]=useState<string|null>(null);
  const[comboEditor,setComboEditor]=useState<{comboId:string;lineKey?:string;initialSelections:ComboSelection[]}|null>(null);
  const noteRefs=useRef<Record<string,HTMLInputElement|null>>({});
  const submitting=useRef(false);
  const freeTables=allTables.filter(t=>!t.order||t.id===v.tableId);
  const selTable=allTables.find(t=>t.id===v.tableId);
  const tableName=selTable?.name??"";
  const qtyByProduct=v.lines.reduce<Record<string,number>>((acc,line)=>{
    acc[line.productId]=(acc[line.productId]??0)+line.qty;
    return acc;
  },{});
  const patchLine=(i:number,p:Partial<LineDraft>)=>{if(busy||submitting.current)return;setV(prev=>({...prev,lines:prev.lines.map((l,n)=>n===i?{...l,...p}:l)}))};
  const tap=(p:CatalogProduct)=>{
    if(busy||submitting.current)return;
    setTicketOpen(true);
    const existingLine=v.lines.find(l=>l.itemType==="product"&&l.productId===p.id);
    setFocusNoteKey(existingLine?.lineKey??p.id);
    setV(prev=>{
      const i=prev.lines.findIndex(l=>l.itemType==="product"&&l.productId===p.id);
      if(i>=0)return{...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+1}:l)};
      return{...prev,lines:[...prev.lines,{lineKey:p.id,itemType:"product",productId:p.id,name:p.name,qty:1,unitPrice:Number(p.price)||0,note:"",selections:[]}]};
    });
  };
  useEffect(()=>{
    if(!focusNoteKey)return;
    const input=noteRefs.current[focusNoteKey];
    if(!input)return;
    input.focus({preventScroll:true});
    input.scrollIntoView({block:"nearest",behavior:"smooth"});
    setFocusNoteKey(null);
  },[focusNoteKey,v.lines]);
  useEffect(()=>{
    if(salon||(!v.lines.length&&!v.notes))return;
    const prevent=(event:BeforeUnloadEvent)=>{event.preventDefault();event.returnValue=""};
    window.addEventListener("beforeunload",prevent);return()=>window.removeEventListener("beforeunload",prevent);
  },[salon,v.lines.length,v.notes]);
  const untap=(p:CatalogProduct)=>{if(busy||submitting.current)return;setV(prev=>({...prev,lines:prev.lines.map(l=>l.itemType==="product"&&l.productId===p.id?{...l,qty:l.qty-1}:l).filter(l=>l.qty>0)}))};
  const step=(i:number,d:number)=>{if(busy||submitting.current)return;setV(prev=>({...prev,lines:prev.lines.map((l,n)=>n===i?{...l,qty:l.qty+d}:l).filter(l=>l.qty>0)}))};
  const configureCombo=(comboId:string)=>{if(!busy&&!submitting.current)setComboEditor({comboId,initialSelections:[]})};
  const editCombo=(line:LineDraft)=>{if(!busy&&!submitting.current)setComboEditor({comboId:line.productId,lineKey:line.lineKey,initialSelections:line.selections})};
  const applyCombo=(combo:ConfiguredCombo)=>{
    if(busy||submitting.current)return;
    const editingLineKey=comboEditor?.lineKey;
    const generated=typeof crypto!=="undefined"&&typeof crypto.randomUUID==="function"?crypto.randomUUID():`${Date.now()}-${Math.random()}`;
    const lineKey=editingLineKey??`combo-${combo.productId}-${generated}`;
    setV(prev=>{
      if(editingLineKey){
        return{...prev,lines:prev.lines.map(line=>line.lineKey===editingLineKey?{...line,name:combo.name,unitPrice:combo.unitPrice,selections:combo.selections,repriceCombo:true}:line)};
      }
      return{...prev,lines:[...prev.lines,{lineKey,itemType:"combo",productId:combo.productId,name:combo.name,qty:1,unitPrice:combo.unitPrice,note:"",selections:combo.selections}]};
    });
    setComboEditor(null);
    setTicketOpen(true);
    setFocusNoteKey(lineKey);
  };
  const subtotal=v.lines.reduce((a,l)=>a+l.qty*l.unitPrice,0);
  const deliveryFee=v.channel==="delivery"?Number(v.deliveryFee)||0:0;
  const total=subtotal+deliveryFee;
  const count=v.lines.reduce((a,l)=>a+l.qty,0);
  const submit=async(sendToKitchen:boolean)=>{
    if(busy||submitting.current)return;
    if(!v.lines.length){notify({tone:"danger",title:"Comanda vacía",message:"Agrega al menos un producto."});return}
    if(salon&&!v.tableId){notify({tone:"danger",title:"Falta mesa",message:"Selecciona la mesa del pedido."});return}
    submitting.current=true;
    try{await save(salon?v:{...v,tableId:""},salon?(adding||sendToKitchen):true)}finally{submitting.current=false}
  };
  return(
    <Dialog as="div" className="salon-comanda-shell" role="dialog" aria-modal="true" aria-busy={busy} aria-label={adding?"Agregar productos":editing?"Editar comanda":"Nueva comanda"}>
      <header className="salon-comanda-header">
        <div className="salon-comanda-header-main">
          <button type="button" className="salon-comanda-icon-button" disabled={busy} aria-label={onBack?"Editar datos del pedido":"Volver al salón"} onClick={()=>{if(!busy&&!submitting.current)(onBack??close)(v)}}>
            <Icon name="chevronLeft" size={20}/>
          </button>
          <div className="salon-comanda-heading">
            <h2>{adding?"Agregar productos":editing?"Editar comanda":"Nueva comanda"}</h2>
          </div>
        </div>
        <div className="salon-comanda-header-actions">
          <span className="salon-comanda-channel"><i/>{channelLabel}</span>
          <button type="button" className="salon-comanda-icon-button" disabled={busy} aria-label="Cerrar comanda" onClick={()=>{if(!busy&&!submitting.current)close(v)}}>
            <Icon name="close" size={18}/>
          </button>
        </div>
      </header>

      <div className="salon-comanda-main">
        <section className="salon-comanda-catalog" aria-label="Carta del restaurante">
          <div className={"salon-comanda-context"+(!salon?" salon-comanda-remote-context":"")}>
            <span className="salon-comanda-context-icon"><Icon name={salon?"utensils":"truck"} size={18}/></span>
            <div className="salon-comanda-context-copy">
              <span>{salon?"Mesa":channelLabel}</span>
              <strong>{salon?(tableName||"Pendiente de seleccionar"):(v.customerName||channelLabel)}</strong>
              {!salon&&<small className="salon-comanda-contact">{[v.customerPhone,v.channel==="delivery"?v.address:"",v.channel==="delivery"?v.reference:""].filter(Boolean).join(" · ")}</small>}
            </div>
            {salon?<><div className="salon-comanda-context-customer">
              <Input
                disabled={busy}
                autoFocus
                value={v.customerName}
                onChange={e=>setV({...v,customerName:e.target.value})}
                placeholder="Cliente / familia (opcional)"
                aria-label="Nombre del cliente o familia"
              />
            </div>
            <div className="salon-comanda-context-stat">
              <small>Personas</small>
              <span className="salon-comanda-context-stat-value">
                <Icon name="users" size={14}/>
                <b>{selTable?.seats??"—"}</b>
              </span>
            </div></>:onBack&&<Button kind="ghost" icon="edit" disabled={busy} onClick={()=>{if(!busy&&!submitting.current)onBack(v)}}>Editar datos</Button>}
          </div>
          <ComandaCatalog qtyByProduct={qtyByProduct} onPick={tap} onRemove={untap} onConfigureCombo={configureCombo} currencySymbol={currencySymbol} variant="salon"/>
        </section>

        <aside className={"salon-comanda-summary"+(ticketOpen?" open":"")} aria-label="Resumen de la comanda">
          <i className="salon-comanda-summary-grip" aria-hidden="true"/>
          <header className="salon-comanda-summary-head">
            <div className="salon-comanda-summary-title">
              <span>Pedido actual</span>
              <h3>Resumen de la comanda</h3>
            </div>
            <span className="salon-comanda-count-badge">{count} ítem{count===1?"":"s"}</span>
            <button type="button" className="salon-comanda-sheet-close" onClick={()=>setTicketOpen(false)} aria-label="Volver a la carta">
              <Icon name="close" size={16}/>
            </button>
          </header>

          {salon&&!v.tableId&&(
            <div className="salon-comanda-table-card">
              <span className="salon-comanda-table-icon"><Icon name="utensils" size={17}/></span>
              <Select disabled={busy} className="salon-comanda-table-select" value={v.tableId} onChange={e=>setV({...v,tableId:e.target.value})} aria-label="Mesa">
                <option value="">Selecciona una mesa</option>
                {freeTables.map(t=><option key={t.id} value={t.id}>{t.zone?t.zone+" · ":""}{t.name}</option>)}
              </Select>
            </div>
          )}

          <div className="salon-comanda-summary-scroll">
            {!v.lines.length?(
              <div className="salon-comanda-empty">
                <span className="salon-comanda-empty-icon"><Icon name="receipt" size={21}/></span>
                <strong>La comanda está vacía</strong>
                <p>Selecciona platos, menús o combos. Aquí aparecerán cantidades, opciones, notas y el total.</p>
              </div>
            ):(
              <>
                <div className="salon-comanda-lines">
                  {v.lines.map((l,i)=>(
                    <article className={"salon-comanda-line"+(l.itemType==="combo"?" combo":"")} key={l.lineKey}>
                      <div className="salon-comanda-line-main">
                        <div className="salon-comanda-line-copy">
                          <div className="salon-comanda-line-title">
                            <span className="salon-comanda-line-inline-qty">{l.qty}×</span>
                            <strong>{l.name}</strong>
                          </div>
                          {l.qty>1&&<small>{currencySymbol} {money(l.unitPrice)} c/u</small>}
                          {l.itemType==="combo"&&(
                            <div className="salon-comanda-line-selections">
                              {l.selections.map(sel=>
                                <span className="salon-comanda-line-selection" key={sel.groupId+sel.productId}>
                                  <b>{sel.groupName}</b>
                                  <span>{sel.name}{sel.surcharge>0?` (+${currencySymbol} ${money(sel.surcharge)})`:""}</span>
                                </span>
                              )}
                            </div>
                          )}
                        </div>
                        <em className="salon-comanda-line-total">{currencySymbol} {money(l.qty*l.unitPrice)}</em>
                      </div>
                      {l.itemType==="combo"&&(
                        <button type="button" disabled={busy} className="salon-comanda-change-combo" onClick={()=>editCombo(l)}>
                          <Icon name="edit" size={13}/><span>Cambiar elección</span>
                        </button>
                      )}
                      <div className="salon-comanda-line-controls">
                        <div className="salon-comanda-stepper">
                          <button type="button" disabled={busy} onClick={()=>step(i,-1)} aria-label={"Quitar un "+l.name}><Icon name="minus" size={12}/></button>
                          <b>{l.qty}</b>
                          <button type="button" disabled={busy} onClick={()=>step(i,1)} aria-label={"Agregar un "+l.name}><Icon name="plus" size={12}/></button>
                        </div>
                        <Input disabled={busy} maxLength={240} ref={node=>{noteRefs.current[l.lineKey]=node}} className="salon-comanda-line-note" value={l.note} onChange={e=>patchLine(i,{note:e.target.value})} placeholder="Nota del ítem, ej. sin cebolla" aria-label={"Nota para "+l.name}/>
                      </div>
                    </article>
                  ))}
                </div>
                <div className="salon-comanda-general-notes">
                  <label htmlFor="salon-comanda-notes">Notas generales</label>
                  <Textarea disabled={busy} maxLength={500} id="salon-comanda-notes" rows={3} value={v.notes} onChange={e=>setV({...v,notes:e.target.value})} placeholder="Indicaciones para cocina o atención"/>
                </div>
              </>
            )}
          </div>

          <footer className="salon-comanda-summary-foot">
            {v.channel==="delivery"&&<dl className="salon-comanda-charges"><div><dt>Subtotal</dt><dd>{currencySymbol} {money(subtotal)}</dd></div><div><dt>Envío</dt><dd>{currencySymbol} {money(deliveryFee)}</dd></div></dl>}
            <div className="salon-comanda-total">
              <div className="salon-comanda-total-copy"><span>{v.channel==="delivery"?"Total":"Subtotal"}</span></div>
              <strong>{currencySymbol} {money(total)}</strong>
            </div>
            {salon&&!editing&&<Button icon="check" kind="secondary" className="salon-comanda-draft" onClick={()=>submit(false)} disabled={busy||!v.lines.length}>
              {busy?"Guardando…":"Guardar borrador"}
            </Button>}
            <Button icon={editing?"check":"chefHat"} className="salon-comanda-submit" onClick={()=>submit(!editing)} disabled={busy||!v.lines.length}>
              {busy?(editing?"Guardando…":"Enviando…"):(adding?"Agregar y enviar":editing?"Guardar":"Registrar y enviar")}
            </Button>
          </footer>
        </aside>
      </div>

      <button type="button" className={"salon-comanda-summary-overlay"+(ticketOpen?" open":"")} onClick={()=>setTicketOpen(false)} aria-label="Cerrar resumen"/>

      <div className="salon-comanda-mobile-bar">
        <button type="button" className="salon-comanda-mobile-summary" onClick={()=>setTicketOpen(true)} aria-expanded={ticketOpen}>
          <Icon name="receipt" size={17}/>
          <span><small>Comanda · {count} ítem{count===1?"":"s"}</small><b>{currencySymbol} {money(total)}</b></span>
          <Icon name="chevron" size={15}/>
        </button>
        {v.lines.length>0&&(
          <button type="button" className="salon-comanda-mobile-submit" onClick={()=>submit(!editing)} disabled={busy}>
            {busy?"…":<><Icon name={editing?"save":"receipt"} size={14}/><span>{adding?"Agregar":editing?"Guardar":"Registrar"}</span></>}
          </button>
        )}
      </div>

      {comboEditor&&(
        <ComboConfigurator
          key={comboEditor.lineKey??comboEditor.comboId}
          comboId={comboEditor.comboId}
          initialSelections={comboEditor.initialSelections}
          editing={Boolean(comboEditor.lineKey)}
          currencySymbol={currencySymbol}
          onClose={()=>setComboEditor(null)}
          onConfirm={applyCombo}
        />
      )}
    </Dialog>
  );
}

