"use client";
import {Dialog} from "@/design-system/dialog";
import "./combo-wizard.css";
import "./combo-rules.css";
import {useEffect,useRef,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import type {Combo,ComboDetail,Draft,Group,Product} from "../domain/types";
import {createComboOption,isReservationValid,reservationLimit} from "../domain/option-quantity";
import {validateComboDraft,validateComboStep,type WizardIssue} from "../domain/wizard-validation";
import {getCombo,listComboProducts,listCombos,saveCombo,setComboActive} from "../infrastructure/combos-api";

const blank:Draft={name:"",description:"",price:"",groups:[],availableFrom:"",availableUntil:"",availableDays:[]};
const templates:Group[]=[{name:"Entrada",required:true,minSelections:1,maxSelections:1,options:[]},{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[]},{name:"Postre",required:false,minSelections:0,maxSelections:1,options:[]},{name:"Bebidas",required:false,minSelections:0,maxSelections:1,options:[]}];
const DAYS=[{d:1,n:"Lun"},{d:2,n:"Mar"},{d:3,n:"Mié"},{d:4,n:"Jue"},{d:5,n:"Vie"},{d:6,n:"Sáb"},{d:0,n:"Dom"}];
const STEP_LABELS=["Nombre y precio","Qué incluye","Cuándo se vende","Confirmar"];
const STEP_HINTS=["Presentación y precio","Platos y opciones","Días, horario y cupos","Revisa antes de guardar"];
const STEP_ICONS=["utensils","kitchen","clock","check"] as const;
const normalize=(value:string)=>value.normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLowerCase().trim();
function belongsToGroup(product:Product,groupName:string){
  const category=normalize(product.categoryName??"");
  const group=normalize(groupName);
  return category===group||category===`${group}s`||category.startsWith(`${group} `)||category.includes(group);
}

export function CombosPage(){
  const{notify}=useFeedback();
  const{organization,location,can}=useSession();
  const canManage=can("menu.manage");
  const settings=useSettings();
  const client=useQueryClient();
  const[draft,setDraft]=useState<Draft|null>(null);
  const[editingId,setEditingId]=useState<string|null>(null);
  const cancelledEdit=useRef<string|null>(null);
  const[selected,setSelected]=useState<string|null>(null);
  const[statusTarget,setStatusTarget]=useState<Combo|null>(null);
  const[step,setStep]=useState(1);
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[search,setSearch]=useState("");
  const[status,setStatus]=useState("");
  const debouncedSearch=useDebouncedValue(search);
  const combos=useQuery({queryKey:["combos",page,size,debouncedSearch,status],queryFn:()=>listCombos(page,size,debouncedSearch,status)});
  const products=useQuery({queryKey:["products","combo-picker",organization?.id,location?.id],queryFn:listComboProducts,enabled:Boolean(canManage&&draft&&location?.id),staleTime:0,refetchInterval:draft?10000:false});
  const detail=useQuery({queryKey:["combo-detail",selected],queryFn:()=>getCombo(selected!),enabled:Boolean(selected)});
  const save=useMutation({
    mutationFn:(value:Draft)=>saveCombo(value,editingId),
    onSuccess:()=>{const edited=Boolean(editingId);setDraft(null);setEditingId(null);setStep(1);void client.invalidateQueries({queryKey:["combos"]});void client.invalidateQueries({queryKey:["products"]});notify({tone:"success",title:edited?"Menú actualizado":"Menú registrado",message:edited?"Los cambios ya están disponibles para la operación.":"El menú ya está disponible para la operación."})},
    onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})
  });
  const loadForEdit=useMutation({
    mutationFn:(id:string)=>getCombo(id),
    onSuccess:value=>{
      if(cancelledEdit.current===value.id){cancelledEdit.current=null;return}
      const toLocalInput=(source:string|null)=>{if(!source)return"";const date=new Date(source);const offset=date.getTimezoneOffset()*60000;return new Date(date.getTime()-offset).toISOString().slice(0,16)};
      setEditingId(value.id);
      setDraft({name:value.name,description:value.description,price:value.price,availableFrom:toLocalInput(value.availableFrom),availableUntil:toLocalInput(value.availableUntil),availableDays:value.availableDays??[],groups:value.groups.map(group=>({name:group.name,required:group.required,minSelections:group.minSelections,maxSelections:group.maxSelections,options:group.options.map(option=>({productId:option.productId,surcharge:option.surcharge==="0.00"?"":option.surcharge,quota:option.quota===null?"":String(option.quota)}))}))});
      setStep(1);
    },
    onError:(error,requestedId)=>{if(editingId===requestedId)setEditingId(null);notify({tone:"danger",title:"No se pudo abrir el menú",message:error.message})}
  });
  const changeStatus=useMutation({
    mutationFn:(item:Combo)=>setComboActive(item.id,!item.active),
    onSuccess:(_,item)=>{
      setStatusTarget(null);
      void client.invalidateQueries({queryKey:["combos"]});
      void client.invalidateQueries({queryKey:["combo-detail",item.id]});
      notify({tone:"success",title:item.active?"Menú desactivado":"Menú activado",message:item.active?"Ya no estará disponible para nuevas ventas.":"Volvió a estar disponible para la operación."});
    },
    onError:error=>{
      setStatusTarget(null);
      notify({tone:"danger",title:"No se pudo cambiar el estado",message:error.message});
    }
  });
  const open=()=>{if(!canManage)return;setEditingId(null);setDraft(blank);setStep(1)};
  const closeWizard=()=>{setDraft(null);setEditingId(null);setStep(1)};
  return <>
    <PageHeader eyebrow="CARTA Y PRODUCCIÓN" title="Menús y combos" description="Define las partes del menú y qué opciones puede elegir el cliente en cada una." action={canManage?<Button icon="plus" onClick={open}>Nuevo menú o combo</Button>:undefined}/>
    <section className="panel management catalog-panel standardized-management combo-list">
      <div className="toolbar">
        <label><Icon name="search" size={17}/><input value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar menú o combo..." aria-label="Buscar menú o combo"/></label>
        <Select value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}} aria-label="Filtrar por estado"><option value="">Todos los estados</option><option value="active">Activos</option><option value="inactive">Inactivos</option></Select>
      </div>
      {combos.isLoading?<ComboSkeleton/>
      :combos.isError?<div className="combo-empty"><Icon name="alert" size={26}/><b>No pudimos cargar los menús y combos</b><p>{combos.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>combos.refetch()}>Reintentar</Button></div>
      :!combos.data?.items.length?
        <div className="combo-empty"><Icon name="menu" size={26}/><b>{search||status?"No encontramos resultados":"Aún no hay menús compuestos"}</b><p>{search||status?"Prueba con otro nombre o cambia el filtro de estado.":canManage?"Usa la plantilla Menú del día para comenzar rápidamente.":"Los menús registrados aparecerán aquí."}</p>{canManage&&!search&&!status&&<Button icon="plus" onClick={open}>Crear menú</Button>}</div>
      :<>
        <div className="table-wrap hover-scroll"><table><thead><tr><th>MENÚ O COMBO</th><th>PRECIO</th><th title="Entrada, plato principal, bebida o postre">PARTES DEL MENÚ</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>
          {combos.data.items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}>
            <td><span className={`row-icon r${index%3}`}><Icon name="menu" size={18}/></span><b>{item.name}</b><small>{item.description||"Sin descripción"}</small></td>
            <td><b>{settings.currencyPosition==="before"?`${settings.currencySymbol} ${Number(item.price).toFixed(settings.currencyDecimals)}`:`${Number(item.price).toFixed(settings.currencyDecimals)} ${settings.currencySymbol}`}</b></td>
            <td>{item.groupCount} {item.groupCount===1?"parte":"partes"}</td>
            <td><Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status></td>
            <td><div className="standard-actions"><RowActionButton action="view" label={`Ver ${item.name}`} onClick={()=>setSelected(item.id)}/>{canManage&&<><RowActionButton action="edit" label={`Editar ${item.name}`} disabled={loadForEdit.isPending} onClick={()=>{cancelledEdit.current=null;setEditingId(item.id);loadForEdit.mutate(item.id)}}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado de ${item.name}`} label={`${item.active?"Desactivar":"Activar"} ${item.name}`} onClick={()=>setStatusTarget(item)}/></>}</div></td>
          </tr>)}
        </tbody></table></div>
      </>}
      {!combos.isLoading&&!combos.isError&&<Pagination page={page} size={size} total={combos.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    </section>
    {selected&&<ComboDetailDialog query={detail} currencySymbol={settings.currencySymbol} close={()=>setSelected(null)}/>}
    <ConfirmDialog open={canManage&&Boolean(statusTarget)} title={`${statusTarget?.active?"Desactivar":"Activar"} menú`} description={statusTarget?.active?`“${statusTarget.name}” dejará de estar disponible para nuevas ventas, pero conservará su historial.`:`“${statusTarget?.name??""}” volverá a estar disponible para la operación.`} confirmLabel={statusTarget?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>canManage&&statusTarget&&changeStatus.mutate(statusTarget)}/>
    {canManage&&!draft&&editingId&&loadForEdit.isPending&&<RemoteModalSkeleton className="combo-wizard" label="Cargando menú o combo" rows={8} close={()=>{cancelledEdit.current=editingId;setEditingId(null)}}/>}
    {canManage&&draft&&<ComboWizard draft={draft} setDraft={setDraft} step={step} setStep={setStep} products={products.data?.items??[]} productsLoading={products.isLoading} productsError={products.isError} retryProducts={()=>products.refetch()} currencySymbol={settings.currencySymbol} busy={save.isPending} editing={Boolean(editingId)} close={closeWizard} finish={()=>{if(canManage)save.mutate(draft)}}/>}
  </>
}

function ComboDetailDialog({query,currencySymbol,close}:{query:ReturnType<typeof useQuery<ComboDetail,Error>>;currencySymbol:string;close:()=>void}){
  const{location}=useSession();
  const value=query.data;
  const from=value?.availableFrom?new Date(value.availableFrom):null;
  const until=value?.availableUntil?new Date(value.availableUntil):null;
  const singleDay=Boolean(from&&until&&until.getTime()>from.getTime()&&until.getTime()-from.getTime()<=24*60*60*1000);
  const days=singleDay?"Solo hoy":value?.availableDays?.length?value.availableDays.map(day=>DAYS.find(item=>item.d===day)?.n).filter(Boolean).join(", "):"Todos los días";
  const formatDate=(date:string|null|undefined)=>date?formatRegionalDateTime(date,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"}):"Sin límite";
  const formatCalendarDate=(date:string|null|undefined)=>date?formatRegionalCalendarDate(date,location?.country,{dateStyle:"long"}):"";
  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal combo-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="combo-detail-title" aria-busy={query.isLoading}>
      <div className="modal-accent"/>
      {query.isLoading?<ComboDetailSkeleton close={close}/>:<>
        <header><span className="modal-title-icon"><Icon name="eye" size={18}/></span><div><h2 id="combo-detail-title">{value?.name??"Menú o combo"}</h2><small>DETALLE DEL MENÚ</small></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
        <div className="combo-detail-body">
          {query.isError?<div className="combo-detail-state"><Icon name="alert" size={24}/><b>No pudimos cargar el menú</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>:value&&<>
            <section className="combo-detail-summary"><div><small>PRECIO</small><b>{currencySymbol} {Number(value.price).toFixed(2)}</b></div><div><small>PARTES DEL MENÚ</small><b>{value.groups.length} {value.groups.length===1?"parte":"partes"}</b></div><Status active={value.active}>{value.active?"Activo":"Inactivo"}</Status></section>
            {value.description&&<section className="combo-detail-description"><small>DESCRIPCIÓN</small><p>{value.description}</p></section>}
            <section className="combo-detail-section"><header><div><small>QUÉ INCLUYE</small><h3>Platos y opciones del menú</h3></div></header>{value.groups.map(group=><article className="combo-detail-group" key={group.id}><div><b>{group.name}</b><small>{group.required?"Obligatorio":"Opcional"} · El cliente elige {group.minSelections}–{group.maxSelections}</small></div><ul>{group.options.map(option=><li key={option.productId}><span>{option.name}</span>{Number(option.surcharge)>0&&<small>+ {currencySymbol} {Number(option.surcharge).toFixed(2)}</small>}</li>)}</ul></article>)}</section>
            <section className={`combo-detail-availability${singleDay?" single":""}`}>{singleDay?<div><small>CUÁNDO SE VENDE</small><b>Solo hoy · {formatCalendarDate(value.availableFrom)}</b></div>:<><div><small>DÍAS DE VENTA</small><b>{days}</b></div><div><small>DESDE</small><b>{formatDate(value.availableFrom)}</b></div><div><small>HASTA</small><b>{formatDate(value.availableUntil)}</b></div></>}</section>
          </>}
        </div>
      </>}
    </Dialog>
  </div>
}

function ComboDetailSkeleton({close}:{close:()=>void}){
 return <>
  <header className="combo-detail-skeleton-head" aria-hidden="true">
   <span className="combo-detail-skeleton-block combo-detail-skeleton-icon"/>
   <div className="combo-detail-skeleton-copy"><span/><b/></div>
   <button aria-label="Cerrar" onClick={close}><Icon name="close"/></button>
  </header>
  <div className="combo-detail-body combo-detail-skeleton-body" aria-label="Cargando detalle del menú">
   <section className="combo-detail-summary combo-detail-skeleton-summary">
    <div><span/><b/></div><div><span/><b/></div><i className="combo-detail-skeleton-status"/>
   </section>
   <section className="combo-detail-description combo-detail-skeleton-description"><span/><b/><b/></section>
   <section className="combo-detail-section combo-detail-skeleton-section">
    <header><div><span/><b/></div></header>
    {Array.from({length:3},(_,i)=><div className="combo-detail-skeleton-group" key={i}><div><b/><span/></div><div><i/><i/><i/></div></div>)}
   </section>
   <section className="combo-detail-availability combo-detail-skeleton-availability">
    {Array.from({length:3},(_,i)=><div key={i}><span/><b/></div>)}
   </section>
  </div>
 </>;
}

function ComboWizard({draft,setDraft,step,setStep,products,productsLoading,productsError,retryProducts,currencySymbol,busy,editing,close,finish}:{draft:Draft;setDraft:(value:Draft)=>void;step:number;setStep:(value:number)=>void;products:Product[];productsLoading:boolean;productsError:boolean;retryProducts:()=>void;currencySymbol:string;busy:boolean;editing:boolean;close:()=>void;finish:()=>void}){
  const [issue,setIssue]=useState<WizardIssue|null>(null);
  const [discardOpen,setDiscardOpen]=useState(false);
  const [initial]=useState(()=>JSON.stringify(draft));
  const dirty=initial!==JSON.stringify(draft);
  useEffect(()=>{
    if(!dirty)return;
    const warn=(event:BeforeUnloadEvent)=>{event.preventDefault();event.returnValue="";};
    window.addEventListener("beforeunload",warn);
    return()=>window.removeEventListener("beforeunload",warn);
  },[dirty]);
  const requestClose=()=>{if(busy)return;if(dirty)setDiscardOpen(true);else close();};
  const showIssue=(value:WizardIssue)=>{setIssue(value);setStep(value.step);requestAnimationFrame(()=>document.getElementById(value.field)?.focus());};
  const navigate=(value:number)=>{setIssue(null);setStep(value);};
  const quotaError=(groupIndex:number,optIndex:number)=>{const option=draft.groups[groupIndex]?.options[optIndex];if(!option)return false;const product=products.find(p=>p.id===option.productId);return Boolean(product&&!isReservationValid(option.quota,product))};
  function next(){const error=validateComboStep(draft,products,step);if(error){showIssue(error);return;}navigate(Math.min(4,step+1));}
  function submit(){if(busy)return;const error=validateComboDraft(draft,products);if(error){showIssue(error);return;}finish();}
  function addTemplate(){setDraft({...draft,groups:templates.map(group=>({...group,options:products.filter(product=>belongsToGroup(product,group.name)).map(createComboOption)}))})}
  function addGroup(){setDraft({...draft,groups:[...draft.groups,{name:"Nueva parte",required:true,minSelections:1,maxSelections:1,options:[]}]})}
  function groupAt(index:number,next:Group){setDraft({...draft,groups:draft.groups.map((group,position)=>position===index?next:group)})}
  return <><div className="modal-backdrop modal-overlay-in">
    <Dialog onResponseClose={close} className="crud-modal combo-wizard modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="combo-wizard-title" aria-busy={busy}>
      <header className="combo-wizard-header">
        <div className="combo-wizard-title">
          <span className="combo-wizard-title-icon"><Icon name="chefHat" size={21}/></span>
          <div>
            <small>{editing?"EDITAR MENÚ O COMBO":"NUEVO MENÚ O COMBO"}</small>
            <h2 id="combo-wizard-title">{editing?"Actualiza tu menú o combo":"Arma tu menú o combo"}</h2>
            <p>{STEP_LABELS[step-1]} · {STEP_HINTS[step-1]}</p>
          </div>
        </div>
        <button className="combo-wizard-close" aria-label="Cerrar" onClick={requestClose} disabled={busy}><Icon name="close" size={18}/></button>
      </header>

      <nav className="combo-wizard-steps" aria-label="Pasos para crear el menú">
        {STEP_LABELS.map((label,index)=><button type="button" key={label} className={step===index+1?"active":step>index+1?"complete":""} aria-current={step===index+1?"step":undefined} disabled={busy||index+1>step} onClick={()=>navigate(index+1)}>
          <span className="combo-step-marker"><Icon name={STEP_ICONS[index]} size={16}/></span>
          <span className="combo-step-copy"><b>{label}</b><small>{STEP_HINTS[index]}</small></span>
        </button>)}
      </nav>

      <main className="combo-wizard-stage">
        <div className="combo-wizard-body">
          {issue&&<p className="form-error" id="combo-wizard-error" role="alert"><Icon name="alert" size={18}/>{issue.message}</p>}
          {step===1&&<InfoStep draft={draft} setDraft={setDraft} currencySymbol={currencySymbol}/>}
          {step===2&&<CompositionStep draft={draft} setDraft={setDraft} products={products} productsLoading={productsLoading} productsError={productsError} retryProducts={retryProducts} currencySymbol={currencySymbol} addTemplate={addTemplate} addGroup={addGroup} groupAt={groupAt} quotaError={quotaError}/>}
          {step===3&&<AvailabilityStep draft={draft} setDraft={setDraft}/>}
          {step===4&&<ReviewStep draft={draft} products={products} currencySymbol={currencySymbol}/>}
        </div>

        <footer className="combo-wizard-footer">
          <button type="button" className="button ghost" onClick={step===1?requestClose:()=>navigate(step-1)} disabled={busy}>{step===1?<><Icon name="close" size={16}/>Cancelar</>:<><Icon name="chevronLeft" size={16}/>Anterior</>}</button>
          <div className="combo-wizard-footer-copy">
            <small>{step<4?"Puedes volver a modificar pasos anteriores.":"Revisa los datos antes de guardar."}</small>
          </div>
          {step<4?<button type="button" className="button primary" onClick={next} disabled={busy||(step===2&&(productsLoading||productsError||!products.length))}>Continuar<Icon name="chevron" size={16}/></button>:<button type="button" className="button primary" onClick={submit} disabled={busy}><Icon name="check" size={16}/>{busy?"Guardando…":"Guardar"}</button>}
        </footer>
      </main>
    </Dialog>
  </div><ConfirmDialog open={discardOpen} title="¿Descartar los cambios?" description="Los cambios del menú o combo no se han guardado." confirmLabel="Descartar" onCancel={()=>setDiscardOpen(false)} onConfirm={close}/></>
}

function InfoStep({draft,setDraft,currencySymbol}:{draft:Draft;setDraft:(v:Draft)=>void;currencySymbol:string}){
  return <div className="form-grid">
    <div className="combo-section-heading span-2"><span className="combo-section-heading-icon"><Icon name="utensils" size={18}/></span><div><h3>Nombre y precio</h3><p>Define cómo verá el cliente este menú y cuánto costará.</p></div></div>
    <label>Nombre<Input id="combo-name" autoFocus value={draft.name} onChange={e=>setDraft({...draft,name:e.target.value})} placeholder="Ej. Menú ejecutivo"/></label>
    <label className="combo-price-field">Precio del menú<div className="money-input"><span>{currencySymbol}</span><Input id="combo-price" inputMode="decimal" value={draft.price} onChange={e=>setDraft({...draft,price:e.target.value})} placeholder="0.00"/></div></label>
    <label className="span-2">Descripción<Textarea value={draft.description} onChange={e=>setDraft({...draft,description:e.target.value})} placeholder="Ej. Incluye entrada, segundo y postre" rows={2}/></label>
  </div>
}

function CompositionStep({draft,setDraft,products,productsLoading,productsError,retryProducts,currencySymbol,addTemplate,addGroup,groupAt,quotaError}:{draft:Draft;setDraft:(v:Draft)=>void;products:Product[];productsLoading:boolean;productsError:boolean;retryProducts:()=>void;currencySymbol:string;addTemplate:()=>void;addGroup:()=>void;groupAt:(i:number,g:Group)=>void;quotaError:(gi:number,oi:number)=>boolean}){
  const categories=Array.from(new Set(products.map(product=>product.categoryName??"Sin categoría"))).sort((a,b)=>a.localeCompare(b,"es"));
  if(productsLoading)return <div className="table-skeleton" aria-label="Cargando productos">{Array.from({length:3},(_,index)=><div className="sk-row" key={index}><i/><i/><i/></div>)}</div>;
  if(productsError)return <div className="combo-empty-groups"><Icon name="alert" size={24}/><b>No pudimos cargar los productos</b><p>Revisa tu conexión e inténtalo de nuevo.</p><Button icon="refresh" onClick={retryProducts}>Reintentar</Button></div>;
  if(products.length===0)return <div className="combo-empty-groups"><Icon name="box" size={24}/><b>No hay productos registrados</b><p>Primero crea productos en la sección Productos.</p></div>;
  return <div className="combo-composition">
    <div className="combo-composition-toolbar">
      <div className="combo-composition-copy"><span className="combo-section-heading-icon"><Icon name="kitchen" size={18}/></span><div><b>Qué incluye el menú</b><small>Agrega entrada, plato principal, bebida o postre y define qué podrá elegir el cliente.</small></div></div>
      <div><Button kind="secondary" icon="menu" onClick={addTemplate}>Plantilla Menú del día</Button><Button id="combo-add-group" icon="plus" onClick={addGroup}>Agregar parte</Button></div>
    </div>
    {draft.groups.length===0?
      <div className="combo-empty-groups"><Icon name="menu" size={24}/><b>Agrega las partes del menú</b><p>Usa la plantilla o crea partes como Entrada, Plato principal o Postre.</p></div>
    :draft.groups.map((group,index)=>
      <div className="combo-group" key={index}>
        <div className="combo-group-header">
          <Input id={`combo-group-${index}-name`} aria-label="Nombre de la parte del menú" value={group.name} onChange={e=>groupAt(index,{...group,name:e.target.value})} placeholder="Ej. Entrada o Plato principal"/>
          <label className="combo-toggle"><input type="checkbox" checked={group.required} onChange={e=>groupAt(index,{...group,required:e.target.checked,minSelections:e.target.checked?1:0,maxSelections:1})}/><span/><b>Obligatorio</b></label>
          <button className="combo-remove" onClick={()=>setDraft({...draft,groups:draft.groups.filter((_,p)=>p!==index)})} aria-label="Quitar parte del menú"><Icon name="close" size={16}/></button>
        </div>
        <div className="combo-options">
          {group.options.map((option,optIndex)=>{
            const product=products.find(p=>p.id===option.productId);
            const limit=product?reservationLimit(product):null;
            return <div className="combo-option" key={optIndex}>
              <span className="combo-option-name">{product?.name??"Producto eliminado"}<small>{product?.categoryName??"Sin categoría"}</small></span>
              <label className="combo-mini">Reservar<Input id={`combo-group-${index}-quota-${optIndex}`} type="number" min="0" step="1" max={limit??undefined} value={option.quota} onChange={e=>groupAt(index,{...group,options:group.options.map((o,p)=>p===optIndex?{...o,quota:e.target.value}:o)})} placeholder="Sin límite" aria-invalid={quotaError(index,optIndex)}/>{quotaError(index,optIndex)&&<small className="field-error">Ingresa un entero sin superar la cantidad disponible ({limit??"sin límite"}).</small>}</label>
              <label className="combo-mini">Recargo<div className="money-input"><span>{currencySymbol}</span><Input id={`combo-group-${index}-surcharge-${optIndex}`} inputMode="decimal" value={option.surcharge} onChange={e=>groupAt(index,{...group,options:group.options.map((o,p)=>p===optIndex?{...o,surcharge:e.target.value}:o)})} placeholder="0.00"/></div></label>
              <button className="combo-remove" onClick={()=>groupAt(index,{...group,options:group.options.filter((_,p)=>p!==optIndex)})} aria-label="Quitar opción"><Icon name="close" size={14}/></button>
            </div>;
          })}
          <div className="combo-add-option">
            <Select id={`combo-group-${index}-product`} aria-label={`Agregar producto a ${group.name}`} value="" onChange={e=>{const product=products.find(p=>p.id===e.target.value);if(product)groupAt(index,{...group,options:[...group.options,createComboOption(product)]});e.target.value=""}}>
              <option value="">+ Agregar opción</option>
              {categories.map(category=><optgroup label={category} key={category}>{products.filter(product=>(product.categoryName??"Sin categoría")===category&&!group.options.some(option=>option.productId===product.id)).map(product=><option value={product.id} key={product.id}>{product.name}</option>)}</optgroup>)}
            </Select>
          </div>
        </div>
      </div>
    )}
  </div>
}

function AvailabilityStep({draft,setDraft}:{draft:Draft;setDraft:(v:Draft)=>void}){
  const pad=(n:number)=>String(n).padStart(2,"0");
  const today=new Date();
  const todayStr=`${today.getFullYear()}-${pad(today.getMonth()+1)}-${pad(today.getDate())}`;
  const isToday=draft.availableFrom.startsWith(todayStr)&&draft.availableUntil.startsWith(todayStr);
  const setToday=()=>setDraft({...draft,availableDays:[],availableFrom:`${todayStr}T00:00`,availableUntil:`${todayStr}T23:59`});
  return <div className="form-grid combo-availability">
    <div className="combo-section-heading span-2"><span className="combo-section-heading-icon"><Icon name="clock" size={18}/></span><div><h3>Cuándo se vende</h3><p>Elige los días y fechas. Si no seleccionas nada, estará disponible siempre.</p></div></div>
    <label className="span-2">Días disponibles
      <div className="days-chips">
        <button type="button" className={isToday?"day-chip today active":"day-chip today"} onClick={setToday}>Hoy</button>
        {DAYS.map(day=><button type="button" key={day.d} className={draft.availableDays.includes(day.d)?"day-chip active":"day-chip"} onClick={()=>setDraft({...draft,availableDays:draft.availableDays.includes(day.d)?draft.availableDays.filter((x:number)=>x!==day.d):[...draft.availableDays,day.d]})}>{day.n}</button>)}
      </div>
    </label>
    <label>Fecha desde<Input id="combo-from" type="datetime-local" value={draft.availableFrom} onChange={e=>setDraft({...draft,availableFrom:e.target.value})}/></label>
    <label>Fecha hasta<Input id="combo-until" type="datetime-local" value={draft.availableUntil} onChange={e=>setDraft({...draft,availableUntil:e.target.value})}/></label>
  </div>
}

function ReviewStep({draft,products,currencySymbol}:{draft:Draft;products:Product[];currencySymbol:string}){
  const pad=(n:number)=>String(n).padStart(2,"0");
  const today=new Date();
  const todayStr=`${today.getFullYear()}-${pad(today.getMonth()+1)}-${pad(today.getDate())}`;
  const isToday=draft.availableFrom.startsWith(todayStr)&&draft.availableUntil.startsWith(todayStr);
  const fmt=(v:string)=>{if(!v)return"";const[d,t]=v.split("T");const[y,m,dd]=d.split("-");return `${dd}/${m}/${y} ${t??""}`.trim()};
  const daysLabel=isToday?"Solo hoy":draft.availableDays.length?draft.availableDays.map(d=>DAYS.find(x=>x.d===d)?.n).join(", "):"Todos los días";
  const rangeLabel=isToday?`Solo hoy (${fmt(draft.availableFrom).split(" ")[0]})`:draft.availableFrom||draft.availableUntil?`${fmt(draft.availableFrom)||"..."} — ${fmt(draft.availableUntil)||"..."}`:"Sin restricción";
  return <div className="combo-review">
    <div className="combo-section-heading"><span className="combo-section-heading-icon"><Icon name="check" size={18}/></span><div><h3>Todo listo para guardar</h3><p>Revisa el menú como quedará antes de publicarlo para la operación.</p></div></div>
    <div className="combo-review-section">
      <b>Nombre y precio</b>
      <dl>
        <div><dt>Nombre</dt><dd>{draft.name||"Sin nombre"}</dd></div>
        <div><dt>Precio</dt><dd>{currencySymbol} {draft.price||"0.00"}</dd></div>
        <div><dt>Descripción</dt><dd>{draft.description||"Sin descripción"}</dd></div>
      </dl>
    </div>
    <div className="combo-review-section">
      <b>Qué incluye</b>
      {draft.groups.map((group,index)=>
        <div key={index} className="combo-review-group">
          <dt>{group.name} {group.required?"(obligatorio)":"(opcional)"}</dt>
          <dd>{group.options.length?group.options.map(o=>products.find(p=>p.id===o.productId)?.name??"?").join(", "):"Sin opciones"}</dd>
        </div>
      )}
    </div>
    <div className="combo-review-section">
      <b>Cuándo se vende</b>
      <dl>
        <div><dt>Días</dt><dd>{daysLabel}</dd></div>
        <div><dt>Fechas</dt><dd>{rangeLabel}</dd></div>
      </dl>
    </div>
  </div>
}

function ComboSkeleton(){return <div className="table-skeleton" aria-label="Cargando menús"><div className="sk-head"><i/><i/><i/><i/><i/></div>{Array.from({length:5},(_,index)=><div className="sk-row" key={index}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/><i/></div>)}</div>}
