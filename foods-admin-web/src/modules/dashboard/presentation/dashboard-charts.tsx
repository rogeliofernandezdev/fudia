"use client";
import {useEffect,useId,useRef,useState,type CSSProperties,type KeyboardEvent,type PointerEvent} from "react";
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

const prefersReducedMotion=()=>typeof window!=="undefined"&&window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;

/** Duración de la transformación de una línea (aparición y cambio de datos). */
const LINE_TWEEN_MS=750;

/** Reparte una serie en `length` puntos por interpolación lineal, para transformar series de distinta longitud. */
function resample(source:number[],length:number,base:number){
 if(!length)return [];
 if(!source.length)return Array.from({length},()=>base);
 if(source.length===1||length===1)return Array.from({length},()=>source[source.length-1]);
 return Array.from({length},(_,index)=>{
  const position=(index/(length-1))*(source.length-1);
  const lower=Math.floor(position),upper=Math.min(source.length-1,lower+1);
  return source[lower]+(source[upper]-source[lower])*(position-lower);
 });
}

/**
 * Línea dinámica: al aparecer la serie crece desde `base` y, cuando llegan datos
 * nuevos (otro periodo o el refresco de 30 s), se transforma desde la serie que
 * estaba en pantalla en lugar de redibujarse. Con movimiento reducido salta al final.
 */
function useTweenedSeries(values:number[],base:number){
 const[frame,setFrame]=useState(()=>({values:values.map(()=>base),animating:values.length>0}));
 const shown=useRef<number[]>(values.map(()=>base));
 const signature=values.join("|");
 useEffect(()=>{
  const target=signature?signature.split("|").map(Number):[];
  const from=resample(shown.current,target.length,base);
  const duration=prefersReducedMotion()?0:LINE_TWEEN_MS;
  const start=performance.now();
  let request=0;
  const tick=(now:number)=>{
   const progress=duration?Math.min(1,(now-start)/duration):1;
   const eased=1-(1-progress)**3;
   const next=target.map((value,index)=>from[index]+(value-from[index])*eased);
   shown.current=next;
   setFrame({values:next,animating:progress<1});
   if(progress<1)request=requestAnimationFrame(tick);
  };
  request=requestAnimationFrame(tick);
  return()=>cancelAnimationFrame(request);
 },[signature,base]);
 // Hasta el primer cuadro de una serie de otra longitud, se reparte la que estaba en pantalla.
 return frame.values.length===values.length?frame:{values:resample(frame.values,values.length,base),animating:true};
}

/**
 * Onda de ventas de una sola serie: curva suave con degradado índigo, el pico
 * etiquetado y un cursor que muestra importe y pedidos del tramo más cercano.
 * Las flechas del teclado recorren los tramos.
 */
