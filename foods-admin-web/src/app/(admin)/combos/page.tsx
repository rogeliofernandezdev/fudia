"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ConfirmDialog,Icon,Input,PageHeader,Pagination,RowActionButton,Select,Status,Textarea} from "@/design-system";
import {useFeedback} from "@/providers";
import {useSettings} from "@/providers/settings-context";
import {apiFetch} from "@/shared/api/client";

type Product={id:string;name:string;categoryName:string|null;price:string;active:boolean;defaultDailyQuota:number|null};
type Option={productId:string;surcharge:string;quota:string};
type Group={name:string;required:boolean;minSelections:number;maxSelections:number;options:Option[]};
type Draft={name:string;description:string;price:string;groups:Group[];availableFrom:string;availableUntil:string;availableDays:number[]};
type Combo={id:string;name:string;description:string;price:string;active:boolean;groupCount:number};
type ComboDetailOption={productId:string;name:string;surcharge:string;quota:number|null};
type ComboDetailGroup={id:string;name:string;required:boolean;minSelections:number;maxSelections:number;options:ComboDetailOption[]};
type ComboDetail=Combo&{availableFrom:string|null;availableUntil:string|null;availableDays:number[]|null;groups:ComboDetailGroup[]};

const blank:Draft={name:"",description:"",price:"",groups:[],availableFrom:"",availableUntil:"",availableDays:[]};
const templates:Group[]=[{name:"Entrada",required:true,minSelections:1,maxSelections:1,options:[]},{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[]},{name:"Postre",required:false,minSelections:0,maxSelections:1,options:[]},{name:"Refresco",required:false,minSelections:0,maxSelections:1,options:[]}];
const DAYS=[{d:1,n:"Lun"},{d:2,n:"Mar"},{d:3,n:"Mié"},{d:4,n:"Jue"},{d:5,n:"Vie"},{d:6,n:"Sáb"},{d:0,n:"Dom"}];
const STEP_LABELS=["Información","Composición","Disponibilidad","Revisión"];
const STEP_HINTS=["Datos del menú","Partes y opciones","Horario y cupos","Confirma y guarda"];
const normalize=(value:string)=>value.normalize("NFD").replace(/[\u0300-\u036f]/g,"").toLowerCase().trim();
function belongsToGroup(product:Product,groupName:string){
  const category=normalize(product.categoryName??"");
  const group=normalize(groupName);
  return category===group||category===`${group}s`||category.startsWith(`${group} `)||category.includes(group);
}

