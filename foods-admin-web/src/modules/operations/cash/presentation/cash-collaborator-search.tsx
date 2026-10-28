"use client";
import SearchSelect,{createFilter} from "react-select";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import type {CashUserOption} from "../domain/types";

const styles=createLookupSelectStyles<CashUserOption>({
  option:(base,state)=>({...base,minHeight:"var(--size-44)",display:"flex",alignItems:"center",overflowWrap:"anywhere",backgroundColor:state.isSelected?"var(--primary-600)":state.isFocused?"var(--primary-100)":"var(--surface)",color:state.isSelected?"var(--surface)":"var(--ink-700)"}),
});
const filterOption=createFilter<CashUserOption>({ignoreAccents:true,ignoreCase:true,trim:true,stringify:option=>option.label});

/** Busca en el equipo del local ya autorizado por el API, sin peticiones por tecla. */
export function CashCollaboratorSearch({options,value,onChange,disabled,loading}:{options:CashUserOption[];value:string;onChange:(id:string)=>void;disabled:boolean;loading:boolean}){
  return <SearchSelect<CashUserOption,false>
    inputId="cash-team-user" instanceId="cash-team-user-search"
    aria-label="Agregar al equipo"
    options={options} getOptionLabel={option=>option.name} getOptionValue={option=>option.id}
    value={options.find(option=>option.id===value)??null}
    onChange={option=>onChange(option?.id??"")}
    filterOption={filterOption}
    isSearchable isClearable isDisabled={disabled} isLoading={loading}
    placeholder={loading?"Cargando usuarios…":options.length?"Escribe un nombre…":"No hay usuarios disponibles"}
    loadingMessage={()=>"Cargando usuarios…"}
    noOptionsMessage={({inputValue})=>inputValue.trim()?"No hay colaboradores con ese nombre":"No hay usuarios disponibles"}
    styles={styles} className="react-select-container" classNamePrefix="rs"
    menuPortalTarget={typeof document==="undefined"?undefined:document.body}
    menuPosition="fixed" menuPlacement="auto"
  />;
}
