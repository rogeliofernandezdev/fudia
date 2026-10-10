import type {StylesConfig} from "react-select";

/** Shared presentation for single-value lookup selects. */
export function createLookupSelectStyles<Option>(overrides:StylesConfig<Option,false>={}):StylesConfig<Option,false>{
  return {
    control:(base,state)=>({...base,height:"var(--control-height)",minHeight:"var(--control-height)",borderColor:state.isFocused?"var(--primary-600)":"var(--line)",borderRadius:"var(--radius-control)",backgroundColor:"var(--surface)",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-semibold)",boxShadow:state.isFocused?"var(--ring-focus)":"none","&:hover":{borderColor:state.isFocused?"var(--primary-600)":"var(--line-strong)"}}),
    valueContainer:base=>({...base,height:"var(--control-height)",padding:"0 var(--space-2-5)"}),
    input:base=>({...base,color:"var(--ink-950)",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-semibold)",margin:0,padding:0}),
    singleValue:base=>({...base,color:"var(--ink-950)",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-semibold)"}),
    placeholder:base=>({...base,color:"var(--ink-500)",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-regular)"}),
    indicatorSeparator:base=>({...base,display:"none"}),
    dropdownIndicator:base=>({...base,color:"var(--ink-500)",padding:"0 var(--space-2)"}),
    menu:base=>({...base,zIndex:20,border:"var(--stroke-1) solid var(--line)",borderRadius:"var(--radius-control)",backgroundColor:"var(--surface)",boxShadow:"var(--elevation-1)",overflow:"hidden"}),
    menuPortal:base=>({...base,zIndex:200}),
    option:(base,state)=>({...base,cursor:"pointer",padding:"var(--space-2) var(--space-2-5)",fontSize:"var(--font-size-control)",fontWeight:"var(--weight-semibold)",backgroundColor:state.isSelected?"var(--primary-600)":state.isFocused?"var(--primary-100)":"var(--surface)",color:state.isSelected?"var(--surface)":"var(--ink-700)"}),
    noOptionsMessage:base=>({...base,color:"var(--ink-500)",fontSize:"var(--font-size-label)"}),
    ...overrides,
  };
}
