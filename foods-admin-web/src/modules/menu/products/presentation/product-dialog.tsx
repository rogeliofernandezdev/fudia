"use client";
import {Dialog} from "@/design-system/dialog";
import "./product-wizard.css";
import {useRef,useState} from "react";
import {Controller,useForm} from "react-hook-form";
import {FormField,InlineNotice,Input,Select,ModalHeader,ModalForm,ModalBody,ModalFooter} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {RichTextEditor} from "@/design-system/rich-text-editor";
import {ProductImageField} from "./product-image-field";
import {AllergenField,type AllergenOption} from "./allergen-field";
import type {Category,ProductDraft,ServiceDestinationOption} from "../domain/types";
import {productResolver} from "../domain/product-schema";


export function ProductDialog({draft,categories,destinations,loadAllergens,createAllergen,currencySymbol,busy,close,save,imageScope=""}:{imageScope?:string;draft:ProductDraft;categories:Category[];destinations:ServiceDestinationOption[];loadAllergens:(q:string)=>Promise<AllergenOption[]>;createAllergen?:(name:string)=>Promise<string>;currencySymbol:string;busy:boolean;close:()=>void;save:(d:ProductDraft,f:File|null)=>Promise<void>}){
 const[showExtras,setShowExtras]=useState(Boolean(draft.prepMinutes)||draft.allergens.length>0||draft.featured);const[pendingFile,setPendingFile]=useState<File|null>(null);const[imageValidating,setImageValidating]=useState(false);const imageInvalid=useRef(false);
 const{register,control,watch,setValue,handleSubmit,formState:{errors}}=useForm<ProductDraft>({defaultValues:{...draft,initialPortionQuantity:draft.initialPortionQuantity??"",serviceDestination:draft.serviceDestination??"kitchen"},resolver:productResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const submitting=useRef(false);
 const value=watch();
 const inventoryLocked=Boolean(draft.id)&&draft.quantityControl==="inventory";
 // Lock before async validation so rapid activations cannot queue duplicate requests.
 async function submit(){
  if(busy||submitting.current||imageValidating||imageInvalid.current)return;
  submitting.current=true;
  try{await handleSubmit(values=>save(values,pendingFile))()}
  finally{submitting.current=false}
 }
 return <div className="modal-backdrop modal-overlay-in" role="presentation"><Dialog onResponseClose={close} className="crud-modal product-form modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="product-title" aria-busy={busy}><ModalHeader title="Información del producto" titleId="product-title" icon="box" subtitle={draft.id?"EDITAR PRODUCTO":"NUEVO PRODUCTO"} onClose={close} busy={busy} closeLabel="Cerrar"/><ModalForm onSubmit={event=>event.preventDefault()} noValidate><ModalBody className="product-form-body"><div className="form-grid">
  <div className="product-identity span-2">
  <div className="form-grid product-identity-fields">
  <FormField className="span-2" label="Nombre del producto" error={errors.name?.message}><Input autoFocus maxLength={160} {...register("name")} placeholder="Ej. Lomo saltado"/></FormField>
  <FormField label="Precio de venta" error={errors.price?.message}><div className={"money-input"+(errors.price?" has-error":"")}><span>{currencySymbol}</span><input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div></FormField>
  <FormField label="Categoría"><Select {...register("categoryId")}><option value="">Sin categoría</option>{categories.filter(c=>(c.active||c.id===value.categoryId)).map(c=><option value={c.id} key={c.id}>{c.name}</option>)}</Select></FormField>
  <FormField className="span-2" label="Destino de atención" error={errors.serviceDestination?.message}><Select disabled={busy||!destinations.length} {...register("serviceDestination")}>{destinations.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}</Select></FormField>
  {value.serviceDestination==="direct"&&(draft.hasRecipe||Number(value.prepMinutes)>0)&&<InlineNotice className="span-2" tone="warning" title="Este producto parece prepararse">{draft.hasRecipe?"Tiene receta":"Tiene tiempo de preparación"}. Con «Sin preparación» no llegará a Cocina ni a Barra: entra al pedido como Listo.</InlineNotice>}
  </div>
  <ProductImageField draft={draft} name={value.name} busy={busy} scopeKey={imageScope} onSelect={setPendingFile} onRemove={()=>{setValue("imageUrl",null);setValue("removeImage",true)}} onValidation={(pending,error)=>{setImageValidating(pending);imageInvalid.current=pending||Boolean(error)}}/>
  </div>
  <fieldset className="product-stock-mode span-2"><legend>Control de cantidad <small>{inventoryLocked?"Gestionado desde Inventario":"Define de dónde sale la existencia"}</small></legend>{inventoryLocked?<div className="product-stock-locked"><span><Icon name="stock" size={16}/></span><b>Inventario físico<small>Las entradas y existencias se administran desde Inventario. Aquí solo editas la información comercial.</small></b></div>:<div className="product-stock-options is-two"><label><input className="product-radio" type="radio" value="none" {...register("quantityControl")}/><b>Sin control<small>Se vende sin contar unidades.</small></b></label><label><input className="product-radio" type="radio" value="portions" {...register("quantityControl")}/><b>Cupo del día<small>{draft.id?"Platos o bebidas con cantidad limitada. Se ajusta desde Disponibilidad.":"Platos o bebidas con cantidad limitada, sin usar Inventario."}</small></b></label></div>}</fieldset>
  {!draft.id&&value.quantityControl==="portions"&&<FormField className="span-2" label="Cantidad disponible hoy" error={errors.initialPortionQuantity?.message}><Input type="number" min="1" max="2147483647" step="1" inputMode="numeric" disabled={busy} {...register("initialPortionQuantity")} placeholder="Ej. 15"/></FormField>}
  <FormField as="div" className="product-description span-2" label="Descripción"><Controller control={control} name="description" render={({field})=><RichTextEditor value={field.value} onChange={field.onChange} placeholder="Ingredientes, presentación o información útil para el equipo"/>}/></FormField>
 </div><button type="button" className={showExtras?"product-extras-toggle open":"product-extras-toggle"} onClick={()=>setShowExtras(!showExtras)} aria-expanded={showExtras}><Icon name="chevron" size={16}/><b>Información adicional</b><small>Opcional</small></button>{showExtras&&<div className="form-grid product-extras"><FormField label="Tiempo de preparación (min)" error={errors.prepMinutes?.message}><Input type="number" min="0" inputMode="numeric" {...register("prepMinutes")} placeholder="Ej. 15"/></FormField><FormField label="Precio de costo" optional error={errors.costPrice?.message}><div className={"money-input"+(errors.costPrice?" has-error":"")}><span>{currencySymbol}</span><input inputMode="decimal" {...register("costPrice")} placeholder="0.00" aria-invalid={Boolean(errors.costPrice)}/></div></FormField><Controller control={control} name="allergens" render={({field})=><AllergenField value={field.value} onChange={field.onChange} loadAllergens={loadAllergens} createAllergen={createAllergen} disabled={busy}/>}/><label className="switch-row compact span-2"><input type="checkbox" {...register("featured")}/><span/><b>Recomendar en la carta digital</b></label></div>}</ModalBody><ModalFooter><button type="button" className="button ghost" onClick={close} disabled={busy}>Cancelar</button><button type="button" className="button primary" onClick={()=>void submit()} disabled={busy||imageValidating}>{busy?"Guardando…":imageValidating?"Validando imagen…":"Guardar"}</button></ModalFooter></ModalForm></Dialog></div>
}
