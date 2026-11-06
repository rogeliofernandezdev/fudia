"use client";

import {useCallback,useEffect,useId,useRef,useState} from "react";
import {createPortal} from "react-dom";
import {placeTooltip} from "./tooltip";

type LabelTarget={element:HTMLElement;text:string};

/** One layer for every data-tooltip label, including controls added by remote lists. */
export function TooltipLayer(){
  const id=useId();
  const bubble=useRef<HTMLDivElement>(null);
  const [active,setActive]=useState<LabelTarget|null>(null);
  const [position,setPosition]=useState<ReturnType<typeof placeTooltip>|null>(null);
  const hide=useCallback(()=>{setActive(null);setPosition(null)},[]);
  const show=useCallback((target:EventTarget|null)=>{
    const element=target instanceof Element?target.closest<HTMLElement>("[data-tooltip]"):null;
    const text=element?.dataset.tooltip?.trim();
    if(!element||!text)return;
    setActive(current=>current?.element===element&&current.text===text?current:{element,text});
  },[]);

  useEffect(()=>{
    const enter=(event:PointerEvent)=>{if(event.pointerType==="mouse")show(event.target)};
    const leave=(event:PointerEvent)=>{
      const element=event.target instanceof Element?event.target.closest("[data-tooltip]"):null;
      if(element&&!(event.relatedTarget instanceof Node&&element.contains(event.relatedTarget)))hide();
    };
    const focus=(event:FocusEvent)=>{if(event.target instanceof Element&&event.target.matches(":focus-visible"))show(event.target)};
    document.addEventListener("pointerover",enter);
    document.addEventListener("pointerout",leave);
    document.addEventListener("focusin",focus);
    document.addEventListener("focusout",hide);
    document.addEventListener("pointerdown",hide,true);
    window.addEventListener("scroll",hide,true);
    return()=>{
      document.removeEventListener("pointerover",enter);
      document.removeEventListener("pointerout",leave);
      document.removeEventListener("focusin",focus);
      document.removeEventListener("focusout",hide);
      document.removeEventListener("pointerdown",hide,true);
      window.removeEventListener("scroll",hide,true);
    };
  },[show,hide]);

  const measure=useCallback(()=>{
    if(!active?.element.isConnected||!bubble.current){hide();return;}
    const anchor=active.element.getBoundingClientRect();
    const box=bubble.current.getBoundingClientRect();
    if(!anchor.width||!anchor.height){hide();return;}
    setPosition(placeTooltip({left:anchor.left+(anchor.width-box.width)/2,top:anchor.top,bottom:anchor.bottom},box,{width:document.documentElement.clientWidth,height:window.innerHeight}));
  },[active,hide]);
  const bindBubble=useCallback((element:HTMLDivElement|null)=>{
    bubble.current=element;
    if(element)measure();
  },[measure]);

  useEffect(()=>{
    if(!active)return;
    const {element}=active;
    const describe=()=>{
      const ids=new Set((element.getAttribute("aria-describedby")??"").split(/\s+/).filter(Boolean));
      ids.add(id);element.setAttribute("aria-describedby",[...ids].join(" "));
    };
    describe();
    const escape=(event:KeyboardEvent)=>{if(event.key==="Escape"){event.stopPropagation();hide()}};
    const observer=new MutationObserver(()=>{
      if(!element.isConnected||!element.dataset.tooltip?.trim()){hide();return;}
      if(element.dataset.tooltip.trim()!==active.text)show(element);
    });
    observer.observe(document.body,{childList:true,subtree:true,attributes:true,attributeFilter:["data-tooltip"]});
    document.addEventListener("keydown",escape,true);
    window.addEventListener("resize",measure);
    return()=>{
      observer.disconnect();
      const ids=(element.getAttribute("aria-describedby")??"").split(/\s+/).filter(value=>value&&value!==id);
      if(ids.length)element.setAttribute("aria-describedby",ids.join(" "));else element.removeAttribute("aria-describedby");
      document.removeEventListener("keydown",escape,true);
      window.removeEventListener("resize",measure);
    };
  },[active,id,hide,show,measure]);

  if(!active)return null;
  return createPortal(<div ref={bindBubble} id={id} role="tooltip" className="ds-tooltip ds-label-tooltip" data-ready={position?"true":undefined} data-placement={position?.placement} style={position?{left:position.left,top:position.top}:undefined}>{active.text}</div>,document.body);
}
