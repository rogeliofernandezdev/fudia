import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {InventoryEntryDraft} from "./types";

const quantity=z.string().trim().refine(value=>{
  const parsed=Number(value);
  return value!==""&&Number.isFinite(parsed)&&parsed>0;
},"Ingresa una cantidad mayor que cero.");

const minimumStock=z.string().trim().refine(value=>{
  const parsed=Number(value);
  return value!==""&&Number.isFinite(parsed)&&parsed>=0;
},"El stock mínimo no puede ser negativo.");

const common={
  inventoryItemId:z.string(),
  productId:z.string(),
  categoryId:z.string(),
  sku:z.string().max(40),
  description:z.string().max(1000),
  quantity,
  unit:z.string().trim().min(1,"Selecciona una unidad base."),
  presentationType:z.enum(["unit","package","box"]),
  unitsPerPresentation:z.string(),
  minimumStock,
  note:z.string().max(240),
};

const existingSchema=z.object({
  mode:z.literal("existing"),
  ...common,
  inventoryItemId:z.string().trim().min(1,"Selecciona el artículo que estás recibiendo."),
  name:z.string().max(160),
  price:z.string(),
});

const newProductSchema=z.object({
  mode:z.literal("new_product"),
  ...common,
  name:z.string().trim().min(1,"Ingresa el nombre del producto.").max(160),
  categoryId:z.string().trim().min(1,"Selecciona una categoría."),
  price:z.string().trim().regex(/^[0-9]+([.][0-9]{1,2})?$/,"Ingresa un precio válido."),
});

const newIngredientSchema=z.object({
  mode:z.literal("new_ingredient"),
  ...common,
  name:z.string().trim().min(1,"Ingresa el nombre del insumo.").max(160),
  price:z.string(),
});

export const inventoryEntrySchema=z.discriminatedUnion("mode",[
  existingSchema,
  newProductSchema,
  newIngredientSchema,
]).superRefine((value,ctx)=>{
  const parsedQuantity=Number(value.quantity);
  if(value.presentationType!=="unit"&&!Number.isInteger(parsedQuantity)){
    ctx.addIssue({
      code:"custom",
      path:["quantity"],
      message:"La cantidad de paquetes o cajas debe ser un número entero.",
    });
  }

  if(value.presentationType!=="unit"){
    const factor=Number(value.unitsPerPresentation);
    if(value.unitsPerPresentation.trim()===""||!Number.isFinite(factor)||factor<=1){
      ctx.addIssue({
        code:"custom",
        path:["unitsPerPresentation"],
        message:"Debe contener más de una unidad base.",
      });
    }
  }
});

export const inventoryEntryResolver:Resolver<InventoryEntryDraft>=(values)=>{
  const result=inventoryEntrySchema.safeParse(values);
  if(result.success){
    return {values:result.data,errors:{}};
  }

  const flatErrors:Record<string,FieldError>={};
  for(const issue of result.error.issues){
    const field=issue.path[0];
    if(typeof field!=="string"||flatErrors[field])continue;
    flatErrors[field]={type:issue.code,message:issue.message};
  }

  return {
    values:{},
    errors:flatErrors as FieldErrors<InventoryEntryDraft>,
  };
};
