"use client";
import {Dialog} from "@/design-system/dialog";
import "./product-wizard.css";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Controller,useForm} from "react-hook-form";
import {Button,FormDisclosure,FormField,InlineNotice,Input,Select,ModalHeader,ModalForm,ModalBody,ModalFooter,Skeleton} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {RichTextEditor} from "@/design-system/rich-text-editor";
import {ProductImageField} from "./product-image-field";
import {AllergenField,type AllergenOption} from "./allergen-field";
import {ProductOptionsField} from "./product-options-field";
import {getProductModifiers} from "../infrastructure/products-api";
import {groupsFromConfig,modifiersPayload,validateModifierGroups,type ModifierErrors,type ModifierGroupDraft} from "../domain/modifiers";
import type {Category,ProductDraft,ServiceDestinationOption} from "../domain/types";
import {productResolver} from "../domain/product-schema";


export function ProductDialog({draft,categories,destinations,loadAllergens,createAllergen,currencySymbol,busy,close,save,imageScope=""}:{imageScope?:string;draft:ProductDraft;categories:Category[];destinations:ServiceDestinationOption[];loadAllergens:(q:string)=>Promise<AllergenOption[]>;createAllergen?:(name:string)=>Promise<string>;currencySymbol:string;busy:boolean;close:()=>void;save:(d:ProductDraft,f:File|null,modifiers:ModifierGroupDraft[]|null)=>Promise<void>}){
 const[showExtras,setShowExtras]=useState(Boolean(draft.prepMinutes)||draft.allergens.length>0||draft.featured);const[pendingFile,setPendingFile]=useState<File|null>(null);const[imageValidating,setImageValidating]=useState(false);const imageInvalid=useRef(false);
 // Every product needs a category: inactive ones only stay listed for the product that already uses them.
 const categoryOptions=categories.filter(c=>c.active||c.id===draft.categoryId);
 const onlyCategory=!draft.categoryId&&categoryOptions.length===1?categoryOptions[0].id:"";
 const{register,control,watch,setValue,handleSubmit,formState:{errors}}=useForm<ProductDraft>({defaultValues:{...draft,categoryId:draft.categoryId||onlyCategory,initialPortionQuantity:draft.initialPortionQuantity??"",serviceDestination:draft.serviceDestination??"kitchen"},resolver:productResolver,mode:"onSubmit",reValidateMode:"onChange"});
 const submitting=useRef(false);
 const value=watch();
 // Opciones al pedir: se cargan al editar y solo se envían si cambiaron.
 const modifiers=useQuery({queryKey:["product-modifiers",draft.id],queryFn:()=>getProductModifiers(draft.id as string),enabled:Boolean(draft.id),staleTime:0});
 const[groups,setGroups]=useState<ModifierGroupDraft[]>([]);
 const[loadedGroups,setLoadedGroups]=useState<ModifierGroupDraft[]>([]);
 const[groupErrors,setGroupErrors]=useState<ModifierErrors>({});
 const[showOptions,setShowOptions]=useState(false);
 const loadedOnce=useRef(false);
 useEffect(()=>{if(!modifiers.data||loadedOnce.current)return;loadedOnce.current=true;const next=groupsFromConfig(modifiers.data);setGroups(next);setLoadedGroups(next)},[modifiers.data]);
 const optionsReady=!draft.id||modifiers.isSuccess;
 const optionsDirty=optionsReady&&JSON.stringify(modifiersPayload(groups))!==JSON.stringify(modifiersPayload(loadedGroups));
 const optionsSummary=draft.id&&modifiers.isLoading?"Cargando…":groups.length?`${groups.length} ${groups.length===1?"grupo":"grupos"}`:"Opcional";
 const inventoryLocked=Boolean(draft.id)&&draft.quantityControl==="inventory";
 // Lock before async validation so rapid activations cannot queue duplicate requests.
 async function submit(){
  if(busy||submitting.current||imageValidating||imageInvalid.current)return;
  const optionErrors=optionsDirty?validateModifierGroups(groups):{};
  setGroupErrors(optionErrors);
  if(optionErrors.form){setShowOptions(true);return}
  submitting.current=true;
  try{await handleSubmit(values=>save(values,pendingFile,optionsDirty?groups:null))()}
  finally{submitting.current=false}
 }
 return <div className="modal-backdrop modal-overlay-in" role="presentation"><Dialog onResponseClose={close} className="crud-modal product-form modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="product-title" aria-busy={busy}><ModalHeader title={draft.id?"Editar producto":"Nuevo producto"} titleId="product-title" icon="box" onClose={close} busy={busy} closeLabel="Cerrar"/><ModalForm onSubmit={event=>event.preventDefault()} noValidate><ModalBody className="product-form-body"><div className="form-grid">
  <div className="product-identity span-2">
  <div className="form-grid product-identity-fields">
  <FormField className="span-2" label="Nombre del producto" error={errors.name?.message}><Input autoFocus maxLength={160} {...register("name")} placeholder="Ej. Lomo saltado"/></FormField>
  <FormField label="Precio de venta" error={errors.price?.message}><div className={"money-input"+(errors.price?" has-error":"")}><span>{currencySymbol}</span><input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div></FormField>
  <FormField label="Categoría" error={errors.categoryId?.message}><Select aria-invalid={Boolean(errors.categoryId)} {...register("categoryId")}><option value="">Selecciona una categoría</option>{categoryOptions.map(c=><option value={c.id} key={c.id}>{c.name}</option>)}</Select></FormField>
  <FormField className="span-2" label="Destino de atención" error={errors.serviceDestination?.message}><Select disabled={busy||!destinations.length} {...register("serviceDestination")}>{destinations.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}</Select></FormField>
  {value.serviceDestination==="direct"&&(draft.hasRecipe||Number(value.prepMinutes)>0)&&<InlineNotice className="span-2" tone="warning" title="Este producto parece prepararse">{draft.hasRecipe?"Tiene receta":"Tiene tiempo de preparación"}. Con «Sin preparación» no llegará a Cocina ni a Barra: entra al pedido como Listo.</InlineNotice>}
  </div>
  <ProductImageField draft={draft} name={value.name} busy={busy} scopeKey={imageScope} onSelect={setPendingFile} onRemove={()=>{setValue("imageUrl",null);setValue("removeImage",true)}} onValidation={(pending,error)=>{setImageValidating(pending);imageInvalid.current=pending||Boolean(error)}}/>
  </div>
  <fieldset className="product-stock-mode span-2"><legend>Control de cantidad <small>{inventoryLocked?"Gestionado desde Inventario":"Define de dónde sale la existencia"}</small></legend>{inventoryLocked?<div className="product-stock-locked"><span><Icon name="stock" size={16}/></span><b>Inventario físico<small>Las entradas y existencias se administran desde Inventario. Aquí solo editas la información comercial.</small></b></div>:<div className="product-stock-options is-two"><label><input className="product-radio" type="radio" value="none" {...register("quantityControl")}/><b>Sin control<small>Se vende sin contar unidades.</small></b></label><label><input className="product-radio" type="radio" value="portions" {...register("quantityControl")}/><b>Cupo del día<small>{draft.id?"Platos o bebidas con cantidad limitada. Se ajusta desde Disponibilidad.":"Platos o bebidas con cantidad limitada, sin usar Inventario."}</small></b></label></div>}</fieldset>
  {!draft.id&&value.quantityControl==="portions"&&<FormField className="span-2" label="Cantidad disponible hoy" error={errors.initialPortionQuantity?.message}><Input type="number" min="1" max="2147483647" step="1" inputMode="numeric" disabled={busy} {...register("initialPortionQuantity")} placeholder="Ej. 15"/></FormField>}
  <FormField as="div" className="product-description span-2" label="Descripción"><Controller control={control} name="description" render={({field})=><RichTextEditor value={field.value} onChange={field.onChange} placeholder="Ingredientes, presentación o información útil para el equipo"/>}/></FormField>
 </div><FormDisclosure title="Información adicional" open={showExtras} onToggle={()=>setShowExtras(!showExtras)}><div className="form-grid product-extras"><FormField label="Tiempo de preparación (min)" error={errors.prepMinutes?.message}><Input type="number" min="0" inputMode="numeric" {...register("prepMinutes")} placeholder="Ej. 15"/></FormField><FormField label="Precio de costo" optional error={errors.costPrice?.message}><div className={"money-input"+(errors.costPrice?" has-error":"")}><span>{currencySymbol}</span><input inputMode="decimal" {...register("costPrice")} placeholder="0.00" aria-invalid={Boolean(errors.costPrice)}/></div></FormField><Controller control={control} name="allergens" render={({field})=><AllergenField value={field.value} onChange={field.onChange} loadAllergens={loadAllergens} createAllergen={createAllergen} disabled={busy}/>}/><label className="switch-row compact span-2"><input type="checkbox" {...register("featured")}/><span/><b>Recomendar en la carta digital</b></label></div></FormDisclosure><FormDisclosure title="Opciones al pedir" summary={optionsSummary} open={showOptions} onToggle={()=>setShowOptions(!showOptions)}>{draft.id&&modifiers.isLoading?<div className="product-options" aria-busy="true" aria-label="Cargando opciones"><Skeleton width="60%" height="var(--size-14)"/><Skeleton width="100%" height="var(--control-height)"/><Skeleton width="100%" height="var(--control-height)"/></div>:draft.id&&modifiers.isError?<InlineNotice tone="danger" title="No pudimos cargar las opciones" action={<Button kind="secondary" icon="refresh" onClick={()=>void modifiers.refetch()} loading={modifiers.isFetching}>Reintentar</Button>}>Puedes guardar el resto del producto; sus opciones no se modificarán.</InlineNotice>:<>{groupErrors.form&&<InlineNotice tone="danger">{groupErrors.form}</InlineNotice>}<ProductOptionsField groups={groups} onChange={next=>{setGroups(next);if(groupErrors.form)setGroupErrors(validateModifierGroups(next))}} errors={groupErrors} currencySymbol={currencySymbol} disabled={busy}/></>}</FormDisclosure></ModalBody><ModalFooter><button type="button" className="button ghost" onClick={close} disabled={busy}>Cancelar</button><button type="button" className="button primary" onClick={()=>void submit()} disabled={busy||imageValidating}>{busy?"Guardando…":imageValidating?"Validando imagen…":"Guardar"}</button></ModalFooter></ModalForm></Dialog></div>
}
