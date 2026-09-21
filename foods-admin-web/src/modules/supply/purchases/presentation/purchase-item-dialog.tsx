"use client";
import {useDeferredValue,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {useForm} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {purchaseInventoryItemResolver} from "../domain/purchase-schema";
import type {PresentationType,PurchaseInventoryItemDraft,PurchaseInventoryOption} from "../domain/types";
import {listPurchaseInventory,listPurchaseItemCategories} from "../infrastructure/purchases-api";

type ItemMode="existing"|"new_product"|"new_ingredient";

const defaults:PurchaseInventoryItemDraft={
  mode:"new_product",
  categoryId:"",
  name:"",
  description:"",
  price:"",
  unit:"und",
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
  save:(draft:PurchaseInventoryItemDraft,file:File|null)=>void;
}){
  const{location}=useSession();
  const[mode,setMode]=useState<ItemMode>("existing");
  const[search,setSearch]=useState("");
  const deferredSearch=useDeferredValue(search.trim());
  const[pendingFile,setPendingFile]=useState<File|null>(null);
  const[previewUrl,setPreviewUrl]=useState<string|null>(null);
  const[imageError,setImageError]=useState("");
  const fileRef=useRef<HTMLInputElement>(null);

  const items=useQuery({
    queryKey:["purchase-item-picker",deferredSearch],
    queryFn:()=>listPurchaseInventory(deferredSearch),
    enabled:mode==="existing",
    staleTime:10000,
  });
  const categories=useQuery({
    queryKey:["purchase-item-categories"],
    queryFn:listPurchaseItemCategories,
    enabled:mode==="new_product",
    staleTime:30000,
  });

  const{register,handleSubmit,watch,setValue,formState:{errors,isSubmitted}}=useForm<PurchaseInventoryItemDraft>({
    defaultValues:defaults,
    resolver:purchaseInventoryItemResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const value=watch();

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

  function changePresentation(type:PresentationType){
    setValue("presentationType",type,{shouldDirty:true,shouldValidate:isSubmitted});
    setValue("unitsPerPresentation",type==="unit"?"1":"",{shouldDirty:true,shouldValidate:isSubmitted});
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
    <section className="crud-modal purchase-item-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-item-title" aria-busy={busy}>
      <div className="modal-accent"/>
      <header>
        <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
        <div><small>ORDEN DE COMPRA</small><h2 id="purchase-item-title">Agregar artículo</h2></div>
        <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
      </header>

      <form onSubmit={mode==="existing"?event=>event.preventDefault():handleSubmit(draft=>save(draft,pendingFile))} noValidate>
        <div className="purchase-item-body">
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
            <div className="purchase-section-title"><span><Icon name="search" size={17}/></span><div><b>Buscar artículo</b><small>Selecciona uno existente para agregarlo a la orden.</small></div></div>
            <label className="purchase-existing-search">Artículo
              <span className="ds-input-shell purchase-existing-search-control"><Icon name="search" size={16}/><Input autoFocus value={search} onChange={event=>setSearch(event.target.value)} placeholder="Escribe el nombre del producto o insumo"/></span>
            </label>

            {items.isLoading?<div className="purchase-item-results-loading">{Array.from({length:4},(_,index)=><i key={index}/>)}</div>
            :items.isError?<div className="purchase-item-results-state"><Icon name="alert" size={20}/><b>No pudimos cargar los artículos</b><small>{items.error.message}</small><Button type="button" kind="secondary" icon="refresh" onClick={()=>items.refetch()}>Reintentar</Button></div>
            :existing.length?<div className="purchase-existing-list">
              {existing.map(item=><button type="button" key={item.id} onClick={()=>choose(item)}>
                <span className="purchase-existing-icon"><Icon name={item.kind==="ingredient"?"stock":"box"} size={17}/></span>
                <span className="purchase-existing-copy"><b>{item.name}</b><small>{item.kind==="ingredient"?"Insumo":"Producto vendible"} · Unidad base: {item.unit}</small></span>
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
                <label className="span-2">Nombre<Input autoFocus maxLength={160} {...register("name")} placeholder={mode==="new_product"?"Ej. Coca-Cola 500 ml":"Ej. Carne de res"} aria-invalid={Boolean(errors.name)}/>{errors.name?.message&&<small className="wizard-field-error">{errors.name.message}</small>}</label>
                {mode==="new_product"&&<>
                  <label>Categoría<Select {...register("categoryId")} disabled={categories.isLoading||categories.isError||!categories.data?.length} aria-invalid={Boolean(errors.categoryId)||categories.isError}><option value="">{categories.isLoading?"Cargando categorías...":categories.isError?"No pudimos cargar las categorías":categories.data?.length?"Selecciona una categoría":"No hay categorías para mercadería vendible"}</option>{categories.data?.map(category=><option value={category.id} key={category.id}>{category.name}</option>)}</Select>{categories.isError?<small className="wizard-field-error">{categories.error.message}</small>:errors.categoryId?.message&&<small className="wizard-field-error">{errors.categoryId.message}</small>}</label>
                  <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div>{errors.price?.message&&<small className="wizard-field-error">{errors.price.message}</small>}</label>
                  <label className="span-2">Descripción opcional<Textarea maxLength={1000} rows={2} {...register("description")} placeholder="Presentación o detalle comercial"/></label>
                  <label className="purchase-item-image span-2">Imagen del producto (opcional)
                    <input ref={fileRef} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={event=>{const file=event.target.files?.[0];if(file)selectFile(file)}}/>
                    <div className="purchase-image-drop" role="button" tabIndex={0} onClick={()=>fileRef.current?.click()} onKeyDown={event=>{if(event.key==="Enter"||event.key===" ")fileRef.current?.click()}}>
                      {previewUrl?<><img src={previewUrl} alt={value.name||"Vista previa del producto"}/><button type="button" className="purchase-image-clear" onClick={event=>{event.stopPropagation();clearImage()}} aria-label="Quitar imagen"><Icon name="close" size={15}/></button></>:<div><Icon name="box" size={23}/><b>Seleccionar imagen</b><small>PNG, JPEG o WebP · máximo 5 MB</small></div>}
                    </div>
                    {imageError&&<small className="wizard-field-error">{imageError}</small>}
                  </label>
                </>}
              </div>
            </section>

            <section className="purchase-item-section">
              <div className="purchase-section-title"><span><Icon name="box" size={17}/></span><div><b>Unidad y presentación</b><small>Define cómo se almacenará y cómo se comprará este artículo.</small></div></div>
              <div className="form-grid">
                <label>Unidad base<Select {...register("unit")} aria-invalid={Boolean(errors.unit)}><option value="und">Unidad</option><option value="botella">Botella</option><option value="lata">Lata</option><option value="caja">Caja</option><option value="kg">Kilogramo</option><option value="l">Litro</option></Select>{errors.unit?.message&&<small className="wizard-field-error">{errors.unit.message}</small>}</label>
                <label>Presentación<Select value={value.presentationType} onChange={event=>changePresentation(event.target.value as PresentationType)} aria-invalid={Boolean(errors.presentationType)||Boolean(errors.unitsPerPresentation)}><option value="unit">Unidad base</option><option value="package">Paquete</option><option value="box">Caja</option></Select></label>
                {value.presentationType!=="unit"&&<label>Unidades por {value.presentationType==="box"?"caja":"paquete"}<Input type="number" min="1.001" step="0.001" inputMode="decimal" {...register("unitsPerPresentation")} placeholder="Ej. 12" aria-invalid={Boolean(errors.unitsPerPresentation)}/>{errors.unitsPerPresentation?.message&&<small className="wizard-field-error">{errors.unitsPerPresentation.message}</small>}</label>}
                <label>Stock mínimo<Input type="number" min="0" step="0.001" inputMode="decimal" {...register("minimumStock")} aria-invalid={Boolean(errors.minimumStock)}/>{errors.minimumStock?.message&&<small className="wizard-field-error">{errors.minimumStock.message}</small>}</label>
              </div>
            </section>

            <div className="purchase-item-zero-note"><Icon name="lock" size={16}/><p><b>Stock inicial: 0</b>Crear el artículo y agregarlo a la orden no mueve existencias. El stock cambia únicamente al confirmar una recepción.</p></div>
          </>}
        </div>
        <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button>{mode!=="existing"&&<Button type="submit" disabled={busy||Boolean(imageError)}>{busy?"Guardando…":"Crear y agregar"}</Button>}</footer>
      </form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </section>
  </div>;
}
