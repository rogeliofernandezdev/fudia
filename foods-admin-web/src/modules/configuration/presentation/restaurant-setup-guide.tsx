"use client";

import "./restaurant-setup-guide.css";
import Link from "next/link";
import {useCallback,useState,useSyncExternalStore} from "react";
import {Button,PageHeader} from "@/design-system/page-header";
import {Icon,type IconName} from "@/design-system/icons";
import {useSession} from "@/providers";
import {canOpenNavigationItem,navigationItemForPath} from "@/shell/navigation";

type GuideStep={
  id:string;
  title:string;
  icon:IconName;
  href:string;
  tooltip:string;
  details:string[];
};

// These are instructions for existing screens, never substitute business data from the API.
const guideSteps:GuideStep[]=[
  {id:"empresa",title:"Empresa",icon:"store",href:"/configuracion/empresa",tooltip:"Revisa la identidad legal del restaurante antes de operar.",details:["Confirma la razón social, el nombre comercial y la identificación fiscal.","Revisa la zona horaria predeterminada; el primer administrador ya fue creado al dar de alta la empresa."]},
  {id:"fiscal",title:"Fiscal y moneda",icon:"receipt",href:"/configuracion/fiscal",tooltip:"Comprueba el país, la moneda y el impuesto del perfil fiscal.",details:["Revisa el perfil fiscal inicial: país, moneda, nombre y porcentaje del impuesto.","Registra tipos de cambio solo si vas a operar en más de una moneda."]},
  {id:"locales",title:"Locales",icon:"store",href:"/locales",tooltip:"Completa los datos del local donde atenderás a los clientes.",details:["Verifica el local principal y completa su dirección, teléfono, horario y zona horaria.","Asigna el perfil fiscal que corresponda; agrega otros locales solo si los necesitas."]},
  {id:"usuarios",title:"Usuarios y roles",icon:"users",href:"/configuracion/usuarios",tooltip:"Da acceso al equipo según su función y el local donde trabaja.",details:["Invita a las personas que atenderán pedidos, cocinarán o cobrarán.","Revisa sus roles, permisos y asignaciones por local antes de compartir el acceso."]},
  {id:"pagos",title:"Medios de pago",icon:"payment",href:"/configuracion/medios-pago",tooltip:"Elige los medios con los que aceptarás pagos y registrarás gastos.",details:["Revisa los medios iniciales y deja activos los que realmente aceptarás.","Comprueba cuáles afectan el efectivo de caja; agrega otros medios si hacen falta."]},
  {id:"productos",title:"Carta y productos",icon:"utensils",href:"/productos",tooltip:"Registra lo que el restaurante ofrecerá y cuánto cuesta.",details:["Crea las categorías y los primeros platos o bebidas con nombre y precio.","Configura la disponibilidad de cada producto según cómo trabajará tu cocina."]},
  {id:"mesas",title:"Mesas y zonas",icon:"grid",href:"/mesas",tooltip:"Organiza el salón y registra las mesas que usarás.",details:["Revisa la zona Principal y añade las zonas que tenga tu local.","Registra las mesas, su zona y capacidad para empezar a tomar comandas."]},
  {id:"caja",title:"Caja y turnos",icon:"sales",href:"/caja",tooltip:"Prepara la caja antes de recibir el primer cobro.",details:["Revisa la Caja principal creada al iniciar la empresa.","Asigna el equipo que la operará y abre un turno con el monto inicial cuando estés listo para vender."]},
];

