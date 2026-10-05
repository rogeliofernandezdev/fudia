"use client";

import {useMemo,useId,type Ref} from "react";
import SearchSelect,{type SelectInstance,type StylesConfig} from "react-select";
import {timezoneOptions,type TimezoneOption} from "@/shared/format/timezones";

const styles:StylesConfig<TimezoneOption,false>={
  control:(base,state)=>({...base,minHeight:"var(--control-height)",height:"var(--control-height)",borderRadius:6,borderColor:state.selectProps["aria-invalid"]?"var(--danger)":state.isFocused?"var(--primary-600)":"var(--line)",boxShadow:state.isFocused?"0 0 0 2px var(--primary-100)":"none",backgroundColor:"var(--surface)",fontSize:11,"&:hover":{borderColor:"var(--primary-600)"}}),
  valueContainer:base=>({...base,padding:"0 10px"}),
  input:base=>({...base,margin:0,padding:0,color:"var(--ink-950)",fontSize:11}),
  singleValue:base=>({...base,color:"var(--ink-950)",fontSize:11,fontWeight:600}),
  placeholder:base=>({...base,color:"var(--ink-400)",fontSize:11,fontWeight:400}),
  indicatorSeparator:base=>({...base,display:"none"}),
  dropdownIndicator:base=>({...base,padding:"0 9px"}),
  menu:base=>({...base,backgroundColor:"var(--surface)",borderRadius:6,overflow:"hidden"}),
  menuPortal:base=>({...base,zIndex:200}),
  option:(base,state)=>({...base,minHeight:"var(--control-height)",display:"flex",alignItems:"center",fontSize:11,overflowWrap:"anywhere",backgroundColor:state.isSelected?"var(--primary-600)":state.isFocused?"var(--primary-100)":"var(--surface)",color:state.isSelected?"var(--surface)":"var(--ink-700)"}),
  noOptionsMessage:base=>({...base,fontSize:11}),
};

type Props={
  value:string;
  onChange:(value:string)=>void;
  onBlur?:()=>void;
  name?:string;
  disabled?:boolean;
  invalid?:boolean;
  inputRef?:Ref<SelectInstance<TimezoneOption,false>>;
};

export function TimezoneSelect({value,onChange,onBlur,name,disabled,invalid,inputRef}:Props){
  const id=useId();
  const options=useMemo(()=>timezoneOptions(value),[value]);
  return <SearchSelect<TimezoneOption,false>
    ref={inputRef} inputId={`${id}-timezone`} instanceId={id} name={name}
    aria-label="Zona horaria" aria-invalid={invalid}
    options={options} value={options.find(option=>option.value===value)??null}
    onChange={option=>onChange(option?.value??"")} onBlur={onBlur}
    isSearchable isClearable={false} isDisabled={disabled}
    placeholder="Buscar ciudad o zona..." noOptionsMessage={()=>"No hay coincidencias"}
    styles={styles} className="react-select-container" classNamePrefix="rs"
    menuPortalTarget={typeof document==="undefined"?undefined:document.body} menuPosition="fixed"
  />;
}
