import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const root=new URL("../src/modules/menu/combos/",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file,resolve=require){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,requestAnimationFrame:fn=>fn(),document:{getElementById:()=>null}});
 return exports;
}
const quantities=compile("domain/option-quantity.ts");
const validation=compile("domain/wizard-validation.ts",name=>name==="./option-quantity"?quantities:require(name));
const saleWindow=compile("domain/sale-window.ts");
const optionGroups=compile("domain/option-groups.ts",name=>name==="./option-quantity"?quantities:require(name));
const base={id:"lomo",name:"Lomo saltado",categoryName:"Segundos",price:"20",active:true,availableQuantity:15};
const cause={...base,id:"causa",name:"Causa rellena",categoryName:"Entradas",availableQuantity:12};
function nodes(tree,result=[]){if(Array.isArray(tree))tree.forEach(node=>nodes(node,result));else if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
function mount({products=[base,cause],locationId="local-a",organizationId="org-a",canManage=true,combos=[],autoOpen=true}={}){
 const frames=new Map(),queries=[],invalidations=[];let active,mutations=[];
 const {CombosPage}=compile("presentation/combos-page.tsx",name=>{
  if(name==="react")return{
   useEffect:()=>{},
   useState:initial=>{const owner=active,i=owner.cursor++;if(!(i in owner.state))owner.state[i]=typeof initial==="function"?initial():initial;return[owner.state[i],next=>{owner.state[i]=typeof next==="function"?next(owner.state[i]):next}]},
   useRef:initial=>{const i=active.refCursor++;return active.refs[i]??(active.refs[i]={current:initial})},
  };
  if(name==="@tanstack/react-query")return{
   useQueryClient:()=>({invalidateQueries:async options=>{invalidations.push(options.queryKey)}}),
   useQuery:options=>{queries.push(options);return{data:{items:options.queryKey.includes("combo-picker")?products:combos,total:products.length},isLoading:false,isError:false}},
   useMutation:options=>{mutations.push(options);return{isPending:false,mutate:()=>{}}},
  };
  if(name==="@/providers")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="@/providers/session-context")return{useSession:()=>({organization:{id:organizationId},location:{id:locationId,country:"PE",timezone:"America/Lima"},can:permission=>permission==="menu.manage"?canManage:true})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2})};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Dialog","Button","ConfirmDialog","EmptyState","ErrorState","FieldLabel","FormField","Icon","IconButton","Input","MoneyInput","PageHeader","Pagination","RemoteModalSkeleton","RowActionButton","Select","Status","Textarea","ModalHeader","ModalFooter","ModalError","ModalItemComponents"].map(key=>[key,key]));
  if(name==="../domain/option-quantity")return quantities;
  if(name==="../domain/wizard-validation")return validation;
  if(name==="../domain/sale-window")return saleWindow;
  if(name==="../domain/option-groups")return optionGroups;
  if(name==="./combo-option-picker")return{ComboOptionPicker:"ComboOptionPicker"};
  if(name.includes("regional-format"))return{formatRegionalCalendarDate:value=>`fecha ${value}`,formatRegionalDateTime:value=>value};
  if(name.includes("infrastructure")||name.endsWith(".css"))return{};
  return require(name);
 });
 function render(fn,props,owner){
  const frame=frames.get(owner)??{state:[],refs:[]};frames.set(owner,frame);frame.cursor=frame.refCursor=0;active=frame;
  return nodes(fn(props));
 }
 function page(){mutations=[];return render(CombosPage,{},"page")}
 function wizard(){const node=page().find(node=>node.type?.name==="ComboEditor");return{node,view:render(node.type,node.props,"wizard")}}
 function composition(){const node=wizard().view.find(node=>node.type?.name==="CompositionStep");return render(node.type,node.props,"composition")}
 if(autoOpen)page().find(node=>node.type==="PageHeader").props.action.props.onClick();
 return{page,wizard,composition,queries,invalidations,setCanManage:value=>{canManage=value},setProducts:value=>{products=value},edit:value=>{page();mutations[1].onSuccess(value)},saved:value=>{page();mutations[0].onSuccess({id:"menu"},value)}};
}
const quotaInput=view=>view.find(node=>node.type==="Input"&&node.props.id?.includes("-quota-"));
// Reservar y Costo extra están siempre a la vista en cada línea de opción.
const adjust=ui=>ui.composition();
const template=ui=>ui.composition().find(node=>node.type==="Button"&&node.props.children==="Usar plantilla Menú del día").props.onClick();

