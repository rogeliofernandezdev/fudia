import type {StylesConfig} from "react-select";

/** Shared presentation for single-value lookup selects. */
export function createLookupSelectStyles<Option>(overrides:StylesConfig<Option,false>={}):StylesConfig<Option,false>{
  return {
    control:base=>({...base,height:"var(--control-height)",minHeight:"var(--control-height)",borderColor:"var(--line)",borderRadius:"var(--radius-control)",backgroundColor:"var(--surface)",fontSize:"var(--font-size-control)",fontWeight:"var(--font-weight-600)",boxShadow:"none","&:hover":{borderColor:"var(--line-strong)"}}),
    valueContainer:base=>({...base,height:"var(--control-height)",padding:"0 var(--space-10)"}),
    input:base=>({...base,color:"var(--ink-950)",fontSize:"var(--font-size-control)",fontWeight:"var(--font-weight-600)",margin:0,padding:0}),
    singleValue:base=>({...base,color:"var(--ink-950)",fontSize:"var(--font-size-control)",fontWeight:"var(--font-weight-600)"}),
    placeholder:base=>({...base,color:"var(--ink-400)",fontSize:"var(--font-size-control)",fontWeight:"var(--font-weight-400)"}),
    indicatorSeparator:base=>({...base,display:"none"}),
    dropdownIndicator:base=>({...base,color:"var(--ink-500)",padding:"0 var(--space-9)"}),
    menu:base=>({...base,zIndex:20,border:"var(--stroke-1) solid var(--line)",borderRadius:"var(--radius-control)",backgroundColor:"var(--surface)",boxShadow:"var(--shadow)",overflow:"hidden"}),
    menuPortal:base=>({...base,zIndex:200}),
    option:(base,state)=>({...base,cursor:"pointer",padding:"var(--space-8) var(--space-10)",fontSize:"var(--font-size-control)",fontWeight:"var(--font-weight-600)",backgroundColor:state.isSelected?"var(--primary-600)":state.isFocused?"var(--primary-100)":"var(--surface)",color:state.isSelected?"var(--surface)":"var(--ink-700)"}),
    noOptionsMessage:base=>({...base,color:"var(--ink-500)",fontSize:"var(--font-size-label)"}),
    ...overrides,
  };
}
