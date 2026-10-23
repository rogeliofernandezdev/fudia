"use client";
import dynamic from "next/dynamic";
import {Skeleton} from "@/design-system/skeleton";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import {searchInventoryProducts} from "../infrastructure/inventory-api";
import type {InventoryProductOption} from "../domain/types";

// react-select + emotion only load when a lookup is on screen.
const AsyncSelect=dynamic(()=>import("react-select/async"),{ssr:false,loading:()=><Skeleton height="var(--control-height)"/>}) as unknown as typeof import("react-select/async").default;

type LookupOption={value:string;label:string;item:InventoryProductOption};
const lookupStyles=createLookupSelectStyles<LookupOption>({
  clearIndicator:base=>({...base,color:"var(--ink-400)",padding:"0 var(--space-1-5)","&:hover":{color:"var(--danger-600)"}}),
  loadingMessage:base=>({...base,color:"var(--ink-500)",fontSize:"var(--font-size-label)"}),
});

function toOption(item:InventoryProductOption):LookupOption{
  return {value:item.id,label:item.name+(item.kind==="ingredient"?" · Insumo":""),item};
}

// The server searches every inventory item of the company by name or SKU, so the
// picker is not limited to the first page of the catalog.
async function loadOptions(q:string):Promise<LookupOption[]>{
  const response=await searchInventoryProducts(q);
  return response.items.map(toOption);
}

export function InventoryItemLookup({value,onChange,ariaLabel,placeholder="Buscar artículo...",clearable=false,invalid=false,disabled=false,excludeIds=[],autoFocus=false}:{
  value:InventoryProductOption|null;
  onChange:(item:InventoryProductOption|null)=>void;
  ariaLabel:string;
  placeholder?:string;
  clearable?:boolean;
  invalid?:boolean;
  disabled?:boolean;
  excludeIds?:string[];
  autoFocus?:boolean;
}){
  return <AsyncSelect<LookupOption,false>
    cacheOptions
    defaultOptions
    loadOptions={loadOptions}
    value={value?toOption(value):null}
    onChange={option=>onChange(option?.item??null)}
    isOptionDisabled={option=>excludeIds.includes(option.value)}
    isDisabled={disabled}
    isClearable={clearable}
    isSearchable
    autoFocus={autoFocus}
    aria-label={ariaLabel}
    aria-invalid={invalid}
    placeholder={placeholder}
    loadingMessage={()=>"Buscando artículos..."}
    noOptionsMessage={({inputValue})=>inputValue.trim()?"No hay coincidencias":"No hay artículos de inventario"}
    styles={lookupStyles}
    className={"react-select-container inventory-item-lookup"+(invalid?" invalid":"")}
    classNamePrefix="rs"
    menuPortalTarget={typeof document==="undefined"?undefined:document.body}
    menuPosition="fixed"
  />;
}
