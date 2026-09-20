"use client";
import {useMemo} from "react";
import {useForm} from "react-hook-form";
import type {FieldPath,FieldPathValue} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {inventoryEntryResolver} from "../domain/inventory-entry-schema";
import type {InventoryCategoryOption,InventoryEntryDraft,InventoryPresentationType,InventoryProductOption} from "../domain/types";

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
const defaultValues:InventoryEntryDraft={
 mode:"existing",inventoryItemId:"",productId:"",categoryId:"",sku:"",name:"",description:"",price:"",
 quantity:"",unit:"und",presentationType:"unit",unitsPerPresentation:"1",minimumStock:"0",note:"",
};

function presentationLabel(type:InventoryPresentationType,factor:string,country?:string|null){
 if(type==="unit")return"Unidad base";
 return `${presentationLabels[type]} x ${formatRegionalNumber(Number(factor),country,{maximumFractionDigits:3})}`;
}

export function InventoryEntryDialog({products,categories,categoryError,currencySymbol,busy,close,save}:{products:InventoryProductOption[];categories:InventoryCategoryOption[];categoryError:string|null;currencySymbol:string;busy:boolean;close:()=>void;save:(draft:InventoryEntryDraft)=>void}){
 const{location}=useSession();
 const{
  register,handleSubmit,watch,setValue,getValues,reset,
  formState:{errors,isSubmitted},
 }=useForm<InventoryEntryDraft>({
  defaultValues,
  resolver:inventoryEntryResolver,
  mode:"onSubmit",
  reValidateMode:"onChange",
 });
 const value=watch();
 const selected=useMemo(()=>products.find(item=>item.id===value.inventoryItemId),[products,value.inventoryItemId]);
 const quantity=Number(value.quantity);
 const factor=value.presentationType==="unit"?1:Number(value.unitsPerPresentation);
 const previewQuantityValid=Number.isFinite(quantity)&&quantity>0&&(value.presentationType==="unit"||Number.isInteger(quantity));
 const previewFactorValid=value.presentationType==="unit"||(Number.isFinite(factor)&&factor>1);
 const stockQuantity=previewQuantityValid&&previewFactorValid?quantity*factor:0;
 const matchedPresentation=(selected?.presentations??[]).find(item=>
  item.presentationType===value.presentationType&&
  Number(item.unitsPerPresentation)===factor
 );
 const presentationChoice=value.presentationType==="unit"?"unit":matchedPresentation?.id??`new-${value.presentationType}`;
 const baseUnitLabel=unitLabels[value.unit]??value.unit;
 const quantityLabel=value.presentationType==="package"?"Cantidad de paquetes":value.presentationType==="box"?"Cantidad de cajas":"Cantidad";
 const hasErrors=Object.keys(errors).length>0;

 function updateField<K extends FieldPath<InventoryEntryDraft>>(field:K,next:FieldPathValue<InventoryEntryDraft,K>){
  setValue(field,next,{shouldDirty:true,shouldValidate:isSubmitted});
 }

 function setMode(mode:InventoryEntryDraft["mode"]){
  const current=getValues();
  reset({
   ...current,
   mode,
   inventoryItemId:"",
   productId:"",
   categoryId:"",
   name:"",
   description:"",
   price:"",
   presentationType:"unit",
   unitsPerPresentation:"1",
  });
 }

 function chooseExistingItem(inventoryItemId:string){
  const item=products.find(option=>option.id===inventoryItemId);
  updateField("inventoryItemId",inventoryItemId);
  updateField("productId",item?.productId??"");
  updateField("unit",item?.unit??getValues("unit"));
  updateField("minimumStock",item?.minimumStock??getValues("minimumStock"));
  updateField("presentationType","unit");
  updateField("unitsPerPresentation","1");
 }

 function chooseSavedPresentation(choice:string){
  if(choice==="unit"){
   updateField("presentationType","unit");
   updateField("unitsPerPresentation","1");
   return;
  }
  const saved=(selected?.presentations??[]).find(item=>item.id===choice);
  if(saved){
   updateField("presentationType",saved.presentationType);
   updateField("unitsPerPresentation",saved.unitsPerPresentation);
   return;
  }
  const type=choice==="new-box"?"box":"package";
  updateField("presentationType",type);
  updateField("unitsPerPresentation","");
 }

 return <div className="modal-backdrop modal-overlay-in" role="presentation">
  <section className="crud-modal inventory-entry-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="inventory-entry-title" aria-busy={busy}>
   <div className="modal-accent"/>
   <header>
    <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
    <div><small>INVENTARIO</small><h2 id="inventory-entry-title">Nueva entrada</h2></div>
    <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
   </header>
   <form onSubmit={handleSubmit(save)} noValidate>
    <div className="inventory-entry-body">
     <section className="inventory-entry-choice" aria-label="Tipo de artículo">
      <button type="button" className={value.mode==="existing"?"active":""} onClick={()=>setMode("existing")}>
       <span><Icon name="search" size={18}/></span><b>Artículo existente<small>Registrar más stock de un producto o insumo ya creado.</small></b>
      </button>
      <button type="button" className={value.mode==="new_product"?"active":""} onClick={()=>setMode("new_product")}>
       <span><Icon name="plus" size={18}/></span><b>Nuevo producto vendible<small>Gaseosa, agua u otra mercadería que también se vende.</small></b>
      </button>
      <button type="button" className={value.mode==="new_ingredient"?"active":""} onClick={()=>setMode("new_ingredient")}>
       <span><Icon name="stock" size={18}/></span><b>Nuevo insumo<small>Carne, papa, zanahoria, aceite u otro ingrediente interno.</small></b>
      </button>
     </section>

     {value.mode==="existing"?<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="box" size={17}/></span><div><b>Qué ingresó</b><small>La entrada se sumará al saldo actual del artículo en este local.</small></div></div>
      <label>Artículo
       <Select autoFocus value={value.inventoryItemId} onChange={event=>chooseExistingItem(event.target.value)} aria-invalid={Boolean(errors.inventoryItemId)}>
        <option value="">{products.length?"Selecciona un producto o insumo":"No hay artículos de inventario"}</option>
        {products.map(item=><option value={item.id} key={item.id}>{item.name}{item.kind==="ingredient"?" · Insumo":""}</option>)}
       </Select>
       {errors.inventoryItemId?.message&&<small className="wizard-field-error">{errors.inventoryItemId.message}</small>}
      </label>
      {selected&&<div className="inventory-entry-product-note"><Icon name="check" size={15}/><span><b>{selected.name}</b><small>{selected.kind==="ingredient"?"Insumo":"Producto vendible"} · Stock controlado en {unitLabels[selected.unit]??selected.unit}.</small></span></div>}
     </section>:value.mode==="new_product"?<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="plus" size={17}/></span><div><b>Crear producto vendible</b><small>Se creará como mercadería vendible con Inventario físico, no como plato preparado.</small></div></div>
      <div className="form-grid">
       <label className="span-2">Nombre del producto<Input autoFocus maxLength={160} {...register("name")} placeholder="Ej. Coca-Cola 500 ml" aria-invalid={Boolean(errors.name)}/>{errors.name?.message&&<small className="wizard-field-error">{errors.name.message}</small>}</label>
       <label>Categoría comercial
        <Select {...register("categoryId")} disabled={Boolean(categoryError)||categories.length===0} aria-invalid={Boolean(errors.categoryId)||Boolean(categoryError)}>
         <option value="">{categoryError?"No pudimos cargar las categorías":categories.length?"Selecciona una categoría":"No hay categorías activas"}</option>
         {categories.map(category=><option value={category.id} key={category.id}>{category.name}</option>)}
        </Select>
        {categoryError?<small className="wizard-field-error">{categoryError}</small>:categories.length===0?<small className="wizard-field-error">Crea una categoría activa en Carta y productos antes de registrar este producto.</small>:errors.categoryId?.message&&<small className="wizard-field-error">{errors.categoryId.message}</small>}
       </label>
       <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div>{errors.price?.message&&<small className="wizard-field-error">{errors.price.message}</small>}</label>
       <label className="span-2">Descripción opcional<Textarea maxLength={1000} {...register("description")} placeholder="Presentación o detalle comercial"/></label>
      </div>
     </section>:<section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="stock" size={17}/></span><div><b>Crear insumo</b><small>Será inventario interno para compras y recetas; no tendrá precio de venta.</small></div></div>
      <div className="form-grid">
       <label className="span-2">Nombre del insumo<Input autoFocus maxLength={160} {...register("name")} placeholder="Ej. Carne de res" aria-invalid={Boolean(errors.name)}/>{errors.name?.message&&<small className="wizard-field-error">{errors.name.message}</small>}</label>
      </div>
     </section>}

     <section className="inventory-entry-section">
      <div className="inventory-entry-section-title"><span><Icon name="stock" size={17}/></span><div><b>Cuánto ingresó</b><small>El saldo siempre se guarda en la unidad base; paquetes y cajas se convierten automáticamente.</small></div></div>
      <div className="form-grid">
       <label>Unidad base de stock
        <Select {...register("unit")} disabled={value.mode==="existing"&&Boolean(selected?.unit)} aria-invalid={Boolean(errors.unit)}>
         <option value="und">Unidad</option><option value="botella">Botella</option><option value="lata">Lata</option><option value="caja">Caja</option><option value="kg">Kilogramo</option><option value="l">Litro</option>
        </Select>
        {errors.unit?.message&&<small className="wizard-field-error">{errors.unit.message}</small>}
       </label>
       {value.mode==="existing"&&selected?<label>Presentación de ingreso
        <Select value={presentationChoice} onChange={event=>chooseSavedPresentation(event.target.value)} aria-invalid={Boolean(errors.presentationType)||Boolean(errors.unitsPerPresentation)}>
         <option value="unit">Unidad base</option>
         {(selected.presentations??[]).filter(item=>item.presentationType!=="unit").map(item=><option value={item.id} key={item.id}>{presentationLabel(item.presentationType,item.unitsPerPresentation,location?.country)}</option>)}
         <option value="new-package">+ Nuevo paquete</option>
         <option value="new-box">+ Nueva caja</option>
        </Select>
       </label>:<label>Presentación de ingreso
        <Select value={value.presentationType} onChange={event=>{const type=event.target.value as InventoryPresentationType;updateField("presentationType",type);updateField("unitsPerPresentation",type==="unit"?"1":"")}} aria-invalid={Boolean(errors.presentationType)||Boolean(errors.unitsPerPresentation)}>
         <option value="unit">Unidad base</option><option value="package">Paquete</option><option value="box">Caja</option>
        </Select>
       </label>}
       <label>{quantityLabel}<Input type="number" min={value.presentationType==="unit"?"0.001":"1"} step={value.presentationType==="unit"?"0.001":"1"} inputMode="decimal" {...register("quantity")} placeholder="0" aria-invalid={Boolean(errors.quantity)}/>{errors.quantity?.message&&<small className="wizard-field-error">{errors.quantity.message}</small>}</label>
       {value.presentationType!=="unit"&&<label>Unidades por {value.presentationType==="box"?"caja":"paquete"}<Input type="number" min="1.001" step="0.001" inputMode="decimal" {...register("unitsPerPresentation")} placeholder="Ej. 12" aria-invalid={Boolean(errors.unitsPerPresentation)}/>{errors.unitsPerPresentation?.message&&<small className="wizard-field-error">{errors.unitsPerPresentation.message}</small>}</label>}
       <label>Stock mínimo ({baseUnitLabel})<Input type="number" min="0" step="0.001" inputMode="decimal" {...register("minimumStock")} aria-invalid={Boolean(errors.minimumStock)}/>{errors.minimumStock?.message&&<small className="wizard-field-error">{errors.minimumStock.message}</small>}</label>
       <label className="span-2">Nota opcional<Textarea maxLength={240} {...register("note")} placeholder="Ej. Ingreso de proveedor, lote o referencia"/>{errors.note?.message&&<small className="wizard-field-error">{errors.note.message}</small>}</label>
      </div>
      {value.presentationType!=="unit"&&previewQuantityValid&&previewFactorValid&&<div className="inventory-entry-product-note"><Icon name="check" size={15}/><span><b>{formatRegionalNumber(quantity,location?.country,{maximumFractionDigits:3})} {value.presentationType==="box"?"cajas":"paquetes"} × {formatRegionalNumber(factor,location?.country,{maximumFractionDigits:3})}</b><small>Se sumarán {formatRegionalNumber(stockQuantity,location?.country,{maximumFractionDigits:3})} {baseUnitLabel} al stock.</small></span></div>}
     </section>

     {value.mode!=="existing"&&<div className="inventory-entry-atomic-note"><Icon name="lock" size={16}/><p><b>Una sola operación</b>{value.mode==="new_ingredient"?"Insumo":"Producto"}, presentación, saldo, entrada y Kárdex se guardan juntos. Si algo falla, no se crea nada parcialmente.</p></div>}
     {isSubmitted&&hasErrors&&<div className="inventory-entry-validation" role="alert"><Icon name="alert" size={15}/><span>Revisa los campos marcados antes de guardar.</span></div>}
    </div>
    <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></footer>
   </form>
   {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
  </section>
 </div>;
}
