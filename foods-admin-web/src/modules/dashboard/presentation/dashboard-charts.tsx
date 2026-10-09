"use client";
import {useId,useRef,useState,type KeyboardEvent,type PointerEvent} from "react";
import {Icon,type IconName} from "@/design-system/icons";

/** Redondea el máximo del eje a un valor limpio (1, 2, 2.5 o 5 × 10ⁿ). */
export function niceMax(value:number){
 if(value<=0)return 1;
 const magnitude=10**Math.floor(Math.log10(value));
 const step=[1,2,2.5,5,10].find(candidate=>candidate*magnitude>=value)??10;
 return step*magnitude;
}

export type TrendPoint={key:string;tick:string;title:string;value:number;orders:number};

/**
 * Curva monótona (Fritsch–Carlson) como trazado SVG: suaviza sin crear picos ni
 * valles que no existen en los datos, por lo que nunca baja de la línea base.
 */
function smoothPath(coords:Array<readonly [number,number]>){
 if(coords.length<3)return coords.map(([px,py],index)=>`${index?"L":"M"}${px},${py}`).join(" ");
 const slopes=coords.slice(1).map(([px,py],index)=>(py-coords[index][1])/(px-coords[index][0]));
 const tangents=coords.map((_,index)=>{
  if(index===0)return slopes[0];
  if(index===coords.length-1)return slopes[slopes.length-1];
  const before=slopes[index-1],after=slopes[index];
  return before*after<=0?0:(before+after)/2;
 });
 for(let index=0;index<slopes.length;index++){
  if(slopes[index]===0){tangents[index]=0;tangents[index+1]=0;continue}
  const a=tangents[index]/slopes[index],b=tangents[index+1]/slopes[index],h=a*a+b*b;
  if(h>9){const t=3/Math.sqrt(h);tangents[index]=t*a*slopes[index];tangents[index+1]=t*b*slopes[index]}
 }
 return coords.map(([px,py],index)=>{
  if(index===0)return `M${px},${py}`;
  const[px0,py0]=coords[index-1];
  const dx=(px-px0)/3;
  return `C${px0+dx},${py0+tangents[index-1]*dx} ${px-dx},${py-tangents[index]*dx} ${px},${py}`;
 }).join(" ");
}

/**
 * Onda de ventas de una sola serie: curva suave con degradado índigo, el pico
 * etiquetado y un cursor que muestra importe y pedidos del tramo más cercano.
 * Las flechas del teclado recorren los tramos.
 */
