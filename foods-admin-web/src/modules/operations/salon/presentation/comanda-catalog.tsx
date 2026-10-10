"use client";
/* eslint-disable @next/next/no-img-element -- Menu images come from tenant-configured product URLs. */

import {Dialog} from "@/design-system/dialog";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button,EmptyState,ErrorState,Icon,Pagination,Skeleton} from "@/design-system";
import {useMoney} from "@/providers";
import type {CatalogProduct,ComboGroup,ComboOption,ComboSelection,ComboSummary,ConfiguredCombo} from "../domain/catalog-types";
import {getOrderCombo,listOrderCategories,listOrderCombos,listSalonProducts} from "../infrastructure/catalog-api";

export type {CatalogProduct,ComboSelection,ConfiguredCombo} from "../domain/catalog-types";

type Money=(value:string|number)=>string;

export function ComandaCatalog({qtyByProduct,onPick,onConfigureCombo}:{
  qtyByProduct:Record<string,number>;
  onPick:(p:CatalogProduct)=>void;
  onConfigureCombo?:(comboId:string)=>void;
}){
  const money=useMoney();
  const[pq,setPq]=useState("");
  const[cat,setCat]=useState("");
  const[page,setPage]=useState(1);
  const[catalogKind,setCatalogKind]=useState<"products"|"combos">("products");
  const pageSize=10;
  const scrollRef=useRef<HTMLDivElement>(null);
  const categories=useQuery({queryKey:["order-categories"],queryFn:listOrderCategories,staleTime:60000});
  const cats=categories.data?.items??[];
  const searching=pq.trim().length>0;
  const salonCatalog=useQuery({
    queryKey:["order-catalog","salon",cat,pq.trim(),page,pageSize],
    queryFn:()=>listSalonProducts({categoryId:cat,q:pq,page,pageSize}),
    enabled:catalogKind==="products",
    staleTime:60000,
  });
  const comboCatalog=useQuery({
    queryKey:["order-catalog","combos",pq.trim(),page,pageSize],
    queryFn:()=>listOrderCombos({q:pq,page,pageSize}),
    enabled:catalogKind==="combos",
    staleTime:30000,
  });

  const salonItems=salonCatalog.data?.items??[];
  const salonTotal=salonCatalog.data?.total??0;
  const comboItems=comboCatalog.data?.items??[];
  const comboTotal=comboCatalog.data?.total??0;
  const combos=catalogKind==="combos";
  const displayCount=combos?comboTotal:salonTotal;
  const countLabel=combos?"menús y combos":"platos";
  const booting=combos?comboCatalog.isLoading:salonCatalog.isLoading;
  const goToPage=(next:number)=>{
    setPage(next);
    scrollRef.current?.scrollTo({top:0,behavior:"smooth"});
  };
  const switchKind=(kind:"products"|"combos")=>{
    setCatalogKind(kind);
    setCat("");
    setPq("");
    setPage(1);
    scrollRef.current?.scrollTo({top:0});
  };

  return(
    <div className="comanda-menu comanda-menu-paged comanda-menu-salon">
      <div className="comanda-menu-hero">
        <div className="comanda-menu-hero-row">
          <h3>La carta</h3>
          <span className="comanda-menu-count">{displayCount} {countLabel}</span>
        </div>

        <div className="comanda-source-tabs" role="tablist" aria-label="Tipo de producto">
          <button type="button" role="tab" aria-selected={catalogKind==="products"} className={catalogKind==="products"?"active":""} onClick={()=>switchKind("products")}>
            <span className="comanda-source-icon"><Icon name="utensils" size={17}/></span>
            <span className="comanda-source-copy"><b>Platos</b><small>Productos de la carta</small></span>
          </button>
          <button type="button" role="tab" aria-selected={combos} className={combos?"active":""} onClick={()=>switchKind("combos")}>
            <span className="comanda-source-icon"><Icon name="combo" size={17}/></span>
            <span className="comanda-source-copy"><b>Menús y combos</b><small>Opciones configurables</small></span>
          </button>
        </div>

        <div className="comanda-search-bar">
          <label className="comanda-search"><Icon name="search" size={18}/><input value={pq} onChange={e=>{setPq(e.target.value);setPage(1)}} placeholder={combos?"Buscar menú o combo...":"Buscar en la carta..."} aria-label={combos?"Buscar menú o combo":"Buscar en la carta"}/></label>
          {pq&&<button type="button" className="comanda-clear-search" onClick={()=>{setPq("");setPage(1)}} aria-label="Limpiar búsqueda"><Icon name="close" size={15}/></button>}
        </div>

        {!combos&&(
          <div className="comanda-seg" role="tablist" aria-label="Categorías de la carta">
            <button type="button" role="tab" aria-selected={cat===""&&!searching} className={cat===""&&!searching?"active":""} onClick={()=>{setCat("");setPq("");setPage(1)}}>Todos</button>
            {cats.map(c=>(
              <button type="button" role="tab" key={c.id} aria-selected={cat===c.id} className={cat===c.id?"active":""} onClick={()=>{setCat(c.id);setPq("");setPage(1)}}>{c.name}</button>
            ))}
          </div>
        )}
      </div>

      <div className="comanda-scroll" ref={scrollRef}>
        {booting?(
          <MenuRows n={8}/>
        ):combos?(
          comboCatalog.isError?(
            <ErrorState className="empty-catalog-card" title="No pudimos cargar los menús" text={comboCatalog.error.message} onRetry={()=>comboCatalog.refetch()} retrying={comboCatalog.isFetching}/>
          ):comboItems.length?(
            <div className="comanda-dishes">
              {comboItems.map(combo=>(
                <ComboItem key={combo.id} combo={combo} qty={qtyByProduct[combo.id]??0} money={money} onConfigure={()=>onConfigureCombo?.(combo.id)}/>
              ))}
            </div>
          ):(
            <EmptyState className="empty-catalog-card" icon="menu" title={searching?"Sin coincidencias":"Sin menús ni combos"} text={searching?"Ningún menú coincide con la búsqueda.":"No hay menús o combos disponibles en este momento."}/>
          )
        ):salonCatalog.isError?(
          <ErrorState className="empty-catalog-card" title="No pudimos cargar la carta" text={salonCatalog.error.message} onRetry={()=>salonCatalog.refetch()} retrying={salonCatalog.isFetching}/>
        ):salonItems.length?(
          <div className="comanda-dishes">
            {salonItems.map(p=>(
              <MenuItem key={p.id} p={p} qty={qtyByProduct[p.id]??0} money={money} onPick={onPick}/>
            ))}
          </div>
        ):(
          <EmptyState className="empty-catalog-card" icon="search" title={searching?"Sin coincidencias":"Sin platos"} text={searching?"Ningún producto coincide con la búsqueda.":"No hay productos activos para mostrar."}/>
        )}
      </div>

      {((!combos&&salonItems.length>0)||(combos&&comboItems.length>0))&&(
        <div className="salon-comanda-pagination-shell">
          <Pagination page={page} size={pageSize} total={displayCount} onPage={goToPage} onSize={()=>{}} mode="compact"/>
        </div>
      )}
    </div>
  );
}

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
  const combo=useQuery({
    queryKey:["order-combo",comboId],
    queryFn:()=>getOrderCombo(comboId),
    staleTime:15000,
  });
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
    <div className="combo-config-backdrop" role="presentation">
      <Dialog onResponseClose={onClose} className="combo-config-modal" role="dialog" aria-modal="true" aria-labelledby="combo-config-title">
        <header className="combo-config-head">
          <div>
            <small>ELIGE LAS OPCIONES</small>
            <h3 id="combo-config-title">{data?.name??"Elegir opciones"}</h3>
            {data?.description&&<p>{data.description}</p>}
          </div>
          <button type="button" aria-label="Cerrar selección de opciones" onClick={onClose}><Icon name="close" size={17}/></button>
        </header>

        <div className="combo-config-body">
          {combo.isLoading?(
            <ComboConfiguratorSkeleton/>
          ):combo.isError?(
            <ErrorState title="No pudimos cargar este menú" text={combo.error.message} onRetry={()=>combo.refetch()} retrying={combo.isFetching}/>
          ):data?(
            data.groups.map(group=>{
              const selectedIds=selected[group.id]??[];
              const minimum=group.required?Math.max(1,group.minSelections):group.minSelections;
              const ready=selectedIds.length>=minimum&&selectedIds.length<=group.maxSelections;
              return(
                <section className="combo-config-group" key={group.id}>
                  <header>
                    <div>
                      <h4>{group.name}</h4>
                      <small>{minimum>0?`Elige ${minimum}${group.maxSelections>minimum?` a ${group.maxSelections}`:""} de ${group.options.length}`:`${group.options.length} opciones · elección opcional`}</small>
                    </div>
                    <span className={ready?"ready":"pending"}>{minimum===0&&selectedIds.length===0?"Opcional":ready?<><Icon name="check" size={12}/>Listo</>:<>{selectedIds.length}/{minimum}</>}</span>
                  </header>
                  <div className="combo-config-options">
                    {group.options.map(option=>{
                      const active=selectedIds.includes(option.productId);
                      return(
                        <button type="button" key={option.productId} className={active?"active":""} aria-pressed={active} disabled={!option.available&&!active} onClick={()=>toggle(group,option)}>
                          <span className="combo-option-check">{active?<Icon name="check" size={13}/>:null}</span>
                          <span className="combo-option-copy"><b>{option.name}</b>{!option.available&&<small>Agotado</small>}</span>
                          <span className="combo-option-price">{Number(option.surcharge)>0?`+${money(option.surcharge)}`:"Incluido"}</span>
                        </button>
                      );
                    })}
                  </div>
                </section>
              );
            })
          ):null}
        </div>

        <footer className="combo-config-foot">
          <div className="combo-config-total"><small>Total del menú</small><strong>{combo.isLoading?<Skeleton width="var(--size-72)" height="var(--size-18)"/>:money(finalPrice)}</strong></div>
          {!editing&&(
            <div className="combo-config-multiple">
              <Icon name="users" size={15}/>
              <span><b>¿Más de un menú?</b><small>Mismas opciones: aumenta la cantidad en la comanda. Opciones distintas: agrega otro menú por separado.</small></span>
            </div>
          )}
          <Button icon={editing?"check":"plus"} disabled={!valid||combo.isLoading||combo.isError} onClick={()=>data&&onConfirm({productId:data.id,name:data.name,unitPrice:finalPrice,selections})}>
            {editing?"Guardar":"Agregar menú"}
          </Button>
        </footer>
      </Dialog>
    </div>
  );
}

