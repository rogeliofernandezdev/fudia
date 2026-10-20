"use client";
import SearchSelect,{createFilter} from "react-select";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import type {Product} from "../domain/types";
import {buildOptionGroups,type ComboPickGroup,type ComboPickOption} from "../domain/option-groups";

const pickStyles=createLookupSelectStyles<ComboPickOption>({
  control:(base,state)=>({...base,minHeight:"var(--control-height)",borderStyle:state.isFocused?"solid":"dashed",borderColor:state.isFocused?"var(--primary-600)":"var(--line-strong)",borderRadius:"var(--radius-control)",backgroundColor:"var(--surface)",fontSize:"var(--font-size-control)",boxShadow:state.isFocused?"var(--ring-focus)":"none",cursor:"text","&:hover":{borderColor:state.isFocused?"var(--primary-600)":"var(--primary-300)"}}),
  placeholder:base=>({...base,color:"var(--ink-600)",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-semibold)"}),
  menuList:base=>({...base,maxHeight:"min(var(--size-320), 45dvh)",padding:"var(--space-1) 0"}),
  group:base=>({...base,padding:"var(--space-1) 0"}),
  groupHeading:base=>({...base,margin:0,padding:"var(--space-1-5) var(--space-3)",color:"var(--ink-800)",fontSize:"var(--font-size-label)",fontWeight:"var(--weight-bold)",letterSpacing:"0",textTransform:"none"}),
  option:(base,state)=>({...base,display:"flex",alignItems:"center",justifyContent:"space-between",gap:"var(--space-3)",minHeight:"var(--size-44)",padding:"var(--space-2) var(--space-3) var(--space-2) var(--space-5)",cursor:"pointer",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-regular)",color:"var(--ink-800)",backgroundColor:state.isFocused?"var(--primary-50)":"var(--surface)","&:active":{backgroundColor:"var(--primary-100)"}}),
});

// Busca por nombre o categoría, sin distinguir mayúsculas ni tildes.
const filterOption=createFilter<ComboPickOption>({ignoreAccents:true,ignoreCase:true,trim:true,stringify:option=>`${option.label} ${option.data.category}`});

function stockLabel(limit:number|null){
  if(limit===null)return null;
  return limit===0?<small className="combo-pick-stock sold-out">Agotado</small>:<small className="combo-pick-stock">{limit} disp.</small>;
}

/**
 * Buscador «Agregar opción»: filtra la carta al escribir, conserva las secciones por categoría (primero las de la
 * parte) y muestra el saldo de cada plato. Al elegir agrega la opción, limpia la búsqueda y se cierra: si quedaba
 * abierto, el resaltado saltaba al plato siguiente y la lista crecía bajo el cursor. El foco sigue en el buscador, así
 * que escribir o pulsar ↓ lo vuelve a abrir; los productos ya agregados no se repiten.
 */
export function ComboOptionSearch({inputId,partName,products,excluded,invalid,describedBy,onPick}:{inputId:string;partName:string;products:Product[];excluded:string[];invalid:boolean;describedBy?:string;onPick:(product:Product)=>void}){
  const groups:ComboPickGroup[]=buildOptionGroups(products,excluded,partName);
  return <SearchSelect<ComboPickOption,false>
    inputId={inputId}
    instanceId={`${inputId}-picker`}
    options={groups}
    value={null}
    onChange={option=>{const product=option&&products.find(item=>item.id===option.value);if(product)onPick(product)}}
    filterOption={filterOption}
    formatOptionLabel={option=><><span className="combo-pick-name">{option.label}</span>{stockLabel(option.limit)}</>}
    placeholder="+ Agregar opción: busca un plato…"
    noOptionsMessage={({inputValue})=>inputValue.trim()?"No hay productos con ese nombre":"Ya agregaste todos los productos"}
    aria-label={`Agregar opción a ${partName.trim()||"esta parte"}`}
    aria-invalid={invalid}
    aria-errormessage={invalid?describedBy:undefined}
    closeMenuOnSelect
    blurInputOnSelect={false}
    isSearchable
    styles={pickStyles}
    className={`react-select-container combo-pick${invalid?" invalid":""}`}
    classNamePrefix="rs"
    menuPortalTarget={typeof document==="undefined"?undefined:document.body}
    menuPosition="fixed"
    menuPlacement="auto"
  />;
}
