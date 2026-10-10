"use client";
/* eslint-disable @next/next/no-img-element -- Menu images come from tenant-configured product URLs. */

import {useEffect,useRef,useState,type KeyboardEvent,type RefObject} from "react";
import {useInfiniteQuery,useQuery} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,Icon,Modal,ModalBody,ModalFooter,NumberStepper,SearchField,Skeleton} from "@/design-system";
import {useMoney} from "@/providers";
import type {CatalogProduct,ComboGroup,ComboOption,ComboSelection,ComboSummary,ConfiguredCombo} from "../domain/catalog-types";
import {getOrderCombo,listOrderCategories,listOrderCombos,listSalonProducts} from "../infrastructure/catalog-api";

export type {CatalogProduct,ComboSelection,ConfiguredCombo} from "../domain/catalog-types";

type Money=(value:string|number)=>string;
const COMBOS="__combos";
const pageSize=24;

/** Acento visual estable por UUID, independiente del nombre y del orden del API. */
function categoryTone(id:string){
  if(!id)return "indigo";
  if(id===COMBOS)return "violet";
  const tones=["indigo","sky","violet","teal"] as const;
  let hash=0;
  for(let index=0;index<id.length;index++)hash=(hash*31+id.charCodeAt(index))>>>0;
  return tones[hash%tones.length];
}

/**
 * Carta de la comanda: categorías deslizables con lista completa accesible y controles de cantidad.
 * Las páginas del API se cargan al llegar al final de la lista.
 * Enter en el buscador agrega el primer plato disponible.
 */
