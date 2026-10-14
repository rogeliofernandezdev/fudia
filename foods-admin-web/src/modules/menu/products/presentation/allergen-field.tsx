"use client";
import {useState} from "react";
import AsyncCreatableSelect from "react-select/async-creatable";
import type {StylesConfig} from "react-select";
import {FormField} from "@/design-system";

export type AllergenOption={label:string;value:string};
const selectStyles:StylesConfig<AllergenOption,true>={control:(base)=>({...base,height:"var(--control-height)",minHeight:"var(--control-height)",borderColor:"var(--line)",borderRadius:"var(--radius-control)",fontSize:"var(--font-size-control)",fontWeight:"var(--control-value-weight,var(--weight-semibold))",boxShadow:"none",outline:"none","&:hover":{borderColor:"var(--line-strong)"}}),valueContainer:(base)=>({...base,padding:"0 var(--space-2-5)",height:"var(--control-height)"}),multiValue:(base)=>({...base,backgroundColor:"var(--primary-100)",borderRadius:"var(--radius-chip)",margin:"var(--space-0-5)"}),multiValueLabel:(base)=>({...base,color:"var(--primary-700)",fontSize:"var(--font-size-label)",fontWeight:"var(--control-value-weight,var(--weight-semibold))",padding:"var(--space-0-5) var(--space-1-5)"}),multiValueRemove:(base)=>({...base,color:"var(--primary-700)",borderRadius:"var(--radius-chip)",padding:0,width:"var(--size-18)","&:hover":{backgroundColor:"var(--danger)",color:"var(--surface)"}}),option:(base,state)=>({...base,fontSize:"var(--font-size-control)",fontWeight:"var(--control-value-weight,var(--weight-semibold))",backgroundColor:state.isSelected?"var(--primary-600)":state.isFocused?"var(--primary-100)":"transparent",color:state.isSelected?"#fff":"var(--ink-700)",cursor:"pointer",padding:"var(--space-1-5) var(--space-2-5)"}),menu:(base)=>({...base,zIndex:5,fontSize:"var(--font-size-control)",border:"var(--stroke-1) solid var(--line)",borderRadius:"var(--radius-control)",boxShadow:"var(--elevation-2)"}),input:(base)=>({...base,fontSize:"var(--font-size-control)",fontWeight:"var(--control-value-weight,var(--weight-semibold))",margin:0,padding:0,border:"none",outline:"none",boxShadow:"none",background:"transparent",opacity:1}),placeholder:(base)=>({...base,fontSize:"var(--font-size-control)",fontWeight:"var(--weight-regular)",color:"var(--ink-400)",margin:0,textAlign:"left"}),indicatorSeparator:(base)=>({...base,display:"none"}),dropdownIndicator:(base)=>({...base,padding:"0 var(--space-2)"})};

/** Mirrors normalize_allergen_name in the backend: trim, collapse spaces, lowercase, drop acute/diaeresis (ñ stays). */
export function allergenKey(name:string){return name.normalize("NFD").replace(/[\u0301\u0308]/g,"").normalize("NFC").trim().replace(/\s+/g," ").toLowerCase()}
const minLength=2,maxLength=60;

export function AllergenField({value,onChange,loadAllergens,createAllergen,disabled=false}:{value:string[];onChange:(next:string[])=>void;loadAllergens:(q:string)=>Promise<AllergenOption[]>;createAllergen?:(name:string)=>Promise<string>;disabled?:boolean}){
 const[creating,setCreating]=useState(false);
 const[error,setError]=useState("");
 const[cacheVersion,setCacheVersion]=useState(0);
 const selected=new Set(value.map(allergenKey));
 // Creation is offered only when no visible option or selected chip already has the same normalized name; the backend re-validates.
 function canCreate(input:string,options:readonly AllergenOption[]){
  const name=input.trim().replace(/\s+/g," ");
  if(!createAllergen||name.length<minLength||name.length>maxLength)return false;
  const key=allergenKey(name);
  return !selected.has(key)&&!options.some(option=>allergenKey(option.label)===key);
 }
 async function create(input:string){
  if(!createAllergen||creating)return;
  setCreating(true);setError("");
  try{const name=await createAllergen(input.trim().replace(/\s+/g," "));onChange([...value,name]);setCacheVersion(version=>version+1)}
  catch(failure){setError(failure instanceof Error?failure.message:"No pudimos registrar el alérgeno.")}
  finally{setCreating(false)}
 }
 return <FormField as="div" className="span-2" label="Alérgenos" error={error}><AsyncCreatableSelect<AllergenOption,true>
  isMulti cacheOptions={cacheVersion} loadOptions={loadAllergens} defaultOptions={false} isDisabled={disabled||creating} isLoading={creating}
  placeholder="Escribir para buscar alérgenos..." value={value.map(a=>({label:a,value:a}))}
  onChange={items=>{setError("");onChange(items.map(option=>option.value))}}
  onCreateOption={input=>void create(input)} isValidNewOption={(input,_value,options)=>canCreate(input,options as readonly AllergenOption[])}
  formatCreateLabel={input=>`Agregar «${input.trim().replace(/\s+/g," ")}»`} createOptionPosition="last"
  styles={selectStyles} className="react-select-container" classNamePrefix="rs" aria-invalid={Boolean(error)}
  loadingMessage={()=>"Buscando…"}
  noOptionsMessage={({inputValue})=>{const name=inputValue.trim();if(!name)return value.length?"No hay más coincidencias":"Escribe para buscar";if(selected.has(allergenKey(name)))return "Ya está seleccionado";if(name.length<minLength)return `Escribe al menos ${minLength} caracteres`;if(name.length>maxLength)return `Máximo ${maxLength} caracteres`;return "No hay coincidencias"}}/></FormField>
}