export default function Page(){
  const{notify}=useFeedback();
  const settings=useSettings();
  const client=useQueryClient();
  const[draft,setDraft]=useState<Draft|null>(null);
  const[editingId,setEditingId]=useState<string|null>(null);
  const[selected,setSelected]=useState<string|null>(null);
  const[statusTarget,setStatusTarget]=useState<Combo|null>(null);
  const[step,setStep]=useState(1);
  const[page,setPage]=useState(1);
  const[size,setSize]=useState(10);
  const[search,setSearch]=useState("");
  const[status,setStatus]=useState("");
  const combos=useQuery({queryKey:["combos",page,size,search,status],queryFn:()=>apiFetch<{items:Combo[];total:number;page:number;pageSize:number}>(`combos?page=${page}&pageSize=${size}&q=${encodeURIComponent(search)}&status=${status}`)});
  const products=useQuery({queryKey:["products","combo-picker"],queryFn:()=>apiFetch<{items:Product[]}>("products?page=1&pageSize=100&status=active")});
  const detail=useQuery({queryKey:["combo-detail",selected],queryFn:()=>apiFetch<ComboDetail>(`combos/${selected}`),enabled:Boolean(selected)});
  const save=useMutation({
    mutationFn:(value:Draft)=>apiFetch<{id:string}>(editingId?`combos/${editingId}`:"combos",{method:editingId?"PATCH":"POST",body:JSON.stringify({
      ...value,
      availableFrom:value.availableFrom?new Date(value.availableFrom).toISOString():null,
      availableUntil:value.availableUntil?new Date(value.availableUntil).toISOString():null,
      availableDays:value.availableDays.length?value.availableDays:null,
      groups:value.groups.map(g=>({...g,options:g.options.map(o=>({...o,surcharge:o.surcharge.trim()||"0",quota:o.quota?Number(o.quota):null}))}))
    })}),
    onSuccess:()=>{const edited=Boolean(editingId);setDraft(null);setEditingId(null);setStep(1);void client.invalidateQueries({queryKey:["combos"]});void client.invalidateQueries({queryKey:["products"]});notify({tone:"success",title:edited?"Menú actualizado":"Menú registrado",message:edited?"Los cambios ya están disponibles para la operación.":"La composición ya está disponible para la operación."})},
    onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})
  });
  const loadForEdit=useMutation({
    mutationFn:(id:string)=>apiFetch<ComboDetail>(`combos/${id}`),
    onSuccess:value=>{
      const toLocalInput=(source:string|null)=>{if(!source)return"";const date=new Date(source);const offset=date.getTimezoneOffset()*60000;return new Date(date.getTime()-offset).toISOString().slice(0,16)};
      setEditingId(value.id);
      setDraft({name:value.name,description:value.description,price:value.price,availableFrom:toLocalInput(value.availableFrom),availableUntil:toLocalInput(value.availableUntil),availableDays:value.availableDays??[],groups:value.groups.map(group=>({name:group.name,required:group.required,minSelections:group.minSelections,maxSelections:group.maxSelections,options:group.options.map(option=>({productId:option.productId,surcharge:option.surcharge==="0.00"?"":option.surcharge,quota:option.quota===null?"":String(option.quota)}))}))});
      setStep(1);
    },
    onError:error=>notify({tone:"danger",title:"No se pudo abrir el menú",message:error.message})
  });
  const changeStatus=useMutation({
    mutationFn:(item:Combo)=>apiFetch<void>(`combos/${item.id}/status`,{method:"PATCH",body:JSON.stringify({active:!item.active})}),
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
  const open=()=>{setEditingId(null);setDraft(blank);setStep(1)};
  const closeWizard=()=>{setDraft(null);setEditingId(null);setStep(1)};
  return <>
    <PageHeader eyebrow="CARTA Y PRODUCCIÓN" title="Menús y combos" description="Define las partes del menú y qué opciones puede elegir el cliente en cada una." action={<Button icon="plus" onClick={open}>Nuevo menú o combo</Button>}/>
    <section className="panel management catalog-panel standardized-management combo-list">
      <div className="toolbar">
        <label><Icon name="search" size={17}/><input value={search} onChange={event=>{setSearch(event.target.value);setPage(1)}} placeholder="Buscar menú o combo..." aria-label="Buscar menú o combo"/></label>
        <Select value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}} aria-label="Filtrar por estado"><option value="">Todos los estados</option><option value="active">Activos</option><option value="inactive">Inactivos</option></Select>
      </div>
      {combos.isLoading?<ComboSkeleton/>
      :combos.isError?<div className="combo-empty"><Icon name="alert" size={26}/><b>No pudimos cargar los menús y combos</b><p>{combos.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>combos.refetch()}>Reintentar</Button></div>
      :!combos.data?.items.length?
        <div className="combo-empty"><Icon name="menu" size={26}/><b>{search||status?"No encontramos resultados":"Aún no hay menús compuestos"}</b><p>{search||status?"Prueba con otro nombre o cambia el filtro de estado.":"Usa la plantilla Menú del día para comenzar rápidamente."}</p>{!search&&!status&&<Button icon="plus" onClick={open}>Crear menú</Button>}</div>
      :<div className="table-wrap hover-scroll"><table><thead><tr><th>MENÚ O COMBO</th><th>PRECIO</th><th title="Entrada, plato principal, bebida o postre">PARTES DEL MENÚ</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>
        {combos.data.items.map((item,index)=><tr className={index%2?"alternate":""} key={item.id}>
          <td><span className={`row-icon r${index%3}`}><Icon name="menu" size={18}/></span><b>{item.name}</b><small>{item.description||"Sin descripción"}</small></td>
          <td><b>{settings.currencyPosition==="before"?`${settings.currencySymbol} ${Number(item.price).toFixed(settings.currencyDecimals)}`:`${Number(item.price).toFixed(settings.currencyDecimals)} ${settings.currencySymbol}`}</b></td>
          <td>{item.groupCount} {item.groupCount===1?"parte":"partes"}</td>
          <td><Status tone={item.active?"green":"gray"}>{item.active?"Activo":"Inactivo"}</Status></td>
          <td><div className="standard-actions"><RowActionButton action="view" label={`Ver ${item.name}`} onClick={()=>setSelected(item.id)}/><RowActionButton action="edit" label={`Editar ${item.name}`} disabled={loadForEdit.isPending} onClick={()=>loadForEdit.mutate(item.id)}/><RowActionButton action={item.active?"deactivate":"activate"} label={`${item.active?"Desactivar":"Activar"} ${item.name}`} onClick={()=>setStatusTarget(item)}/></div></td>
        </tr>)}
      </tbody></table></div>}
      {!combos.isLoading&&!combos.isError&&<Pagination page={page} size={size} total={combos.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>}
    </section>
    {selected&&<ComboDetailDialog query={detail} currencySymbol={settings.currencySymbol} close={()=>setSelected(null)}/>}
    <ConfirmDialog open={Boolean(statusTarget)} title={`${statusTarget?.active?"Desactivar":"Activar"} menú`} description={statusTarget?.active?`“${statusTarget.name}” dejará de estar disponible para nuevas ventas, pero conservará su historial.`:`“${statusTarget?.name??""}” volverá a estar disponible para la operación.`} confirmLabel={statusTarget?.active?"Desactivar":"Activar"} pending={changeStatus.isPending} onCancel={()=>setStatusTarget(null)} onConfirm={()=>statusTarget&&changeStatus.mutate(statusTarget)}/>
    {draft&&<ComboWizard draft={draft} setDraft={setDraft} step={step} setStep={setStep} products={products.data?.items??[]} productsLoading={products.isLoading} productsError={products.isError} retryProducts={()=>products.refetch()} currencySymbol={settings.currencySymbol} busy={save.isPending} editing={Boolean(editingId)} close={closeWizard} finish={()=>save.mutate(draft)}/>}
  </>
}