export function ComandaCatalog({qtyByProduct,onPick,onDecrease,onConfigureCombo,disabled=false}:{
  qtyByProduct:Record<string,number>;
  onPick:(p:CatalogProduct)=>void;
  onDecrease:(productId:string)=>void;
  onConfigureCombo?:(comboId:string)=>void;
  disabled?:boolean;
}){
  const money=useMoney();
  const[pq,setPq]=useState("");
  const[cat,setCat]=useState("");
  const[categoriesOpen,setCategoriesOpen]=useState(false);
  const categoryTrack=useRef<HTMLDivElement>(null);
  const scrollRef=useRef<HTMLDivElement>(null);
  const sentinel=useRef<HTMLDivElement>(null);
  const combos=cat===COMBOS;
  const q=pq.trim();
  const searching=q.length>0;
  const categories=useQuery({queryKey:["order-categories"],queryFn:listOrderCategories,staleTime:60000});
  const cats=categories.data?.items??[];
  const salonCatalog=useInfiniteQuery({
    queryKey:["order-catalog","salon",cat,q,pageSize],
    queryFn:({pageParam})=>listSalonProducts({categoryId:cat,q,page:pageParam,pageSize}),
    initialPageParam:1,
    getNextPageParam:(last,all)=>all.length*pageSize<last.total?all.length+1:undefined,
    enabled:!combos,
    staleTime:60000,
  });
  const comboCatalog=useInfiniteQuery({
    queryKey:["order-catalog","combos",q,pageSize],
    queryFn:({pageParam})=>listOrderCombos({q,page:pageParam,pageSize}),
    initialPageParam:1,
    getNextPageParam:(last,all)=>all.length*pageSize<last.total?all.length+1:undefined,
    enabled:combos,
    staleTime:30000,
  });
  const active=combos?comboCatalog:salonCatalog;
  const salonItems=salonCatalog.data?.pages.flatMap(page=>page.items)??[];
  const comboItems=comboCatalog.data?.pages.flatMap(page=>page.items)??[];

  useEffect(()=>{
    const node=sentinel.current;
    if(!node||typeof IntersectionObserver==="undefined")return;
    const observer=new IntersectionObserver(entries=>{
      if(entries.some(entry=>entry.isIntersecting)&&active.hasNextPage&&!active.isFetchingNextPage)void active.fetchNextPage();
    },{root:scrollRef.current,rootMargin:"0px 0px 240px 0px"});
    observer.observe(node);
    return()=>observer.disconnect();
  },[active]);

  useEffect(()=>{
    const track=categoryTrack.current;
    const selected=track?.querySelector<HTMLButtonElement>('[aria-pressed="true"]');
    if(!track||!selected)return;
    const left=selected.offsetLeft;
    if(left<track.scrollLeft||left+selected.offsetWidth>track.scrollLeft+track.clientWidth)track.scrollTo({left:Math.max(0,left)});
  },[cat]);
  const choose=(next:string)=>{setCat(next);setPq("");setCategoriesOpen(false);scrollRef.current?.scrollTo({top:0})};
  const quickAdd=(event:KeyboardEvent<HTMLInputElement>)=>{
    if(event.key!=="Enter"||combos||disabled)return;
    event.preventDefault();
    const first=salonItems.find(item=>{const status=item.availabilityStatus??"available";return status!=="sold_out"&&status!=="unavailable"});
    if(first)onPick(first);
  };

  return(
    <div className="cmd-catalog">
      <div className="cmd-toolbar">
        <div className="cmd-catalog-heading"><Icon name={combos?"combo":"dish"} size={20}/><h3>{combos?"Menús y combos":cats.find(category=>category.id===cat)?.name||"Carta del restaurante"}</h3></div>
        <SearchField value={pq} onChange={event=>setPq(event.target.value)} onKeyDown={quickAdd} enterKeyHint="done" placeholder={combos?"Buscar menú o combo":"Buscar plato o bebida"} label={combos?"Buscar menú o combo":"Buscar en la carta"}/>
        <CategoryFilters categories={cats} category={cat} loading={categories.isLoading} error={categories.isError} retrying={categories.isFetching} onRetry={()=>void categories.refetch()} onChange={choose} trackRef={categoryTrack} expanded={categoriesOpen} onExpand={()=>setCategoriesOpen(true)} onDismiss={()=>setCategoriesOpen(false)}/>
      </div>

      <div className="cmd-list-scroll" ref={scrollRef}>
        {active.isLoading?(
          <MenuRows n={9}/>
        ):combos?(
          comboCatalog.isError?(
            <ErrorState title="No pudimos cargar los menús" text={comboCatalog.error.message} onRetry={()=>comboCatalog.refetch()} retrying={comboCatalog.isFetching}/>
          ):comboItems.length?(
            <ul className="cmd-list">
              {comboItems.map(combo=><ComboItem key={combo.id} combo={combo} qty={qtyByProduct[combo.id]??0} money={money} disabled={disabled} onConfigure={()=>onConfigureCombo?.(combo.id)}/>)}
            </ul>
          ):(
            <EmptyState icon="combo" title={searching?"Sin coincidencias":"Aún no hay menús ni combos"} text={searching?"Ningún menú coincide con la búsqueda.":"Cuando la carta tenga menús o combos activos aparecerán aquí."}/>
          )
        ):salonCatalog.isError?(
          <ErrorState title="No pudimos cargar la carta" text={salonCatalog.error.message} onRetry={()=>salonCatalog.refetch()} retrying={salonCatalog.isFetching}/>
        ):salonItems.length?(
          <ul className="cmd-list">
            {salonItems.map(p=><MenuItem key={p.id} p={p} qty={qtyByProduct[p.id]??0} money={money} onPick={onPick} onDecrease={onDecrease} disabled={disabled}/>)}
          </ul>
        ):(
          <EmptyState icon={searching?"search":"dish"} title={searching?"Sin coincidencias":"Aún no hay platos"} text={searching?"Ningún plato coincide con la búsqueda.":"Activa productos en Carta y productos para venderlos aquí."}/>
        )}
        <div ref={sentinel} className="cmd-list-sentinel" aria-hidden="true"/>
        {active.isFetchingNextPage&&<MenuRows n={3}/>}
        {active.hasNextPage&&!active.isFetchingNextPage&&<Button kind="secondary" className="cmd-load-more" onClick={()=>void active.fetchNextPage()}>Ver más</Button>}
      </div>
    </div>
  );
}

