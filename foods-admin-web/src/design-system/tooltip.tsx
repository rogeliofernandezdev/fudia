"use client";

import {useCallback,useEffect,useId,useLayoutEffect,useRef,useState,type ReactNode} from "react";
import {createPortal} from "react-dom";
import "./styles/tooltip.css";

/** Separación entre el disparador y la burbuja, y margen mínimo respecto del viewport. */
const GAP=8;
const VIEWPORT_MARGIN=16;

type Placement="top"|"bottom";
type Position={left:number;top:number;placement:Placement};

/** Coloca la burbuja sobre el disparador (o debajo si no cabe) y la acota al viewport. */
export function placeTooltip(trigger:{left:number;top:number;bottom:number},bubble:{width:number;height:number},viewport:{width:number;height:number}):Position{
  const above=trigger.top-GAP-bubble.height;
  const below=trigger.bottom+GAP;
  const fitsAbove=above>=VIEWPORT_MARGIN;
  const fitsBelow=below+bubble.height<=viewport.height-VIEWPORT_MARGIN;
  const placement:Placement=fitsAbove||(!fitsBelow&&trigger.top>viewport.height-trigger.bottom)?"top":"bottom";
  const maxLeft=Math.max(VIEWPORT_MARGIN,viewport.width-VIEWPORT_MARGIN-bubble.width);
  const maxTop=Math.max(VIEWPORT_MARGIN,viewport.height-VIEWPORT_MARGIN-bubble.height);
  return{
    placement,
    left:Math.min(Math.max(trigger.left,VIEWPORT_MARGIN),maxLeft),
    top:Math.min(Math.max(placement==="top"?above:below,VIEWPORT_MARGIN),maxTop),
  };
}

/**
 * Tooltip de contenido (texto de varias líneas o listas) para un disparador visual.
 * Abre con hover de puntero fino, foco de teclado y toque o pulsación larga en táctil;
 * se cierra con Escape o al tocar fuera, y acompaña al disparador al desplazar. Las etiquetas de una sola línea
 * de botones iconográficos usan `[data-tooltip]` y la capa global `TooltipLayer`.
 */
export function Tooltip({content,label,children,className=""}:{
  content:ReactNode;
  /** Nombre accesible del disparador: «Ver descripción y alérgenos de Ají de gallina». */
  label:string;
  children:ReactNode;
  className?:string;
}){
  const id=useId();
  const trigger=useRef<HTMLButtonElement>(null);
  const bubble=useRef<HTMLDivElement>(null);
  const pointer=useRef("");
  const[open,setOpen]=useState(false);
  const[position,setPosition]=useState<Position|null>(null);
  const show=useCallback(()=>setOpen(true),[]);
  const hide=useCallback(()=>{setOpen(false);setPosition(null)},[]);

  const measure=useCallback(()=>{
    const anchor=trigger.current?.getBoundingClientRect();
    const box=bubble.current?.getBoundingClientRect();
    if(!anchor||!box)return;
    setPosition(placeTooltip(anchor,box,{width:document.documentElement.clientWidth,height:window.innerHeight}));
  },[]);
  useLayoutEffect(()=>{if(open)measure()},[open,measure]);

  useEffect(()=>{
    if(!open)return;
    const outside=(event:PointerEvent)=>{
      const target=event.target as Node;
      if(!trigger.current?.contains(target)&&!bubble.current?.contains(target))hide();
    };
    const escape=(event:KeyboardEvent)=>{if(event.key==="Escape"){event.stopPropagation();hide()}};
    document.addEventListener("pointerdown",outside,true);
    document.addEventListener("keydown",escape,true);
    window.addEventListener("scroll",measure,true);
    window.addEventListener("resize",measure);
    return()=>{
      document.removeEventListener("pointerdown",outside,true);
      document.removeEventListener("keydown",escape,true);
      window.removeEventListener("scroll",measure,true);
      window.removeEventListener("resize",measure);
    };
  },[open,hide,measure]);

  return(
    <>
      <button
        type="button"
        ref={trigger}
        className={`ds-tooltip-trigger ${className}`.trim()}
        aria-label={label}
        aria-describedby={open?id:undefined}
        onPointerDown={event=>{pointer.current=event.pointerType}}
        onPointerEnter={event=>{if(event.pointerType==="mouse")show()}}
        onPointerLeave={event=>{if(event.pointerType==="mouse")hide()}}
        onFocus={event=>{if(event.currentTarget.matches(":focus-visible"))show()}}
        onBlur={hide}
        onKeyDown={()=>{pointer.current="keyboard"}}
        // Un clic de mouse ya abrió con hover; toque y teclado alternan la burbuja.
        onClick={()=>{if(pointer.current!=="mouse")setOpen(value=>!value)}}
        // Pulsación larga: abre la burbuja en lugar del menú nativo de la imagen.
        onContextMenu={event=>{if(pointer.current!=="mouse"){event.preventDefault();show()}}}
      >
        {children}
      </button>
      {open&&createPortal(
        <div id={id} ref={bubble} role="tooltip" className="ds-tooltip" data-placement={position?.placement} data-ready={position?"true":undefined} style={position?{left:position.left,top:position.top}:undefined} onClick={hide}>
          {content}
        </div>,
        document.body,
      )}
    </>
  );
}
