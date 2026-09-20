"use client";

import {useRef,useState} from "react";
import {useQueries,useQuery} from "@tanstack/react-query";
import {Icon,Pagination} from "@/design-system";
import {apiFetch} from "@/shared/api/client";

export type CatalogProduct={id:string;name:string;price:string;categoryId:string|null;imageUrl:string|null};
type ProductList={items:CatalogProduct[];total:number};
type CatalogQuery={data?:ProductList;isLoading:boolean;error?:{message:string}|null};
type Section={id:string;title:string;items:CatalogProduct[];loading:boolean;error?:{message:string}|null};

const money=(v:string)=>Number(v).toFixed(2);

export function ComandaCatalog({qtyByProduct,onPick,onRemove,currencySymbol,variant="default"}:{qtyByProduct:Record<string,number>;onPick:(p:CatalogProduct)=>void;onRemove:(p:CatalogProduct)=>void;currencySymbol:string;variant?:"default"|"salon"}){
  const[pq,setPq]=useState("");
  const[cat,setCat]=useState("");
  const[page,setPage]=useState(1);
  const[pageSize,setPageSize]=useState(10);
  const scrollRef=useRef<HTMLDivElement>(null);
  const categories=useQuery({queryKey:["order-categories"],queryFn:()=>apiFetch<{items:{id:string;name:string}[]}>("categories?pageSize=100"),staleTime:60000});
  const cats=categories.data?.items??[];
  const catIndex=cats.findIndex(c=>c.id===cat);
  const perCat=useQueries({queries:variant==="default"?cats.map(c=>({queryKey:["order-catalog",c.id],queryFn:()=>apiFetch<ProductList>(`products?status=active&categoryId=${c.id}&page=1&pageSize=100`),staleTime:60000})):[]});
  const todos=useQuery({queryKey:["order-catalog","todos"],queryFn:()=>apiFetch<ProductList>("products?status=active&page=1&pageSize=100"),enabled:variant==="default",staleTime:60000});
  const searching=pq.trim().length>0;
  const search=useQuery({queryKey:["order-catalog","search",pq],queryFn:()=>apiFetch<ProductList>(`products?status=active&q=${encodeURIComponent(pq.trim())}&page=1&pageSize=20`),enabled:variant==="default"&&searching,staleTime:60000});
  const salonCatalog=useQuery({
    queryKey:["order-catalog","salon",cat,pq.trim(),page,pageSize],
    queryFn:()=>{
      const filters=[`status=active`,`page=${page}`,`pageSize=${pageSize}`];
      if(cat)filters.push(`categoryId=${encodeURIComponent(cat)}`);
      if(searching)filters.push(`q=${encodeURIComponent(pq.trim())}`);
      return apiFetch<ProductList>(`products?${filters.join("&")}`);
    },
    enabled:variant==="salon",
    staleTime:60000,
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
  const displayCount=variant==="salon"?salonTotal:totalItems;
  const booting=variant==="salon"?salonCatalog.isLoading:(!searching&&(categories.isLoading||(cats.length>0&&perCat.every(q=>q.isLoading))));
  const goToPage=(next:number)=>{
    setPage(next);
    scrollRef.current?.scrollTo({top:0,behavior:"smooth"});
  };
  const changePageSize=(next:number)=>{
    setPageSize(next);
    setPage(1);
    scrollRef.current?.scrollTo({top:0,behavior:"smooth"});
  };

  return(
    <div className={"comanda-menu comanda-menu-paged"+(variant==="salon"?" comanda-menu-salon":"")}>
      <div className="comanda-menu-hero">
        <div className="comanda-menu-hero-row">
          <h3>La carta</h3>
          <span className="comanda-menu-count">{displayCount} platos</span>
        </div>
        <div className="comanda-search-bar">
          <label className="comanda-search"><Icon name="search" size={18}/><input value={pq} onChange={e=>{setPq(e.target.value);setPage(1)}} placeholder="Buscar en la carta..."/></label>
          {pq&&<button type="button" className="comanda-clear-search" onClick={()=>{setPq("");setPage(1)}} aria-label="Limpiar búsqueda"><Icon name="close" size={15}/></button>}
        </div>
        <div className="comanda-seg" role="tablist" aria-label="Categorías de la carta">
          <button type="button" role="tab" aria-selected={cat===""&&!searching} className={cat===""&&!searching?"active":""} onClick={()=>{setCat("");setPq("");setPage(1)}}>Todos</button>
          {cats.map(c=>(
            <button type="button" role="tab" key={c.id} aria-selected={cat===c.id} className={cat===c.id?"active":""} onClick={()=>{setCat(c.id);setPq("");setPage(1)}}>{c.name}</button>
          ))}
        </div>
      </div>
      <div className="comanda-scroll" ref={scrollRef}>
        {variant==="salon"?(
          booting?(
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
      {variant==="salon"&&salonItems.length>0&&(
        <div className="salon-comanda-pagination-shell">
          <Pagination page={page} size={pageSize} total={salonTotal} onPage={goToPage} onSize={changePageSize} mode="simple"/>
        </div>
      )}
    </div>
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
