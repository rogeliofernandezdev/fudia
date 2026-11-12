"use client";

import {useEffect,useId,useRef,useState} from "react";
import type {ComponentPropsWithoutRef,ReactNode} from "react";

type DataTableProps = Omit<ComponentPropsWithoutRef<"table">,"children"> & {
  children: ReactNode;
  label: string;
  containerClassName?: string;
  scrollbarPosition?: "bottom" | "top";
};

/** A native table with accessible horizontal scrolling and no mobile card substitution. */
export function DataTable({label,containerClassName="",scrollbarPosition="bottom",children,className="",...tableProps}:DataTableProps){
  const viewportRef=useRef<HTMLDivElement>(null);
  const topScrollbarRef=useRef<HTMLDivElement>(null);
  const scrollbarContentRef=useRef<HTMLDivElement>(null);
  const viewportId=useId();
  const hintId=useId();
  const [hasOverflow,setHasOverflow]=useState(false);
  const [atEnd,setAtEnd]=useState(false);

  useEffect(()=>{
    const viewport=viewportRef.current;
    if(!viewport)return;
    const update=()=>{
      const overflowing=viewport.scrollWidth>viewport.clientWidth+1;
      const topScrollbar=topScrollbarRef.current;
      if(scrollbarContentRef.current)scrollbarContentRef.current.style.width=`${viewport.scrollWidth}px`;
      if(topScrollbar&&topScrollbar.scrollLeft!==viewport.scrollLeft)topScrollbar.scrollLeft=viewport.scrollLeft;
      setHasOverflow(overflowing);
      setAtEnd(!overflowing||viewport.scrollLeft+viewport.clientWidth>=viewport.scrollWidth-1);
    };
    update();
    const observer=typeof ResizeObserver==="undefined"?null:new ResizeObserver(update);
    observer?.observe(viewport);
    const table=viewport.querySelector("table");
    if(table)observer?.observe(table);
    viewport.addEventListener("scroll",update,{passive:true});
    const topScrollbar=topScrollbarRef.current;
    const scrollFromTop=()=>{
      if(topScrollbar&&viewport.scrollLeft!==topScrollbar.scrollLeft)viewport.scrollLeft=topScrollbar.scrollLeft;
    };
    topScrollbar?.addEventListener("scroll",scrollFromTop,{passive:true});
    window.addEventListener("resize",update);
    return ()=>{
      observer?.disconnect();
      viewport.removeEventListener("scroll",update);
      topScrollbar?.removeEventListener("scroll",scrollFromTop);
      window.removeEventListener("resize",update);
    };
  },[scrollbarPosition]);

  const viewportClassName=["table-wrap","data-table-viewport","min-w-0","max-w-full",containerClassName,scrollbarPosition==="top"&&"data-table-viewport--top"].filter(Boolean).join(" ");
  return (
    <div className="data-table-shell min-w-0 max-w-full">
      {hasOverflow&&(
        <p id={hintId} className="data-table-hint">
          <span aria-hidden="true">{atEnd?"←":"↔"}</span>
          <span>{atEnd?"Desliza a la izquierda para ver las primeras columnas":"Desliza horizontalmente para ver todas las columnas"}</span>
        </p>
      )}
      {scrollbarPosition==="top"&&(
        <div ref={topScrollbarRef} className="data-table-scrollbar-top" hidden={!hasOverflow} role="region" aria-label={`Desplazamiento horizontal: ${label}`} aria-controls={viewportId} tabIndex={0}>
          <div ref={scrollbarContentRef} className="data-table-scrollbar-content"/>
        </div>
      )}
      <div id={viewportId} ref={viewportRef} className={viewportClassName} role="region" aria-label={label} aria-describedby={hasOverflow?hintId:undefined} tabIndex={0}>
        <table {...tableProps} className={["data-table",className].filter(Boolean).join(" ")}>{children}</table>
      </div>
    </div>
  );
}