test("menús y combos: lectura no habilita crear, editar ni cambiar estado",()=>{
 const combos=[{id:"combo",name:"Menú del día",price:"25",groupCount:2,active:true},{id:"inactive",name:"Menú inactivo",price:"20",groupCount:1,active:false}];
 const ui=mount({canManage:false,autoOpen:false,combos});
 const view=ui.page();
 assert.equal(view.find(node=>node.type==="PageHeader").props.action,undefined);
 assert.deepEqual(view.filter(node=>node.type==="RowActionButton").map(node=>node.props.action),["view","view"]);
 assert.equal(view.some(node=>node.type?.name==="ComboWizard"),false);
 assert.ok(view.some(node=>node.type==="Pagination"));
 assert.equal(mount({canManage:false,autoOpen:false}).page().some(node=>node.type==="Button"&&node.props.icon==="plus"),false);
});

test("menús y combos: las acciones siguen el permiso asignado, también al revocarlo",()=>{
 const ui=mount({combos:[{id:"combo",name:"Menú",price:"25",groupCount:2,active:true}]});
 assert.deepEqual(ui.page().filter(node=>node.type==="RowActionButton").map(node=>node.props.action),["view","edit","deactivate"]);
 ui.page().find(node=>node.type==="RowActionButton"&&node.props.action==="deactivate").props.onClick();
 ui.setCanManage(false);
 const view=ui.page();
 assert.equal(view.some(node=>node.type?.name==="ComboWizard"),false);
 assert.equal(view.find(node=>node.type==="ConfirmDialog").props.open,false);
 assert.deepEqual(view.filter(node=>node.type==="RowActionButton").map(node=>node.props.action),["view"]);
 assert.equal(ui.queries.filter(query=>query.queryKey.includes("combo-picker")).at(-1).enabled,false);
});

test("Reservar parte del saldo disponible del API, incluidos cero y stock fraccionario",()=>{
 for(const [availableQuantity,expected] of [[15,"15"],[12,"12"],[0,"0"],[2.75,"2"],[null,""]]){
  assert.equal(quantities.createComboOption({...base,availableQuantity}).quota,expected);
 }
});

test("agregar una opción precarga el saldo actual y permite reservar menos",()=>{
 const ui=mount({products:[{...base,availableQuantity:12}]});
 ui.composition().find(node=>node.type==="Button"&&node.props.id==="combo-add-group").props.onClick();
 const picker=ui.composition().find(node=>node.type==="ComboOptionPicker");
 picker.props.onPick(picker.props.products.find(product=>product.id==="lomo"));
 let input=quotaInput(adjust(ui));assert.equal(input.props.value,"12");assert.equal(input.props.max,12);assert.equal(input.props.step,"1");
 input.props.onChange({target:{value:"7"}});assert.equal(quotaInput(ui.composition()).props.value,"7");
 ui.setProducts([{...base,availableQuantity:10}]);input=quotaInput(ui.composition());
 assert.equal(input.props.value,"7","el refresco no pisa una cantidad elegida");assert.equal(input.props.max,10);
});

test("la plantilla Menú del día aplica la misma regla a entrada y segundo",()=>{
 const ui=mount();
 template(ui);
 const draft=ui.wizard().node.props.draft;
 assert.equal(draft.groups.find(group=>group.name==="Entrada").options[0].quota,"12");
 assert.equal(draft.groups.find(group=>group.name==="Segundo").options[0].quota,"15");
});

