"use client";
/* eslint-disable @next/next/no-img-element -- Local blob previews must bypass Next image optimization. */
import {Dialog} from "@/design-system/dialog";
import {useDeferredValue,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {useForm,useWatch} from "react-hook-form";
import {Button,Icon,Input,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {purchaseInventoryItemResolver} from "../domain/purchase-schema";
import type {PurchaseInventoryItemDraft,PurchaseInventoryOption} from "../domain/types";
import {listPurchaseInventory} from "../infrastructure/purchases-api";
import {PurchaseUnitField} from "./purchase-unit-field";
import {PurchaseCategoryField} from "./purchase-category-field";
import {PurchasePresentationsField} from "./purchase-presentations-field";

type ItemMode="existing"|"new_product"|"new_ingredient";

const defaults:PurchaseInventoryItemDraft={
  mode:"new_product",
  categoryId:"",
  name:"",
  description:"",
  price:"",
  unit:"",
  presentationType:"unit",
  unitsPerPresentation:"1",
  minimumStock:"0",
};

export function PurchaseItemDialog({
  currencySymbol,busy,close,choose,save,
}:{
  currencySymbol:string;
  busy:boolean;
  close:()=>void;
  choose:(item:PurchaseInventoryOption)=>void;
  save:(draft:PurchaseInventoryItemDraft,file:File|null)=>Promise<void>;
}){
  const{location,organization}=useSession();
  const[mode,setMode]=useState<ItemMode>("existing");
  const[search,setSearch]=useState("");
  const deferredSearch=useDeferredValue(search.trim());
  const[pendingFile,setPendingFile]=useState<File|null>(null);
  const[previewUrl,setPreviewUrl]=useState<string|null>(null);
  const[imageError,setImageError]=useState("");
  const[catalogBusy,setCatalogBusy]=useState(false);
  const fileRef=useRef<HTMLInputElement>(null);
  const submitting=useRef(false);

  const items=useQuery({
    queryKey:["purchase-item-picker",organization?.id,deferredSearch],
    queryFn:()=>listPurchaseInventory(deferredSearch),
    enabled:mode==="existing",
    staleTime:10000,
  });
  const{control,register,handleSubmit,setValue,formState:{errors,isSubmitted,isSubmitting}}=useForm<PurchaseInventoryItemDraft>({
    defaultValues:defaults,
    resolver:purchaseInventoryItemResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const watchedName=useWatch({control,name:"name"});
  const categoryId=useWatch({control,name:"categoryId"})??"";
  const unit=useWatch({control,name:"unit"})??"";
  const presentations=useWatch({control,name:"presentations"})??[];
  const locked=busy||isSubmitting;
  const validationMessage=Object.values(errors).find(error=>typeof error?.message==="string")?.message;

  async function submit(){
    if(mode==="existing"||busy||catalogBusy||submitting.current||imageError)return;
    // Lock before asynchronous validation; await the request before allowing a retry.
    submitting.current=true;
    try{await handleSubmit(draft=>save(draft,pendingFile))()}
    finally{submitting.current=false}
  }

  function changeMode(next:ItemMode){
    setMode(next);
    setImageError("");
    if(next==="new_product"||next==="new_ingredient"){
      setValue("mode",next,{shouldDirty:true,shouldValidate:isSubmitted});
    }
    if(next!=="new_product"){
      setPendingFile(null);
      setPreviewUrl(null);
    }
    if(next==="new_ingredient"){
      setValue("categoryId","",{shouldDirty:true});
      setValue("price","",{shouldDirty:true});
      setValue("description","",{shouldDirty:true});
    }
  }

  function selectFile(file:File){
    if(file.size>5*1024*1024){
      setImageError("La imagen no puede pesar más de 5 MB.");
      return;
    }
    setImageError("");
    setPendingFile(file);
    setPreviewUrl(URL.createObjectURL(file));
  }

  function clearImage(){
    setPendingFile(null);
    setPreviewUrl(null);
    setImageError("");
    if(fileRef.current)fileRef.current.value="";
  }

  const existing=items.data?.items??[];

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal purchase-item-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-item-title" aria-busy={locked}>
      <div className="modal-accent"/>
      <header>
        <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
        <div><small>ORDEN DE COMPRA</small><h2 id="purchase-item-title">Agregar artículo</h2></div>
        <button type="button" aria-label="Cerrar" onClick={close} disabled={locked||catalogBusy}><Icon name="close"/></button>
      </header>

      <form onSubmit={event=>event.preventDefault()} noValidate>
        <div className="purchase-item-body" inert={busy}>
          <section className="purchase-item-choice" aria-label="Cómo agregar el artículo">
            <button type="button" className={mode==="existing"?"active":""} onClick={()=>changeMode("existing")}>
              <span><Icon name="search" size={18}/></span>
              <b>Artículo existente<small>Busca un producto o insumo que ya está creado.</small></b>
            </button>
            <button type="button" className={mode==="new_product"?"active":""} onClick={()=>changeMode("new_product")}>
              <span><Icon name="plus" size={18}/></span>
              <b>Nuevo producto vendible<small>Mercadería que también se vende y queda vinculada a Producto.</small></b>
            </button>
            <button type="button" className={mode==="new_ingredient"?"active":""} onClick={()=>changeMode("new_ingredient")}>
              <span><Icon name="stock" size={18}/></span>
              <b>Nuevo insumo<small>Artículo interno para compras y recetas; no se vuelve vendible.</small></b>
            </button>
          </section>

          {mode==="existing"?<section className="purchase-item-section purchase-existing-section">
            <label className="purchase-existing-search">Buscar artículo
              <span className="ds-input-shell purchase-existing-search-control"><Icon name="search" size={16}/><Input autoFocus value={search} onChange={event=>setSearch(event.target.value)} placeholder="Escribe el nombre del producto o insumo"/></span>
            </label>

            {items.isLoading?<div className="purchase-item-results-loading">{Array.from({length:4},(_,index)=><i key={index}/>)}</div>
            :items.isError?<div className="purchase-item-results-state"><Icon name="alert" size={20}/><b>No pudimos cargar los artículos</b><small>{items.error.message}</small><Button type="button" kind="secondary" icon="refresh" onClick={()=>items.refetch()}>Reintentar</Button></div>
            :existing.length?<div className="purchase-existing-list">
              {existing.map(item=><button type="button" key={item.id} onClick={()=>choose(item)}>
                <span className="purchase-existing-icon"><Icon name={item.kind==="ingredient"?"stock":"box"} size={17}/></span>
                <span className="purchase-existing-copy"><b>{item.name}</b><small>{item.kind==="ingredient"?"Insumo":"Producto vendible"} · Unidad de inventario: {item.unit}</small></span>
                <span className="purchase-existing-stock"><small>STOCK</small><b>{formatRegionalNumber(Number(item.quantity??0),location?.country,{maximumFractionDigits:3})}</b></span>
                <Icon name="chevron" size={15}/>
              </button>)}
            </div>
            :<div className="purchase-item-results-state purchase-item-not-found">
              <Icon name="search" size={21}/><b>{deferredSearch?"No encontramos ese artículo":"No hay artículos disponibles"}</b>
              <small>{deferredSearch?"Puedes crearlo ahora y quedará seleccionado en la orden.":"Crea el primer artículo para abastecimiento."}</small>
              <div><Button type="button" kind="secondary" icon="plus" onClick={()=>changeMode("new_product")}>Nuevo producto vendible</Button><Button type="button" kind="secondary" icon="stock" onClick={()=>changeMode("new_ingredient")}>Nuevo insumo</Button></div>
            </div>}
          </section>:<>
            <section className="purchase-item-section">
              <div className="purchase-section-title"><span><Icon name={mode==="new_product"?"plus":"stock"} size={17}/></span><div><b>{mode==="new_product"?"Nuevo producto vendible":"Nuevo insumo"}</b><small>{mode==="new_product"?"Se crea como mercadería vendible con Inventario físico.":"Se crea solo como inventario interno; no tendrá precio de venta."}</small></div></div>
              <div className="form-grid">
                <label className="span-2">Nombre<Input autoFocus maxLength={160} {...register("name")} placeholder={mode==="new_product"?"Ej. Coca-Cola 500 ml":"Ej. Carne de res"} aria-invalid={Boolean(errors.name)}/>{errors.name?.message&&<small className="field-error">{errors.name.message}</small>}</label>
                {mode==="new_product"&&<>
                  <PurchaseCategoryField value={categoryId} onChange={value=>setValue("categoryId",value,{shouldDirty:true,shouldValidate:isSubmitted})} error={errors.categoryId?.message} disabled={busy} onBusyChange={setCatalogBusy}/>
                  <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div>{errors.price?.message&&<small className="field-error">{errors.price.message}</small>}</label>
                  <label className="span-2">Descripción opcional<Textarea maxLength={1000} rows={2} {...register("description")} placeholder="Presentación o detalle comercial"/></label>
                  <label className="purchase-item-image span-2">Imagen del producto (opcional)
                    <input ref={fileRef} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={event=>{const file=event.target.files?.[0];if(file)selectFile(file)}}/>
                    <div className="purchase-image-drop" role="button" tabIndex={0} onClick={()=>fileRef.current?.click()} onKeyDown={event=>{if(event.key==="Enter"||event.key===" ")fileRef.current?.click()}}>
                      {previewUrl?<><img src={previewUrl} alt={watchedName||"Vista previa del producto"}/><button type="button" className="purchase-image-clear" onClick={event=>{event.stopPropagation();clearImage()}} aria-label="Quitar imagen"><Icon name="close" size={15}/></button></>:<div><Icon name="box" size={23}/><b>Seleccionar imagen</b><small>PNG, JPEG o WebP · máximo 5 MB</small></div>}
                    </div>
                    {imageError&&<small className="field-error">{imageError}</small>}
                  </label>
                </>}
              </div>
            </section>

            <section className="purchase-item-section">
              <div className="purchase-section-title"><span><Icon name="box" size={17}/></span><div><b>Unidad y presentación</b><small>Define cómo se almacenará y cómo se comprará este artículo.</small></div></div>
              <div className="form-grid">
                <PurchaseUnitField value={unit} onChange={value=>{setValue("unit",value,{shouldDirty:true,shouldValidate:isSubmitted});setValue("presentations",[],{shouldDirty:true})}} error={errors.unit?.message} disabled={locked||catalogBusy} onBusyChange={setCatalogBusy}/>
                <label>Stock mínimo<Input type="number" min="0" step="0.001" inputMode="decimal" {...register("minimumStock")} aria-invalid={Boolean(errors.minimumStock)}/>{errors.minimumStock?.message&&<small className="field-error">{errors.minimumStock.message}</small>}</label>
              </div>
              <PurchasePresentationsField unit={unit} value={presentations} disabled={locked} onBusyChange={setCatalogBusy} error={errors.presentations?.message??errors.unitsPerPresentation?.message} onChange={rows=>{
                setValue("presentations",rows,{shouldDirty:true,shouldValidate:isSubmitted});
                const preferred=rows.find(row=>row.isDefault)??rows[0];
                setValue("presentationType",preferred?.presentationType??"unit",{shouldDirty:true});
                setValue("unitsPerPresentation",preferred?.unitsPerPresentation??"1",{shouldDirty:true});
              }}/>
            </section>

            <div className="purchase-item-zero-note"><Icon name="lock" size={16}/><p><b>Stock inicial: 0</b>Crear el artículo y agregarlo a la orden no mueve existencias. El stock cambia únicamente al confirmar una recepción.</p></div>
          </>}
        </div>
        <footer>
          {mode!=="existing"&&typeof validationMessage==="string"&&<div className="purchase-validation" role="alert"><Icon name="alert" size={15}/>{validationMessage}</div>}
          <Button type="button" kind="ghost" onClick={close} disabled={locked||catalogBusy}>Cancelar</Button>
          {mode!=="existing"&&<Button type="button" onClick={()=>void submit()} disabled={locked||catalogBusy}>{locked?"Guardando…":"Crear y agregar"}</Button>}
        </footer>
      </form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </Dialog>
  </div>;
}
