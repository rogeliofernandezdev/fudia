"use client";
import {useMemo,useState} from "react";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {InventoryEntryDraft,InventoryPresentationType,InventoryProductOption} from "../domain/types";

const unitLabels:Record<string,string>={
 und:"unidades",
 botella:"botellas",
 lata:"latas",
 caja:"cajas",
 kg:"kg",
 l:"L",
};
const presentationLabels:Record<InventoryPresentationType,string>={
 unit:"Unidad",
 package:"Paquete",
 box:"Caja",
};
function presentationLabel(type:InventoryPresentationType,factor:string,country?:string|null){
 if(type==="unit")return"Unidad base";
 return `${presentationLabels[type]} x ${formatRegionalNumber(Number(factor),country,{maximumFractionDigits:3})}`;
}

export function InventoryEntryDialog({products,currencySymbol,busy,close,save}:{products:InventoryProductOption[];currencySymbol:string;busy:boolean;close:()=>void;save:(draft:InventoryEntryDraft)=>void}){
 const{location}=useSession();
 const[value,setValue]=useState<InventoryEntryDraft>({
  mode:"existing",productId:"",sku:"",name:"",description:"",price:"",
  quantity:"",unit:"und",presentationType:"unit",unitsPerPresentation:"1",minimumStock:"0",note:"",
 });
 const[attempted,setAttempted]=useState(false);
 const selected=useMemo(()=>products.find(product=>product.id===value.productId),[products,value.productId]);
 const quantity=Number(value.quantity);
 const factor=value.presentationType==="unit"?1:Number(value.unitsPerPresentation);
 const minimum=Number(value.minimumStock||0);
 const priceValid=/^[0-9]+([.][0-9]{1,2})?$/.test(value.price);
 const validQuantity=Number.isFinite(quantity)&&quantity>0&&(value.presentationType==="unit"||Number.isInteger(quantity));
 const validFactor=value.presentationType==="unit"||(Number.isFinite(factor)&&factor>1);
 const validMinimum=Number.isFinite(minimum)&&minimum>=0;
 const stockQuantity=validQuantity&&validFactor?quantity*factor:0;
 const valid=value.mode==="existing"
  ?Boolean(value.productId)&&validQuantity&&validFactor&&validMinimum
  :Boolean(value.name.trim())&&priceValid&&validQuantity&&validFactor&&validMinimum;
 const matchedPresentation=(selected?.presentations??[]).find(item=>
  item.presentationType===value.presentationType&&
  Number(item.unitsPerPresentation)===factor
 );
 const presentationChoice=value.presentationType==="unit"?"unit":matchedPresentation?.id??`new-${value.presentationType}`;
 const baseUnitLabel=unitLabels[value.unit]??value.unit;
 const quantityLabel=value.presentationType==="package"?"Cantidad de paquetes":value.presentationType==="box"?"Cantidad de cajas":"Cantidad";

 function submit(event:React.FormEvent){
  event.preventDefault();
  setAttempted(true);
  if(valid)save(value);
 }

 function chooseExistingProduct(productId:string){
  const product=products.find(item=>item.id===productId);
  setValue(current=>({
   ...current,
   productId,
   unit:product?.unit??current.unit,
   minimumStock:product?.minimumStock??current.minimumStock,
   presentationType:"unit",
   unitsPerPresentation:"1",
  }));
 }

 function chooseSavedPresentation(choice:string){
  if(choice==="unit"){
   setValue(current=>({...current,presentationType:"unit",unitsPerPresentation:"1"}));
   return;
  }
  const saved=(selected?.presentations??[]).find(item=>item.id===choice);
  if(saved){
   setValue(current=>({...current,presentationType:saved.presentationType,unitsPerPresentation:saved.unitsPerPresentation}));
   return;
  }
  const type=choice==="new-box"?"box":"package";
  setValue(current=>({...current,presentationType:type,unitsPerPresentation:""}));
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
       <span><Icon name="search" size={18}/></span><b>Producto existente<small>Registrar más stock de un producto con Inventario físico.</small></b>
      </button>
      <button type="button" className={value.mode==="new"?"active":""} onClick={()=>setValue(current=>({...current,mode:"new",productId:"",presentationType:"unit",unitsPerPresentation:"1"}))}>
       <span><Icon name="plus" size={18}/></span><b>Nuevo producto físico<small>Crearlo con Inventario físico y registrar su primera entrada.</small></b>
      </button>
     </section>

     {value.mode==="existing"?<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="box" size={17}/></span><div><b>Qué producto ingresó</b><small>La entrada se sumará a su saldo actual en este local.</small></div></div>
      <label>Producto
       <Select autoFocus value={value.productId} onChange={event=>chooseExistingProduct(event.target.value)} aria-invalid={attempted&&!value.productId}>
        <option value="">{products.length?"Selecciona un producto de inventario":"No hay productos con Inventario físico"}</option>
        {products.map(product=><option value={product.id} key={product.id}>{product.name}</option>)}
       </Select>
       {attempted&&!value.productId&&<small className="wizard-field-error">Selecciona el producto que estás recibiendo.</small>}
      </label>
      {selected&&<div className="inventory-entry-product-note"><Icon name="check" size={15}/><span><b>{selected.name}</b><small>Stock controlado en {unitLabels[selected.unit??"und"]??selected.unit??"unidades"}.</small></span></div>}
     </section>:<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="plus" size={17}/></span><div><b>Crear producto</b><small>Se creará en el catálogo único de Productos y quedará configurado como Inventario físico.</small></div></div>
      <div className="form-grid">
       <label className="span-2">Nombre del producto<Input autoFocus maxLength={160} value={value.name} onChange={event=>setValue(current=>({...current,name:event.target.value}))} placeholder="Ej. Coca-Cola 500 ml" aria-invalid={attempted&&!value.name.trim()}/>{attempted&&!value.name.trim()&&<small className="wizard-field-error">Ingresa el nombre del producto.</small>}</label>
       <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" value={value.price} onChange={event=>setValue(current=>({...current,price:event.target.value}))} placeholder="0.00" aria-invalid={attempted&&!priceValid}/></div>{attempted&&!priceValid&&<small className="wizard-field-error">Ingresa un precio válido.</small>}</label>
       <label className="span-2">Descripción opcional<Textarea value={value.description} onChange={event=>setValue(current=>({...current,description:event.target.value}))} placeholder="Presentación o detalle comercial"/></label>
      </div>
     </section>}

     <section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="stock" size={17}/></span><div><b>Cuánto ingresó</b><small>El saldo siempre se guarda en la unidad base; paquetes y cajas se convierten automáticamente.</small></div></div>
      <div className="form-grid">
       <label>Unidad base de stock
        <Select value={value.unit} disabled={value.mode==="existing"&&Boolean(selected?.unit)} onChange={event=>setValue(current=>({...current,unit:event.target.value}))}>
         <option value="und">Unidad</option><option value="botella">Botella</option><option value="lata">Lata</option><option value="caja">Caja</option><option value="kg">Kilogramo</option><option value="l">Litro</option>
        </Select>
       </label>
       {value.mode==="existing"&&selected?<label>Presentación de ingreso
        <Select value={presentationChoice} onChange={event=>chooseSavedPresentation(event.target.value)}>
         <option value="unit">Unidad base</option>
         {(selected.presentations??[]).filter(item=>item.presentationType!=="unit").map(item=><option value={item.id} key={item.id}>{presentationLabel(item.presentationType,item.unitsPerPresentation,location?.country)}</option>)}
         <option value="new-package">+ Nuevo paquete</option>
         <option value="new-box">+ Nueva caja</option>
        </Select>
       </label>:<label>Presentación de ingreso
        <Select value={value.presentationType} onChange={event=>{const type=event.target.value as InventoryPresentationType;setValue(current=>({...current,presentationType:type,unitsPerPresentation:type==="unit"?"1":""}))}}>
         <option value="unit">Unidad base</option><option value="package">Paquete</option><option value="box">Caja</option>
        </Select>
       </label>}
       <label>{quantityLabel}<Input type="number" min={value.presentationType==="unit"?"0.001":"1"} step={value.presentationType==="unit"?"0.001":"1"} inputMode="decimal" value={value.quantity} onChange={event=>setValue(current=>({...current,quantity:event.target.value}))} placeholder="0" aria-invalid={attempted&&!validQuantity}/>{attempted&&!validQuantity&&<small className="wizard-field-error">{value.presentationType==="unit"?"Ingresa una cantidad mayor que cero.":"Ingresa una cantidad entera de paquetes o cajas."}</small>}</label>
       {value.presentationType!=="unit"&&<label>Unidades por {value.presentationType==="box"?"caja":"paquete"}<Input type="number" min="1.001" step="0.001" inputMode="decimal" value={value.unitsPerPresentation} onChange={event=>setValue(current=>({...current,unitsPerPresentation:event.target.value}))} placeholder="Ej. 12" aria-invalid={attempted&&!validFactor}/>{attempted&&!validFactor&&<small className="wizard-field-error">Debe contener más de una unidad base.</small>}</label>}
       <label>Stock mínimo ({baseUnitLabel})<Input type="number" min="0" step="0.001" inputMode="decimal" value={value.minimumStock} onChange={event=>setValue(current=>({...current,minimumStock:event.target.value}))} aria-invalid={attempted&&!validMinimum}/>{attempted&&!validMinimum&&<small className="wizard-field-error">El mínimo no puede ser negativo.</small>}</label>
       <label className="span-2">Nota opcional<Textarea maxLength={240} value={value.note} onChange={event=>setValue(current=>({...current,note:event.target.value}))} placeholder="Ej. Ingreso de proveedor, lote o referencia"/></label>
      </div>
      {value.presentationType!=="unit"&&validQuantity&&validFactor&&<div className="inventory-entry-product-note"><Icon name="check" size={15}/><span><b>{formatRegionalNumber(quantity,location?.country,{maximumFractionDigits:3})} {value.presentationType==="box"?"cajas":"paquetes"} × {formatRegionalNumber(factor,location?.country,{maximumFractionDigits:3})}</b><small>Se sumarán {formatRegionalNumber(stockQuantity,location?.country,{maximumFractionDigits:3})} {baseUnitLabel} al stock.</small></span></div>}
     </section>

     {value.mode==="new"&&<div className="inventory-entry-atomic-note"><Icon name="lock" size={16}/><p><b>Una sola operación</b>Producto, presentación, saldo, entrada y Kárdex se guardan juntos. Si algo falla, no se crea nada parcialmente.</p></div>}
    </div>
    <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></footer>
   </form>
   {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
  </section>
 </div>;
}