test("la plantilla reconoce sinónimos: sopas y ensaladas van a la entrada, refrescos a bebidas",()=>{
 const soup={...base,id:"sopa",name:"Sopa criolla",categoryName:"Sopas"};
 const salad={...base,id:"ensalada",name:"Ensalada rusa",categoryName:"Ensaladas"};
 const drink={...base,id:"chicha",name:"Chicha morada",categoryName:"Refrescos"};
 const ui=mount({products:[base,cause,soup,salad,drink]});
 template(ui);
 const options=name=>ui.wizard().node.props.draft.groups.find(group=>group.name===name).options.map(option=>option.productId);
 assert.deepEqual(options("Entrada").sort(),["causa","ensalada","sopa"]);
 assert.deepEqual(options("Segundo"),["lomo"]);
 assert.deepEqual(options("Bebidas"),["chicha"]);
 const {belongsToGroup}=optionGroups;
 assert.equal(belongsToGroup(soup,"Sopas"),true,"una parte propia usa su nombre en singular");
 assert.equal(belongsToGroup(soup,"Segundo"),false);
 assert.equal(belongsToGroup({...base,categoryName:null},"Entrada"),false);
});

test("un producto sin control no se confunde con cero disponible",()=>{
 const ui=mount({products:[{...base,availableQuantity:null}]});
 template(ui);
 const input=quotaInput(adjust(ui));assert.equal(input.props.value,"");assert.equal(input.props.placeholder,"Sin límite");assert.equal(input.props.max,undefined);
});

test("la validación compartida rechaza excedentes, negativos y fracciones antes de continuar",()=>{
 for(const quota of ["16","-1","1.5","abc"]){
  const value={groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[{productId:"lomo",quota,surcharge:""}]}]};
  assert.equal(validation.validateComboStep(value,[base],2).field,"combo-group-0-quota-0");
 }
 const ui=mount();
 ui.wizard().view.find(node=>node.type==="Input"&&node.props.id==="combo-name").props.onChange({target:{value:"Menú"}});
 ui.wizard().view.find(node=>node.type==="MoneyInput"&&node.props.id==="combo-price").props.onChange({target:{value:"15,50"}});
 assert.equal(ui.wizard().node.props.draft.price,"15.50","la coma decimal se normaliza");
 template(ui);
 quotaInput(adjust(ui)).props.onChange({target:{value:"13"}});
 assert.equal(quotaInput(ui.composition()).props["aria-invalid"],true);
 const lineError=ui.composition().find(node=>node.props?.className==="field-error combo-option-error");
 assert.match(lineError.props.children,/sin superar la cantidad disponible/,"la línea avisa sin abrir nada");
 ui.wizard().view.find(node=>node.type==="Button"&&node.props.id==="combo-submit").props.onClick();
 assert.equal(ui.page().some(node=>node.props?.title==="Menú registrado"),false);
 assert.equal(quotaInput(ui.composition()).props["aria-describedby"],"combo-group-0-option-0-error","el error queda junto al campo, no arriba de la hoja");
});

test("editar conserva la reserva persistida sin sustituirla por el saldo",()=>{
 const ui=mount();ui.edit({id:"combo",name:"Menú",description:"",price:"25",availableFrom:null,availableUntil:null,groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[{productId:"lomo",quota:4,surcharge:"0.00"}]}]});
 assert.equal(quotaInput(adjust(ui)).props.value,"4");assert.equal(quotaInput(ui.composition()).props.max,15);
});

test("las cantidades del selector se aíslan por empresa/local y se refrescan con el wizard abierto",()=>{
 const a=mount(),b=mount({locationId:"local-b"}),c=mount({organizationId:"org-b"});
 for(const ui of [a,b,c])ui.composition();
 const query=ui=>ui.queries.filter(query=>query.queryKey.includes("combo-picker")).at(-1);
 assert.notDeepEqual(query(a).queryKey,query(b).queryKey);assert.notDeepEqual(query(a).queryKey,query(c).queryKey);
 assert.equal(query(a).refetchInterval,10000);assert.equal(query(a).enabled,true);
});

