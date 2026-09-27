"use client";
import "./restaurant-setup.css";
import Link from "next/link";
import {useMemo,useState} from "react";
import {useRouter} from "next/navigation";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,PageHeader,Status} from "@/design-system";
import type {IconName} from "@/design-system";
import {useFeedback} from "@/providers/feedback-provider";
import type {RestaurantSetup,ServiceMode} from "../domain/types";
import {completeRestaurantSetup,getRestaurantSetup,updateRestaurantServiceMode} from "../infrastructure/setup-api";

type SetupStep={
  key:string;
  title:string;
  description:string;
  icon:IconName;
  required:boolean;
  complete:boolean;
  href?:string;
};

const serviceModes:Array<{value:Exclude<ServiceMode,"">;title:string;description:string;icon:IconName}>=[
  {value:"counter",title:"Mostrador y para llevar",description:"Atención sin asignar mesas a los pedidos.",icon:"store"},
  {value:"dine_in",title:"Atención en salón",description:"Los pedidos se organizan por mesas y zonas.",icon:"grid"},
  {value:"mixed",title:"Operación mixta",description:"Combina salón, mostrador y pedidos para llevar.",icon:"utensils"},
];

function buildSteps(setup:RestaurantSetup):SetupStep[]{
  const tablesRequired=setup.serviceMode==="dine_in"||setup.serviceMode==="mixed";
  const steps:SetupStep[]=[
    {key:"operation",title:"Forma de atención",description:"Define cómo recibirá pedidos este local.",icon:"store",required:true,complete:Boolean(setup.serviceMode)&&(!tablesRequired||setup.counts.tables>0)},
    {key:"catalog",title:"Carta inicial",description:"Crea las categorías y productos que venderás.",icon:"utensils",required:true,complete:setup.counts.categories>0&&setup.counts.products>0,href:"/productos"},
    {key:"cash",title:"Caja",description:"Confirma que el local tenga una caja disponible.",icon:"sales",required:true,complete:setup.counts.cashRegisters>0,href:"/caja"},
    {key:"team",title:"Equipo",description:"Invita cajeros, meseros o cocineros cuando los necesites.",icon:"users",required:false,complete:setup.counts.users>1,href:"/configuracion/usuarios"},
  ];
  if(setup.modules.inventario)steps.push({key:"inventory",title:"Inventario",description:"Registra insumos o mercadería y sus existencias iniciales.",icon:"stock",required:false,complete:setup.counts.inventoryItems>0,href:"/inventario"});
  if(setup.modules.recetas&&setup.modules.inventario)steps.push({key:"recipes",title:"Recetas",description:"Vincula platos con sus insumos para controlar consumo y costo.",icon:"cookingPot",required:false,complete:setup.counts.recipes>0,href:"/recetas"});
  if(setup.modules.compras)steps.push({key:"purchases",title:"Proveedores y compras",description:"Registra proveedores para organizar el abastecimiento.",icon:"truck",required:false,complete:setup.counts.suppliers>0,href:"/compras"});
  steps.push({key:"finish",title:"Revisión",description:"Comprueba lo esencial y habilita la operación.",icon:"check",required:true,complete:setup.coreReady});
  return steps;
}

