"use client";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import "./combo-editor.css";
import "./combo-rules.css";
import {useEffect,useRef,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,EmptyState,ErrorState,FieldLabel,FormField,Icon,IconButton,Input,MoneyInput,PageHeader,Pagination,RemoteModalSkeleton,RowActionButton,Select,Status,Textarea,ModalHeader,ModalFooter,ModalError,ModalItemList,ModalWorkspace,IconTile,Skeleton,type IconTileTone} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {useSession} from "@/providers/session-context";
import {formatRegionalCalendarDate,formatRegionalDateTime} from "@/shared/i18n/regional-format";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import type {Combo,ComboDetail,Draft,Group,Product} from "../domain/types";
import {createComboOption,isReservationValid,reservationLimit} from "../domain/option-quantity";
import {validateComboDraft,type WizardIssue} from "../domain/wizard-validation";
import {applySaleWindow,changeSaleDate,fullDaySchedule,isOutsideDates,isoToZonedInput,saleWindowOf,zonedInputToIso,type SaleWindow} from "../domain/sale-window";
import {belongsToGroup,normalize} from "../domain/option-groups";
import {ComboOptionPicker} from "./combo-option-picker";
import {getCombo,listComboProducts,listCombos,saveCombo,setComboActive} from "../infrastructure/combos-api";

const blank:Draft={name:"",description:"",price:"",groups:[],availableFrom:"",availableUntil:"",availableDays:[]};
const templates:Group[]=[{name:"Entrada",required:true,minSelections:1,maxSelections:1,options:[]},{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[]},{name:"Postre",required:false,minSelections:0,maxSelections:1,options:[]},{name:"Bebidas",required:false,minSelections:0,maxSelections:1,options:[]}];
/** Ayuda de «Costo extra»: «Recargo» sonaba a penalización. */
const EXTRA_COST_HINT="Monto que se suma si se elige este plato.";
const DAYS=[{d:1,n:"Lun"},{d:2,n:"Mar"},{d:3,n:"Mié"},{d:4,n:"Jue"},{d:5,n:"Vie"},{d:6,n:"Sáb"},{d:0,n:"Dom"}];
const SALE_WINDOWS:{value:SaleWindow;label:string}[]=[{value:"always",label:"Siempre"},{value:"days",label:"Elegir días"}];
/** Identidad de cada parte: tono suave e icono propio, como las tarjetas del Dashboard. Las partes nuevas rotan los tonos.
 *  Sin ámbar ni rojo: esos tonos significan advertencia o error en el sistema. */
