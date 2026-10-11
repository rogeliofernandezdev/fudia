import "./styles/table-actions.css";
import "./styles/pagination.css";
import Link from "next/link";
import {ButtonHTMLAttributes,forwardRef,InputHTMLAttributes,SelectHTMLAttributes,TextareaHTMLAttributes,type ReactNode} from "react";
import {Icon,type IconName,type IconSize} from "@/design-system/icons";

/**
 * primary: una sola acción dominante por superficie · secondary: alternativa, Reintentar, Limpiar filtros ·
 * ghost: salir sin guardar y Anterior · danger: confirmar dentro de ConfirmDialog · danger-outline: disparadores destructivos
 * (Cancelar pedido, Anular, Cerrar turno). El verde no es un botón: se reserva para estados confirmados.
 */
export type ButtonKind="primary"|"secondary"|"ghost"|"danger"|"danger-outline";
export type ButtonAction="cancel"|"cancel-record"|"continue"|"back";
/** La intención fija variante e icono; no inferirla de la etiqueta visible. */
const buttonActions:Record<ButtonAction,{kind:ButtonKind;icon:IconName;iconPosition:"start"|"end"}>={
 cancel:{kind:"ghost",icon:"close",iconPosition:"start"},
 "cancel-record":{kind:"danger-outline",icon:"cancel",iconPosition:"start"},
 continue:{kind:"primary",icon:"chevron",iconPosition:"end"},
 back:{kind:"ghost",icon:"chevronLeft",iconPosition:"start"},
};
type ButtonProps=ButtonHTMLAttributes<HTMLButtonElement>&{icon?:IconName;kind?:ButtonKind;action?:ButtonAction;/** Bloquea y anuncia la petición en curso; el texto debe cambiar al gerundio del mismo verbo. */loading?:boolean};
export function Button({children,icon,kind="primary",action,type="button",className="",loading=false,disabled,...props}:ButtonProps){
 const definition=action?buttonActions[action]:undefined;
 const variant=definition?.kind??kind,glyph=definition?.icon??icon,trailing=definition?.iconPosition==="end";
 return <button type={type} className={`button ${variant} ${className}`.trim()} disabled={disabled||loading} aria-busy={loading||undefined} {...props}>{glyph&&!trailing&&<Icon name={glyph} size={18}/>}<span>{children}</span>{glyph&&trailing&&<Icon name={glyph} size={18}/>}</button>;
}
export function IconButton({icon,label,accessibleLabel,iconSize=18,className="",type="button",...props}:{icon:IconName;label:string;accessibleLabel?:string;iconSize?:IconSize}&Omit<ButtonHTMLAttributes<HTMLButtonElement>,"aria-label">){return <button type={type} className={`ds-icon-button ${className}`.trim()} aria-label={accessibleLabel??label} data-tooltip={label} {...props}><Icon name={icon} size={iconSize}/></button>}
type RowAction="view"|"review"|"charge"|"edit"|"activate"|"deactivate"|"cancel"|"remove";
const rowActions:Record<RowAction,{icon:IconName;label:string;tone:"default"|"danger"}>={view:{icon:"eye",label:"Ver detalle",tone:"default"},review:{icon:"check",label:"Revisar orden",tone:"default"},charge:{icon:"cash",label:"Cobrar pedido",tone:"default"},edit:{icon:"edit",label:"Editar",tone:"default"},activate:{icon:"toggleOff",label:"Activar",tone:"default"},deactivate:{icon:"toggleOn",label:"Desactivar",tone:"default"},cancel:{icon:"cancel",label:"Cancelar",tone:"danger"},remove:{icon:"close",label:"Quitar fila",tone:"danger"}};
export function RowActionButton({action,label,stateLabel="Estado activo",className="",...props}:{action:RowAction;label?:string;stateLabel?:string}&Omit<ButtonHTMLAttributes<HTMLButtonElement>,"aria-label">){
 const definition=rowActions[action];
 if(action==="activate"||action==="deactivate")return <IconButton icon={definition.icon} iconSize={28} label={label??definition.label} accessibleLabel={stateLabel} role="switch" aria-checked={action==="deactivate"} className={`status-toggle ${className}`.trim()} {...props}/>;
 return <IconButton icon={definition.icon} label={label??definition.label} className={`${definition.tone==="danger"?"danger":""} ${className}`.trim()} {...props}/>;
}
export function FieldLabel({children,hint}:{children:React.ReactNode;hint:string}){return <span className="field-label">{children}<i className="field-hint" data-tooltip={hint} tabIndex={0} role="note" aria-label={hint}>i</i></span>}
export const Input=forwardRef<HTMLInputElement,InputHTMLAttributes<HTMLInputElement>>(function Input({className="",...props},ref){return <input ref={ref} className={`ds-input ${className}`.trim()} {...props}/>});
export const Select=forwardRef<HTMLSelectElement,SelectHTMLAttributes<HTMLSelectElement>>(function Select({className="",children,...props},ref){return <select ref={ref} className={`ds-select ${className}`.trim()} {...props}>{children}</select>});
export const Textarea=forwardRef<HTMLTextAreaElement,TextareaHTMLAttributes<HTMLTextAreaElement>>(function Textarea({className="",...props},ref){return <textarea ref={ref} className={`ds-textarea ${className}`.trim()} {...props}/>});
/**
 * Encabezado de página: el título repite el nombre de la navegación; sin eyebrow (el menú ya indica el grupo).
 * `back` enlaza a la ruta padre, `meta` muestra un estado junto al título y `action` admite una acción
 * primaria y, como máximo, una secundaria. Los filtros nunca van en el encabezado.
 */