test("el payload persiste la reserva precargada y no quedan campos antiguos en el wizard",async()=>{
 let payload;
 const api=compile("infrastructure/combos-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(_,options)=>{payload=JSON.parse(options.body);return{id:"combo"}}}:name==="../domain/sale-window"?saleWindow:require(name));
 await api.saveCombo({name:"Menú",description:"",price:"25",availableFrom:"2026-10-18T00:00",availableUntil:"2026-10-24T23:59",availableDays:[],groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[quantities.createComboOption(base)]}]},null,"America/Lima");
 assert.equal(payload.groups[0].options[0].quota,15);
 assert.deepEqual([payload.availableFrom,payload.availableUntil],["2026-10-18T05:00:00.000Z","2026-10-25T04:59:00.000Z"],"el API recibe las fechas en la hora del local");
 for(const file of ["domain/types.ts","domain/wizard-validation.ts","presentation/combos-page.tsx"]){assert.equal(read(file).includes("defaultDailyQuota"),false,file)}
});

test("la hoja empieza con la plantilla y quitar una parte con opciones pide confirmación",()=>{
 const ui=mount();
 const start=ui.composition();
 assert.equal(start.find(node=>node.type==="Button"&&node.props.children==="Usar plantilla Menú del día").props.kind,undefined,"la plantilla es la recomendada: botón relleno índigo (primario)");
 assert.equal(start.find(node=>node.type==="Button"&&node.props.id==="combo-add-group").props.kind,"ghost","«Agregar una parte» es neutra: se distingue de la plantilla");
 assert.equal(start.filter(node=>node.type==="Button"&&(node.props.kind??"primary")==="primary").length,1,"solo la plantilla se rellena; «Agregar una parte» queda neutra");
 assert.equal(start.some(node=>node.type==="ComboOptionPicker"),false,"sin partes no hay buscador de opciones");
 template(ui);
 assert.deepEqual(Array.from(ui.wizard().node.props.draft.groups,group=>group.name),["Entrada","Segundo","Postre","Bebidas"]);
 ui.composition().find(node=>node.type==="IconButton"&&node.props.label==="Quitar Segundo").props.onClick();
 assert.equal(ui.wizard().node.props.draft.groups.length,4,"una parte con opciones no se quita sin confirmar");
 const confirm=ui.wizard().view.find(node=>node.type==="ConfirmDialog"&&node.props.open);
 assert.equal(confirm.props.title,"¿Quitar Segundo?");
 confirm.props.onConfirm();
 assert.deepEqual(Array.from(ui.wizard().node.props.draft.groups,group=>group.name),["Entrada","Postre","Bebidas"]);
 ui.composition().find(node=>node.type==="IconButton"&&node.props.label==="Quitar Bebidas").props.onClick();
 assert.deepEqual(Array.from(ui.wizard().node.props.draft.groups,group=>group.name),["Entrada","Postre"],"una parte vacía se quita al instante");
});