function ComboDetailDialog({query,currencySymbol,close}:{query:ReturnType<typeof useQuery<ComboDetail,Error>>;currencySymbol:string;close:()=>void}){
  const value=query.data;
  const from=value?.availableFrom?new Date(value.availableFrom):null;
  const until=value?.availableUntil?new Date(value.availableUntil):null;
  const singleDay=Boolean(from&&until&&until.getTime()>from.getTime()&&until.getTime()-from.getTime()<=24*60*60*1000);
  const days=singleDay?"Solo hoy":value?.availableDays?.length?value.availableDays.map(day=>DAYS.find(item=>item.d===day)?.n).filter(Boolean).join(", "):"Todos los días";
  const formatDate=(date:string|null|undefined)=>date?new Intl.DateTimeFormat("es-PE",{dateStyle:"medium",timeStyle:"short"}).format(new Date(date)):"Sin límite";
  const formatCalendarDate=(date:string|null|undefined)=>{if(!date)return"";const[y,m,d]=date.slice(0,10).split("-").map(Number);return new Intl.DateTimeFormat("es-PE",{dateStyle:"long",timeZone:"UTC"}).format(new Date(Date.UTC(y,m-1,d)))};
  return <div className="modal-backdrop modal-overlay-in" role="presentation">
    <section className="crud-modal combo-detail modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="combo-detail-title">
      <div className="modal-accent"/>
      <header><span className="modal-title-icon"><Icon name="eye" size={18}/></span><div><h2 id="combo-detail-title">{value?.name??"Cargando menú"}</h2><small>DETALLE DEL MENÚ</small></div><button aria-label="Cerrar" onClick={close}><Icon name="close"/></button></header>
      <div className="combo-detail-body">
        {query.isLoading?<ComboDetailSkeleton/>:query.isError?<div className="combo-detail-state"><Icon name="alert" size={24}/><b>No pudimos cargar el menú</b><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>query.refetch()}>Reintentar</Button></div>:value&&<>
          <section className="combo-detail-summary"><div><small>PRECIO</small><b>{currencySymbol} {Number(value.price).toFixed(2)}</b></div><div><small>PARTES DEL MENÚ</small><b>{value.groups.length} {value.groups.length===1?"parte":"partes"}</b></div><Status tone={value.active?"green":"gray"}>{value.active?"Activo":"Inactivo"}</Status></section>
          {value.description&&<section className="combo-detail-description"><small>DESCRIPCIÓN</small><p>{value.description}</p></section>}
          <section className="combo-detail-section"><header><div><small>COMPOSICIÓN</small><h3>Partes del menú y opciones disponibles</h3></div></header>{value.groups.map(group=><article className="combo-detail-group" key={group.id}><div><b>{group.name}</b><small>{group.required?"Obligatorio":"Opcional"} · El cliente elige {group.minSelections}–{group.maxSelections}</small></div><ul>{group.options.map(option=><li key={option.productId}><span>{option.name}</span>{Number(option.surcharge)>0&&<small>+ {currencySymbol} {Number(option.surcharge).toFixed(2)}</small>}</li>)}</ul></article>)}</section>
          <section className={`combo-detail-availability${singleDay?" single":""}`}>{singleDay?<div><small>VIGENCIA</small><b>Solo hoy · {formatCalendarDate(value.availableFrom)}</b></div>:<><div><small>DÍAS DISPONIBLES</small><b>{days}</b></div><div><small>DESDE</small><b>{formatDate(value.availableFrom)}</b></div><div><small>HASTA</small><b>{formatDate(value.availableUntil)}</b></div></>}</section>
        </>}
      </div>
    </section>
  </div>
}

