"use client";
import {Dialog} from "@/design-system/dialog";
import "./styles/feedback.css";
import {createContext,useCallback,useContext,useEffect,useMemo,useRef,useState} from "react";import {Icon} from "@/design-system/icons";
type Tone="success"|"danger"|"info";type Input={tone:Tone;title:string;message:string;duration?:number};type Item=Input&{id:number};
const Context=createContext<{notify:(input:Input)=>void}|null>(null);
export function FeedbackProvider({children}:{children:React.ReactNode}){
  const [item,setItem]=useState<Item|null>(null);
  const [closing,setClosing]=useState(false);
  const sequence=useRef(0);
  const button=useRef<HTMLButtonElement>(null);
  const exitTimer=useRef<number|undefined>(undefined);
  const close=useCallback(()=>{
    window.clearTimeout(exitTimer.current);
    setClosing(true);
    exitTimer.current=window.setTimeout(()=>{setItem(null);setClosing(false);},180);
  },[]);
  const notify=useCallback((input:Input)=>{
    window.clearTimeout(exitTimer.current);
    setClosing(false);
    setItem({...input,id:++sequence.current});
  },[]);
  useEffect(()=>()=>window.clearTimeout(exitTimer.current),[]);
  useEffect(()=>{
    if(!item)return;
    button.current?.focus();
    if(item.tone!=="success")return;
    const timer=window.setTimeout(close,item.duration??4200);
    return()=>window.clearTimeout(timer);
  },[item,close]);
  const value=useMemo(()=>({notify}),[notify]);
  return <Context.Provider value={value}>{children}{item&&<div className={`feedback-overlay ${closing?"out":"in"}`}>
    <Dialog className={`feedback-card ${item.tone} ${closing?"out":"in"}`} role={item.tone==="danger"?"alertdialog":"dialog"} aria-modal="true" aria-labelledby={`feedback-title-${item.id}`} aria-describedby={`feedback-message-${item.id}`}>
      <button type="button" className="feedback-close" onClick={close} aria-label="Cerrar notificación"><Icon name="close"/></button>
      <span className={`feedback-icon ${item.tone}`}><Icon name={item.tone==="success"?"check":"alert"} size={36}/></span>
      <h2 id={`feedback-title-${item.id}`}>{item.title}</h2><p id={`feedback-message-${item.id}`}>{item.message}</p>
      <button type="button" ref={button} data-dialog-initial-focus className="feedback-accept" onClick={close}>Aceptar</button>
    </Dialog>
  </div>}</Context.Provider>;
}
export function useFeedback(){const value=useContext(Context);if(!value)throw new Error("useFeedback debe utilizarse dentro de FeedbackProvider");return value}