export function TrendChart({label,points,format,compact,live=false}:{label:string;points:TrendPoint[];format:(value:number)=>string;compact:(value:number)=>string;live?:boolean}){
 const[active,setActive]=useState<number|null>(null);
 const wave=useTweenedSeries(points.map(point=>point.value),0);
 const plot=useRef<HTMLDivElement>(null);
 const id=useId().replace(/:/g,"");
 const max=niceMax(Math.max(0,...points.map(point=>point.value)));
 const x=(index:number)=>points.length>1?(index/(points.length-1))*100:50;
 const y=(value:number)=>100-(Math.max(0,value)/max)*100;
 const line=smoothPath(wave.values.map((value,index)=>[x(index),y(value)] as const));
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
 const last=points.length-1;
 // Pico y punto en vivo esperan a que la línea se asiente.
 const settled=active===null&&!wave.animating;

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
     <svg viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
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
     {/* El pico se funde cuando la línea se asienta; con el cursor activo solo se oculta. */}
     {points[peak]?.value>0&&<span className={"dash-peak"+(settled?"":" hidden")} aria-hidden="true">
      <span className="dash-trend-dot" style={{left:`${x(peak)}%`,top:`${y(points[peak].value)}%`}}/>
      <span className="dash-peak-label" style={{left:`${x(peak)}%`,top:`${y(points[peak].value)}%`}}>{format(points[peak].value)}</span>
     </span>}
     {/* «Hoy»: la hora en curso late mientras sigan llegando cobros. */}
     {live&&points[last]&&<span className={"dash-trend-live"+(settled?"":" hidden")} style={{left:`${x(last)}%`,top:`${y(points[last].value)}%`}} aria-hidden="true"/>}
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

/** Minigráfico de tendencia: curva recesiva con área tenue y el último valor resaltado; crece y se transforma como la onda. */
export function Sparkline({values}:{values:number[]}){
 const id=useId().replace(/:/g,"");
 const min=values.length?Math.min(...values):0;
 const spark=useTweenedSeries(values,min);
 if(values.length<2)return null;
 const max=Math.max(...values);
 const range=max-min||1;
 const coords=spark.values.map((value,index)=>[(index/(values.length-1))*100,Math.min(100,Math.max(0,92-((value-min)/range)*84))] as const);
 const line=smoothPath(coords);
 const last=coords[coords.length-1];
 return <span className="dash-spark" aria-hidden="true">
  <svg viewBox="0 0 100 100" preserveAspectRatio="none">
   <defs><linearGradient id={`${id}-spark`} x1="0" y1="0" x2="0" y2="1"><stop offset="0%" className="dash-spark-stop"/><stop offset="100%" className="dash-spark-stop fade"/></linearGradient></defs>
   <path className="dash-spark-area" d={`${line} L100,100 L0,100 Z`} fill={`url(#${id}-spark)`}/>
   <path className="dash-spark-line" d={line}/>
  </svg>
  <i style={{left:`${last[0]}%`,top:`${last[1]}%`}}/>
 </span>;
}

/**
 * Cifra que cuenta desde el valor anterior hasta el nuevo (desde 0 al aparecer)
 * para señalar que el dato cambió. `children` es el texto final: lo leen los
 * lectores de pantalla y evita anunciar los valores intermedios.
 */
export function CountUp({value,format,children}:{value:number;format:(value:number)=>string;children:string}){
 const[display,setDisplay]=useState(0);
 const current=useRef(0);
 useEffect(()=>{
  const from=current.current;
  const duration=prefersReducedMotion()||from===value?0:650;
  const start=performance.now();
  let frame=0;
  const tick=(now:number)=>{
   const progress=duration?Math.min(1,(now-start)/duration):1;
   const next=from+(value-from)*(1-(1-progress)**3);
   current.current=next;
   setDisplay(next);
   if(progress<1)frame=requestAnimationFrame(tick);
  };
  frame=requestAnimationFrame(tick);
  return()=>cancelAnimationFrame(frame);
 },[value]);
 return <><span className="dash-count" aria-hidden="true">{format(display)}</span><span className="sr-only">{children}</span></>;
}

export type BarListItem={key:string;label:string;value:number;detail?:string};

/** La barra ocupa toda la pista y se desplaza con transform (nunca con width); cada fila entra 60 ms después de la anterior. */
const barStyle=(ratio:number,index:number):CSSProperties=>({transform:`translateX(${(Math.min(1,Math.max(0,ratio))-1)*100}%)`,animationDelay:`calc(var(--duration-slow) + ${index*60}ms)`});

/** Barras horizontales de una sola serie con valor y participación como etiqueta directa. */
export function BarList({label,items,format,empty}:{label:string;items:BarListItem[];format:(value:number)=>string;empty:{icon:IconName;title:string;text:string}}){
 if(!items.length)return <div className="dashboard-empty compact"><Icon name={empty.icon} size={20}/><b>{empty.title}</b><p>{empty.text}</p></div>;
 const positive=items.reduce((sum,item)=>sum+Math.max(0,item.value),0);
 const max=Math.max(1,...items.map(item=>item.value));
 return <ul className="dash-bars" aria-label={label}>
  {items.map((item,index)=>{
   const share=positive>0&&item.value>0?Math.round((item.value/positive)*100):0;
   return <li key={item.key}>
    <div className="dash-bar-copy"><span><b>{item.label}</b>{item.detail&&<small>{item.detail}</small>}</span><span className="dash-bar-value">{format(item.value)}<small>{share}%</small></span></div>
    <div className="dash-bar-track" aria-hidden="true"><i style={barStyle(item.value>0?Math.max(.01,item.value/max):0,index)}/></div>
   </li>;
  })}
 </ul>;
}

/** Barra de progreso de una fila de ranking, con la misma animación que BarList. */
export function RankTrack({ratio,index}:{ratio:number;index:number}){
 return <div className="top-track" aria-hidden="true"><i style={barStyle(ratio,index)}/></div>;
}

const DONUT_SLICES=4;
const DONUT_RADIUS=46;
const DONUT_LENGTH=2*Math.PI*DONUT_RADIUS;

/**
 * Anillo de participación en la rampa índigo (de mayor a menor tono), con el
 * total al centro y una leyenda que da importe, detalle y porcentaje. Agrupa
 * desde el quinto elemento en «Otros». El anillo es decorativo: la leyenda es
 * la lectura accesible.
 */
export function DonutChart({label,items,format,empty}:{label:string;items:BarListItem[];format:(value:number)=>string;empty:{icon:IconName;title:string;text:string}}){
 if(!items.length)return <div className="dashboard-empty compact"><Icon name={empty.icon} size={20}/><b>{empty.title}</b><p>{empty.text}</p></div>;
 const sorted=[...items].sort((a,b)=>b.value-a.value);
 const rest=sorted.slice(DONUT_SLICES);
 const slices=rest.length>1?[...sorted.slice(0,DONUT_SLICES),{key:"__other",label:`Otros ${rest.length}`,value:rest.reduce((sum,item)=>sum+item.value,0)}]:sorted;
 const total=slices.reduce((sum,item)=>sum+item.value,0);
 const positive=slices.reduce((sum,item)=>sum+Math.max(0,item.value),0);
 const gap=slices.filter(item=>item.value>0).length>1?2:0;
 const lengths=slices.map(item=>positive>0?(Math.max(0,item.value)/positive)*DONUT_LENGTH:0);
 const arcs=slices.map((item,index)=>({key:item.key,index,dash:Math.max(0,lengths[index]-gap),offset:lengths.slice(0,index).reduce((sum,length)=>sum+length,0)})).filter(arc=>arc.dash>0);
 return <div className="dash-donut">
  <div className="dash-donut-ring" aria-hidden="true">
   <svg key={slices.map(item=>item.key+item.value).join("|")} viewBox="0 0 120 120">
    <circle className="dash-donut-base" cx="60" cy="60" r={DONUT_RADIUS}/>
    <g transform="rotate(-90 60 60)">{arcs.map(arc=><circle key={arc.key} className={"dash-donut-arc s"+arc.index} cx="60" cy="60" r={DONUT_RADIUS} strokeDasharray={`${arc.dash} ${DONUT_LENGTH-arc.dash}`} strokeDashoffset={-arc.offset}/>)}</g>
   </svg>
   <span className="dash-donut-total"><small>Total</small><b>{format(total)}</b></span>
  </div>
  <ul className="dash-donut-legend" aria-label={label}>
   {slices.map((item,index)=>{
    const share=positive>0&&item.value>0?Math.round((item.value/positive)*100):0;
    return <li key={item.key}>
     <i className={"dash-donut-swatch s"+index} aria-hidden="true"/>
     <span className="dash-donut-copy"><b>{item.label}</b>{item.detail&&<small>{item.detail}</small>}</span>
     <span className="dash-bar-value">{format(item.value)}<small>{share}%</small></span>
    </li>;
   })}
  </ul>
 </div>;
}
