"use client";

import Image from "next/image";
import {useEffect,useRef,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {IconAdjustmentsHorizontal,IconCheck,IconChecks,IconChefHat,IconDotsVertical,IconMapPin,IconMenu2,IconNote,IconPlus,IconReceipt,IconSend2,IconShoppingBag,IconTrash,IconX} from "@tabler/icons-react";
import {Button,Input,ModalBody,ModalFooter,ModalHeader,Skeleton} from "@/design-system";
import {Dialog} from "@/design-system/dialog";
import type {ChatAction,ChatInput,ChatMessage,ChatProduct,ProductOptions} from "../domain/types";
import {loadChat,loadChatMenu,loadProductOptions,newChatRequestId,openChat,sendChatInput} from "../infrastructure/chat-api";
import "./restaurant-chat.css";

// `first` marca el primer mensaje de un grupo del mismo emisor: solo ese lleva la punta de la burbuja.
function Message({message,first}:{message:ChatMessage;first:boolean}) {
  const outgoing=message.role==="user";
  return <article className={`restaurant-chat-message ${outgoing?"is-outgoing":"is-incoming"}${first?" is-first":""}`} aria-label={outgoing?"Tu mensaje":"Mensaje del restaurante"}>
    {message.imageUrl&&<Image className="restaurant-chat-message-image" src={message.imageUrl} alt={message.text||"Imagen compartida por el restaurante"} width={480} height={320} unoptimized/>}
    {message.text&&<p>{message.text}</p>}
    <footer><time dateTime={new Date(message.createdAt*1000).toISOString()}>{new Date(message.createdAt*1000).toLocaleTimeString(undefined,{hour:"2-digit",minute:"2-digit"})}</time>{outgoing&&(message.status==="sent"?<IconChecks className="restaurant-chat-read" size={16} aria-label="Procesado"/>:<IconCheck size={16} aria-label="Recibido"/>)}</footer>
  </article>;
}

function OptionsEditor({qr,product,currency,onClose,onAdd}:{qr:string;product:ChatProduct;currency:string;onClose:()=>void;onAdd:(action:ChatAction)=>void}) {
  const options=useQuery({queryKey:["chat-options",qr,product.productId],queryFn:()=>loadProductOptions(qr,product.productId),staleTime:0,retry:false});
  const [selected,setSelected]=useState<Record<string,string[]>>({});
  const [note,setNote]=useState("");
  const valid=options.data?.detail.groups.every(group=>(selected[group.id]?.length??0)>=Math.max(group.minSelections,group.required?1:0)&&(selected[group.id]?.length??0)<=group.maxSelections);
  const toggle=(group:ProductOptions["detail"]["groups"][number],id:string)=>setSelected(current=>{
    const values=current[group.id]??[];
    return {...current,[group.id]:values.includes(id)?values.filter(value=>value!==id):group.maxSelections===1?[id]:[...values,id]};
  });
  const add=()=>{
    if (!options.data||!valid) return;
    const combo=options.data.kind==="combo";
    const selections=Object.entries(selected).flatMap(([groupId,ids])=>ids.map(id=>combo?{groupId,productId:id}:{groupId,optionId:id}));
    onAdd({name:combo?"add_combo_item":"add_modified_item",args:{productId:product.productId,quantity:1,note,[combo?"selections":"modifiers"]:selections}});
  };
  return <div className="modal-backdrop modal-overlay-in"><Dialog className="crud-modal restaurant-chat-options" role="dialog" aria-modal="true" aria-labelledby="chat-options-title" onResponseClose={onClose}>
    <ModalHeader title={product.name} titleId="chat-options-title" onClose={onClose}/>
    <ModalBody>{options.isPending?<Skeleton className="restaurant-chat-options-skeleton"/>:options.isError?<div role="alert"><p>{options.error.message}</p><Button kind="secondary" onClick={()=>void options.refetch()}>Reintentar</Button></div>:<>
      {options.data.detail.groups.length===0&&<p>Puedes agregar este plato sin opciones adicionales.</p>}
      {options.data.detail.groups.map(group=><fieldset key={group.id} className="restaurant-chat-option-group"><legend>{group.name}</legend><p>Elige {Math.max(group.minSelections,group.required?1:0)} a {group.maxSelections}</p>{group.options.map(option=>{
        const id=option.productId??option.id??"";
        return <label key={id}><input type="checkbox" checked={selected[group.id]?.includes(id)??false} disabled={option.available===false||(!(selected[group.id]?.includes(id))&&group.maxSelections>1&&(selected[group.id]?.length??0)>=group.maxSelections)} onChange={()=>toggle(group,id)}/><span>{option.name}{option.available===false?" · No disponible":""}</span>{Number(option.surcharge)>0&&<span>+ {currency} {Number(option.surcharge).toFixed(2)}</span>}</label>;
      })}</fieldset>)}
      <label className="restaurant-chat-note">Nota para cocina<Input maxLength={240} value={note} onChange={event=>setNote(event.target.value)} placeholder="Por ejemplo, sin ají"/></label>
    </>}</ModalBody>
    <ModalFooter><Button kind="ghost" onClick={onClose}>Cancelar</Button><Button disabled={!valid||options.isPending} onClick={add}>Agregar al pedido</Button></ModalFooter>
  </Dialog></div>;
}

export function RestaurantChat({qr,scan=""}:{qr:string;scan?:string}) {
  const client=useQueryClient();
  const [text,setText]=useState("");
  const [search,setSearch]=useState("");
  // La carta no se abre sola: el comensal la pide con el atajo «Carta».
  const [showMenu,setShowMenu]=useState(false);
  const [showCart,setShowCart]=useState(false);
  const [showInfo,setShowInfo]=useState(false);
  const [product,setProduct]=useState<ChatProduct|null>(null);
  const [failed,setFailed]=useState<ChatInput|null>(null);
  const [locating,setLocating]=useState(false);
  const [locationError,setLocationError]=useState("");
  const bottom=useRef<HTMLDivElement>(null);
  const feed=useRef<HTMLDivElement>(null);
  const following=useRef(true);
  const composer=useRef<HTMLInputElement>(null);
  const activeAt=useRef(0);
  const session=useQuery({queryKey:["web-chat-session",qr],queryFn:()=>openChat(qr,scan),retry:false,staleTime:Infinity,gcTime:0});
  // Sondeo según la conversación: rápido mientras el local responde, normal durante el minuto siguiente a la última actividad y espaciado cuando el chat está quieto.
  const chat=useQuery({queryKey:["web-chat",qr],queryFn:()=>loadChat(qr),enabled:session.isSuccess,retry:false,refetchInterval:query=>query.state.data?.pending?1200:Date.now()-activeAt.current<60_000?4000:12_000});
  const menu=useQuery({queryKey:["web-chat-menu",qr],queryFn:()=>loadChatMenu(qr),enabled:session.isSuccess,retry:false,refetchInterval:showMenu?30_000:false});
  const send=useMutation({mutationFn:(input:ChatInput)=>sendChatInput(qr,input),onSuccess:()=>{setFailed(null);void client.invalidateQueries({queryKey:["web-chat",qr]});void client.invalidateQueries({queryKey:["web-chat-menu",qr]})},onError:(_error,input)=>setFailed(input)});
  const state=chat.data;
  const busy=send.isPending||Boolean(state?.pending)||Boolean(failed)||locating;
  const disabled=busy||!state?.active||Boolean(chat.isError);
  const messages=state?.messages??[];
  const currency=menu.data?.currencySymbol??"";
  const items=menu.data?.items.filter(item=>!search||`${item.name} ${item.categoryName??""}`.toLocaleLowerCase().includes(search.toLocaleLowerCase()))??[];
  const quantity=state?.cart.items.reduce((sum,item)=>sum+item.quantity,0)??0;
  // La foto del plato sale de la carta ya cargada: el carrito solo trae nombres e importes.
  const photos=new Map(menu.data?.items.map(item=>[item.productId,item.imageUrl]));
  // Unidades de cada plato ya en el pedido (suma sus variantes): la tarjeta de la carta las muestra sobre la foto.
  const inCart=new Map<string,number>();
  for (const item of state?.cart.items??[]) inCart.set(item.productId,(inCart.get(item.productId)??0)+item.quantity);
  useEffect(()=>{
    activeAt.current=Date.now();
    if (following.current) bottom.current?.scrollIntoView({block:"end"});
  },[messages.length,state?.pending,showMenu,showCart]);
  // Armar el pedido no cierra el panel abierto: se agregan varios platos seguidos desde la carta y se quitan desde «Mi pedido».
  const submit=(input:ChatInput,keepPanel=false)=>{following.current=true;if(!keepPanel){setShowMenu(false);setShowCart(false)}send.mutate(input)};
  const act=(action:ChatAction,keepPanel=false)=>submit({requestId:newChatRequestId(),action},keepPanel);
  const add=(item:ChatProduct)=>{
    if (item.isCombo||item.hasModifiers) setProduct(item);
    else act({name:"add_item",args:{productId:item.productId,quantity:1}},true);
  };
  const sendText=()=>{
    if (!text.trim()||disabled) return;
    const input={requestId:newChatRequestId(),text:text.trim()};
    setText("");submit(input);composer.current?.focus();
  };
  const shareLocation=()=>{
    setLocationError("");
    if(!navigator.geolocation){setLocationError("Este navegador no permite compartir la ubicación.");return}
    setLocating(true);
    navigator.geolocation.getCurrentPosition(position=>{
      setLocating(false);
      submit({requestId:newChatRequestId(),location:{latitude:position.coords.latitude,longitude:position.coords.longitude}});
    },()=>{setLocating(false);setLocationError("No pudimos obtener tu ubicación. Revisa el permiso del navegador y vuelve a intentarlo.")},{enableHighAccuracy:true,timeout:15_000,maximumAge:0});
  };
  const error=session.error??chat.error;
  if (!state) return <main className="restaurant-chat-stage"><section className="restaurant-chat-loading" aria-busy={!error}>
    <IconChefHat size={32}/>{error?<><h1>No pudimos abrir el chat</h1><p role="alert">{error.message}</p><button type="button" className="restaurant-chat-primary" onClick={()=>window.location.reload()}>Volver a intentar</button></>:<><h1>Abriendo el chat del restaurante</h1><Skeleton className="restaurant-chat-skeleton restaurant-chat-loading-line"/><Skeleton className="restaurant-chat-skeleton restaurant-chat-loading-line"/><span className="sr-only">Cargando conversación</span></>}
  </section></main>;
  return <main className="restaurant-chat-stage"><section className="restaurant-chat" aria-label={`Chat de ${state.context.organizationName}`}>
    <header className="restaurant-chat-header">
      <div className="restaurant-chat-avatar">{menu.data?.restaurantLogoUrl?<Image src={menu.data.restaurantLogoUrl} alt="" width={40} height={40} unoptimized/>:<IconChefHat size={24}/>}</div>
      <div className="restaurant-chat-heading"><h1>{state.context.organizationName}</h1><p>{state.context.locationName} · {state.mode==="table"?state.context.name:state.mode==="counter"?"Pedido en tienda":"Delivery y recojo"}</p></div>
      <button className="restaurant-chat-icon" type="button" aria-label="Información del chat" aria-expanded={showInfo} onClick={()=>setShowInfo(!showInfo)}><IconDotsVertical size={22}/></button>
    </header>
    {showInfo&&<aside className="restaurant-chat-info"><strong>Asistente de {state.context.organizationName}</strong><p>Consulta la carta, arma tu pedido y confírmalo aquí. Tus mensajes se atienden con asistencia de inteligencia artificial. El personal del local revisa tu pedido.</p><p>Las actualizaciones aparecen mientras tengas este chat abierto.</p><button type="button" onClick={()=>{setShowInfo(false);act({name:"request_human"})}} disabled={disabled}>Solicitar atención del personal</button></aside>}
    <div ref={feed} className="restaurant-chat-feed" role="log" aria-label="Conversación" aria-live="polite" aria-relevant="additions" onScroll={()=>{const element=feed.current;if(element)following.current=element.scrollHeight-element.scrollTop-element.clientHeight<100}}>
      <div className="restaurant-chat-date">Hoy</div>
      {messages.map((message,index)=><Message key={message.id} message={message} first={messages[index-1]?.role!==message.role}/>)}
      {!messages.length&&!state.pending&&<p className="restaurant-chat-system">Escríbenos o elige un plato de la carta para empezar.</p>}
      {state.pending&&<div className="restaurant-chat-typing" role="status"><span className="sr-only">El restaurante está respondiendo</span><span className="restaurant-chat-dot" aria-hidden="true"/><span className="restaurant-chat-dot" aria-hidden="true"/><span className="restaurant-chat-dot" aria-hidden="true"/></div>}
      {state.failed&&<p className="restaurant-chat-system" role="alert">No pudimos procesar un mensaje. Revisa tu pedido antes de volver a enviarlo.</p>}
      {showMenu&&<section className="restaurant-chat-menu" aria-label="Carta del restaurante">
        <div className="restaurant-chat-section-heading"><h2>Carta</h2><button className="restaurant-chat-icon" type="button" aria-label="Ocultar carta" onClick={()=>setShowMenu(false)}><IconX size={20}/></button></div>
        <Input aria-label="Buscar platos" placeholder="Buscar un plato o categoría" value={search} onChange={event=>setSearch(event.target.value)}/>
        {menu.isPending?<div className="restaurant-chat-foods"><Skeleton className="restaurant-chat-skeleton restaurant-chat-food-skeleton"/><Skeleton className="restaurant-chat-skeleton restaurant-chat-food-skeleton"/></div>:menu.isError?<div className="restaurant-chat-system" role="alert"><p>{menu.error.message}</p><button type="button" onClick={()=>void menu.refetch()}>Volver a cargar la carta</button></div>:items.length===0?<p className="restaurant-chat-system">{search?"No encontramos platos con ese nombre.":"La carta de este local todavía no tiene platos disponibles."}</p>:<div className="restaurant-chat-foods">{items.map(item=>{
          const available=["available","low"].includes(item.status)&&(item.remaining===null||Number(item.remaining)>0);
          const choose=item.isCombo||item.hasModifiers;
          const added=inCart.get(item.productId)??0;
          return <article className={`restaurant-chat-food${available?"":" is-unavailable"}`} key={item.productId}>
            <div className="restaurant-chat-food-media">
              {item.imageUrl?<Image src={item.imageUrl} alt={item.name} width={320} height={220} unoptimized/>:<div className="restaurant-chat-food-placeholder"><IconChefHat size={40}/><span>Sin foto</span></div>}
              {item.recommended&&<span className="restaurant-chat-food-badge">Recomendado</span>}
              {/* La clave cambia con la cantidad: la insignia se vuelve a montar y confirma cada «Agregar» con un pequeño rebote. */}
              {added>0&&<span key={added} className="restaurant-chat-food-qty"><span className="sr-only">En tu pedido: </span>{added}×</span>}
            </div>
            <div className="restaurant-chat-food-copy">{item.categoryName&&<span>{item.categoryName}</span>}<h3>{item.name}</h3>{item.description&&<p>{item.description}</p>}</div>
            <div className="restaurant-chat-food-foot"><strong>{currency} {Number(item.price).toFixed(2)}</strong>{available?<button type="button" className="restaurant-chat-add" disabled={disabled} aria-label={`${choose?"Elegir opciones de":"Agregar"} ${item.name}`} onClick={()=>add(item)}>{choose?<IconAdjustmentsHorizontal size={18}/>:<IconPlus size={18}/>}{choose?"Elegir":"Agregar"}</button>:<span className="restaurant-chat-food-out">No disponible</span>}</div>
          </article>;
        })}</div>}
      </section>}
      {showCart&&<section className="restaurant-chat-cart" aria-label="Resumen de tu pedido">
        <div className="restaurant-chat-section-heading"><h2>Tu pedido{quantity>0&&<span className="restaurant-chat-cart-count">{quantity} {quantity===1?"producto":"productos"}</span>}</h2><button className="restaurant-chat-icon" type="button" aria-label="Ocultar pedido" onClick={()=>setShowCart(false)}><IconX size={20}/></button></div>
        {!state.cart.items.length?<div className="restaurant-chat-cart-empty"><IconShoppingBag size={32}/><strong>Tu pedido está vacío</strong><p>Elige algo de la carta para empezar.</p><button type="button" className="restaurant-chat-primary" onClick={()=>{following.current=true;setShowCart(false);setShowMenu(true)}}>Ver la carta</button></div>:<>
          <ul>{state.cart.items.map((item,index)=>{
            const photo=photos.get(item.productId);
            const options=[...item.modifiers,...item.selections];
            return <li key={`${item.productId}-${index}`} className="restaurant-chat-cart-item">
              <div className="restaurant-chat-cart-thumb">{photo?<Image src={photo} alt="" width={56} height={56} unoptimized/>:<IconChefHat size={24}/>}<span className="restaurant-chat-cart-qty"><span className="sr-only">Cantidad: </span>{item.quantity}×</span></div>
              <div className="restaurant-chat-cart-body"><strong>{item.name}</strong>{options.length>0&&<div className="restaurant-chat-cart-options">{options.map((option,i)=><span key={i}>{option.name}</span>)}</div>}{item.note&&<p className="restaurant-chat-cart-note"><IconNote size={14}/>{item.note}</p>}{item.quantity>1&&<small>{currency} {item.unitPrice} c/u</small>}</div>
              <div className="restaurant-chat-cart-side"><span>{currency} {item.lineTotal}</span><button className="restaurant-chat-icon" type="button" disabled={disabled} aria-label={`Quitar ${item.name} y sus variantes`} onClick={()=>act({name:"remove_item",args:{productId:item.productId}},true)}><IconTrash size={20}/></button></div>
            </li>;
          })}</ul>
          <div className="restaurant-chat-total"><span>Subtotal</span><strong>{currency} {state.cart.total}</strong></div>
          <p className="restaurant-chat-cart-hint">El pedido se enviará cuando lo confirmes.</p>
          <button type="button" className="restaurant-chat-primary" disabled={disabled} onClick={()=>act({name:"prepare_confirmation"})}>Revisar para confirmar</button>
        </>}
      </section>}
      <div ref={bottom}/>
    </div>
    {error&&<p className="restaurant-chat-alert" role="alert">{error.message} <button type="button" onClick={()=>void chat.refetch()}>Reintentar</button></p>}
    {!state.active&&<p className="restaurant-chat-alert" role="alert">El canal cambió. Vuelve a escanear el QR para continuar.</p>}
    {locationError&&<p className="restaurant-chat-alert" role="alert">{locationError}</p>}
    {failed&&<div className="restaurant-chat-alert" role="alert"><span>No se pudo enviar. Puedes reintentar el mismo mensaje sin duplicarlo.</span><button type="button" disabled={send.isPending} onClick={()=>send.mutate(failed)}>Reintentar envío</button></div>}
    {state.confirmationRef&&<div className="restaurant-chat-confirm"><p>¿Confirmas el pedido del resumen?</p><button type="button" className="restaurant-chat-primary" disabled={disabled} onClick={()=>act({name:"confirm_order",confirmation_ref:state.confirmationRef!})}>Confirmar pedido</button></div>}
    <nav className="restaurant-chat-shortcuts" aria-label="Acciones del chat"><button type="button" aria-expanded={showMenu} onClick={()=>{following.current=true;setShowMenu(!showMenu);setShowCart(false)}}><IconMenu2 size={16}/>Carta</button><button type="button" aria-expanded={showCart} onClick={()=>{following.current=true;setShowCart(!showCart);setShowMenu(false)}}><IconShoppingBag size={16}/>Mi pedido{quantity>0&&<span>{quantity}</span>}</button>{state.mode==="table"&&<button type="button" disabled={disabled} onClick={()=>act({name:"request_bill"})}><IconReceipt size={16}/>Pedir cuenta</button>}{state.mode==="delivery"&&<button type="button" disabled={disabled} aria-label={locating?"Obteniendo tu ubicación":"Compartir ubicación"} onClick={shareLocation}><IconMapPin size={16}/>{locating?"Ubicando…":"Ubicación"}</button>}</nav>
    <form className="restaurant-chat-composer" onSubmit={event=>{event.preventDefault();sendText()}}>
      <Input ref={composer} aria-label="Escribe un mensaje" placeholder="Escribe un mensaje" value={text} onChange={event=>setText(event.target.value)} maxLength={2000} disabled={!state.active} autoComplete="off"/>
      <button type="submit" className="restaurant-chat-send" aria-label="Enviar mensaje" disabled={disabled||!text.trim()}><IconSend2 size={24}/></button>
    </form>
  </section>{product&&<OptionsEditor qr={qr} product={product} currency={currency} onClose={()=>setProduct(null)} onAdd={action=>{setProduct(null);act(action,true)}}/>}</main>;
}