export function RestaurantSetupGuide(){
  const{user,organization,location,modules,menuAccess,permissions,can}=useSession();
  const context=user&&modules?{user,modules,menuAccess,permissions}:null;
  const steps=context?guideSteps.filter(step=>{
    const item=navigationItemForPath(step.href);
    return item&&canOpenNavigationItem(item,context);
  }):[];
  const stepIds=steps.map(step=>step.id).join("|");
  const storageKey=user&&organization&&location?`fudia-setup-guide:${user.id}:${organization.id}:${location.id}`:null;
  const[progress,setProgress]=useState<{key:string;id:string}|null>(null);
  const subscribe=useCallback((notify:()=>void)=>{
    window.addEventListener("storage",notify);
    window.addEventListener("fudia-setup-progress",notify);
    return()=>{window.removeEventListener("storage",notify);window.removeEventListener("fudia-setup-progress",notify)};
  },[]);
  const getSnapshot=useCallback(()=>{
    try{return storageKey?window.localStorage.getItem(storageKey):null}catch{return null}
  },[storageKey]);
  const storedId=useSyncExternalStore(subscribe,getSnapshot,()=>null);

  function showStep(id:string){
    if(!storageKey)return;
    setProgress({key:storageKey,id});
    try{window.localStorage.setItem(storageKey,id);window.dispatchEvent(new Event("fudia-setup-progress"))}catch{/* The guide still works for this visit. */}
  }

  const savedId=progress?.key===storageKey?progress.id:storedId;
  const selectedId=savedId==="done"||stepIds.split("|").includes(savedId??"")?savedId:steps[0]?.id;
  const index=steps.findIndex(step=>step.id===selectedId);
  const step=index<0?steps[0]:steps[index];
  const finished=selectedId==="done";

  return <>
    <PageHeader eyebrow="CONFIGURACIÓN" title="Puesta en marcha" description="Un recorrido por lo que necesitas revisar o registrar para comenzar a atender en tu restaurante."/>
    {!organization?<section className="setup-guide-empty panel"><Icon name="store" size={28}/><h2>Selecciona una empresa</h2><p>Elige la empresa y el local para ver su recorrido de puesta en marcha.</p></section>:
    !steps.length?<section className="setup-guide-empty panel"><Icon name="lock" size={28}/><h2>Sin opciones disponibles</h2><p>Tu rol aún no puede abrir las pantallas de configuración. Pide al administrador que revise tus accesos.</p></section>:
    <section className="setup-guide" aria-label="Asistente de puesta en marcha">
      <div className="setup-guide-heading"><div><b>{finished?"Recorrido terminado":`Paso ${index+1} de ${steps.length}`}</b><span>{location?.name} · {organization.name}</span></div><progress value={finished?steps.length:index+1} max={steps.length} aria-label="Progreso de la guía"/></div>
      <div className="setup-guide-layout">
        <nav className="setup-guide-steps" aria-label="Opciones de la guía">
          {steps.map((item,i)=><button key={item.id} type="button" className={`setup-guide-step${item.id===selectedId?" active":""}`} aria-current={item.id===selectedId?"step":undefined} aria-label={`${i+1}. ${item.title}. ${item.tooltip}`} data-tooltip={item.tooltip} onClick={()=>showStep(item.id)}>
            <span className="setup-guide-number">{i+1}</span><span className="setup-guide-step-icon"><Icon name={item.icon} size={19}/></span><span className="setup-guide-step-copy"><b>{item.title}</b><small>{item.tooltip}</small></span><Icon name="chevron" size={17}/>
          </button>)}
        </nav>
        {finished?<div className="setup-guide-callout" aria-live="polite"><span className="setup-guide-callout-icon"><Icon name="check" size={24}/></span><small>FIN DEL RECORRIDO</small><h2>Ya conoces las opciones para empezar</h2><p>El recorrido no comprueba los datos guardados. Revisa que la carta, las mesas, los medios de pago y el turno de caja estén listos antes de atender.</p><div className="setup-guide-actions"><Button kind="secondary" onClick={()=>showStep(steps[0].id)}>Repetir guía</Button>{navigationItemForPath("/pos")&&context&&canOpenNavigationItem(navigationItemForPath("/pos")!,context)&&<Link className="button primary" href="/pos"><Icon name="sales" size={18}/>Ir al punto de venta</Link>}</div></div>:
        <div className="setup-guide-callout" aria-live="polite" aria-labelledby="setup-guide-current-title">
          <span className="setup-guide-callout-icon"><Icon name={step.icon} size={24}/></span>
          <small>ASISTENTE · {index+1} / {steps.length}</small>
          <h2 id="setup-guide-current-title">{step.title}</h2>
          <p>{step.tooltip}</p>
          <h3>Qué debes registrar o revisar</h3>
          <ul>{step.details.map(detail=><li key={detail}><Icon name="check" size={16}/>{detail}</li>)}</ul>
          <p className="setup-guide-hint">{can(step.id==="usuarios"?"users.manage":step.id==="mesas"?"tables.manage":step.id==="caja"?"cash.manage":step.id==="productos"?"menu.manage":"organizations.manage")?"Cuando termines en esa pantalla, vuelve aquí y continúa con la siguiente opción.":"Puedes consultar esta opción. Para registrar cambios, pide a un administrador el permiso correspondiente."}</p>
          <div className="setup-guide-actions"><Link className="button primary" href={step.href}><Icon name="chevron" size={18}/>Abrir {step.title}</Link><div><Button kind="secondary" disabled={index===0} onClick={()=>showStep(steps[index-1].id)}>Anterior</Button><Button kind="secondary" onClick={()=>showStep(steps[index+1]?.id??"done")}>{index===steps.length-1?"Finalizar guía":"Siguiente"}</Button></div></div>
        </div>}
      </div>
      <p className="setup-guide-footnote">Tu avance se guarda en este navegador para la empresa y el local seleccionados. La guía no cambia la configuración por sí sola.</p>
    </section>}
  </>;
}
