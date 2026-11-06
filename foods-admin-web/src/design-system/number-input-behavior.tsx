"use client";

import {useEffect} from "react";

type Bounds={min?:string;max?:string;decimal?:boolean};
const finite=(value?:string)=>value?.trim()&&Number.isFinite(Number(value))?Number(value):undefined;
const precision=(value:string)=>{const [coefficient,exponent="0"]=value.toLowerCase().split("e");return Math.max(0,(coefficient.split(".")[1]?.length??0)-Number(exponent));};

/** Arrow increments are independent of the precision allowed for manually entered decimals. */
export function stepNumberValue(value:string,direction:1|-1,{min,max,decimal=false}:Bounds={}):string{
  const current=value.trim()===""?0:Number(value);
  if(!Number.isFinite(current))return value;
  const lower=finite(min),upper=finite(max);
  if(lower!==undefined&&upper!==undefined&&lower>upper)return value;
  let next=current+direction;
  let boundPrecision=0;
  if(lower!==undefined&&next<lower){next=lower;boundPrecision=precision(min!);}
  if(upper!==undefined&&next>upper){next=upper;boundPrecision=precision(max!);}
  const digits=Math.min(12,Math.max(decimal?1:0,precision(value),boundPrecision));
  return Number(next.toFixed(12)).toFixed(digits);
}

/** Keeps native inputs and their validation; only their arrow behavior changes. */
export function NumberInputBehavior(){
  useEffect(()=>{
    let pointer:{input:HTMLInputElement;value:string;direction:1|-1}|null=null;
    let releaseTimer:ReturnType<typeof setTimeout>|undefined;
    let writing=false;
    const numberInput=(target:EventTarget|null)=>target instanceof HTMLInputElement&&target.type==="number"&&!target.disabled&&!target.readOnly?target:null;
    const next=(input:HTMLInputElement,value:string,direction:1|-1)=>stepNumberValue(value,direction,{min:input.min,max:input.max,decimal:input.inputMode==="decimal"});
    const write=(input:HTMLInputElement,value:string)=>Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,"value")?.set?.call(input,value);
    const press=(event:PointerEvent)=>{
      const input=numberInput(event.target);
      if(!input){pointer=null;return;}
      const rect=input.getBoundingClientRect();
      pointer={input,value:input.value,direction:event.clientY<rect.top+rect.height/2?1:-1};
    };
    const release=()=>{
      const previous=pointer;
      clearTimeout(releaseTimer);
      releaseTimer=setTimeout(()=>{if(pointer===previous)pointer=null;},0);
    };
    const input=(event:Event)=>{
      if(writing||!event.isTrusted||!pointer||event.target!==pointer.input)return;
      const target=numberInput(event.target);
      if(!target)return;
      const before=Number(pointer.value),after=target.valueAsNumber;
      const direction=pointer.value!==""&&Number.isFinite(after)&&after!==before?(after>before?1:-1):pointer.direction;
      const value=next(target,pointer.value,direction);
      write(target,value);pointer.value=value;
    };
    const key=(event:KeyboardEvent)=>{
      if(event.defaultPrevented||event.altKey||event.ctrlKey||event.metaKey||!["ArrowUp","ArrowDown"].includes(event.key))return;
      const target=numberInput(event.target);
      if(!target)return;
      event.preventDefault();
      const value=next(target,target.value,event.key==="ArrowUp"?1:-1);
      if(value===target.value)return;
      writing=true;
      try{write(target,value);target.dispatchEvent(new Event("input",{bubbles:true}));}
      finally{writing=false;}
      if(pointer?.input===target)pointer.value=value;
    };
    document.addEventListener("pointerdown",press,true);
    document.addEventListener("pointerup",release,true);
    document.addEventListener("pointercancel",release,true);
    document.addEventListener("input",input,true);
    document.addEventListener("keydown",key);
    return()=>{
      clearTimeout(releaseTimer);
      document.removeEventListener("pointerdown",press,true);
      document.removeEventListener("pointerup",release,true);
      document.removeEventListener("pointercancel",release,true);
      document.removeEventListener("input",input,true);
      document.removeEventListener("keydown",key);
    };
  },[]);
  return null;
}
