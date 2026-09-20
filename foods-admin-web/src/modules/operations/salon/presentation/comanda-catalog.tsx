"use client";

import {useEffect,useRef,useState} from "react";
import {useQueries,useQuery} from "@tanstack/react-query";
import {Button,Icon,Pagination} from "@/design-system";
import type {CatalogProduct,ComboDetail,ComboList,ComboSelection,ConfiguredCombo,ProductList} from "../domain/catalog-types";
import {getOrderCombo,listAllProducts,listOrderCategories,listOrderCombos,listProductsByCategory,listSalonProducts,searchProducts} from "../infrastructure/catalog-api";

export type {CatalogProduct,ComboSelection,ConfiguredCombo} from "../domain/catalog-types";

type CatalogQuery={data?:ProductList;isLoading:boolean;error?:{message:string}|null};
type Section={id:string;title:string;items:CatalogProduct[];loading:boolean;error?:{message:string}|null};
type ComboSummary={id:string;name:string;description:string;price:string;imageUrl:string|null;groupCount:number;optionCount:number};
type ComboList={items:ComboSummary[];total:number};
type ComboOption={productId:string;name:string;surcharge:string;available:boolean};
type ComboGroup={id:string;name:string;required:boolean;minSelections:number;maxSelections:number;options:ComboOption[]};
type ComboDetail={id:string;name:string;description:string;price:string;imageUrl:string|null;groups:ComboGroup[]};

const money=(v:string|number)=>Number(v).toFixed(2);

