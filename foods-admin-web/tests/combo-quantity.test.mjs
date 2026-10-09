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
const base={id:"lomo",name:"Lomo saltado",categoryName:"Segundos",price:"20",active:true,availableQuantity:15};
const cause={...base,id:"causa",name:"Causa rellena",categoryName:"Entradas",availableQuantity:12};
function nodes(tree,result=[]){if(Array.isArray(tree))tree.forEach(node=>nodes(node,result));else if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
function mount({products=[base,cause],locationId="local-a",organizationId="org-a"}={}){
 const frames=new Map(),queries=[];let active,mutations=[];
 const {CombosPage}=compile("presentation/combos-page.tsx",name=>{
  if(name==="react")return{
   useEffect:()=>{},
   useState:initial=>{const owner=active,i=owner.cursor++;if(!(i in owner.state))owner.state[i]=typeof initial==="function"?initial():initial;return[owner.state[i],next=>{owner.state[i]=typeof next==="function"?next(owner.state[i]):next}]},
   useRef:initial=>{const i=active.refCursor++;return active.refs[i]??(active.refs[i]={current:initial})},
  };
  if(name==="@tanstack/react-query")return{
   useQueryClient:()=>({invalidateQueries:async()=>{}}),
   useQuery:options=>{queries.push(options);return{data:{items:options.queryKey.includes("combo-picker")?products:[],total:products.length},isLoading:false,isError:false}},
   useMutation:options=>{mutations.push(options);return{isPending:false,mutate:()=>{}}},
  };
  if(name==="@/providers")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="@/providers/session-context")return{useSession:()=>({organization:{id:organizationId},location:{id:locationId,country:"PE"}})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/"})};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Dialog","Button","ConfirmDialog","Icon","Input","PageHeader","Pagination","RemoteModalSkeleton","RowActionButton","Select","Status","Textarea"].map(key=>[key,key]));
  if(name==="../domain/option-quantity")return quantities;
  if(name==="../domain/wizard-validation")return validation;
  if(name.includes("infrastructure")||name.endsWith(".css")||name.includes("regional-format"))return{};
  return require(name);
 });
 function render(fn,props,owner){
  const frame=frames.get(owner)??{state:[],refs:[]};frames.set(owner,frame);frame.cursor=frame.refCursor=0;active=frame;
  return nodes(fn(props));
 }
 function page(){mutations=[];return render(CombosPage,{},"page")}
 function wizard(){const node=page().find(node=>node.type?.name==="ComboWizard");return{node,view:render(node.type,node.props,"wizard")}}
 function composition(){const node=wizard().view.find(node=>node.type?.name==="CompositionStep");return render(node.type,node.props,"composition")}
 page().find(node=>node.type==="PageHeader").props.action.props.onClick();
 wizard().node.props.setStep(2);
 return{page,wizard,composition,queries,setProducts:value=>{products=value},edit:value=>{page();mutations[1].onSuccess(value);wizard().node.props.setStep(2)}};
}
const quotaInput=view=>view.find(node=>node.type==="Input"&&node.props.id?.includes("-quota-"));

test("Reservar parte del saldo disponible del API, incluidos cero y stock fraccionario",()=>{
 for(const [availableQuantity,expected] of [[15,"15"],[12,"12"],[0,"0"],[2.75,"2"],[null,""]]){
  assert.equal(quantities.createComboOption({...base,availableQuantity}).quota,expected);
 }
});

test("agregar una opción precarga el saldo actual y permite reservar menos",()=>{
 const ui=mount({products:[{...base,availableQuantity:12}]});
 ui.composition().find(node=>node.type==="Button"&&node.props.id==="combo-add-group").props.onClick();
 ui.composition().find(node=>node.type==="Select").props.onChange({target:{value:"lomo"}});
 let input=quotaInput(ui.composition());assert.equal(input.props.value,"12");assert.equal(input.props.max,12);assert.equal(input.props.step,"1");
 input.props.onChange({target:{value:"7"}});assert.equal(quotaInput(ui.composition()).props.value,"7");
 ui.setProducts([{...base,availableQuantity:10}]);input=quotaInput(ui.composition());
 assert.equal(input.props.value,"7","el refresco no pisa una cantidad elegida");assert.equal(input.props.max,10);
});

test("la plantilla Menú del día aplica la misma regla a entrada y segundo",()=>{
 const ui=mount();
 ui.composition().find(node=>node.type==="Button"&&node.props.children==="Plantilla Menú del día").props.onClick();
 const draft=ui.wizard().node.props.draft;
 assert.equal(draft.groups.find(group=>group.name==="Entrada").options[0].quota,"12");
 assert.equal(draft.groups.find(group=>group.name==="Segundo").options[0].quota,"15");
});

test("un producto sin control no se confunde con cero disponible",()=>{
 const ui=mount({products:[{...base,availableQuantity:null}]});
 ui.composition().find(node=>node.type==="Button"&&node.props.children==="Plantilla Menú del día").props.onClick();
 const input=quotaInput(ui.composition());assert.equal(input.props.value,"");assert.equal(input.props.placeholder,"Sin límite");assert.equal(input.props.max,undefined);
});

test("la validación compartida rechaza excedentes, negativos y fracciones antes de continuar",()=>{
 for(const quota of ["16","-1","1.5","abc"]){
  const value={groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[{productId:"lomo",quota,surcharge:""}]}]};
  assert.equal(validation.validateComboStep(value,[base],2).field,"combo-group-0-quota-0");
 }
 const ui=mount();ui.composition().find(node=>node.type==="Button"&&node.props.children==="Plantilla Menú del día").props.onClick();
 quotaInput(ui.composition()).props.onChange({target:{value:"13"}});
 assert.equal(quotaInput(ui.composition()).props["aria-invalid"],true);
 ui.wizard().view.find(node=>node.type==="button"&&node.props.className==="button primary").props.onClick();
 const wizard=ui.wizard();assert.equal(wizard.node.props.step,2);assert.ok(wizard.view.some(node=>node.props.role==="alert"));
});

test("editar conserva la reserva persistida sin sustituirla por el saldo",()=>{
 const ui=mount();ui.edit({id:"combo",name:"Menú",description:"",price:"25",availableFrom:null,availableUntil:null,groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[{productId:"lomo",quota:4,surcharge:"0.00"}]}]});
 assert.equal(quotaInput(ui.composition()).props.value,"4");assert.equal(quotaInput(ui.composition()).props.max,15);
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
 const api=compile("infrastructure/combos-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(_,options)=>{payload=JSON.parse(options.body);return{id:"combo"}}}:require(name));
 await api.saveCombo({name:"Menú",description:"",price:"25",availableFrom:"",availableUntil:"",availableDays:[],groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[quantities.createComboOption(base)]}]},null);
 assert.equal(payload.groups[0].options[0].quota,15);
 for(const file of ["domain/types.ts","domain/wizard-validation.ts","presentation/combos-page.tsx"]){assert.equal(read(file).includes("defaultDailyQuota"),false,file)}
});