test("«Empezar de nuevo» vuelve al recuadro inicial: confirma si hay opciones y conserva nombre y precio",()=>{
 const ui=mount();
 const reset=()=>ui.wizard().view.find(node=>node.type==="Button"&&node.props.children==="Empezar de nuevo");
 assert.equal(reset(),undefined,"sin partes no hay nada que limpiar");
 const editor=ui.wizard().view;
 editor.find(node=>node.type==="Input"&&node.props.id==="combo-name").props.onChange({target:{value:"Menú ejecutivo"}});
 template(ui);
 assert.equal(reset().props.kind,"ghost","acción neutra, no compite con «Crear menú»");
 reset().props.onClick();
 assert.equal(ui.wizard().node.props.draft.groups.length,4,"con opciones no se limpia sin confirmar");
 const confirm=ui.wizard().view.find(node=>node.type==="ConfirmDialog"&&node.props.open);
 assert.equal(confirm.props.title,"¿Empezar de nuevo?");
 assert.match(confirm.props.description,/4 partes y sus 2 opciones/);
 confirm.props.onConfirm();
 const draft=ui.wizard().node.props.draft;
 assert.equal(draft.groups.length,0);assert.equal(draft.name,"Menú ejecutivo");
 assert.ok(ui.composition().some(node=>node.type==="Button"&&node.props.children==="Usar plantilla Menú del día"),"vuelven los dos botones de inicio");
 assert.equal(ui.wizard().view.some(node=>node.type==="ConfirmDialog"&&node.props.open),false);
 ui.composition().find(node=>node.type==="Button"&&node.props.id==="combo-add-group").props.onClick();
 reset().props.onClick();
 assert.equal(ui.wizard().node.props.draft.groups.length,0,"partes vacías se quitan al instante");
});