export function ComandaCatalog({qtyByProduct,onPick,onRemove,onConfigureCombo,currencySymbol,variant="default"}:{
  qtyByProduct:Record<string,number>;
  onPick:(p:CatalogProduct)=>void;
  onRemove:(p:CatalogProduct)=>void;
  onConfigureCombo?:(comboId:string)=>void;
  currencySymbol:string;
  variant?:"default"|"salon";
}){
  const[pq,setPq]=useState("");
  const[cat,setCat]=useState("");
  const[page,setPage]=useState(1);
  const[catalogKind,setCatalogKind]=useState<"products"|"combos">("products");
  const pageSize=10;
  const scrollRef=useRef<HTMLDivElement>(null);
  const categories=useQuery({queryKey:["order-categories"],queryFn:listOrderCategories,staleTime:60000});
  const cats=categories.data?.items??[];
  const catIndex=cats.findIndex(c=>c.id===cat);
  const perCat=useQueries({queries:variant==="default"?cats.map(c=>({queryKey:["order-catalog",c.id],queryFn:()=>listProductsByCategory(c.id),staleTime:60000})):[]});
  const todos=useQuery({queryKey:["order-catalog","todos"],queryFn:listAllProducts,enabled:variant==="default",staleTime:60000});
  const searching=pq.trim().length>0;
  const search=useQuery({queryKey:["order-catalog","search",pq],queryFn:()=>searchProducts(pq),enabled:variant==="default"&&searching,staleTime:60000});
  const salonCatalog=useQuery({
    queryKey:["order-catalog","salon",cat,pq.trim(),page,pageSize],
    queryFn:()=>listSalonProducts({categoryId:cat,q:pq,page,pageSize}),
    enabled:variant==="salon"&&catalogKind==="products",
    staleTime:60000,
  });
  const comboCatalog=useQuery({
    queryKey:["order-catalog","combos",pq.trim(),page,pageSize],
    queryFn:()=>listOrderCombos({q:pq,page,pageSize}),
    enabled:variant==="salon"&&catalogKind==="combos",
    staleTime:30000,
  });

  const norm=(q:CatalogQuery|undefined)=>({items:q?.data?.items??[],loading:q?q.isLoading:true,error:q?.error});
  const orphans=(todos.data?.items??[]).filter(p=>!p.categoryId||!cats.some(c=>c.id===p.categoryId));
  const sections:Section[]=searching
    ?[{id:"search",title:"Resultados",...norm(search)}]
    :cat
      ?[{id:cat,title:cats[catIndex]?.name??"Categoría",...norm(perCat[catIndex])}]
      :[
          ...cats.map((c,i)=>({id:c.id,title:c.name,...norm(perCat[i])})),
          ...(orphans.length||todos.isLoading?[{id:"otros",title:"Otros",items:orphans,loading:todos.isLoading,error:todos.error}]:[]),
        ];
  const single=searching||Boolean(cat);
  const visible=sections.filter(s=>single||s.loading||s.error||s.items.length>0);
  const totalItems=sections.reduce((a,s)=>a+s.items.length,0);
  const salonItems=salonCatalog.data?.items??[];
  const salonTotal=salonCatalog.data?.total??0;
  const comboItems=comboCatalog.data?.items??[];
  const comboTotal=comboCatalog.data?.total??0;
  const displayCount=variant==="salon"?(catalogKind==="combos"?comboTotal:salonTotal):totalItems;
  const countLabel=variant==="salon"&&catalogKind==="combos"?"menús y combos":"platos";
  const booting=variant==="salon"
    ?(catalogKind==="combos"?comboCatalog.isLoading:salonCatalog.isLoading)
    :(!searching&&(categories.isLoading||(cats.length>0&&perCat.every(q=>q.isLoading))));
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
    <div className={"comanda-menu comanda-menu-paged"+(variant==="salon"?" comanda-menu-salon":"")}>
      <div className="comanda-menu-hero">
        <div className="comanda-menu-hero-row">
          <h3>La carta</h3>
          <span className="comanda-menu-count">{displayCount} {countLabel}</span>
        </div>

        {variant==="salon"&&(
          <div className="comanda-source-tabs" role="tablist" aria-label="Tipo de producto">
            <button type="button" role="tab" aria-selected={catalogKind==="products"} className={catalogKind==="products"?"active":""} onClick={()=>switchKind("products")}>
              <span className="comanda-source-icon"><Icon name="utensils" size={17}/></span>
              <span className="comanda-source-copy"><b>Platos</b><small>Productos de la carta</small></span>
            </button>
            <button type="button" role="tab" aria-selected={catalogKind==="combos"} className={catalogKind==="combos"?"active":""} onClick={()=>switchKind("combos")}>
              <span className="comanda-source-icon"><Icon name="combo" size={17}/></span>
              <span className="comanda-source-copy"><b>Menús y combos</b><small>Opciones configurables</small></span>
            </button>
          </div>
        )}

        <div className="comanda-search-bar">
          <label className="comanda-search"><Icon name="search" size={18}/><input value={pq} onChange={e=>{setPq(e.target.value);setPage(1)}} placeholder={catalogKind==="combos"?"Buscar menú o combo...":"Buscar en la carta..."}/></label>
          {pq&&<button type="button" className="comanda-clear-search" onClick={()=>{setPq("");setPage(1)}} aria-label="Limpiar búsqueda"><Icon name="close" size={15}/></button>}
        </div>

        {(variant!=="salon"||catalogKind==="products")&&(
          <div className="comanda-seg" role="tablist" aria-label="Categorías de la carta">
            <button type="button" role="tab" aria-selected={cat===""&&!searching} className={cat===""&&!searching?"active":""} onClick={()=>{setCat("");setPq("");setPage(1)}}>Todos</button>
            {cats.map(c=>(
              <button type="button" role="tab" key={c.id} aria-selected={cat===c.id} className={cat===c.id?"active":""} onClick={()=>{setCat(c.id);setPq("");setPage(1)}}>{c.name}</button>
            ))}
          </div>
        )}
      </div>

      <div className="comanda-scroll" ref={scrollRef}>
        {variant==="salon"?(
          catalogKind==="combos"?(
            booting?(
              <MenuRows n={8}/>
            ):comboCatalog.isError?(
              <div className="catalog-state empty-catalog-card"><span><Icon name="alert"/></span><b>No pudimos cargar los menús</b><p>{comboCatalog.error.message}</p></div>
            ):comboItems.length?(
              <div className="comanda-dishes">
                {comboItems.map(combo=>(
                  <ComboItem key={combo.id} combo={combo} qty={qtyByProduct[combo.id]??0} currencySymbol={currencySymbol} onConfigure={()=>onConfigureCombo?.(combo.id)}/>
                ))}
              </div>
            ):(
              <div className="catalog-state empty-catalog-card"><span><Icon name="menu"/></span><b>Sin menús ni combos</b><p>{searching?"Ningún menú coincide con la búsqueda.":"No hay menús o combos disponibles en este momento."}</p></div>
            )
          ):booting?(
            <MenuRows n={8}/>
          ):salonCatalog.isError?(
            <div className="catalog-state empty-catalog-card"><span><Icon name="alert"/></span><b>No pudimos cargar la carta</b><p>{salonCatalog.error.message}</p></div>
          ):salonItems.length?(
            <div className="comanda-dishes">
              {salonItems.map(p=>(
                <MenuItem key={p.id} p={p} qty={qtyByProduct[p.id]??0} currencySymbol={currencySymbol} onPick={onPick} onRemove={onRemove} variant={variant}/>
              ))}
            </div>
          ):(
            <div className="catalog-state empty-catalog-card"><span><Icon name="search"/></span><b>Sin platos</b><p>{searching?"Ningún producto coincide con la búsqueda.":"No hay productos activos para mostrar."}</p></div>
          )
        ):booting?(
          <MenuSkeleton sections={4}/>
        ):visible.map(sec=>(
          <section className="comanda-menusec" key={sec.id}>
            <header className="comanda-menusec-head"><span>{sec.title}</span><i aria-hidden="true"/><small>{sec.loading?"…":sec.items.length}</small></header>
            {sec.loading?(
              <MenuRows n={3}/>
            ):sec.error?(
              <div className="catalog-state empty-catalog-card"><span><Icon name="alert"/></span><b>No pudimos cargar la carta</b><p>{sec.error.message}</p></div>
            ):sec.items.length?(
              <div className="comanda-dishes">
                {sec.items.map(p=>(
                  <MenuItem key={p.id} p={p} qty={qtyByProduct[p.id]??0} currencySymbol={currencySymbol} onPick={onPick} onRemove={onRemove} variant={variant}/>
                ))}
              </div>
            ):(
              <div className="catalog-state empty-catalog-card"><span><Icon name="search"/></span><b>Sin platos</b><p>{searching?"Ningún producto coincide con la búsqueda.":"Esta categoría aún no tiene productos activos."}</p></div>
            )}
          </section>
        ))}
        {variant!=="salon"&&!booting&&!visible.length&&(
          <div className="catalog-state empty-catalog-card"><span><Icon name="search"/></span><b>Sin platos</b><p>La carta aún no tiene productos activos.</p></div>
        )}
      </div>

      {variant==="salon"&&((catalogKind==="products"&&salonItems.length>0)||(catalogKind==="combos"&&comboItems.length>0))&&(
        <div className="salon-comanda-pagination-shell">
          <Pagination page={page} size={pageSize} total={catalogKind==="combos"?comboTotal:salonTotal} onPage={goToPage} onSize={()=>{}} mode="compact"/>
        </div>
      )}
    </div>
  );
}