function CategoryFilters({categories,category,loading,error,retrying,onRetry,onChange,trackRef,expanded=false,onExpand,onDismiss}:{categories:{id:string;name:string}[];category:string;loading:boolean;error:boolean;retrying:boolean;onRetry:()=>void;onChange:(id:string)=>void;trackRef?:RefObject<HTMLDivElement|null>;expanded?:boolean;onExpand?:()=>void;onDismiss?:()=>void}){
  return <><nav className="cmd-category-nav" aria-label="Filtrar la carta">
    <div className="cmd-chips" ref={trackRef} role="group" aria-label="Categorías de productos">
      <button type="button" className="cmd-chip cmd-tone-indigo" aria-pressed={category===""} onClick={()=>onChange("")}>{category===""&&<Icon name="check" size={14}/>}Todo</button>
      <button type="button" className="cmd-chip cmd-chip-combos cmd-tone-violet" aria-pressed={category===COMBOS} onClick={()=>onChange(COMBOS)}><Icon name={category===COMBOS?"check":"combo"} size={18}/>Menús y combos</button>
      {loading?<span className="cmd-category-loading" role="status" aria-label="Cargando categorías"><Skeleton width="var(--size-88)" height="var(--control-height)"/><Skeleton width="var(--size-88)" height="var(--control-height)"/></span>:categories.map(item=><button type="button" key={item.id} className={"cmd-chip cmd-tone-"+categoryTone(item.id)} aria-pressed={category===item.id} onClick={()=>onChange(item.id)}>{category===item.id&&<Icon name="check" size={14}/>} {item.name}</button>)}
    </div>
    <Button kind="secondary" className="cmd-category-all" icon="menu" aria-label="Ver todas las categorías" aria-haspopup="dialog" aria-expanded={expanded} onClick={onExpand}><span>Categorías</span></Button>
    {error&&<div className="cmd-category-error" role="alert"><span>No pudimos cargar las categorías.</span><Button kind="ghost" onClick={onRetry} loading={retrying}>Reintentar</Button></div>}
  </nav>{expanded&&<Modal title="Categorías de la carta" icon="menu" size="sm" onClose={()=>onDismiss?.()} closeLabel="Cerrar categorías">
    <ModalBody><div className="cmd-category-options" role="group" aria-label="Todas las categorías">
      {[{id:"",name:"Todo"},{id:COMBOS,name:"Menús y combos"},...categories].map(item=><button type="button" className="cmd-category-option" key={item.id} aria-pressed={category===item.id} onClick={()=>onChange(item.id)}><span className={"cmd-category-label cmd-tone-"+categoryTone(item.id)}><i className="cmd-category-dot" aria-hidden="true"/>{item.name}</span>{category===item.id&&<Icon name="check" size={18}/>}</button>)}
      {loading&&<Skeleton height="var(--control-height)"/>}
      {error&&<ErrorState title="No pudimos cargar las categorías" text="Reintenta para consultar la lista completa." onRetry={onRetry} retrying={retrying}/>}
    </div></ModalBody>
  </Modal>}</>;
}