test("«Costo extra» reemplaza a «Recargo» y explica que se suma si eligen el plato",()=>{
 const ui=mount();
 template(ui);
 const view=ui.composition();
 const hints=view.filter(node=>node.type==="FieldLabel");
 assert.ok(hints.length>1,"ayuda en la fila de títulos y en cada línea (visible en móvil)");
 assert.ok(hints.every(node=>node.props.hint==="Monto que se suma si se elige este plato."));
 assert.equal(hints[0].props.children,"Costo extra");
 const label=view.find(node=>node.type==="label"&&node.props.className==="combo-cell-label");
 assert.equal(label.props.children,"Costo extra");assert.match(label.props.htmlFor,/-surcharge-/);
 assert.doesNotMatch(read("presentation/combos-page.tsx"),/>Recargo</,"sin «Recargo» en la interfaz");
 const css=read("presentation/combo-editor.css");
 assert.match(css,/\.combo-option-table \{[^}]*overflow: clip;/,"clip: enfocar la ayuda no desplaza las columnas");
 assert.match(css,/\.combo-option-head > :not\(:first-child\) \{\s*justify-self: end;/,"Reservar y Costo extra terminan en el borde derecho de su campo");
 assert.match(css,/\.combo-option-head > :nth-child\(2\) \{\s*padding-right: calc\(var\(--space-2-5\) \+ var\(--stroke-1\)\);/,"Reservar termina con el texto del sufijo");
});

test("elegir un plato cierra el buscador: no salta al siguiente ni crece la lista bajo el cursor",()=>{
 const picker=read("presentation/combo-option-search.tsx");
 assert.match(picker,/closeMenuOnSelect\s/);
 assert.doesNotMatch(picker,/closeMenuOnSelect=\{false\}/);
 assert.match(picker,/blurInputOnSelect=\{false\}/,"el foco sigue en el buscador para agregar otro");
});

test("cuándo se vende: solo Siempre y Elegir días; sin «Solo hoy» que obligaba a crear un menú por día",()=>{
 const ui=mount();
 const radios=()=>ui.wizard().view.filter(node=>node.type==="button"&&node.props.role==="radio");
 assert.deepEqual(radios().map(node=>[node.props.children,node.props["aria-checked"]]),[["Siempre",true],["Elegir días",false]]);
 ui.wizard().view.find(node=>node.type==="Input"&&node.props.id==="combo-name").props.onChange({target:{value:"Menú"}});
 ui.wizard().view.find(node=>node.type==="MoneyInput"&&node.props.id==="combo-price").props.onChange({target:{value:"15"}});
 template(ui);
 radios()[1].props.onClick();
 const chips=ui.wizard().view.filter(node=>node.type==="button"&&node.props.className?.startsWith("day-chip"));
 assert.equal(chips.length,7);assert.equal(chips[0].props["aria-pressed"],false);
 ui.wizard().view.find(node=>node.type==="Button"&&node.props.id==="combo-submit").props.onClick();
 assert.ok(ui.wizard().view.some(node=>node.props?.className==="field-error"&&node.props.children==="Elige al menos un día o una fecha."),"«Elegir días» sin días ni fechas no se guarda como Siempre");
 radios()[0].props.onClick();
 const draft=ui.wizard().node.props.draft;
 assert.deepEqual(JSON.parse(JSON.stringify([draft.availableDays,draft.availableFrom,draft.availableUntil])),[[],"",""]);
 assert.equal(read("presentation/combos-page.tsx").includes("Solo hoy"),false);
});

test("las fechas de venta se convierten con la zona horaria del local, no la del navegador",()=>{
 assert.equal(saleWindow.zonedInputToIso("2026-10-18T23:59","America/Lima"),"2026-10-19T04:59:00.000Z","Lima es UTC-5");
 assert.equal(saleWindow.zonedInputToIso("2026-10-18T00:00","Europe/Madrid"),"2026-10-17T22:00:00.000Z","Madrid en horario de verano es UTC+2");
 assert.equal(saleWindow.zonedInputToIso("2026-12-18T00:00","Europe/Madrid"),"2026-12-17T23:00:00.000Z","y en invierno UTC+1");
 assert.equal(saleWindow.isoToZonedInput("2026-10-19T04:59:00Z","America/Lima"),"2026-10-18T23:59","editar muestra la hora del local");
 assert.equal(saleWindow.zonedInputToIso("","America/Lima"),null);assert.equal(saleWindow.isoToZonedInput(null,"America/Lima"),"");
});

test("el calendario completa el día elegido sin pedir horas ni agregar controles",()=>{
 const ui=mount();
 ui.wizard().view.find(node=>node.type==="button"&&node.props.role==="radio"&&node.props.children==="Elegir días").props.onClick();
 const from=ui.wizard().view.find(node=>node.type==="Input"&&node.props.id==="combo-from");
 assert.equal(from.props.type,"date");
 from.props.onChange({target:{value:"2026-10-21"}});
 const state=ui.wizard();
 assert.equal(state.node.props.draft.availableFrom,"2026-10-21T00:00");
 assert.equal(state.node.props.draft.availableUntil,"2026-10-21T23:59:59.999");
 const until=state.view.find(node=>node.type==="Input"&&node.props.id==="combo-until");
 assert.equal(until.props.type,"date");assert.equal(until.props.value,"2026-10-21");
 until.props.onChange({target:{value:"2026-10-23"}});
 assert.equal(ui.wizard().node.props.draft.availableUntil,"2026-10-23T23:59:59.999");
 assert.equal(state.view.filter(node=>node.type==="button"&&node.props.role==="radio").length,2);
});

test("guardar refresca también el detalle: no conserva Fuera de fecha de antes de editar",()=>{
 const ui=mount();
 const draft=ui.wizard().node.props.draft;
 ui.saved(draft);
 assert.ok(ui.invalidations.some(key=>key[0]==="combo-detail"));
});

test("editar presenta días completos y seleccionar hoy sustituye un rango vencido",()=>{
 const ui=mount({autoOpen:false});
 ui.edit({id:"menu",name:"Menú",description:"",price:"15",groups:[],availableDays:[],availableFrom:"2026-10-20T17:00:00Z",availableUntil:"2026-10-20T19:00:00Z"});
 let state=ui.wizard();
 assert.equal(state.node.props.draft.availableFrom,"2026-10-20T00:00");
 assert.equal(state.node.props.draft.availableUntil,"2026-10-20T23:59:59.999");
 state.view.find(node=>node.type==="Input"&&node.props.id==="combo-from").props.onChange({target:{value:"2026-10-21"}});
 state=ui.wizard();
 assert.equal(state.node.props.draft.availableFrom,"2026-10-21T00:00");
 assert.equal(state.node.props.draft.availableUntil,"2026-10-21T23:59:59.999");
});

test("un menú activo fuera de sus fechas se lista como «Fuera de fecha», no como Activo",()=>{
 const ui=mount({autoOpen:false,combos:[{id:"vencido",name:"Especial",price:"20",groupCount:1,active:true,outsideDates:true},{id:"vigente",name:"Menú del día",price:"15",groupCount:2,active:true,outsideDates:false}]});
 const statuses=ui.page().filter(node=>node.type==="Status").map(node=>[node.props.tone??null,node.props.children]);
 assert.deepEqual(JSON.parse(JSON.stringify(statuses)),[["gray","Fuera de fecha"],[null,"Activo"]]);
});

test("agregar opción: buscador con secciones por categoría, primero las de la parte y sin repetir",()=>{
 const products=[
  {...base,id:"lomo",name:"Lomo Saltado Criollo",categoryName:"segundos",availableQuantity:12},
  {...base,id:"arroz",name:"Arroz con Mariscos",categoryName:"segundos",availableQuantity:null},
  {...base,id:"sopa",name:"Sopa Fuchifú",categoryName:"Sopas",availableQuantity:0},
  {...base,id:"causa",name:"Causa rellena",categoryName:"Entradas",availableQuantity:3},
  {...base,id:"puca",name:"puca picante",categoryName:null,availableQuantity:null},
  {...base,id:"helado",name:"Helado de vainilla",categoryName:"Postres",availableQuantity:5},
 ];
 const groups=JSON.parse(JSON.stringify(optionGroups.buildOptionGroups(products,["lomo"],"Segundo")));
 assert.deepEqual(groups.map(group=>group.label),["Segundos","Entradas","Postres","Sopas","Sin categoría"],"la categoría de la parte va primero y «Sin categoría» al final");
 assert.deepEqual(groups[0].options.map(option=>option.label),["Arroz con Mariscos"],"lo ya agregado no se repite");
 assert.equal(groups.find(group=>group.label==="Sopas").options[0].limit,0,"el saldo viaja con cada opción para mostrar Agotado");
 assert.equal(JSON.parse(JSON.stringify(optionGroups.buildOptionGroups(products,["causa"],"Entrada"))).some(group=>group.label==="Entradas"),false,"una sección sin productos disponibles no se muestra");
 const ui=mount();template(ui);
 const picker=ui.composition().find(node=>node.type==="ComboOptionPicker"&&node.props.inputId==="combo-group-0-product");
 assert.equal(picker.props.partName,"Entrada");assert.deepEqual(Array.from(picker.props.excluded),["causa"]);
});

test("fuera de fechas: el detalle y el aviso de guardado no dicen que el menú está disponible",()=>{
 const now=Date.parse("2026-10-18T15:00:00Z");
 assert.equal(saleWindow.isOutsideDates("2026-05-10T05:00:00Z","2026-05-11T04:59:00Z",now),true,"vencido");
 assert.equal(saleWindow.isOutsideDates("2026-12-24T05:00:00Z",null,now),true,"aún no empieza");
 assert.equal(saleWindow.isOutsideDates(null,null,now),false,"Siempre");
 assert.equal(saleWindow.isOutsideDates("2026-10-01T05:00:00Z","2026-10-31T04:59:00Z",now),false,"vigente");
 const page=read("presentation/combos-page.tsx");
 assert.match(page,/value\.active&&isOutsideDates\(value\.availableFrom,value\.availableUntil\)\?<Status tone="gray">Fuera de fecha<\/Status>/,"el detalle coincide con el listado");
 assert.match(page,/outside\?"Está fuera de sus fechas de venta: se venderá solo en las fechas indicadas\."/);
});

test("el listado de menús muestra solo el nombre, sin la descripción debajo",()=>{
 const ui=mount({autoOpen:false,combos:[{id:"m",name:"Menú del día",description:"",price:"15",groupCount:2,active:true,outsideDates:false}]});
 const view=ui.page();
 assert.ok(view.some(node=>node.type==="b"&&node.props.children==="Menú del día"));
 assert.equal(view.some(node=>node.type==="small"&&node.props.children==="Sin descripción"),false);
});