function ComboDetailSkeleton(){return <div className="combo-detail-skeleton" aria-label="Cargando detalle"><i/><div><i/><i/><i/></div><i/><i/></div>}

function ComboWizard({draft,setDraft,step,setStep,products,productsLoading,productsError,retryProducts,currencySymbol,busy,editing,close,finish}:{draft:Draft;setDraft:(value:Draft)=>void;step:number;setStep:(value:number)=>void;products:Product[];productsLoading:boolean;productsError:boolean;retryProducts:()=>void;currencySymbol:string;busy:boolean;editing:boolean;close:()=>void;finish:()=>void}){
  const validInfo=draft.name.trim()!==""&&/^\d+(\.\d{1,2})?$/.test(draft.price);
  const validGroups=draft.groups.length>0&&draft.groups.every(group=>group.name.trim()!==""&&(!group.required||group.options.length>=1));
  const quotaError=(groupIndex:number,optIndex:number)=>{const option=draft.groups[groupIndex]?.options[optIndex];if(!option?.quota)return false;const product=products.find(p=>p.id===option.productId);return product?.defaultDailyQuota!=null&&Number(option.quota)>product.defaultDailyQuota};
  const anyQuotaExceeded=draft.groups.some((_,gi)=>draft.groups[gi].options.some((_,oi)=>quotaError(gi,oi)));
  function next(){if(step===1&&!validInfo)return;if(step===2&&(!validGroups||anyQuotaExceeded))return;setStep(Math.min(4,step+1))}
  function addTemplate(){setDraft({...draft,groups:templates.map(group=>({...group,options:products.filter(product=>belongsToGroup(product,group.name)).map(product=>({productId:product.id,surcharge:"",quota:product.defaultDailyQuota!=null?String(product.defaultDailyQuota):""}))}))})}
  function addGroup(){setDraft({...draft,groups:[...draft.groups,{name:"Nueva parte",required:true,minSelections:1,maxSelections:1,options:[]}]})}
  function groupAt(index:number,next:Group){setDraft({...draft,groups:draft.groups.map((group,position)=>position===index?next:group)})}
  return <div className="modal-backdrop modal-overlay-in">
    <section className="crud-modal combo-wizard modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="combo-wizard-title">
      <header className="combo-wizard-header">
        <div className="combo-wizard-title">
          <span className="combo-wizard-title-icon"><Icon name="combo" size={20}/></span>
          <div>
            <small>{editing?"EDITAR MENÚ O COMBO":"NUEVO MENÚ O COMBO"}</small>
            <h2 id="combo-wizard-title">{editing?"Editar menú o combo":"Registrar menú o combo"}</h2>
            <p>{STEP_LABELS[step-1]} · {STEP_HINTS[step-1]}</p>
          </div>
        </div>
        <span className="combo-wizard-progress">Paso {step} de {STEP_LABELS.length}</span>
        <button className="combo-wizard-close" aria-label="Cerrar" onClick={close}><Icon name="close" size={18}/></button>
      </header>

      <div className="combo-wizard-layout">
        <aside className="combo-wizard-sidebar" aria-label="Progreso del registro">
          <div className="combo-wizard-sidebar-head">
            <span>CONFIGURACIÓN</span>
            <b>{editing?"Actualiza el menú":"Completa el menú"}</b>
            <small>Define la información comercial, las elecciones y su disponibilidad.</small>
          </div>

          <nav className="combo-wizard-steps">
            {STEP_LABELS.map((label,index)=><button type="button" key={label} className={step===index+1?"active":step>index+1?"complete":""} aria-current={step===index+1?"step":undefined} disabled={index+1>step} onClick={()=>setStep(index+1)}>
              <span className="combo-step-marker">{step>index+1?<Icon name="check" size={14}/>:index+1}</span>
              <span className="combo-step-copy"><b>{label}</b><small>{STEP_HINTS[index]}</small></span>
            </button>)}
          </nav>

          <div className="combo-wizard-sidebar-tip">
            <Icon name="receipt" size={16}/>
            <span>
              <b>{draft.name||"Menú sin nombre"}</b>
              <small>{draft.price?currencySymbol+" "+Number(draft.price||0).toFixed(2):"Precio pendiente"}</small>
            </span>
          </div>
        </aside>

        <main className="combo-wizard-stage">
          <div className="combo-wizard-body">
            {step===1&&<InfoStep draft={draft} setDraft={setDraft} currencySymbol={currencySymbol}/>}
            {step===2&&<CompositionStep draft={draft} setDraft={setDraft} products={products} productsLoading={productsLoading} productsError={productsError} retryProducts={retryProducts} currencySymbol={currencySymbol} addTemplate={addTemplate} addGroup={addGroup} groupAt={groupAt} quotaError={quotaError}/>}
            {step===3&&<AvailabilityStep draft={draft} setDraft={setDraft}/>}
            {step===4&&<ReviewStep draft={draft} products={products} currencySymbol={currencySymbol}/>}
          </div>

          <footer className="combo-wizard-footer">
            <button type="button" className="button ghost" onClick={step===1?close:()=>setStep(step-1)} disabled={busy}>{step===1?<><Icon name="close" size={16}/>Cancelar</>:<><Icon name="chevronLeft" size={16}/>Anterior</>}</button>
            <div className="combo-wizard-footer-copy">
              <small>{step<4?"Puedes volver a modificar pasos anteriores.":"Revisa los datos antes de guardar."}</small>
            </div>
            {step<4?<button type="button" className="button primary" onClick={next} disabled={busy}>Continuar<Icon name="chevron" size={16}/></button>:<button className="button primary" onClick={finish} disabled={busy}><Icon name="save" size={16}/>{busy?"Guardando…":editing?"Guardar cambios":"Registrar menú"}</button>}
          </footer>
        </main>
      </div>
    </section>
  </div>
}