/** Opciones de un menú o combo: hoja inferior en móvil y modal mediano en escritorio. */
export function ComboConfigurator({comboId,initialSelections=[],editing=false,onClose,onConfirm}:{
  comboId:string;
  initialSelections?:ComboSelection[];
  editing?:boolean;
  onClose:()=>void;
  onConfirm:(combo:ConfiguredCombo)=>void;
}){
  const[selected,setSelected]=useState<Record<string,string[]>>(()=>{
    const result:Record<string,string[]>={};
    for(const sel of initialSelections)result[sel.groupId]=[...(result[sel.groupId]??[]),sel.productId];
    return result;
  });
  const money=useMoney();
  const combo=useQuery({queryKey:["order-combo",comboId],queryFn:()=>getOrderCombo(comboId),staleTime:15000});
  const data=combo.data;
  const initialMapped=useRef(false);

  useEffect(()=>{
    if(!data||initialMapped.current)return;
    const mapped:Record<string,string[]>={};
    for(const selection of initialSelections){
      const normalizedName=selection.groupName.trim().toLowerCase();
      const group=data.groups.find(item=>item.id===selection.groupId||item.name.trim().toLowerCase()===normalizedName);
      if(!group||!group.options.some(option=>option.productId===selection.productId))continue;
      mapped[group.id]=[...(mapped[group.id]??[]),selection.productId];
    }
    setSelected(mapped);
    initialMapped.current=true;
  },[data,initialSelections]);

  const toggle=(group:ComboGroup,option:ComboOption)=>{
    setSelected(prev=>{
      const current=prev[group.id]??[];
      if(current.includes(option.productId))return{...prev,[group.id]:current.filter(id=>id!==option.productId)};
      if(!option.available)return prev;
      if(group.maxSelections===1)return{...prev,[group.id]:[option.productId]};
      if(current.length>=group.maxSelections)return prev;
      return{...prev,[group.id]:[...current,option.productId]};
    });
  };

  const valid=Boolean(data&&data.groups.every(group=>{
    const selectedIds=selected[group.id]??[];
    const count=selectedIds.length;
    const minimum=group.required?Math.max(1,group.minSelections):group.minSelections;
    const allAvailable=selectedIds.every(id=>group.options.find(option=>option.productId===id)?.available);
    return count>=minimum&&count<=group.maxSelections&&allAvailable;
  }));

  const selections:ComboSelection[]=data?data.groups.flatMap(group=>
    (selected[group.id]??[]).map(productId=>{
      const option=group.options.find(o=>o.productId===productId);
      return option?{groupId:group.id,groupName:group.name,productId:option.productId,name:option.name,surcharge:Number(option.surcharge)||0}:null;
    }).filter((value):value is ComboSelection=>Boolean(value))
  ):[];
  const finalPrice=Number(data?.price??0)+selections.reduce((sum,sel)=>sum+sel.surcharge,0);

  return(
    <Modal title={data?.name??"Elige las opciones"} subtitle={data?.description||undefined} icon="combo" size="md" onClose={onClose} closeLabel="Cerrar selección de opciones" className="cmd-combo">
      <ModalBody className="cmd-combo-body">
        {combo.isLoading?(
          <ComboConfiguratorSkeleton/>
        ):combo.isError?(
          <ErrorState title="No pudimos cargar este menú" text={combo.error.message} onRetry={()=>combo.refetch()} retrying={combo.isFetching}/>
        ):data?(
          data.groups.map(group=>{
            const selectedIds=selected[group.id]??[];
            const minimum=group.required?Math.max(1,group.minSelections):group.minSelections;
            const ready=selectedIds.length>=minimum&&selectedIds.length<=group.maxSelections;
            const optional=minimum===0&&selectedIds.length===0;
            return(
              <section className="cmd-combo-group" key={group.id} aria-labelledby={`cmd-group-${group.id}`}>
                <header>
                  <div>
                    <h3 id={`cmd-group-${group.id}`}>{group.name}</h3>
                    <small>{minimum>0?`Elige ${minimum}${group.maxSelections>minimum?` a ${group.maxSelections}`:""}`:`Opcional, hasta ${group.maxSelections}`}</small>
                  </div>
                  <span className={`cmd-combo-progress${optional?" optional":ready?" ready":""}`}>{optional?"Opcional":ready?<><Icon name="check" size={14}/>Listo</>:<>{selectedIds.length} de {minimum}</>}</span>
                </header>
                <div className="cmd-combo-options">
                  {group.options.map(option=>{
                    const picked=selectedIds.includes(option.productId);
                    return(
                      <button type="button" key={option.productId} className="cmd-combo-option" aria-pressed={picked} disabled={!option.available&&!picked} onClick={()=>toggle(group,option)}>
                        <span className="cmd-combo-check" aria-hidden="true">{picked&&<Icon name="check" size={14}/>}</span>
                        <span className="cmd-combo-option-copy"><b>{option.name}</b>{!option.available&&<small>Agotado</small>}</span>
                        <span className="cmd-combo-option-price">{Number(option.surcharge)>0?`+ ${money(option.surcharge)}`:"Incluido"}</span>
                      </button>
                    );
                  })}
                </div>
              </section>
            );
          })
        ):null}
      </ModalBody>
      <ModalFooter className="cmd-combo-foot">
        <p className="cmd-combo-total"><small>Total del menú</small><strong>{combo.isLoading?<Skeleton width="var(--size-72)" height="var(--size-18)"/>:money(finalPrice)}</strong></p>
        <Button icon={editing?"check":"plus"} disabled={!valid||combo.isLoading||combo.isError} onClick={()=>data&&onConfirm({productId:data.id,name:data.name,unitPrice:finalPrice,selections,imageUrl:data.imageUrl})}>
          {editing?"Guardar":"Agregar menú"}
        </Button>
      </ModalFooter>
    </Modal>
  );
}

