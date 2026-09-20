"use client";
import {useMemo,useState} from "react";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import type {InventoryEntryDraft,InventoryProductOption} from "../domain/types";

export function InventoryEntryDialog({products,currencySymbol,busy,close,save}:{products:InventoryProductOption[];currencySymbol:string;busy:boolean;close:()=>void;save:(draft:InventoryEntryDraft)=>void}){
 const[value,setValue]=useState<InventoryEntryDraft>({
  mode:"existing",productId:"",sku:"",name:"",description:"",price:"",
  quantity:"",unit:"und",minimumStock:"0",note:"",
 });
 const[attempted,setAttempted]=useState(false);
 const selected=useMemo(()=>products.find(product=>product.id===value.productId),[products,value.productId]);
 const quantity=Number(value.quantity);
 const minimum=Number(value.minimumStock||0);
 const priceValid=/^[0-9]+([.][0-9]{1,2})?$/.test(value.price);
 const validQuantity=Number.isFinite(quantity)&&quantity>0;
 const validMinimum=Number.isFinite(minimum)&&minimum>=0;
 const valid=value.mode==="existing"
  ?Boolean(value.productId)&&validQuantity&&validMinimum
  :Boolean(value.name.trim())&&priceValid&&validQuantity&&validMinimum;

 function submit(event:React.FormEvent){
  event.preventDefault();
  setAttempted(true);
  if(valid)save(value);
 }

 return <div className="modal-backdrop modal-overlay-in" role="presentation">
  <section className="crud-modal inventory-entry-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-entry-title" aria-busy={busy}>
   <div className="modal-accent"/>
   <header>
    <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
    <div><small>INVENTARIO</small><h2 id="inventory-entry-title">Nueva entrada</h2></div>
    <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
   </header>
   <form onSubmit={submit} noValidate>
    <div className="inventory-entry-body">
     <section className="inventory-entry-choice" aria-label="Origen del producto">
      <button type="button" className={value.mode==="existing"?"active":""} onClick={()=>setValue(current=>({...current,mode:"existing"}))}>
       <span><Icon name="search" size={18}/></span><b>Producto existente<small>Registrar más stock del mismo producto.</small></b>
      </button>
      <button type="button" className={value.mode==="new"?"active":""} onClick={()=>setValue(current=>({...current,mode:"new",productId:""}))}>
       <span><Icon name="plus" size={18}/></span><b>Nuevo producto<small>Crearlo y registrar su primera entrada sin salir de Inventario.</small></b>
      </button>
     </section>

     {value.mode==="existing"?<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="box" size={17}/></span><div><b>Qué producto ingresó</b><small>La entrada se sumará a su saldo actual en este local.</small></div></div>
      <label>Producto
       <Select autoFocus value={value.productId} onChange={event=>{const product=products.find(item=>item.id===event.target.value);setValue(current=>({...current,productId:event.target.value,unit:product?.unit??current.unit,minimumStock:product?.minimumStock??current.minimumStock}))}} aria-invalid={attempted&&!value.productId}>
        <option value="">Selecciona un producto</option>
        {products.map(product=><option value={product.id} key={product.id}>{product.name} · {product.sku}</option>)}
       </Select>
       {attempted&&!value.productId&&<small className="wizard-field-error">Selecciona el producto que estás recibiendo.</small>}
      </label>
      {selected&&<div className="inventory-entry-product-note"><Icon name="check" size={15}/><span><b>{selected.name}</b><small>{selected.quantityControl==="inventory"?"Ya controla existencia física.":"Esta primera entrada activará su control por Inventario."}</small></span></div>}
     </section>:<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="plus" size={17}/></span><div><b>Crear producto</b><small>Se creará en el catálogo único de Productos y quedará vinculado al inventario.</small></div></div>
      <div className="form-grid">
       <label className="span-2">Nombre del producto<Input autoFocus maxLength={160} value={value.name} onChange={event=>setValue(current=>({...current,name:event.target.value}))} placeholder="Ej. Coca-Cola 500 ml" aria-invalid={attempted&&!value.name.trim()}/>{attempted&&!value.name.trim()&&<small className="wizard-field-error">Ingresa el nombre del producto.</small>}</label>
       <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" value={value.price} onChange={event=>setValue(current=>({...current,price:event.target.value}))} placeholder="0.00" aria-invalid={attempted&&!priceValid}/></div>{attempted&&!priceValid&&<small className="wizard-field-error">Ingresa un precio válido.</small>}</label>
       <label>SKU opcional<Input maxLength={40} value={value.sku} onChange={event=>setValue(current=>({...current,sku:event.target.value}))} placeholder="Se genera si lo dejas vacío"/></label>
       <label className="span-2">Descripción opcional<Textarea value={value.description} onChange={event=>setValue(current=>({...current,description:event.target.value}))} placeholder="Presentación o detalle comercial"/></label>
      </div>
     </section>}

     <section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="stock" size={17}/></span><div><b>Cuánto ingresó</b><small>Este movimiento quedará registrado en Kárdex.</small></div></div>
      <div className="form-grid">
       <label>Cantidad<Input type="number" min="0.001" step="0.001" inputMode="decimal" value={value.quantity} onChange={event=>setValue(current=>({...current,quantity:event.target.value}))} placeholder="0" aria-invalid={attempted&&!validQuantity}/>{attempted&&!validQuantity&&<small className="wizard-field-error">Ingresa una cantidad mayor que cero.</small>}</label>
       <label>Unidad<Select value={value.unit} onChange={event=>setValue(current=>({...current,unit:event.target.value}))}><option value="und">Unidad</option><option value="botella">Botella</option><option value="lata">Lata</option><option value="caja">Caja</option><option value="kg">Kilogramo</option><option value="l">Litro</option></Select></label>
       <label>Stock mínimo<Input type="number" min="0" step="0.001" inputMode="decimal" value={value.minimumStock} onChange={event=>setValue(current=>({...current,minimumStock:event.target.value}))} aria-invalid={attempted&&!validMinimum}/>{attempted&&!validMinimum&&<small className="wizard-field-error">El mínimo no puede ser negativo.</small>}</label>
       <label className="span-2">Nota opcional<Textarea maxLength={240} value={value.note} onChange={event=>setValue(current=>({...current,note:event.target.value}))} placeholder="Ej. Ingreso de proveedor, lote o referencia"/></label>
      </div>
     </section>

     {value.mode==="new"&&<div className="inventory-entry-atomic-note"><Icon name="lock" size={16}/><p><b>Una sola operación</b>Producto, vínculo de inventario, saldo, entrada y Kárdex se guardan juntos. Si algo falla, no se crea nada parcialmente.</p></div>}
    </div>
    <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button icon="plus" disabled={busy}>{busy?"Registrando…":"Registrar entrada"}</Button></footer>
   </form>
   {busy&&<div className="modal-busy" role="status"><i/><span>Registrando entrada…</span></div>}
  </section>
 </div>;
}