type PartTone={tone:"green"|"indigo"|"violet"|"teal";icon:"salad"|"dish"|"dessert"|"drink"|"utensils"};
const PART_TONES:{pattern:RegExp;style:PartTone}[]=[
  {pattern:/entrada|sopa|ensalada|piqueo|aperitivo/,style:{tone:"green",icon:"salad"}},
  {pattern:/segundo|principal|fondo|plato/,style:{tone:"indigo",icon:"dish"}},
  {pattern:/postre|dulce/,style:{tone:"violet",icon:"dessert"}},
  {pattern:/bebida|refresco|jugo|trago|cafe/,style:{tone:"teal",icon:"drink"}},
];
/** Tono de IconTile equivalente a cada parte, para el detalle de lectura. */
const PART_TILE:Record<PartTone["tone"],IconTileTone>={green:"brand",indigo:"primary",violet:"digital",teal:"teal"};
function partTone(name:string,index:number):PartTone{
  const value=normalize(name);
  return PART_TONES.find(item=>item.pattern.test(value))?.style??{tone:PART_TONES[index%PART_TONES.length].style.tone,icon:"utensils"};
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
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[search,setSearch]=useState("");
  const[status,setStatus]=useState("");
  const debouncedSearch=useDebouncedValue(search);
  const combos=useQuery({queryKey:["combos",page,size,debouncedSearch,status],queryFn:()=>listCombos(page,size,debouncedSearch,status)});
  const products=useQuery({queryKey:["products","combo-picker",organization?.id,location?.id],queryFn:listComboProducts,enabled:Boolean(canManage&&draft&&location?.id),staleTime:0,refetchInterval:draft?10000:false});
  const detail=useQuery({queryKey:["combo-detail",selected],queryFn:()=>getCombo(selected!),enabled:Boolean(selected)});
  const save=useMutation({
    mutationFn:(value:Draft)=>saveCombo(value,editingId,location?.timezone),
    onSuccess:(_,value)=>{
      const edited=Boolean(editingId);
      // Un menú guardado fuera de sus fechas no se vende todavía (o ya no): el aviso no debe decir que está disponible.
      const outside=isOutsideDates(zonedInputToIso(value.availableFrom,location?.timezone),zonedInputToIso(value.availableUntil,location?.timezone));
      setDraft(null);setEditingId(null);void client.invalidateQueries({queryKey:["combos"]});void client.invalidateQueries({queryKey:["combo-detail"]});void client.invalidateQueries({queryKey:["products"]});
      notify({tone:"success",title:edited?"Menú actualizado":"Menú registrado",message:outside?"Está fuera de sus fechas de venta: se venderá solo en las fechas indicadas.":edited?"Los cambios ya están disponibles para la operación.":"El menú ya está disponible para la operación."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})
  });
  const loadForEdit=useMutation({
    mutationFn:(id:string)=>getCombo(id),
    onSuccess:value=>{
      if(cancelledEdit.current===value.id){cancelledEdit.current=null;return}
      // Las fechas se muestran en la hora del local, no en la del navegador.
      const toLocalInput=(source:string|null)=>isoToZonedInput(source,location?.timezone);
      setEditingId(value.id);
      const schedule=fullDaySchedule({availableFrom:toLocalInput(value.availableFrom),availableUntil:toLocalInput(value.availableUntil),availableDays:value.availableDays??[]});
      setDraft({name:value.name,description:value.description,price:value.price,...schedule,groups:value.groups.map(group=>({name:group.name,required:group.required,minSelections:group.minSelections,maxSelections:group.maxSelections,options:group.options.map(option=>({productId:option.productId,surcharge:option.surcharge==="0.00"?"":option.surcharge,quota:option.quota===null?"":String(option.quota)}))}))});
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
  const money=(amount:string|number)=>{const fixed=Number(amount).toFixed(settings.currencyDecimals);return settings.currencyPosition==="after"?`${fixed} ${settings.currencySymbol}`:`${settings.currencySymbol} ${fixed}`};
  const open=()=>{if(!canManage)return;setEditingId(null);setDraft(blank)};
  const closeEditor=()=>{setDraft(null);setEditingId(null)};
  return <>
    <PageHeader title="Menús y combos" description="Define las partes del menú y qué opciones puede elegir el cliente en cada una." action={canManage?<Button icon="plus" onClick={open}>Nuevo menú o combo</Button>:undefined}/>
    <section className="panel management catalog-panel standardized-management combo-list">
      <div className="toolbar">
        <label><Icon name="search" size={18}/><input value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar menú o combo..." aria-label="Buscar menú o combo"/></label>
        <Select chip value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}} aria-label="Filtrar por estado"><option value="">Todos los estados</option><option value="active">Activos</option><option value="inactive">Inactivos</option></Select>
      </div>
      {combos.isLoading?<ComboSkeleton/>
      :combos.isError?<div className="combo-empty"><Icon name="alert" size={24}/><b>No pudimos cargar los menús y combos</b><p>{combos.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>combos.refetch()}>Reintentar</Button></div>
      :!combos.data?.items.length?
        <div className="combo-empty"><Icon name="menu" size={24}/><b>{search||status?"No encontramos resultados":"Aún no hay menús compuestos"}</b><p>{search||status?"Prueba con otro nombre o cambia el filtro de estado.":canManage?"Usa la plantilla Menú del día para comenzar rápidamente.":"Los menús registrados aparecerán aquí."}</p>{canManage&&!search&&!status&&<Button icon="plus" onClick={open}>Crear menú</Button>}</div>
      :<>
        <DataTable label="Listado de menús y combos" containerClassName="hover-scroll"><thead><tr><th>Menú o combo</th><th>Precio</th><th title="Entrada, plato principal, bebida o postre">Partes del menú</th><th>Estado</th><th>Acciones</th></tr></thead><tbody>
          {combos.data.items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}>
            <td><span className={`row-icon r${index%3}`}><Icon name="menu" size={18}/></span><b>{item.name}</b></td>
            <td><b>{money(item.price)}</b></td>
            <td>{item.groupCount} {item.groupCount===1?"parte":"partes"}</td>
            <td>{item.active&&item.outsideDates?<Status tone="gray">Fuera de fecha</Status>:<Status active={item.active}>{item.active?"Activo":"Inactivo"}</Status>}</td>
            <td><div className="standard-actions"><RowActionButton action="view" label={`Ver ${item.name}`} onClick={()=>setSelected(item.id)}/>{canManage&&<><RowActionButton action="edit" label={`Editar ${item.name}`} disabled={loadForEdit.isPending} onClick={()=>{cancelledEdit.current=null;setEditingId(item.id);loadForEdit.mutate(item.id)}}/><RowActionButton action={item.active?"deactivate":"activate"} stateLabel={`Estado de ${item.name}`} label={`${item.active?"Desactivar":"Activar"} ${item.name}`} onClick={()=>setStatusTarget(item)}/></>}</div></td>
          </tr>)}
        </tbody></DataTable>
      </>}
      {!combos.isLoading&&!combos.isError&&<Pagination page={page} size={size} total={combos.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    </section>
    {selected&&<ComboDetailDialog query={detail} currencySymbol={settings.currencySymbol} close={()=>setSelected(null)}/>}
    <ConfirmDialog open={canManage&&Boolean(statusTarget)} title={`${statusTarget?.active?"Desactivar":"Activar"} menú`} description={statusTarget?.active?`“${statusTarget.name}” dejará de estar disponible para nuevas ventas, pero conservará su historial.`:`“${statusTarget?.name??""}” volverá a estar disponible para la operación.`} confirmLabel={statusTarget?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>canManage&&statusTarget&&changeStatus.mutate(statusTarget)}/>
    {canManage&&!draft&&editingId&&loadForEdit.isPending&&<RemoteModalSkeleton className="combo-editor" label="Cargando menú o combo" rows={8} close={()=>{cancelledEdit.current=editingId;setEditingId(null)}}/>}
    {canManage&&draft&&<ComboEditor draft={draft} setDraft={setDraft} products={products.data?.items??[]} productsLoading={products.isLoading} productsError={products.isError} retryProducts={()=>products.refetch()} currencySymbol={settings.currencySymbol} currencyPosition={settings.currencyPosition} money={money} busy={save.isPending} editing={Boolean(editingId)} close={closeEditor} finish={()=>{if(canManage)save.mutate(draft)}}/>}
  </>
}

/**
 * Detalle de menú con el patrón de Detalle de pedido: cabecera con el nombre, estado y venta
 * arriba, partes como lista con su icono y un resumen fijo con el precio como importe principal.
 * Es de lectura: sin pie de acciones; la X es el único cierre.
 */
function ComboDetailDialog({query,currencySymbol,close}:{query:ReturnType<typeof useQuery<ComboDetail,Error>>;currencySymbol:string;close:()=>void}){
  const{location}=useSession();
  const value=query.data;
  const localFrom=isoToZonedInput(value?.availableFrom,location?.timezone).slice(0,10);
  const localUntil=isoToZonedInput(value?.availableUntil,location?.timezone).slice(0,10);
  const singleDay=Boolean(localFrom&&localUntil&&localFrom===localUntil);
  const days=value?.availableDays?.length?value.availableDays.map(day=>DAYS.find(item=>item.d===day)?.n).filter(Boolean).join(", "):"Todos los días";
  const money=(amount:string|number)=>`${currencySymbol} ${Number(amount).toFixed(2)}`;
  const formatDate=(date:string|null|undefined)=>date?formatRegionalDateTime(date,{country:location?.country,timeZone:location?.timezone},{dateStyle:"medium",timeStyle:"short"}):"";
  const formatCalendarDate=(date:string|null|undefined)=>date?formatRegionalCalendarDate(date,location?.country,{dateStyle:"long"}):"";
  const schedule=singleDay?`Se vende el ${formatCalendarDate(localFrom)}`:[days,value?.availableFrom&&`desde ${formatDate(value.availableFrom)}`,value?.availableUntil&&`hasta ${formatDate(value.availableUntil)}`].filter(Boolean).join(" · ");
  const optionCount=value?.groups.reduce((total,group)=>total+group.options.length,0)??0;
  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <Dialog onResponseClose={close} className="crud-modal size-md combo-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="combo-detail-title" aria-busy={query.isLoading}>
      {query.isLoading?<ComboDetailSkeleton close={close}/>:<>
        <ModalHeader title={value?.name??"Menú o combo"} titleId="combo-detail-title" icon="combo" onClose={close} closeLabel="Cerrar"/>
        {query.isError?<ModalError title="No pudimos cargar el menú" text={query.error.message} onRetry={()=>query.refetch()} retrying={query.isFetching}/>:value&&<ModalWorkspace summary={<section className="modal-summary" aria-labelledby="combo-detail-summary-title">
          <div className="modal-summary-heading"><h3 id="combo-detail-summary-title" className="modal-section-title">Resumen del menú</h3></div>
          <div className="modal-summary-lines">
            <div className="modal-summary-line"><span>Partes</span><b>{value.groups.length}</b></div>
            <div className="modal-summary-line"><span>Opciones</span><b>{optionCount}</b></div>
          </div>
          <div className="modal-summary-total"><span>Precio</span><strong>{money(value.price)}</strong></div>
        </section>}>
          <div className="modal-body combo-detail-body" tabIndex={0} aria-label="Estado, venta y platos del menú">
            <section className="combo-detail-hero" aria-label="Estado y venta del menú">
              <div className="combo-detail-status">{value.active&&isOutsideDates(value.availableFrom,value.availableUntil)?<Status tone="gray">Fuera de fecha</Status>:<Status active={value.active}>{value.active?"Activo":"Inactivo"}</Status>}</div>
              <p className="combo-detail-schedule"><Icon name="calendar" size={14}/><span>{schedule}</span></p>
              {value.description&&<p className="combo-detail-description">{value.description}</p>}
            </section>
            <section className="combo-detail-section" aria-labelledby="combo-detail-groups"><h3 id="combo-detail-groups" className="modal-section-title">Platos y opciones</h3>
              {value.groups.length?<ModalItemList className="combo-detail-parts">{value.groups.map((group,index)=>{const style=partTone(group.name,index);return <div className="modal-item-row combo-detail-part" role="listitem" key={group.id}>
                <IconTile name={style.icon} tone={PART_TILE[style.tone]} size="sm"/>
                <div className="combo-detail-part-copy"><small>{group.name}</small><span>{group.options.map((option,optionIndex)=><span className="combo-detail-option" key={option.productId}>{optionIndex>0&&<span aria-hidden="true">·</span>}{option.name}{Number(option.surcharge)>0&&<small>+ {money(option.surcharge)}</small>}</span>)}</span></div>
              </div>})}</ModalItemList>:<p className="combo-detail-empty">Este menú aún no tiene platos.</p>}
            </section>
          </div>
        </ModalWorkspace>}
      </>}
    </Dialog>
  </div>
}

/** Skeleton con la geometría final: cabecera, estado y venta, partes y resumen. */
function ComboDetailSkeleton({close}:{close:()=>void}){
 return <>
  <ModalHeader title={<><span className="sr-only">Cargando detalle del menú</span><Skeleton width="var(--size-168)" height="var(--size-16)"/></>} titleId="combo-detail-title" icon="combo" onClose={close} initialFocusClose/>
  <ModalWorkspace summary={<section className="modal-summary" aria-hidden="true">
   <div className="modal-summary-heading"><Skeleton width="var(--size-138)" height="var(--size-13)"/></div>
   <div className="modal-summary-lines">{Array.from({length:2},(_,index)=><div className="modal-summary-line" key={index}><Skeleton width="var(--size-72)" height="var(--size-9)"/><Skeleton width="var(--size-36)" height="var(--size-16)"/></div>)}</div>
   <div className="modal-summary-total"><Skeleton width="var(--size-72)" height="var(--size-10)"/><Skeleton width="var(--size-108)" height="var(--size-24)"/></div>
  </section>}>
   <div className="modal-body combo-detail-body" role="status" aria-label="Cargando detalle del menú">
    <div className="combo-detail-hero" aria-hidden="true"><Skeleton width="var(--size-72)" height="var(--size-24)" radius="var(--radius-pill)"/><Skeleton width="var(--size-220)" height="var(--size-10)"/></div>
    <ModalItemList className="combo-detail-parts">{Array.from({length:3},(_,index)=><div className="modal-item-row combo-detail-part" role="listitem" aria-hidden="true" key={index}>
     <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-tile)"/>
     <span className="combo-detail-part-copy"><Skeleton width="var(--size-62)" height="var(--size-9)"/><Skeleton width="min(var(--size-220),72%)" height="var(--size-11)"/></span>
    </div>)}</ModalItemList>
   </div>
  </ModalWorkspace>
 </>;
}

type ComboEditorProps={draft:Draft;setDraft:(value:Draft)=>void;products:Product[];productsLoading:boolean;productsError:boolean;retryProducts:()=>void;currencySymbol:string;currencyPosition?:"before"|"after";money:(amount:string|number)=>string;busy:boolean;editing:boolean;close:()=>void;finish:()=>void};
const decimal=(value:string)=>value.replace(",",".");
const amountPattern=/^\d+(\.\d{1,2})?$/;

/** Hoja única para crear o editar un menú: datos, partes y venta en un solo recorrido, sin pasos ni revisión aparte. */
function ComboEditor({draft,setDraft,products,productsLoading,productsError,retryProducts,currencySymbol,currencyPosition="before",money,busy,editing,close,finish}:ComboEditorProps){
  const [issue,setIssue]=useState<WizardIssue|null>(null);
  const [discardOpen,setDiscardOpen]=useState(false);
  const [removing,setRemoving]=useState<number|null>(null);
  const [resetting,setResetting]=useState(false);
  const [initial]=useState(()=>JSON.stringify(draft));
  const [saleWindow,setSaleWindow]=useState<SaleWindow>(()=>saleWindowOf(draft));
  const [describing,setDescribing]=useState(()=>Boolean(draft.description));
  const dirty=initial!==JSON.stringify(draft);
  useEffect(()=>{
    if(!dirty)return;
    const warn=(event:BeforeUnloadEvent)=>{event.preventDefault();event.returnValue="";};
    window.addEventListener("beforeunload",warn);
    return()=>window.removeEventListener("beforeunload",warn);
  },[dirty]);
  const requestClose=()=>{if(busy)return;if(dirty)setDiscardOpen(true);else close();};
  const update=(next:Draft)=>{setIssue(null);setDraft(next)};
  // El campo con error se centra en la hoja: un aviso arriba del cuerpo quedaba fuera de vista en menús largos.
  const reveal=(id:string)=>requestAnimationFrame(()=>{
    const field=document.getElementById(id);if(!field)return;
    const calm=typeof window!=="undefined"&&window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    field.focus({preventScroll:true});field.scrollIntoView({block:"center",behavior:calm?"auto":"smooth"});
  });
  const showIssue=(value:WizardIssue)=>{setIssue(value);reveal(value.field);};
  const errorFor=(field:string)=>issue?.field===field?issue.message:undefined;
  function submit(){
    if(busy)return;
    const error=validateComboDraft(draft,products);
    if(error){showIssue(error);return;}
    if(saleWindow==="days"&&!draft.availableDays.length&&!draft.availableFrom&&!draft.availableUntil){showIssue({step:3,field:"combo-days",message:"Elige al menos un día o una fecha."});return;}
    finish();
  }
  function addTemplate(){update({...draft,groups:templates.map(group=>({...group,options:products.filter(product=>belongsToGroup(product,group.name)).map(createComboOption)}))})}
  function addGroup(){const index=draft.groups.length;update({...draft,groups:[...draft.groups,{name:"",required:true,minSelections:1,maxSelections:1,options:[]}]});reveal(`combo-group-${index}-name`)}
  function groupAt(index:number,next:Group){update({...draft,groups:draft.groups.map((group,position)=>position===index?next:group)})}
  function removeGroup(index:number){update({...draft,groups:draft.groups.filter((_,position)=>position!==index)})}
  // Quitar una parte con opciones pide confirmación; una parte vacía se quita al instante.
  const requestRemoveGroup=(index:number)=>{if(draft.groups[index]?.options.length)setRemoving(index);else removeGroup(index)};
  // «Empezar de nuevo» vuelve al recuadro inicial (plantilla o parte); con opciones pide confirmación. Nombre, precio y días se conservan.
  function clearGroups(){setResetting(false);update({...draft,groups:[]})}
  const requestClearGroups=()=>{if(draft.groups.some(group=>group.options.length))setResetting(true);else clearGroups()};
  const optionCount=draft.groups.reduce((total,group)=>total+group.options.length,0);
  function chooseWindow(value:SaleWindow){setSaleWindow(value);update({...draft,...applySaleWindow(draft,value)})}
  const removingGroup=removing===null?null:draft.groups[removing];
  const priceReady=amountPattern.test(draft.price);
  const partCount=draft.groups.length;
  return <><div className="modal-backdrop modal-overlay-in">
    <Dialog onResponseClose={close} className="crud-modal size-lg combo-editor modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="combo-editor-title" aria-busy={busy}>
      <ModalHeader title={editing?"Actualiza tu menú o combo":"Arma tu menú o combo"} titleId="combo-editor-title" icon="chefHat" onClose={requestClose} busy={busy} closeLabel="Cerrar"/>
      <div className="modal-body combo-editor-body">
        <h3 className="form-section-title"><IconTile name="chefHat" size="sm"/>Datos del menú</h3>
        <div className="combo-editor-basics">
          <FormField label="Nombre" error={errorFor("combo-name")}><Input id="combo-name" autoFocus={!editing} value={draft.name} onChange={e=>update({...draft,name:e.target.value})} placeholder="Ej. Menú ejecutivo"/></FormField>
          <FormField label="Precio" className="combo-editor-price" error={errorFor("combo-price")}><MoneyInput id="combo-price" symbol={currencySymbol} position={currencyPosition} value={draft.price} onChange={e=>update({...draft,price:decimal(e.target.value)})} placeholder="0.00"/></FormField>
          {describing
            ?<FormField label="Descripción" optional className="combo-editor-description"><Textarea id="combo-description" value={draft.description} onChange={e=>update({...draft,description:e.target.value})} placeholder="Ej. Incluye entrada, segundo y refresco" rows={2}/></FormField>
            :<button type="button" className="combo-editor-link" onClick={()=>{setDescribing(true);reveal("combo-description")}}><Icon name="plus" size={16}/>Agregar descripción</button>}
        </div>

        <section className="combo-editor-section" aria-labelledby="combo-parts-title">
          <header className="combo-editor-section-head">
            <h3 id="combo-parts-title"><IconTile name="combo" tone="brand" size="sm"/>Qué incluye</h3>
            {partCount>0&&<Button kind="ghost" icon="refresh" className="combo-parts-reset" onClick={requestClearGroups} disabled={busy}>Empezar de nuevo</Button>}
          </header>
          <CompositionStep draft={draft} products={products} productsLoading={productsLoading} productsError={productsError} retryProducts={retryProducts} currencySymbol={currencySymbol} currencyPosition={currencyPosition} errorFor={errorFor} addTemplate={addTemplate} addGroup={addGroup} groupAt={groupAt} removeGroup={requestRemoveGroup}/>
        </section>

        <section className="combo-editor-section" aria-labelledby="combo-window-title">
          <h3 id="combo-window-title"><IconTile name="calendar" tone="digital" size="sm"/>Cuándo se vende</h3>
          <div className="combo-window" role="radiogroup" aria-labelledby="combo-window-title">
            {SALE_WINDOWS.map(option=><button type="button" role="radio" key={option.value} aria-checked={saleWindow===option.value} className={saleWindow===option.value?"combo-window-option active":"combo-window-option"} onClick={()=>chooseWindow(option.value)}>{option.label}</button>)}
          </div>
          {saleWindow==="days"&&<div className="combo-window-days">
            <fieldset className="combo-days">
              <legend>Días</legend>
              <div className="combo-days-chips">
                {DAYS.map((day,index)=>{const active=draft.availableDays.includes(day.d);return <button type="button" key={day.d} id={index===0?"combo-days":undefined} aria-pressed={active} className={active?"day-chip active":"day-chip"} onClick={()=>update({...draft,availableDays:active?draft.availableDays.filter(value=>value!==day.d):[...draft.availableDays,day.d]})}>{day.n}</button>})}
              </div>
              {errorFor("combo-days")&&<small className="field-error" role="alert">{errorFor("combo-days")}</small>}
            </fieldset>
            <div className="combo-window-range">
              <FormField label="Desde" optional error={errorFor("combo-from")}><Input id="combo-from" type="date" value={draft.availableFrom.slice(0,10)} onChange={e=>update({...draft,...changeSaleDate(draft,"availableFrom",e.target.value)})}/></FormField>
              <FormField label="Hasta" optional error={errorFor("combo-until")}><Input id="combo-until" type="date" value={draft.availableUntil.slice(0,10)} onChange={e=>update({...draft,...changeSaleDate(draft,"availableUntil",e.target.value)})}/></FormField>
            </div>
          </div>}
        </section>
      </div>

      <ModalFooter className="combo-editor-footer">
        {(partCount>0||priceReady)&&<p className="combo-editor-summary" aria-live="polite">
          {partCount>0&&<span>{partCount} {partCount===1?"parte":"partes"}</span>}
          {priceReady&&<strong>{money(draft.price)}</strong>}
        </p>}
        <Button action="cancel" onClick={requestClose} disabled={busy}>Cancelar</Button>
        <Button id="combo-submit" icon="check" onClick={submit} disabled={busy||productsLoading} loading={busy}>{busy?(editing?"Guardando…":"Creando…"):(editing?"Guardar cambios":"Crear menú")}</Button>
      </ModalFooter>
    </Dialog>
  </div>
  <ConfirmDialog open={removingGroup!==null} title={`¿Quitar ${removingGroup?.name.trim()||"esta parte"}?`} description={`Se quitarán sus ${removingGroup?.options.length??0} ${removingGroup?.options.length===1?"opción":"opciones"} del menú.`} confirmLabel="Quitar" onCancel={()=>setRemoving(null)} onConfirm={()=>{if(removing!==null)removeGroup(removing);setRemoving(null)}}/>
  <ConfirmDialog open={resetting} title="¿Empezar de nuevo?" description={`Se quitarán las ${partCount} ${partCount===1?"parte":"partes"} y sus ${optionCount} ${optionCount===1?"opción":"opciones"}. El nombre, el precio y los días de venta se conservan.`} confirmLabel="Quitar todo" onCancel={()=>setResetting(false)} onConfirm={clearGroups}/>
  <ConfirmDialog open={discardOpen} title="¿Descartar los cambios?" description="Los cambios del menú o combo no se han guardado." confirmLabel="Descartar" onCancel={()=>setDiscardOpen(false)} onConfirm={close}/></>
}

type CompositionProps={draft:Draft;products:Product[];productsLoading:boolean;productsError:boolean;retryProducts:()=>void;currencySymbol:string;currencyPosition:"before"|"after";errorFor:(field:string)=>string|undefined;addTemplate:()=>void;addGroup:()=>void;groupAt:(index:number,group:Group)=>void;removeGroup:(index:number)=>void};

/** Partes del menú: el nombre es el título editable y cada opción una línea con su reserva y costo extra a la vista. */
function CompositionStep({draft,products,productsLoading,productsError,retryProducts,currencySymbol,currencyPosition,errorFor,addTemplate,addGroup,groupAt,removeGroup}:CompositionProps){
  if(productsLoading)return <div className="combo-parts-loading" role="status" aria-label="Cargando productos">{Array.from({length:3},(_,index)=><i key={index}/>)}</div>;
  if(productsError)return <ErrorState className="combo-parts-state" title="No pudimos cargar los productos" text="Revisa tu conexión e inténtalo de nuevo." onRetry={retryProducts}/>;
  if(products.length===0)return <EmptyState className="combo-parts-state" icon="dish" title="No hay productos registrados" text="Primero crea productos en la sección Productos."/>;
  // Sin partes: recuadro con las dos formas de empezar. La plantilla (recomendada) es un botón relleno índigo y «Agregar
  // una parte» `ghost` neutro. Excepción aprobada: mientras no hay partes conviven dos primarios con «Crear menú».
  if(!draft.groups.length)return <div className="combo-parts-start">
    <p>Arma lo que elegirá el cliente: entrada, segundo, postre o bebida.</p>
    <div className="combo-parts-start-actions">
      <Button icon="combo" onClick={addTemplate}>Usar plantilla Menú del día</Button>
      <Button kind="ghost" icon="plus" id="combo-add-group" onClick={addGroup}>Agregar una parte</Button>
    </div>
    {errorFor("combo-add-group")&&<small className="field-error" role="alert">{errorFor("combo-add-group")}</small>}
  </div>;
  return <div className="combo-parts">
    {draft.groups.map((group,index)=>{
      const prefix=`combo-group-${index}`;
      const label=group.name.trim()||"esta parte";
      const nameError=errorFor(`${prefix}-name`),productError=errorFor(`${prefix}-product`);
      const style=partTone(group.name,index);
      const count=group.options.length;
      return <section className={`combo-part tone-${style.tone}`} key={index} aria-label={group.name.trim()||`Parte ${index+1}`}>
        <header className="combo-part-header">
          <span className="combo-part-icon" aria-hidden="true"><Icon name={style.icon} size={20}/></span>
          <div className="combo-part-heading">
            <label className="combo-part-title">
              <Input id={`${prefix}-name`} className="combo-part-name" aria-label="Nombre de la parte" aria-invalid={Boolean(nameError)} aria-describedby={nameError?`${prefix}-name-error`:undefined} value={group.name} onChange={e=>groupAt(index,{...group,name:e.target.value})} placeholder="Ej. Entrada"/>
              <Icon name="edit" size={16}/>
            </label>
            <span className="combo-part-rule">{count} {count===1?"opción":"opciones"} · {group.required?"El cliente elige 1":"Puede omitirse"}</span>
          </div>
          <label className="combo-toggle"><input type="checkbox" checked={group.required} onChange={e=>groupAt(index,{...group,required:e.target.checked,minSelections:e.target.checked?1:0,maxSelections:1})}/><span aria-hidden="true"/><b>Obligatorio</b></label>
          <IconButton icon="trash" label={`Quitar ${label}`} className="combo-part-remove" onClick={()=>removeGroup(index)}/>
        </header>
        {nameError&&<small id={`${prefix}-name-error`} className="field-error" role="alert">{nameError}</small>}
        {group.options.length>0&&<div className="combo-option-table">
          <div className="combo-option-head"><span aria-hidden="true">Opción</span><span aria-hidden="true">Reservar</span><FieldLabel hint={EXTRA_COST_HINT}>Costo extra</FieldLabel></div>
          <ul className="combo-part-options">
            {group.options.map((option,optIndex)=>{
              const product=products.find(p=>p.id===option.productId);
              const limit=product?reservationLimit(product):null;
              const quotaId=`${prefix}-quota-${optIndex}`,surchargeId=`${prefix}-surcharge-${optIndex}`,errorId=`${prefix}-option-${optIndex}-error`;
              const quotaInvalid=Boolean(product&&!isReservationValid(option.quota,product))||Boolean(errorFor(quotaId));
              const message=product&&!isReservationValid(option.quota,product)?`Ingresa un entero sin superar la cantidad disponible (${limit??"sin límite"}).`:errorFor(quotaId)??errorFor(surchargeId);
              const name=product?.name??"Producto eliminado";
              const soldOut=option.quota==="0"||limit===0;
              const setOption=(next:typeof option)=>groupAt(index,{...group,options:group.options.map((item,position)=>position===optIndex?next:item)});
              return <li className="combo-option" key={option.productId}>
                <span className="combo-option-name"><span className="combo-option-title"><b>{name}</b>{soldOut&&<Status tone="gray">Agotado</Status>}</span><small>{product?.categoryName??"Sin categoría"}</small></span>
                <label className="combo-option-cell"><span className="combo-cell-label">Reservar</span>
                  <span className={quotaInvalid?"money-input suffix combo-quota invalid":"money-input suffix combo-quota"}>
                    <Input id={quotaId} type="number" inputMode="numeric" min="0" step="1" max={limit??undefined} value={option.quota} onChange={e=>setOption({...option,quota:e.target.value})} placeholder="Sin límite" aria-invalid={quotaInvalid} aria-describedby={message?errorId:undefined}/>
                    {limit!==null&&<span>de {limit}</span>}
                  </span>
                </label>
                <div className="combo-option-cell"><FieldLabel hint={EXTRA_COST_HINT}><label htmlFor={surchargeId} className="combo-cell-label">Costo extra</label></FieldLabel>
                  <MoneyInput id={surchargeId} symbol={currencySymbol} position={currencyPosition} value={option.surcharge} onChange={e=>setOption({...option,surcharge:decimal(e.target.value)})} placeholder="0.00" aria-invalid={Boolean(errorFor(surchargeId))}/>
                </div>
                <IconButton icon="trash" label={`Quitar ${name}`} className="combo-option-remove" onClick={()=>groupAt(index,{...group,options:group.options.filter((_,position)=>position!==optIndex)})}/>
                {message&&<small id={errorId} className="field-error combo-option-error" role="alert">{message}</small>}
              </li>;
            })}
          </ul>
        </div>}
        <ComboOptionPicker inputId={`${prefix}-product`} partName={group.name} products={products} excluded={group.options.map(option=>option.productId)} invalid={Boolean(productError)} describedBy={productError?`${prefix}-product-error`:undefined} onPick={product=>groupAt(index,{...group,options:[...group.options,createComboOption(product)]})}/>
        {productError&&<small id={`${prefix}-product-error`} className="field-error" role="alert">{productError}</small>}
      </section>;
    })}
    <Button kind="secondary" icon="plus" id="combo-add-group" className="combo-parts-add" onClick={addGroup}>Agregar parte</Button>
  </div>
}

function ComboSkeleton(){return <div className="table-skeleton" aria-label="Cargando menús"><div className="sk-head"><i/><i/><i/><i/><i/></div>{Array.from({length:5},(_,index)=><div className="sk-row" key={index}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/><i/></div>)}</div>}