function ComboItem({combo,qty,money,onConfigure}:{combo:ComboSummary;qty:number;money:Money;onConfigure:()=>void}){
  return(
    <article className={"comanda-dish comanda-dish-modern comanda-combo-card"+(qty>0?" picked":"")} aria-label={combo.name}>
      <span className={"comanda-dish-thumb comanda-dish-modern-media"+(combo.imageUrl?" has-image":"")}>
        <span className="comanda-dish-image-fallback"><Icon name="combo" size={24}/><small>Menú</small></span>
        {combo.imageUrl&&<img src={combo.imageUrl} alt={combo.name} loading="lazy" onError={e=>{e.currentTarget.hidden=true}}/>}
        {qty>0&&<b className="comanda-dish-selected-qty">{qty}×</b>}
      </span>
      <div className="comanda-dish-modern-content">
        <div className="comanda-combo-copy">
          <strong className="comanda-dish-name" title={combo.name}>{combo.name}</strong>
          {combo.description&&<small>{combo.description}</small>}
          <em>{combo.groupCount} grupo{combo.groupCount===1?"":"s"} de elección</em>
        </div>
        <div className="comanda-dish-modern-footer">
          <b className="comanda-dish-price">Desde {money(combo.price)}</b>
          <button type="button" className="comanda-dish-add-button" onClick={onConfigure} aria-label={`Elegir opciones de ${combo.name}`}>
            <Icon name="combo" size={14}/><span>Elegir opciones</span>
          </button>
        </div>
      </div>
    </article>
  );
}

