"use client";

import {useEffect,useId,useRef,useState} from "react";
import type {ComponentPropsWithoutRef,ReactNode} from "react";

type DataTableProps = Omit<ComponentPropsWithoutRef<"table">,"children"> & {
  children: ReactNode;
  label: string;
  containerClassName?: string;
};

/** A native table with accessible horizontal scrolling and no mobile card substitution. */
export function DataTable({label,containerClassName="",children,className="",...tableProps}:DataTableProps){
  const viewportRef=useRef<HTMLDivElement>(null);
  const hintId=useId();
  const [hasOverflow,setHasOverflow]=useState(false);
  const [atEnd,setAtEnd]=useState(false);

  useEffect(()=>{
    const viewport=viewportRef.current;
    if(!viewport)return;
    const update=()=>{
      const overflowing=viewport.scrollWidth>viewport.clientWidth+1;
      setHasOverflow(overflowing);
      setAtEnd(!overflowing||viewport.scrollLeft+viewport.clientWidth>=viewport.scrollWidth-1);
    };
    update();
    const observer=typeof ResizeObserver==="undefined"?null:new ResizeObserver(update);
    observer?.observe(viewport);
    const table=viewport.querySelector("table");
    if(table)observer?.observe(table);
    viewport.addEventListener("scroll",update,{passive:true});
    window.addEventListener("resize",update);
    return ()=>{
      observer?.disconnect();
      viewport.removeEventListener("scroll",update);
      window.removeEventListener("resize",update);
    };
  },[]);

  const viewportClassName=["table-wrap","data-table-viewport","min-w-0","max-w-full",containerClassName].filter(Boolean).join(" ");
  return (
    <div className="data-table-shell min-w-0 max-w-full">
      {hasOverflow&&(
        <p id={hintId} className="data-table-hint">
          <span aria-hidden="true">{atEnd?"←":"↔"}</span>
          <span>{atEnd?"Desliza a la izquierda para ver las primeras columnas":"Desliza horizontalmente para ver todas las columnas"}</span>
        </p>
      )}
      <div ref={viewportRef} className={viewportClassName} role="region" aria-label={label} aria-describedby={hasOverflow?hintId:undefined} tabIndex={0}>
        <table {...tableProps} className={["data-table",className].filter(Boolean).join(" ")}>{children}</table>
      </div>
    </div>
  );
}
