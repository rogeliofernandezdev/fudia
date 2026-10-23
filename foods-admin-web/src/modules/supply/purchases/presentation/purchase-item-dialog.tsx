"use client";
import {Dialog} from "@/design-system/dialog";
import {useDeferredValue,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {useForm,useWatch} from "react-hook-form";
import {Button,EmptyState,ErrorState,Icon,Input,Skeleton,Status,ModalHeader,ModalFooter} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import {purchaseInventoryItemResolver} from "../domain/purchase-schema";
import {linkableControlLabel} from "../domain/linkable-product";
import type {LinkableProduct,PurchaseInventoryItemDraft,PurchaseInventoryOption} from "../domain/types";
import {listLinkableProducts,listPurchaseInventory} from "../infrastructure/purchases-api";
import {PurchaseUnitField} from "./purchase-unit-field";
import {PurchaseCategoryField} from "./purchase-category-field";
import {PurchasePresentationsField} from "./purchase-presentations-field";

/** Buscar es la vista inicial; crear y vincular son salidas explícitas desde la búsqueda. */
type View={kind:"search"}|{kind:"create"}|{kind:"link";product:LinkableProduct};

const blankDraft:PurchaseInventoryItemDraft={
  mode:"new_product",
  productId:"",
  categoryId:"",
  name:"",
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
  save:(draft:PurchaseInventoryItemDraft)=>Promise<void>;
}){
  const{location,organization,modules}=useSession();
  const usesRecipes=Boolean(modules?.recetas);
  const[view,setView]=useState<View>({kind:"search"});
  const[search,setSearch]=useState("");
  const deferredSearch=useDeferredValue(search.trim());
  const[catalogBusy,setCatalogBusy]=useState(false);
  const submitting=useRef(false);
  const searching=view.kind==="search";
  const linkSearch=deferredSearch.length>=2;

  const items=useQuery({
    queryKey:["purchase-item-picker",organization?.id,deferredSearch],
    queryFn:()=>listPurchaseInventory(deferredSearch),
    enabled:searching,
    staleTime:10000,
  });
  const linkable=useQuery({
    queryKey:["purchase-linkable-products",organization?.id,deferredSearch],
    queryFn:()=>listLinkableProducts(deferredSearch),
    enabled:searching&&linkSearch,
    staleTime:10000,
  });
  // Sin Recetas casi todo se compra para revender; con Recetas lo habitual es un insumo.
  const{control,register,handleSubmit,setValue,reset,formState:{errors,isSubmitted,isSubmitting}}=useForm<PurchaseInventoryItemDraft>({
    defaultValues:{...blankDraft,mode:usesRecipes?"new_ingredient":"new_product"},
    resolver:purchaseInventoryItemResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const mode=useWatch({control,name:"mode"});
  const categoryId=useWatch({control,name:"categoryId"})??"";
  const unit=useWatch({control,name:"unit"})??"";
  const presentations=useWatch({control,name:"presentations"})??[];
  const locked=busy||isSubmitting;
  const validationMessage=Object.values(errors).find(error=>typeof error?.message==="string")?.message;

  async function submit(){
    if(searching||busy||catalogBusy||submitting.current)return;
    // Lock before asynchronous validation; await the request before allowing a retry.
    submitting.current=true;
    try{await handleSubmit(draft=>save(draft))()}
    finally{submitting.current=false}
  }

  function startCreate(){
    reset({...blankDraft,mode:usesRecipes?"new_ingredient":"new_product",name:search.trim()});
    setView({kind:"create"});
  }
  function startLink(product:LinkableProduct){
    reset({...blankDraft,mode:"link_product",productId:product.productId,name:product.name});
    setView({kind:"link",product});
  }
  function setSold(sold:boolean){
    setValue("mode",sold?"new_product":"new_ingredient",{shouldDirty:true,shouldValidate:isSubmitted});
    if(!sold){
      setValue("categoryId","",{shouldDirty:true});
      setValue("price","",{shouldDirty:true});
    }
  }

  const existing=items.data?.items??[];
  const linkableItems=linkSearch?linkable.data?.items??[]:[];
  const nothingFound=!items.isLoading&&!items.isError&&!existing.length&&(!linkSearch||(!linkable.isLoading&&!linkable.isError&&!linkableItems.length));

  const stockFields=<section className="purchase-item-section">
    <div className="purchase-section-title"><span><Icon name="box" size={18}/></span><div><b>Unidad y presentación</b><small>Cómo se almacena y cómo se compra.</small></div></div>
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
  </section>;

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal purchase-item-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-item-title" aria-busy={locked}>

      <ModalHeader title={view.kind==="create"?"Nuevo artículo":view.kind==="link"?"Controlar con inventario":"Agregar artículo"} titleId="purchase-item-title" icon="stock" onClose={close} busy={locked||catalogBusy} closeLabel="Cerrar"/>

      <form onSubmit={event=>event.preventDefault()} noValidate>
        <div className="purchase-item-body" inert={busy}>
          {view.kind==="search"?<section className="purchase-item-section purchase-existing-section">
            <label className="purchase-existing-search">Buscar artículo
              <span className="ds-input-shell purchase-existing-search-control"><Icon name="search" size={16}/><Input autoFocus value={search} onChange={event=>setSearch(event.target.value)} placeholder="Escribe el nombre del producto o insumo"/></span>
            </label>

            {items.isLoading?<div className="purchase-item-results" role="status" aria-busy="true" aria-label="Cargando artículos">{Array.from({length:4},(_,index)=><Skeleton key={index}/>)}</div>
            :items.isError?<ErrorState title="No pudimos cargar los artículos" text={items.error.message} onRetry={()=>void items.refetch()}/>
            :nothingFound?<EmptyState
              icon="search"
              title={deferredSearch?"No encontramos ese artículo":"No hay artículos disponibles"}
              text={deferredSearch?"Créalo ahora y quedará agregado a la orden.":"Crea el primer artículo para abastecimiento."}
              action={<Button type="button" kind="secondary" icon="plus" onClick={startCreate}>Crear artículo nuevo</Button>}
            />:<>
              {existing.length>0&&<div className="purchase-result-list" aria-label="Artículos de inventario">
                {existing.map(item=>{const stock=Number(item.quantity??0);return <button type="button" className="purchase-result" key={item.id} onClick={()=>choose(item)}>
                  <span className="purchase-result-copy"><b>{item.name}</b><small>{item.kind==="ingredient"?"Insumo":"Se vende"} · {item.unit}</small></span>
                  {stock>0?<span className="purchase-result-stock">{formatRegionalNumber(stock,location?.country,{maximumFractionDigits:3})} {item.unit}</span>:<Status tone="gray">Agotado</Status>}
                  <Icon name="chevron" size={16}/>
                </button>})}
              </div>}
              {linkSearch&&(linkable.isLoading?<div className="purchase-item-results" role="status" aria-busy="true" aria-label="Buscando en la carta"><Skeleton/></div>
              :linkable.isError?<div className="purchase-validation" role="alert"><Icon name="alert" size={16}/>{linkable.error.message}<Button type="button" kind="secondary" icon="refresh" onClick={()=>void linkable.refetch()}>Reintentar</Button></div>
              :linkableItems.length>0&&<div className="purchase-linkable-group">
                <p className="purchase-linkable-title">En tu carta, sin inventario</p>
                <div className="purchase-result-list" aria-label="Productos de la carta sin inventario">
                  {linkableItems.map(product=><button type="button" className="purchase-result" key={product.productId} onClick={()=>startLink(product)}>
                    <span className="purchase-result-copy"><b>{product.name}</b><small>{linkableControlLabel(product)}{product.categoryName?" · "+product.categoryName:""}</small></span>
                    <span className="purchase-result-action">Vincular</span>
                    <Icon name="chevron" size={16}/>
                  </button>)}
                </div>
              </div>)}
              <button type="button" className="purchase-add-row" onClick={startCreate}><Icon name="plus" size={18}/>Crear artículo nuevo</button>
            </>}
          </section>:view.kind==="create"?<>
            <section className="purchase-item-section">
              <div className="form-grid">
                <label className="span-2">Nombre<Input autoFocus maxLength={160} {...register("name")} placeholder="Ej. Inca Kola 500 ml o Carne de res" aria-invalid={Boolean(errors.name)}/>{errors.name?.message&&<small className="field-error">{errors.name.message}</small>}</label>
                <label className="switch-row compact span-2"><input type="checkbox" checked={mode==="new_product"} disabled={locked} onChange={event=>setSold(event.target.checked)}/><span/><b>Se vende a clientes<small>{mode==="new_product"?"Aparecerá en la carta y su disponibilidad saldrá del stock.":"Queda como insumo: no aparece en la carta."}</small></b></label>
                {mode==="new_product"&&<>
                  <PurchaseCategoryField value={categoryId} onChange={value=>setValue("categoryId",value,{shouldDirty:true,shouldValidate:isSubmitted})} error={errors.categoryId?.message} disabled={busy} onBusyChange={setCatalogBusy}/>
                  <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div>{errors.price?.message&&<small className="field-error">{errors.price.message}</small>}</label>
                </>}
              </div>
            </section>
            {stockFields}
            <div className="purchase-item-zero-note"><Icon name="lock" size={16}/><p><b>Stock inicial: 0</b>El stock cambia únicamente al confirmar una recepción.{usesRecipes?" Todo artículo puede usarse como ingrediente en Recetas.":""}</p></div>
          </>:<>
            <section className="purchase-item-section purchase-link-summary">
              <span className="purchase-existing-icon"><Icon name="dish" size={18}/></span>
              <div><b>{view.product.name}</b><small>{linkableControlLabel(view.product)}{view.product.categoryName?" · "+view.product.categoryName:""}</small></div>
            </section>
            <div className="purchase-item-zero-note"><Icon name="lock" size={16}/><p><b>Conserva nombre, precio, categoría e historial</b>Desde ahora su stock se llevará con Compras e Inventario y empieza en 0: aparecerá «Agotado» hasta que confirmes una recepción o registres una entrada en Inventario.</p></div>
            {stockFields}
          </>}
        </div>
        <ModalFooter>
          {!searching&&typeof validationMessage==="string"&&<div className="purchase-validation" role="alert"><Icon name="alert" size={16}/>{validationMessage}</div>}
          {searching
            ?<Button type="button" kind="ghost" onClick={close} disabled={locked}>Cancelar</Button>
            :<Button type="button" action="back" onClick={()=>setView({kind:"search"})} disabled={locked||catalogBusy}>Volver</Button>}
          {!searching&&<Button type="button" onClick={()=>void submit()} disabled={locked||catalogBusy} loading={locked}>{locked?"Guardando…":view.kind==="link"?"Vincular y agregar":"Crear y agregar"}</Button>}
        </ModalFooter>
      </form>
    </Dialog>
  </div>;
}
