"use client";
import {useForm} from "react-hook-form";
import {Button,Icon,Input,Select,Textarea} from "@/design-system";
import {purchaseInventoryItemResolver} from "../domain/purchase-schema";
import type {PresentationType,PurchaseInventoryItemDraft,PurchaseItemCategory} from "../domain/types";

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

export function PurchaseItemDialog({categories,categoryError,currencySymbol,busy,close,save}:{categories:PurchaseItemCategory[];categoryError:string|null;currencySymbol:string;busy:boolean;close:()=>void;save:(draft:PurchaseInventoryItemDraft)=>void}){
  const{register,handleSubmit,watch,setValue,formState:{errors,isSubmitted}}=useForm<PurchaseInventoryItemDraft>({
    defaultValues:defaults,
    resolver:purchaseInventoryItemResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });
  const value=watch();

  function changeMode(mode:PurchaseInventoryItemDraft["mode"]){
    setValue("mode",mode,{shouldDirty:true,shouldValidate:isSubmitted});
    if(mode==="new_ingredient"){
      setValue("categoryId","",{shouldDirty:true});
      setValue("price","",{shouldDirty:true});
      setValue("description","",{shouldDirty:true});
    }
  }

  function changePresentation(type:PresentationType){
    setValue("presentationType",type,{shouldDirty:true,shouldValidate:isSubmitted});
    setValue("unitsPerPresentation",type==="unit"?"1":"",{shouldDirty:true,shouldValidate:isSubmitted});
  }

  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <section className="crud-modal purchase-item-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="purchase-item-title" aria-busy={busy}>
      <div className="modal-accent"/>
      <header>
        <span className="modal-title-icon"><Icon name="stock" size={18}/></span>
        <div><small>ORDEN DE COMPRA</small><h2 id="purchase-item-title">Crear artículo</h2></div>
        <button type="button" aria-label="Cerrar" onClick={close} disabled={busy}><Icon name="close"/></button>
      </header>
      <form onSubmit={handleSubmit(save)} noValidate>
        <div className="purchase-item-body">
          <section className="purchase-item-choice" aria-label="Tipo de artículo">
            <button type="button" className={value.mode==="new_product"?"active":""} onClick={()=>changeMode("new_product")}>
              <span><Icon name="plus" size={18}/></span><b>Nuevo producto vendible<small>Mercadería que también se vende y queda vinculada a Producto.</small></b>
            </button>
            <button type="button" className={value.mode==="new_ingredient"?"active":""} onClick={()=>changeMode("new_ingredient")}>
              <span><Icon name="stock" size={18}/></span><b>Nuevo insumo<small>Artículo interno para compras y recetas; no se vuelve vendible.</small></b>
            </button>
          </section>

          <section className="purchase-item-section">
            <div className="purchase-section-title"><span><Icon name={value.mode==="new_product"?"plus":"stock"} size={17}/></span><div><b>{value.mode==="new_product"?"Producto vendible":"Insumo"}</b><small>Se crea como artículo de inventario con stock inicial cero.</small></div></div>
            <div className="form-grid">
              <label className="span-2">Nombre<Input autoFocus maxLength={160} {...register("name")} placeholder={value.mode==="new_product"?"Ej. Coca-Cola 500 ml":"Ej. Carne de res"} aria-invalid={Boolean(errors.name)}/>{errors.name?.message&&<small className="wizard-field-error">{errors.name.message}</small>}</label>
              {value.mode==="new_product"&&<>
                <label>Categoría<Select {...register("categoryId")} disabled={Boolean(categoryError)||categories.length===0} aria-invalid={Boolean(errors.categoryId)||Boolean(categoryError)}><option value="">{categoryError?"No pudimos cargar las categorías":categories.length?"Selecciona una categoría":"No hay categorías para mercadería vendible"}</option>{categories.map(category=><option value={category.id} key={category.id}>{category.name}</option>)}</Select>{categoryError?<small className="wizard-field-error">{categoryError}</small>:errors.categoryId?.message&&<small className="wizard-field-error">{errors.categoryId.message}</small>}</label>
                <label>Precio de venta<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" {...register("price")} placeholder="0.00" aria-invalid={Boolean(errors.price)}/></div>{errors.price?.message&&<small className="wizard-field-error">{errors.price.message}</small>}</label>
                <label className="span-2">Descripción opcional<Textarea maxLength={1000} {...register("description")} placeholder="Presentación o detalle comercial"/></label>
              </>}
            </div>
          </section>

          <section className="purchase-item-section">
            <div className="purchase-section-title"><span><Icon name="box" size={17}/></span><div><b>Control físico</b><small>La presentación convierte a la unidad base cuando se reciba la compra.</small></div></div>
            <div className="form-grid">
              <label>Unidad base<Select {...register("unit")} aria-invalid={Boolean(errors.unit)}><option value="und">Unidad</option><option value="botella">Botella</option><option value="lata">Lata</option><option value="caja">Caja</option><option value="kg">Kilogramo</option><option value="l">Litro</option></Select>{errors.unit?.message&&<small className="wizard-field-error">{errors.unit.message}</small>}</label>
              <label>Presentación<Select value={value.presentationType} onChange={event=>changePresentation(event.target.value as PresentationType)} aria-invalid={Boolean(errors.presentationType)||Boolean(errors.unitsPerPresentation)}><option value="unit">Unidad base</option><option value="package">Paquete</option><option value="box">Caja</option></Select></label>
              {value.presentationType!=="unit"&&<label>Unidades por {value.presentationType==="box"?"caja":"paquete"}<Input type="number" min="1.001" step="0.001" inputMode="decimal" {...register("unitsPerPresentation")} placeholder="Ej. 12" aria-invalid={Boolean(errors.unitsPerPresentation)}/>{errors.unitsPerPresentation?.message&&<small className="wizard-field-error">{errors.unitsPerPresentation.message}</small>}</label>}
              <label>Stock mínimo<Input type="number" min="0" step="0.001" inputMode="decimal" {...register("minimumStock")} aria-invalid={Boolean(errors.minimumStock)}/>{errors.minimumStock?.message&&<small className="wizard-field-error">{errors.minimumStock.message}</small>}</label>
            </div>
          </section>

          <div className="purchase-item-zero-note"><Icon name="lock" size={16}/><p><b>Stock inicial: 0</b>Crear el artículo y agregarlo a la orden no mueve existencias. El stock cambia únicamente cuando se confirma una recepción.</p></div>
        </div>
        <footer><Button type="button" kind="ghost" onClick={close} disabled={busy}>Cancelar</Button><Button type="submit" disabled={busy}>{busy?"Guardando…":"Guardar"}</Button></footer>
      </form>
      {busy&&<div className="modal-busy" role="status"><i/><span>Guardando…</span></div>}
    </section>
  </div>;
}