function MenuItem({p,qty,money,onPick}:{p:CatalogProduct;qty:number;money:Money;onPick:(p:CatalogProduct)=>void}){
  const availability=p.availabilityStatus??"available";
  const unavailable=availability==="sold_out"||availability==="unavailable";
  return(
    <article className={"comanda-dish comanda-dish-modern"+(qty>0?" picked":"")+(unavailable?" unavailable":"")+(availability==="low"?" low":"")} aria-label={p.name}>
      <span className={"comanda-dish-thumb comanda-dish-modern-media"+(p.imageUrl?" has-image":"")}>
        <span className="comanda-dish-image-fallback"><Icon name="utensils" size={24}/><small>Sin imagen</small></span>
        {p.imageUrl&&<img src={p.imageUrl} alt={p.name} loading="lazy" onError={e=>{e.currentTarget.hidden=true}}/>}
        {qty>0&&<b className="comanda-dish-selected-qty">{qty}×</b>}
        {availability==="low"&&<small className="comanda-dish-availability low">{p.remaining!==null&&p.remaining!==undefined?`Quedan ${p.remaining}`:"Pocas unidades"}</small>}
        {availability==="sold_out"&&<small className="comanda-dish-availability blocked">Agotado</small>}
        {availability==="unavailable"&&<small className="comanda-dish-availability blocked">Fuera de horario</small>}
      </span>
      <div className="comanda-dish-modern-content">
        <strong className="comanda-dish-name" title={p.name}>{p.name}</strong>
        <div className="comanda-dish-modern-footer">
          <b className="comanda-dish-price">{money(p.price)}</b>
          <button type="button" className="comanda-dish-add-button" disabled={unavailable} onClick={()=>!unavailable&&onPick(p)} aria-label={unavailable?`${p.name} no disponible`:qty>0?`Agregar otro ${p.name}`:`Agregar ${p.name} a la comanda`}>
            <Icon name={unavailable?"close":"plus"} size={14}/>
            <span>{unavailable?"No disponible":"Agregar"}</span>
          </button>
        </div>
      </div>
    </article>
  );
}