function InfoStep({draft,setDraft,currencySymbol}:{draft:Draft;setDraft:(v:Draft)=>void;currencySymbol:string}){
  return <div className="form-grid">
    <div className="combo-section-heading span-2"><h3>Datos comerciales</h3><p>Define cómo se mostrará y venderá este menú.</p></div>
    <label>Nombre<Input autoFocus value={draft.name} onChange={e=>setDraft({...draft,name:e.target.value})} placeholder="Ej. Menú ejecutivo"/></label>
    <label className="combo-price-field">Precio del menú<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" value={draft.price} onChange={e=>setDraft({...draft,price:e.target.value})} placeholder="0.00"/></div></label>
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
      <div><b>Partes del menú y opciones</b><small>Crea cada parte —Entrada, Plato principal, Bebida o Postre— y define qué productos podrá elegir el cliente.</small></div>
      <div><Button kind="secondary" icon="menu" onClick={addTemplate}>Plantilla Menú del día</Button><Button icon="plus" onClick={addGroup}>Agregar parte</Button></div>
    </div>
    {draft.groups.length===0?
      <div className="combo-empty-groups"><Icon name="menu" size={24}/><b>Agrega las partes del menú</b><p>Usa la plantilla o crea partes como Entrada, Plato principal o Postre.</p></div>
    :draft.groups.map((group,index)=>
      <div className="combo-group" key={index}>
        <div className="combo-group-header">
          <Input value={group.name} onChange={e=>groupAt(index,{...group,name:e.target.value})} placeholder="Ej. Entrada o Plato principal"/>
          <label className="combo-toggle"><input type="checkbox" checked={group.required} onChange={e=>groupAt(index,{...group,required:e.target.checked,minSelections:e.target.checked?1:0,maxSelections:1})}/><span/><b>Obligatorio</b></label>
          <button className="combo-remove" onClick={()=>setDraft({...draft,groups:draft.groups.filter((_,p)=>p!==index)})} aria-label="Quitar parte del menú"><Icon name="close" size={16}/></button>
        </div>
        <div className="combo-options">
          {group.options.map((option,optIndex)=>
            <div className="combo-option" key={optIndex}>
              <span className="combo-option-name">{products.find(p=>p.id===option.productId)?.name??"Producto eliminado"}<small>{products.find(p=>p.id===option.productId)?.categoryName??"Sin categoría"}</small></span>
              <label className="combo-mini">Reservar<Input type="number" min="0" value={option.quota} onChange={e=>groupAt(index,{...group,options:group.options.map((o,p)=>p===optIndex?{...o,quota:e.target.value}:o)})} placeholder="0" aria-invalid={quotaError(index,optIndex)}/>{quotaError(index,optIndex)&&<small className="wizard-field-error">Supera el stock ({products.find(p=>p.id===option.productId)?.defaultDailyQuota})</small>}</label>
              <label className="combo-mini">Recargo<div className="money-input"><span>{currencySymbol}</span><Input inputMode="decimal" value={option.surcharge} onChange={e=>groupAt(index,{...group,options:group.options.map((o,p)=>p===optIndex?{...o,surcharge:e.target.value}:o)})} placeholder="0.00"/></div></label>
              <button className="combo-remove" onClick={()=>groupAt(index,{...group,options:group.options.filter((_,p)=>p!==optIndex)})} aria-label="Quitar opción"><Icon name="close" size={14}/></button>
            </div>
          )}
          <div className="combo-add-option">
            <select value="" onChange={e=>{const product=products.find(p=>p.id===e.target.value);if(product)groupAt(index,{...group,options:[...group.options,{productId:product.id,surcharge:"",quota:product.defaultDailyQuota!=null?String(product.defaultDailyQuota):""}]});e.target.value=""}}>
              <option value="">+ Agregar opción</option>
              {categories.map(category=><optgroup label={category} key={category}>{products.filter(product=>(product.categoryName??"Sin categoría")===category&&!group.options.some(option=>option.productId===product.id)).map(product=><option value={product.id} key={product.id}>{product.name}</option>)}</optgroup>)}
            </select>
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
    <div className="combo-section-heading span-2"><h3>Vigencia del menú</h3><p>Es opcional. Sin fechas ni días seleccionados estará siempre disponible.</p></div>
    <label className="span-2">Días disponibles
      <div className="days-chips">
        <button type="button" className={isToday?"day-chip today active":"day-chip today"} onClick={setToday}>Hoy</button>
        {DAYS.map(day=><button type="button" key={day.d} className={draft.availableDays.includes(day.d)?"day-chip active":"day-chip"} onClick={()=>setDraft({...draft,availableDays:draft.availableDays.includes(day.d)?draft.availableDays.filter((x:number)=>x!==day.d):[...draft.availableDays,day.d]})}>{day.n}</button>)}
      </div>
    </label>
    <label>Fecha desde<Input type="datetime-local" value={draft.availableFrom} onChange={e=>setDraft({...draft,availableFrom:e.target.value})}/></label>
    <label>Fecha hasta<Input type="datetime-local" value={draft.availableUntil} onChange={e=>setDraft({...draft,availableUntil:e.target.value})}/></label>
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
    <div className="combo-section-heading"><h3>Revisa el menú</h3><p>Confirma la información antes de guardar.</p></div>
    <div className="combo-review-section">
      <b>Información</b>
      <dl>
        <div><dt>Nombre</dt><dd>{draft.name||"Sin nombre"}</dd></div>
        <div><dt>Precio</dt><dd>{currencySymbol} {draft.price||"0.00"}</dd></div>
        <div><dt>Descripción</dt><dd>{draft.description||"Sin descripción"}</dd></div>
      </dl>
    </div>
    <div className="combo-review-section">
      <b>Composición</b>
      {draft.groups.map((group,index)=>
        <div key={index} className="combo-review-group">
          <dt>{group.name} {group.required?"(obligatorio)":"(opcional)"}</dt>
          <dd>{group.options.length?group.options.map(o=>products.find(p=>p.id===o.productId)?.name??"?").join(", "):"Sin opciones"}</dd>
        </div>
      )}
    </div>
    <div className="combo-review-section">
      <b>Disponibilidad</b>
      <dl>
        <div><dt>Días</dt><dd>{daysLabel}</dd></div>
        <div><dt>Vigencia</dt><dd>{rangeLabel}</dd></div>
      </dl>
    </div>
  </div>
}

function ComboSkeleton(){return <div className="table-skeleton" aria-label="Cargando menús"><div className="sk-head"><i/><i/><i/><i/><i/></div>{Array.from({length:5},(_,index)=><div className="sk-row" key={index}><i className="sk-name"><span/><b/><small/></i><i/><i/><i/><i/></div>)}</div>}
