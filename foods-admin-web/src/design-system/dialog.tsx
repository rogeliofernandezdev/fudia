"use client";

import {useEffect,useRef,type HTMLAttributes} from "react";

const stack:HTMLElement[]=[];
const responseClosers=new Map<HTMLElement,()=>void>();
/** Replace the entire action stack with its API response, from child to parent. */
export function closeDialogsForFeedback(){
  for(const panel of [...stack].reverse())responseClosers.get(panel)?.();
}
let previousOverflow="";
let rootOpener:HTMLElement|null=null;
const focusableSelector='button:not(:disabled),a[href],input:not(:disabled),select:not(:disabled),textarea:not(:disabled),[tabindex]:not([tabindex="-1"])';

/** Shared modal behavior; presentation and close actions remain owned by callers. */
export function Dialog({as:Element="section",children,onResponseClose,...props}:HTMLAttributes<HTMLElement>&{as?:"section"|"div";onResponseClose?:()=>void}){
  const ref=useRef<HTMLElement>(null);
  const responseClose=useRef(onResponseClose);
  useEffect(()=>{responseClose.current=onResponseClose;},[onResponseClose]);
  const openingFocus=useRef(typeof document!=="undefined"&&document.activeElement instanceof HTMLElement?document.activeElement:null);
  useEffect(()=>{
    const panel=ref.current;
    if(!panel)return;
    const opener=openingFocus.current;
    if(!stack.length){rootOpener=opener;previousOverflow=document.body.style.overflow;document.body.style.overflow="hidden";}
    stack.push(panel);
    responseClosers.set(panel,()=>responseClose.current?.());
    const controls=()=>Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector)).filter(element=>element.getClientRects().length&&!element.closest('[hidden],[inert],[aria-hidden="true"]'));
    const focusInitial=()=>{
      if(panel.contains(document.activeElement))return;
      (panel.querySelector<HTMLElement>("[data-dialog-initial-focus]")??controls()[0]??panel).focus({preventScroll:true});
    };
    focusInitial();
    const ownsPortal=(target:Node)=>Array.from(panel.querySelectorAll<HTMLElement>("[aria-controls]")).some(control=>control.getAttribute("aria-controls")?.split(" ").some(id=>document.getElementById(id)?.contains(target)));
    const focus=(event:FocusEvent)=>{
      if(stack.at(-1)!==panel||!(event.target instanceof Node)||panel.contains(event.target)||ownsPortal(event.target))return;
      (controls()[0]??panel).focus({preventScroll:true});
    };
    const keydown=(event:KeyboardEvent)=>{
      if(stack.at(-1)!==panel||event.defaultPrevented)return;
      if(event.key!=="Tab")return;
      const items=controls(),first=items[0],last=items.at(-1);
      if(!first){event.preventDefault();panel.focus();return;}
      if(event.shiftKey&&(document.activeElement===first||document.activeElement===panel)){event.preventDefault();last?.focus();}
      else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus();}
    };
    document.addEventListener("focusin",focus);
    document.addEventListener("keydown",keydown);
    return()=>{
      document.removeEventListener("focusin",focus);
      document.removeEventListener("keydown",keydown);
      const wasTop=stack.at(-1)===panel;
      const index=stack.indexOf(panel);
      if(index>=0)stack.splice(index,1);
      responseClosers.delete(panel);
      if(!stack.length)document.body.style.overflow=previousOverflow;
      if(wasTop){const next=stack.at(-1);const target=opener?.isConnected&&(!next||next.contains(opener))?opener:next??(rootOpener?.isConnected?rootOpener:null);target?.focus({preventScroll:true});}
      if(!stack.length)rootOpener=null;
    };
  },[]);
  return <Element {...props} ref={element=>{ref.current=element;}} tabIndex={-1} role={props.role??"dialog"} aria-modal="true">{children}</Element>;
}