function MenuRows({n}:{n:number}){
  return(
    <div className="comanda-dishes" aria-hidden="true">
      {Array.from({length:n},(_,i)=>(
        <div className="comanda-skel" key={i}>
          <span className="comanda-skel-media"/>
          <span className="comanda-skel-name"/>
          <span className="comanda-skel-dots"/>
          <span className="comanda-skel-price"/>
          <span className="comanda-skel-add"/>
        </div>
      ))}
    </div>
  );
}

/** Opciones del menú en carga: grupos con encabezado y opciones seleccionables. */
function ComboConfiguratorSkeleton(){
  return(
    <div className="combo-config-skeleton" role="status" aria-label="Cargando opciones del menú">
      {Array.from({length:2},(_,group)=>(
        <section className="combo-config-group" key={group} aria-hidden="true">
          <header>
            <div><Skeleton width="var(--size-138)" height="var(--size-14)"/><Skeleton width="var(--size-92)" height="var(--size-10)"/></div>
            <Skeleton width="var(--size-48)" height="var(--size-22)" radius="var(--radius-999)"/>
          </header>
          <div className="combo-config-options">
            {Array.from({length:3},(_,option)=><Skeleton key={option} height="var(--control-height)" radius="var(--radius-control)"/>)}
          </div>
        </section>
      ))}
    </div>
  );
}