export function PageHeader({title,description,action,back,meta}:{title:string;description?:ReactNode;action?:ReactNode;back?:{href:string;label:string};meta?:ReactNode}){
 return <header className="page-header">
  <div className="page-header-copy">
   {back&&<Link className="page-header-back" href={back.href}><Icon name="chevronLeft" size={16}/>{back.label}</Link>}
   <div className="page-header-title"><h1>{title}</h1>{meta}</div>
   {description&&<p>{description}</p>}
  </div>
  {action&&<div className="page-header-actions">{action}</div>}
 </header>;
}
/** Tonos tenues definidos en docs/02_DESIGN_SYSTEM.md: sky cuenta abierta, blue confirmado, green listo/éxito, violet preparación, orange por resolver, teal entregado, red cancelado/error, gray neutro/finalizado. */
export function Status({tone="green",active,icon,emphasis=false,children}:{tone?:"green"|"blue"|"sky"|"orange"|"red"|"violet"|"gray"|"teal";active?:boolean;icon?:IconName;emphasis?:boolean;children:React.ReactNode}){
 const binary=active!==undefined;
 const glyph=binary?(active?"circleCheck":"circleMinus"):icon;
 return <span className={`status ${binary?(active?"green":"gray"):tone}${emphasis?" emphasized":""}`}>{glyph?<Icon name={glyph} size={emphasis?16:14} className="status-icon"/>:<i aria-hidden="true"/>}{children}</span>
}
export function Pagination({page,size,total,onPage,onSize,mode="full"}:{page:number;size:number;total:number;onPage:(page:number)=>void;onSize:(size:number)=>void;mode?:"full"|"simple"|"compact"}){
 const last=Math.max(1,Math.ceil(total/size));const safePage=Math.min(Math.max(page,1),last);const pages=paginationItems(safePage,last);const start=total?(safePage-1)*size+1:0;const end=Math.min(safePage*size,total);
 return <footer className={`standard-pagination${mode==="compact"?" compact":""}`}>{mode!=="compact"&&<><span>{total?`Mostrando ${start}–${end} de ${total}`:"Sin resultados"}</span><label>Filas <Select value={size} onChange={event=>onSize(Number(event.target.value))}><option>10</option><option>20</option><option>50</option></Select></label></>}<nav aria-label="Paginación"><button type="button" className="pagination-nav" disabled={safePage<=1} onClick={()=>onPage(safePage-1)}><Icon name="chevronLeft" size={14}/><span>Anterior</span></button>{mode==="simple"?<span className="pagination-current" aria-live="polite">Página {safePage} de {last}</span>:pages.map((value,index)=>value==="ellipsis"?<span className="pagination-ellipsis" aria-hidden="true" key={`ellipsis-${index}`}>…</span>:<button type="button" className={`pagination-page${value===safePage?" active":""}`} aria-label={`Página ${value}`} aria-current={value===safePage?"page":undefined} onClick={()=>onPage(value)} key={value}>{value}</button>)}<button type="button" className="pagination-nav" disabled={safePage>=last} onClick={()=>onPage(safePage+1)}><span>Siguiente</span><Icon name="chevron" size={14}/></button></nav></footer>
}
function paginationItems(page:number,total:number):(number|"ellipsis")[]{if(total<=5)return Array.from({length:total},(_,index)=>index+1);const values=new Set([1,total,page-1,page,page+1].filter(value=>value>=1&&value<=total));const sorted=[...values].sort((a,b)=>a-b);const result:(number|"ellipsis")[]=[];sorted.forEach((value,index)=>{if(index&&value-sorted[index-1]>1)result.push("ellipsis");result.push(value)});return result}