function ComboItem({combo,qty,money,onConfigure,disabled=false}:{combo:ComboSummary;qty:number;money:Money;onConfigure:()=>void;disabled?:boolean}){
  return(
    <li>
      <button type="button" className={"cmd-item cmd-item-combo"+(qty>0?" picked":"")} disabled={disabled} onClick={onConfigure} aria-label={`Elegir opciones de ${combo.name}`}>
        <span className="cmd-item-visual" aria-hidden="true"><Icon name="combo" size={24}/>{combo.imageUrl&&<img className="cmd-item-thumb" src={combo.imageUrl} alt="" loading="lazy" onError={event=>{event.currentTarget.hidden=true}}/>}</span>
        <span className="cmd-item-copy">
          <strong>{combo.name}</strong>
          <span className="cmd-item-price">desde {money(combo.price)}</span>
          <small>{combo.groupCount} {combo.groupCount===1?"elección":"elecciones"}</small>
        </span>
        <span className="cmd-item-add" aria-hidden="true">{qty>0?<b>{qty}</b>:<Icon name="chevron" size={16}/>}</span>
      </button>
    </li>
  );
}

function MenuItem({p,qty,money,onPick,onDecrease,disabled=false}:{p:CatalogProduct;qty:number;money:Money;onPick:(p:CatalogProduct)=>void;onDecrease:(productId:string)=>void;disabled?:boolean}){
  const availability=p.availabilityStatus??"available";
  const unavailable=availability==="sold_out"||availability==="unavailable";
  const label=unavailable?`${p.name}, no disponible`:qty>0?`Agregar otro ${p.name}, llevas ${qty}`:`Agregar ${p.name}`;
  return(
    <li>
      <article className={"cmd-item"+(qty>0?" picked":"")+(unavailable?" unavailable":"")} aria-label={p.name}>
        <span className="cmd-item-visual" aria-hidden="true"><Icon name="dish" size={24}/>{p.imageUrl&&<img className="cmd-item-thumb" src={p.imageUrl} alt="" loading="lazy" onError={event=>{event.currentTarget.hidden=true}}/>}</span>
        <span className="cmd-item-copy">
          <strong>{p.name}</strong>
          <span className="cmd-item-price">{money(p.price)}</span>
          {availability==="low"&&<small className="low">{p.remaining!==null&&p.remaining!==undefined?`Quedan ${p.remaining}`:"Pocas unidades"}</small>}
          {availability==="sold_out"&&<small className="blocked">Agotado</small>}
          {availability==="unavailable"&&<small className="blocked">Fuera de horario</small>}
        </span>
        {qty>0?<NumberStepper appearance="soft" value={qty} min={1} label={p.name} disabled={disabled} max={unavailable?qty:undefined} onChange={next=>next>qty?onPick(p):onDecrease(p.id)} onRemove={()=>onDecrease(p.id)}/>:<button type="button" className="cmd-item-add" disabled={disabled||unavailable} aria-label={label} onClick={()=>onPick(p)}><Icon name={unavailable?"circleSlash":"plus"} size={18}/></button>}
      </article>
    </li>
  );
}

/** Filas en carga con la misma geometría que la carta. */
function MenuRows({n}:{n:number}){
  return(
    <ul className="cmd-list" aria-hidden="true">
      {Array.from({length:n},(_,i)=>(
        <li className="cmd-item cmd-item-skeleton" key={i}>
          <Skeleton className="cmd-item-visual" width="var(--size-56)" height="var(--size-56)"/>
          <span className="cmd-item-copy"><Skeleton width={i%3===0?"72%":"58%"} height="var(--size-12)"/><Skeleton width="34%" height="var(--size-9)"/></span>
          <Skeleton width="var(--size-32)" height="var(--size-32)" radius="var(--radius-pill)"/>
        </li>
      ))}
    </ul>
  );
}

/** Opciones del menú en carga: grupos con encabezado y opciones seleccionables. */
function ComboConfiguratorSkeleton(){
  return(
    <div role="status" aria-label="Cargando opciones del menú">
      {Array.from({length:2},(_,group)=>(
        <section className="cmd-combo-group" key={group} aria-hidden="true">
          <header>
            <div><Skeleton width="var(--size-138)" height="var(--size-14)"/><Skeleton width="var(--size-92)" height="var(--size-10)"/></div>
            <Skeleton width="var(--size-48)" height="var(--size-22)" radius="var(--radius-pill)"/>
          </header>
          <div className="cmd-combo-options">
            {Array.from({length:3},(_,option)=><Skeleton key={option} height="var(--size-48)" radius="var(--radius-tile)"/>)}
          </div>
        </section>
      ))}
    </div>
  );
}