export function ComboConfigurator({comboId,initialSelections=[],editing=false,currencySymbol,onClose,onConfirm}:{
  comboId:string;
  initialSelections?:ComboSelection[];
  editing?:boolean;
  currencySymbol:string;
  onClose:()=>void;
  onConfirm:(combo:ConfiguredCombo)=>void;
}){
  const[selected,setSelected]=useState<Record<string,string[]>>(()=>{
    const result:Record<string,string[]>={};
    for(const sel of initialSelections)result[sel.groupId]=[...(result[sel.groupId]??[]),sel.productId];
    return result;
  });
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

  useEffect(()=>{
    const onKeyDown=(event:KeyboardEvent)=>{
      if(event.key==="Escape")onClose();
    };
    window.addEventListener("keydown",onKeyDown);
    return()=>window.removeEventListener("keydown",onKeyDown);
  },[onClose]);

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
    <div className="combo-config-backdrop" role="presentation" onMouseDown={e=>{if(e.target===e.currentTarget)onClose()}}>
      <section className="combo-config-modal" role="dialog" aria-modal="true" aria-labelledby="combo-config-title">
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
            <div className="combo-config-state"><span className="combo-config-spinner"/><b>Cargando opciones…</b></div>
          ):combo.isError?(
            <div className="combo-config-state error"><Icon name="alert" size={20}/><b>No pudimos cargar este menú</b><p>{combo.error.message}</p></div>
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
                          <span className="combo-option-price">{Number(option.surcharge)>0?`+${currencySymbol} ${money(option.surcharge)}`:"Incluido"}</span>
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
          <div className="combo-config-total"><small>Total del menú</small><strong>{currencySymbol} {money(finalPrice)}</strong></div>
          {!editing&&(
            <div className="combo-config-multiple">
              <Icon name="users" size={15}/>
              <span><b>¿Más de un menú?</b><small>Mismas opciones: aumenta la cantidad en la comanda. Opciones distintas: agrega otro menú por separado.</small></span>
            </div>
          )}
          <Button icon={editing?"save":"plus"} disabled={!valid||combo.isLoading||combo.isError} onClick={()=>data&&onConfirm({productId:data.id,name:data.name,unitPrice:finalPrice,selections})}>
            {editing?"Guardar opciones":"Agregar menú"}
          </Button>
        </footer>
      </section>
    </div>
  );
}