export function TrendChart({label,points,format,compact}:{label:string;points:TrendPoint[];format:(value:number)=>string;compact:(value:number)=>string}){
 const[active,setActive]=useState<number|null>(null);
 const plot=useRef<HTMLDivElement>(null);
 const id=useId().replace(/:/g,"");
 const max=niceMax(Math.max(0,...points.map(point=>point.value)));
 const x=(index:number)=>points.length>1?(index/(points.length-1))*100:50;
 const y=(value:number)=>100-(Math.max(0,value)/max)*100;
 const line=smoothPath(points.map((point,index)=>[x(index),y(point.value)] as const));
 const area=`${line} L${x(points.length-1)},100 L${x(0)},100 Z`;
 const peak=points.reduce((best,point,index)=>point.value>(points[best]?.value??0)?index:best,0);
 const pick=(event:PointerEvent<HTMLDivElement>)=>{
  const box=plot.current?.getBoundingClientRect();
  if(!box||!points.length)return;
  setActive(Math.round(Math.min(1,Math.max(0,(event.clientX-box.left)/box.width))*(points.length-1)));
 };
 const move=(event:KeyboardEvent<HTMLDivElement>)=>{
  if(event.key!=="ArrowRight"&&event.key!=="ArrowLeft"&&event.key!=="Home"&&event.key!=="End")return;
  event.preventDefault();
  const last=points.length-1;
  setActive(current=>event.key==="Home"?0:event.key==="End"?last:Math.min(last,Math.max(0,(current??last)+(event.key==="ArrowRight"?1:-1))));
 };
 const point=active===null?null:points[active];
 const align=active===null?"":x(active)<18?" start":x(active)>82?" end":"";
 const ticks=points.length>14?Math.ceil(points.length/7):1;

 return <figure className="dash-trend">
  <div className="dash-trend-frame">
   <div className="dash-trend-y" aria-hidden="true"><span>{compact(max)}</span><span>{compact(max/2)}</span><span>0</span></div>
   <div className="dash-trend-body">
    <div
     className="dash-trend-plot"
     ref={plot}
     tabIndex={0}
     role="img"
     aria-label={`${label}. Usa las flechas para recorrer los valores.`}
     onPointerMove={pick}
     onPointerDown={pick}
     onPointerLeave={()=>setActive(null)}
     onFocus={()=>setActive(points.length-1)}
     onBlur={()=>setActive(null)}
     onKeyDown={move}
    >
     <svg key={points.map(item=>item.key+item.value).join("|")} viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
      <defs>
       <linearGradient id={`${id}-fill`} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" className="dash-wave-stop"/>
        <stop offset="60%" className="dash-wave-stop mid"/>
        <stop offset="100%" className="dash-wave-stop fade"/>
       </linearGradient>
      </defs>
      <path className="dash-wave-area" d={area} fill={`url(#${id}-fill)`}/>
      <path className="dash-wave-line" d={line}/>
     </svg>
     {active===null&&points[peak]?.value>0&&<>
      <span className="dash-trend-dot" style={{left:`${x(peak)}%`,top:`${y(points[peak].value)}%`}} aria-hidden="true"/>
      <span className="dash-peak-label" style={{left:`${x(peak)}%`,top:`${y(points[peak].value)}%`}} aria-hidden="true">{format(points[peak].value)}</span>
     </>}
     {point&&<>
      <span className="dash-trend-cursor" style={{left:`${x(active??0)}%`}} aria-hidden="true"/>
      <span className="dash-trend-dot" style={{left:`${x(active??0)}%`,top:`${y(point.value)}%`}} aria-hidden="true"/>
      <div className={"dash-tooltip"+align} style={{left:`${x(active??0)}%`}} role="status">
       <small>{point.title}</small>
       <b>{format(point.value)}</b>
       <em>{point.orders} {point.orders===1?"pedido con cobro":"pedidos con cobro"}</em>
      </div>
     </>}
    </div>
    <div className="dash-trend-x" aria-hidden="true">{points.map((item,index)=><span key={item.key} style={{left:`${x(index)}%`}} className={(index%ticks===0||index===points.length-1?"":"hidden")+(index===points.length-1?" last":"")}>{item.tick}</span>)}</div>
   </div>
  </div>
  {/* Una tabla no respeta la altura de 1 px de sr-only; el contenedor sí la recorta. */}
  <div className="sr-only">
   <table>
    <caption>{label}</caption>
    <thead><tr><th scope="col">Tramo</th><th scope="col">Ventas</th><th scope="col">Pedidos con cobro</th></tr></thead>
    <tbody>{points.map(item=><tr key={item.key}><th scope="row">{item.title}</th><td>{format(item.value)}</td><td>{item.orders}</td></tr>)}</tbody>
   </table>
  </div>
 </figure>;
}

/** Minigráfico de tendencia: línea recesiva con el último valor resaltado. */
export function Sparkline({values}:{values:number[]}){
 if(values.length<2)return null;
 const max=Math.max(...values);
 const min=Math.min(...values);
 const range=max-min||1;
 const coords=values.map((value,index)=>[(index/(values.length-1))*100,92-((value-min)/range)*84] as const);
 const last=coords[coords.length-1];
 return <span className="dash-spark" aria-hidden="true">
  <svg viewBox="0 0 100 100" preserveAspectRatio="none"><path d={coords.map(([px,py],index)=>`${index?"L":"M"}${px},${py}`).join(" ")}/></svg>
  <i style={{left:`${last[0]}%`,top:`${last[1]}%`}}/>
 </span>;
}

export type BarListItem={key:string;label:string;value:number;detail?:string};

/** Barras horizontales de una sola serie con valor y participación como etiqueta directa. */
export function BarList({label,items,format,empty}:{label:string;items:BarListItem[];format:(value:number)=>string;empty:{icon:IconName;title:string;text:string}}){
 if(!items.length)return <div className="dashboard-empty compact"><Icon name={empty.icon} size={20}/><b>{empty.title}</b><p>{empty.text}</p></div>;
 const positive=items.reduce((sum,item)=>sum+Math.max(0,item.value),0);
 const max=Math.max(1,...items.map(item=>item.value));
 return <ul className="dash-bars" aria-label={label}>
  {items.map(item=>{
   const share=positive>0&&item.value>0?Math.round((item.value/positive)*100):0;
   return <li key={item.key}>
    <div className="dash-bar-copy"><span><b>{item.label}</b>{item.detail&&<small>{item.detail}</small>}</span><span className="dash-bar-value">{format(item.value)}<small>{share}%</small></span></div>
    <div className="dash-bar-track" aria-hidden="true"><i style={{width:`${item.value>0?Math.max(1,(item.value/max)*100):0}%`}}/></div>
   </li>;
  })}
 </ul>;
}