export function RestaurantSetupPage(){
  const router=useRouter();
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const[activeKey,setActiveKey]=useState("operation");
  const query=useQuery({queryKey:["restaurant-setup"],queryFn:getRestaurantSetup});
  const setup=query.data;
  const steps=useMemo(()=>setup?buildSteps(setup):[],[setup]);
  const activeIndex=Math.max(0,steps.findIndex(step=>step.key===activeKey));
  const active=steps[activeIndex];
  const required=steps.filter(step=>step.required&&step.key!=="finish");
  const completedRequired=required.filter(step=>step.complete).length;

  const saveMode=useMutation({
    mutationFn:updateRestaurantServiceMode,
    onSuccess:data=>{
      qc.setQueryData(["restaurant-setup"],data);
      void qc.invalidateQueries({queryKey:["session-context"]});
      notify({tone:"success",title:"Tipo de atención guardado",message:"El asistente ajustó los requisitos del local."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message}),
  });
  const finish=useMutation({
    mutationFn:completeRestaurantSetup,
    onSuccess:async data=>{
      qc.setQueryData(["restaurant-setup"],data);
      await qc.invalidateQueries({queryKey:["session-context"]});
      notify({tone:"success",title:"Restaurante listo",message:"La configuración esencial quedó completa. Ya puedes iniciar la operación."});
      router.replace("/dashboard");
    },
    onError:error=>notify({tone:"danger",title:"Aún faltan datos",message:error.message}),
  });

  if(query.isLoading)return <SetupSkeleton/>;
  if(query.isError)return <><PageHeader eyebrow="CONFIGURACIÓN" title="Puesta en marcha" description="Prepara el restaurante para iniciar su operación."/><section className="restaurant-setup-state"><Icon name="alert" size={25}/><h2>No pudimos cargar el asistente</h2><p>{query.error.message}</p><Button kind="secondary" icon="refresh" onClick={()=>void query.refetch()}>Reintentar</Button></section></>;
  if(!setup||!active)return null;

  return <>
    <PageHeader eyebrow="CONFIGURACIÓN INICIAL" title="Puesta en marcha" description="Completa lo esencial para que el restaurante pueda comenzar a atender."/>
    <section className="restaurant-setup-overview">
      <div className="restaurant-setup-progress-copy"><span><Icon name={setup.completedAt?"check":"arrowRightCircle"} size={19}/></span><div><small>AVANCE OBLIGATORIO</small><b>{completedRequired} de {required.length} requisitos listos</b></div></div>
      <div className="restaurant-setup-progress-track" aria-label={`${completedRequired} de ${required.length} requisitos completos`}><i style={{width:`${required.length?completedRequired/required.length*100:0}%`}}/></div>
      <Status tone={setup.completedAt?"green":setup.coreReady?"blue":"orange"}>{setup.completedAt?"Configurado":setup.coreReady?"Listo para finalizar":"Configuración pendiente"}</Status>
    </section>

    <div className="restaurant-setup-layout">
      <nav className="restaurant-setup-steps" aria-label="Pasos de puesta en marcha">
        {steps.map((step,index)=><button type="button" className={`${step.key===active.key?"active ":""}${step.complete?"complete":""}`.trim()} onClick={()=>setActiveKey(step.key)} aria-current={step.key===active.key?"step":undefined} key={step.key}>
          <span>{step.complete?<Icon name="check" size={16}/>:<Icon name={step.icon} size={16}/>}</span>
          <div><b>{step.title}</b><small>{step.required?"Obligatorio":"Opcional según tu operación"}</small></div>
          <em>{String(index+1).padStart(2,"0")}</em>
        </button>)}
      </nav>

      <section className="restaurant-setup-content">
        <header><span><Icon name={active.icon} size={20}/></span><div><small>{active.required?"REQUISITO OBLIGATORIO":"CONFIGURACIÓN OPCIONAL"}</small><h2>{active.title}</h2><p>{active.description}</p></div>{active.complete&&<Status>Listo</Status>}</header>
        <div className="restaurant-setup-body">{renderStep(active,setup,saveMode.isPending,mode=>saveMode.mutate(mode),()=>finish.mutate(),finish.isPending)}</div>
        <footer>
          <Button kind="ghost" icon="chevronLeft" disabled={activeIndex===0} onClick={()=>setActiveKey(steps[activeIndex-1].key)}>Anterior</Button>
          {activeIndex<steps.length-1&&<Button icon="chevron" onClick={()=>setActiveKey(steps[activeIndex+1].key)}>Siguiente</Button>}
        </footer>
      </section>
    </div>
  </>;
}

function renderStep(step:SetupStep,setup:RestaurantSetup,savingMode:boolean,saveMode:(mode:Exclude<ServiceMode,"">)=>void,finish:()=>void,finishing:boolean){
  const tablesRequired=setup.serviceMode==="dine_in"||setup.serviceMode==="mixed";
  if(step.key==="operation")return <>
    <div className="restaurant-setup-service-modes" role="radiogroup" aria-label="Forma de atención">
      {serviceModes.map(mode=><button type="button" role="radio" aria-checked={setup.serviceMode===mode.value} className={setup.serviceMode===mode.value?"selected":""} disabled={savingMode} onClick={()=>saveMode(mode.value)} key={mode.value}>
        <i><Icon name={mode.icon} size={19}/></i><span><b>{mode.title}</b><small>{mode.description}</small></span><em><u/></em>
      </button>)}
    </div>
    {tablesRequired&&<SetupRequirement icon="grid" title="Mesas del local" value={setup.counts.tables} complete={setup.counts.tables>0} text={setup.counts.tables>0?`${setup.counts.tables} mesas activas en este local.`:"Crea al menos una mesa para atender en salón."} href="/mesas"/>}
  </>;
  if(step.key==="catalog")return <div className="restaurant-setup-requirements"><SetupRequirement icon="menu" title="Categorías" value={setup.counts.categories} complete={setup.counts.categories>0} text="Organizan la carta para encontrar productos con rapidez." href="/productos"/><SetupRequirement icon="utensils" title="Productos disponibles" value={setup.counts.products} complete={setup.counts.products>0} text="Registra al menos un producto activo con categoría y precio." href="/productos"/></div>;
  if(step.key==="cash")return <SetupRequirement icon="sales" title="Caja operativa" value={setup.counts.cashRegisters} complete={setup.counts.cashRegisters>0} text={setup.counts.cashRegisters>0?"La Caja principal está disponible. El turno se abre al comenzar cada jornada.":"El local necesita al menos una caja activa."} href="/caja"/>;
  if(step.key==="team")return <SetupRequirement icon="users" title="Usuarios activos" value={setup.counts.users} complete={setup.counts.users>1} text={setup.counts.users>1?"El equipo ya tiene usuarios además del administrador.":"Puedes operar inicialmente con el administrador e invitar al equipo después."} href="/configuracion/usuarios" optional/>;
  if(step.key==="inventory")return <SetupRequirement icon="stock" title="Artículos de inventario" value={setup.counts.inventoryItems} complete={setup.counts.inventoryItems>0} text="Este módulo está incluido en tu plan. Puedes registrar insumos y stock inicial ahora o después." href="/inventario" optional/>;
  if(step.key==="recipes")return <SetupRequirement icon="cookingPot" title="Recetas activas" value={setup.counts.recipes} complete={setup.counts.recipes>0} text="Recetas aparece porque está incluido en tu plan; no forma parte del plan básico." href="/recetas" optional/>;
  if(step.key==="purchases")return <SetupRequirement icon="truck" title="Proveedores activos" value={setup.counts.suppliers} complete={setup.counts.suppliers>0} text="Configura proveedores si organizarás órdenes y recepciones desde el sistema." href="/compras" optional/>;
  return <div className="restaurant-setup-review">
    <div className={setup.coreReady?"ready":"pending"}><Icon name={setup.coreReady?"check":"alert"} size={24}/><div><h3>{setup.coreReady?"Todo lo esencial está listo":"Aún faltan requisitos obligatorios"}</h3><p>{setup.coreReady?"Puedes finalizar el asistente. Las configuraciones opcionales seguirán disponibles en el menú.":"Revisa los pasos marcados como obligatorios antes de finalizar."}</p></div></div>
    <ul>{buildSteps(setup).filter(item=>item.key!=="finish").map(item=><li key={item.key}><Icon name={item.complete?"check":"clock"} size={15}/><span><b>{item.title}</b><small>{item.required?"Obligatorio":"Opcional"}</small></span><em>{item.complete?"Listo":"Pendiente"}</em></li>)}</ul>
    <Button icon="check" disabled={!setup.coreReady||finishing} onClick={finish}>{finishing?"Finalizando…":"Finalizar puesta en marcha"}</Button>
  </div>;
}

function SetupRequirement({icon,title,value,complete,text,href,optional=false}:{icon:IconName;title:string;value:number;complete:boolean;text:string;href:string;optional?:boolean}){
  return <article className={`restaurant-setup-requirement${complete?" complete":""}`}><span><Icon name={icon} size={20}/></span><div><small>{optional?"OPCIONAL":"REQUISITO"}</small><h3>{title}</h3><p>{text}</p></div><strong>{value}</strong><Link className="button secondary" href={href}><Icon name={complete?"eye":"arrowRightCircle"} size={17}/>{complete?"Revisar":"Configurar"}</Link></article>;
}

function SetupSkeleton(){return <><div className="restaurant-setup-title-skeleton"><i/><i/></div><div className="restaurant-setup-skeleton"><aside>{Array.from({length:5},(_,index)=><i key={index}/>)}</aside><section><i/><i/><i/><i/></section></div></>}