function ComboItem({combo,qty,currencySymbol,onConfigure}:{combo:ComboSummary;qty:number;currencySymbol:string;onConfigure:()=>void}){
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
          <b className="comanda-dish-price">Desde {currencySymbol} {money(combo.price)}</b>
          <button type="button" className="comanda-dish-add-button" onClick={onConfigure} aria-label={`Elegir opciones de ${combo.name}`}>
            <Icon name="combo" size={14}/><span>Elegir opciones</span>
          </button>
        </div>
      </div>
    </article>
  );
}

function MenuItem({p,qty,currencySymbol,onPick,onRemove,variant}:{p:CatalogProduct;qty:number;currencySymbol:string;onPick:(p:CatalogProduct)=>void;onRemove:(p:CatalogProduct)=>void;variant:"default"|"salon"}){
  if(variant==="salon"){
    return(
      <article className={"comanda-dish comanda-dish-modern"+(qty>0?" picked":"")} aria-label={p.name}>
        <span className={"comanda-dish-thumb comanda-dish-modern-media"+(p.imageUrl?" has-image":"")}>
          <span className="comanda-dish-image-fallback"><Icon name="utensils" size={24}/><small>Sin imagen</small></span>
          {p.imageUrl&&<img src={p.imageUrl} alt={p.name} loading="lazy" onError={e=>{e.currentTarget.hidden=true}}/>}
          {qty>0&&<b className="comanda-dish-selected-qty">{qty}×</b>}
        </span>
        <div className="comanda-dish-modern-content">
          <strong className="comanda-dish-name" title={p.name}>{p.name}</strong>
          <div className="comanda-dish-modern-footer">
            <b className="comanda-dish-price">{currencySymbol} {money(p.price)}</b>
            <button type="button" className="comanda-dish-add-button" onClick={()=>onPick(p)} aria-label={qty>0?`Agregar otro ${p.name}`:`Agregar ${p.name} a la comanda`}>
              <Icon name="plus" size={14}/>
              <span>Agregar</span>
            </button>
          </div>
        </div>
      </article>
    );
  }

  return(
    <article className={"comanda-dish"+(qty>0?" picked":"")} onClick={()=>onPick(p)} role="button" tabIndex={0} onKeyDown={e=>{if(e.key==="Enter"||e.key===" ")onPick(p)}} aria-label={p.name}>
      <span className="comanda-dish-thumb">
        {p.imageUrl?<img src={p.imageUrl} alt={p.name}/>:<Icon name="utensils" size={20}/>}
      </span>
      <strong className="comanda-dish-name" title={p.name}>{p.name}</strong>
      <i className="comanda-dish-dots" aria-hidden="true"/>
      <b className="comanda-dish-price">{currencySymbol} {money(p.price)}</b>
      {qty>0?(
        <span className="comanda-dish-step" onClick={e=>e.stopPropagation()} onKeyDown={e=>e.stopPropagation()}>
          <button type="button" onClick={()=>onRemove(p)} aria-label={`Quitar un ${p.name}`}><Icon name="minus" size={12}/></button>
          <b>{qty}</b>
          <button type="button" onClick={()=>onPick(p)} aria-label={`Agregar un ${p.name}`}><Icon name="plus" size={12}/></button>
        </span>
      ):(
        <span className="comanda-dish-add" aria-hidden="true"><Icon name="plus" size={14}/></span>
      )}
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

function MenuSkeleton({sections}:{sections:number}){
  return(
    <>
      {Array.from({length:sections},(_,i)=>(
        <section className="comanda-menusec" key={i}>
          <header className="comanda-menusec-head"><span className="comanda-skel-sect"/><i aria-hidden="true"/><small/></header>
          <MenuRows n={3}/>
        </section>
      ))}
    </>
  );
}
