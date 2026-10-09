import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL("../"+path,import.meta.url),"utf8");
function compile(path,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,URLSearchParams,document:{body:{}}});
 return exports;
}
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
const combos=[{code:"unit",name:"Unidad base",unit:"botella",unitName:"Botella",isDefault:false},{code:"crate",name:"Jaba",unit:"botella",unitName:"Botella",isDefault:true}];
function mount({items=combos,loading=false,error=false,manage=true,pending=false,organization="org-a",initial=[],unit="botella",unitName="Botella"}={}){
 const state=[],refs=[],queries=[],writes=[],mutations=[];let cursor=0,refCursor=0,rows=initial;
 const query={data:loading||error?undefined:{items},isLoading:loading,isError:error,isFetching:false,error:error?new Error("Catálogo no disponible"):null,refetch:()=>writes.push("refetch")};
 const exports=compile("src/modules/supply/purchases/presentation/purchase-presentations-field.tsx",name=>{
  if(name==="react")return{useState:initial=>{const i=cursor++;if(!(i in state))state[i]=initial;return[state[i],value=>{state[i]=value}]},useRef:initial=>{const i=refCursor++;return refs[i]??(refs[i]={current:initial})},useId:()=>"radio-group",useEffect:callback=>callback()};
  if(name==="react-dom")return{createPortal:tree=>tree};
  if(name==="@tanstack/react-query")return{useQuery:options=>{queries.push(options);return query},useQueryClient:()=>({setQueryData:(key,fn)=>writes.push(fn({items})),invalidateQueries:()=>writes.push("invalidate")}),useMutation:options=>({isPending:pending,isError:false,mutate:draft=>{mutations.push(draft);options.onSuccess?.({...draft,unitName:"Botella"})}})};
  if(name==="@/providers/session-context")return{useSession:()=>({organization:{id:organization},can:()=>manage})};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","FormField","Icon","IconButton","Input","Select","Dialog"].map(key=>[key,key]));
  if(name==="../infrastructure/purchases-api")return{};
  return require(name);
 });
 const render=()=>{cursor=0;return nodes(exports.PurchasePresentationsField({unit,value:rows,onChange:value=>{rows=value},disabled:false,onBusyChange:value=>writes.push(value)}))};
 const dialog=()=>{cursor=refCursor=0;return nodes(exports.PurchaseCombinationDialog({unit,unitName,close:()=>writes.push("close"),save:value=>writes.push(value)}))};
 return{render,dialog,queries,writes,mutations,get rows(){return rows}};
}
test("presentaciones: opción preferida remota por unidad y empresa, sin adivinar contenido",()=>{
 const ui=mount();ui.render();const tree=ui.render();
 assert.ok(tree.some(node=>node.type==="FormField"&&node.props.label==="Presentación de compra"));
 assert.equal(tree.find(node=>node.type==="Select").props["aria-label"],"Presentación de compra");
 assert.equal(tree.find(node=>node.type==="option"&&node.props.value==="unit").props.children,"Botella");
 assert.equal(tree.some(node=>node.type==="option"&&node.props.children==="Unidad base"),false);
 assert.equal(ui.rows[0].presentationType,"crate");assert.equal(ui.rows[0].unitsPerPresentation,"");
 assert.ok(tree.some(node=>node.type==="option"&&node.props.value==="crate"&&node.props.children==="Jaba"));
 assert.equal(tree.some(node=>node.type==="option"&&node.props.value==="package"),false);
 assert.deepEqual(Array.from(ui.queries[0].queryKey),["purchase-combinations","org-a","botella"]);
 const other=mount({organization:"org-b"});other.render();assert.equal(other.queries[0].queryKey[1],"org-b");
});
const packageCombos=[
 {code:"unit",name:"Unidad base",unit:"paquete",unitName:"Paquete",isDefault:false},
 {code:"package",name:"Paquete",unit:"paquete",unitName:"Paquete",isDefault:true},
 {code:"box",name:"Caja",unit:"paquete",unitName:"Paquete",isDefault:false},
];
test("misma unidad: Paquete aparece una vez, mantiene Caja y no selecciona la opción oculta",()=>{
 for(const name of ["Paquete","  PAQUETE  ","Paquéte"]){
  const ui=mount({unit:"paquete",items:packageCombos.map(item=>item.code==="package"?{...item,name}:item)});ui.render();
  const tree=ui.render(),options=tree.filter(node=>node.type==="option");
  assert.equal(options.filter(node=>node.props.children==="Paquete").length,1);
  assert.equal(options.some(node=>node.props.value==="package"),false);
  assert.ok(options.some(node=>node.props.value==="box"&&node.props.children==="Caja"));
  assert.equal(ui.rows[0].presentationType,"unit");assert.equal(ui.rows[0].unitsPerPresentation,"1");
 }
});
test("misma unidad: conserva la preferencia Caja y la conversión histórica distinta de uno",()=>{
 const preferred=mount({unit:"paquete",items:packageCombos.map(item=>({...item,isDefault:item.code==="box"}))});preferred.render();
 assert.equal(preferred.rows[0].presentationType,"box");assert.equal(preferred.rows[0].unitsPerPresentation,"");
 const ui=mount({unit:"paquete",items:packageCombos,initial:[{presentationType:"package",unitsPerPresentation:"10",isDefault:true}]});
 const tree=ui.render();assert.equal(ui.rows[0].unitsPerPresentation,"10");assert.equal(ui.rows[0].presentationType,"package");
 assert.equal(tree.find(node=>node.type==="option"&&node.props.value==="package").props.children,"Paquete (contenido: 10)");
 const equivalent=mount({unit:"paquete",items:packageCombos,initial:[{presentationType:"package",unitsPerPresentation:"1",isDefault:true}]});equivalent.render();
 assert.equal(equivalent.rows[0].presentationType,"unit");assert.equal(equivalent.rows[0].unitsPerPresentation,"1");
});
test("misma unidad: el + reutiliza la compra individual sin crear otra presentación",()=>{
 const ui=mount({unit:"paquete",unitName:"Paquete",items:packageCombos});
 ui.dialog().find(node=>node.type==="Input").props.onChange({target:{value:" PAQUETE "}});
 ui.dialog().find(node=>node.props.children==="Guardar").props.onClick();
 assert.equal(ui.mutations.length,0);assert.ok(ui.writes.some(item=>item?.code==="unit"&&item.unit==="paquete"));
});
test("compra individual: nombre del catálogo remoto y conversión uno, sin cambiar códigos",()=>{
 const ui=mount({items:combos.map(item=>({...item,unitName:"Lata"}))});ui.render();
 let tree=ui.render();assert.equal(tree.find(node=>node.type==="option"&&node.props.value==="unit").props.children,"Lata");
 tree.find(node=>node.type==="Select").props.onChange({target:{value:"unit"}});
 assert.equal(ui.rows[0].presentationType,"unit");assert.equal(ui.rows[0].unitsPerPresentation,"1");
 tree=ui.render();assert.equal(tree.find(node=>node.props["aria-label"]==="Contenido de la presentación").props.disabled,true);
});
test("presentaciones: una selección y contenido, sin controles redundantes",()=>{
 const ui=mount();ui.render();
 let tree=ui.render();tree.find(node=>node.props["aria-label"]==="Contenido de la presentación").props.onChange({target:{value:"10"}});
 assert.equal(ui.rows.length,1);assert.equal(ui.rows[0].isDefault,true);assert.equal(ui.rows[0].unitsPerPresentation,"10");
 tree=ui.render();assert.equal(tree.filter(node=>node.type==="Select").length,1);
 assert.equal(tree.some(node=>["Otra presentación","Predeterminada","Nueva combinación"].includes(node.props.children)),false);
 assert.equal(tree.some(node=>node.type==="input"),false);
 const plus=tree.find(node=>node.type==="IconButton"&&node.props.label==="Nueva presentación");assert.equal(plus.props.icon,"plus");
 plus.props.onClick();tree=ui.render();assert.ok(ui.writes.includes(true));
 const dialog=tree.find(node=>typeof node.type==="function");
 dialog.props.save({code:"box",name:"Caja",unit:"botella",unitName:"Botella",isDefault:false});
 assert.equal(ui.rows.length,1);assert.equal(ui.rows[0].presentationType,"box");assert.equal(ui.rows[0].unitsPerPresentation,"");assert.equal(ui.rows[0].isDefault,true);assert.ok(ui.writes.includes(false));
});
test("presentaciones: carga, error recuperable, vacío y solo lectura",()=>{
 assert.ok(mount({loading:true}).render().some(node=>node.props["aria-label"]==="Cargando presentaciones"));
 const ui=mount({error:true}),tree=ui.render();assert.ok(tree.some(node=>node.props.role==="alert"));tree.find(node=>node.props.children==="Reintentar").props.onClick();assert.ok(ui.writes.includes("refetch"));
 assert.ok(mount({items:[]}).render().some(node=>node.props.children==="No hay presentaciones"));
 assert.equal(mount({manage:false}).render().some(node=>node.props.label==="Nueva presentación"),false);
});
test("presentaciones: ambos formularios comparten la versión simple y no conservan CSS antiguo",()=>{
 const dialog=read("src/modules/supply/purchases/presentation/purchase-presentation-dialog.tsx");
 assert.ok(dialog.includes("<PurchasePresentationsField"));
 assert.equal(/multiple=|type="checkbox"|Usar por defecto/.test(dialog),false);
 const css=read("src/modules/supply/purchases/presentation/purchases.css");
 assert.equal(/purchase-presentation-default|purchase-presentation-tools/.test(css),false);
 assert.match(css,/@media\(max-width:600px\)\{\.purchase-presentation-row\{grid-template-columns:minmax\(0,1fr\)\}/);
});
test("presentación nueva: solo nombre, sin Enter ni doble POST",()=>{
 const ui=mount();let tree=ui.dialog();
 assert.equal(tree.filter(node=>node.type==="Input").length,1);assert.equal(tree.some(node=>node.type==="Select"||node.type==="input"),false);
 tree.find(node=>node.type==="Input").props.onChange({target:{value:"Fardo grande"}});
 tree=ui.dialog();const save=tree.find(node=>node.props.children==="Guardar");save.props.onClick();save.props.onClick();
 assert.equal(ui.mutations.length,1);assert.equal(ui.mutations[0].code,"fardo-grande");assert.equal(ui.mutations[0].unit,"botella");assert.equal(ui.mutations[0].isDefault,false);
 let prevented=false;tree.find(node=>node.type==="form").props.onSubmit({preventDefault:()=>{prevented=true},stopPropagation:()=>{}});assert.ok(prevented);assert.equal(ui.mutations.length,1);
 assert.ok(mount({pending:true}).dialog().filter(node=>node.type==="Button"||node.type==="Input"||node.props["aria-label"]==="Cerrar").every(node=>node.props.disabled));
});
test("presentación nueva: reutiliza nombres existentes sin duplicar catálogos",()=>{
 const ui=mount({items:[...combos,{code:"box",name:"Caja",unit:"lata",unitName:"Lata",isDefault:false}]});
 ui.dialog().find(node=>node.type==="Input").props.onChange({target:{value:"  CAJA  "}});
 ui.dialog().find(node=>node.props.children==="Guardar").props.onClick();
 assert.equal(ui.mutations[0].code,"box");assert.equal(ui.mutations[0].name,"Caja");
});
test("presentación nueva: nombre obligatorio, catálogo recuperable y códigos automáticos seguros",()=>{
 const ui=mount();ui.dialog().find(node=>node.props.children==="Guardar").props.onClick();
 assert.equal(ui.mutations.length,0);assert.ok(ui.dialog().some(node=>node.props.role==="alert"));
 ui.dialog().find(node=>node.type==="Input").props.onChange({target:{value:"24 unidades"}});
 ui.dialog().find(node=>node.props.children==="Guardar").props.onClick();assert.equal(ui.mutations[0].code,"presentation-24-unidades");
 for(const state of [{loading:true},{error:true}]){
  const remote=mount(state),tree=remote.dialog();const save=tree.find(node=>node.props.children==="Guardar");
  assert.equal(save.props.disabled,true);save.props.onClick();assert.equal(remote.mutations.length,0);
  if(state.error){tree.find(node=>node.props.children==="Reintentar").props.onClick();assert.ok(remote.writes.includes("refetch"))}
 }
 const collision=mount({items:[{...combos[1],code:"fardo-grande",name:"Fardo"}]});
 collision.dialog().find(node=>node.type==="Input").props.onChange({target:{value:"Fardo grande"}});
 collision.dialog().find(node=>node.props.children==="Guardar").props.onClick();assert.equal(collision.mutations[0].code,"fardo-grande-2");
});
test("conversiones: admite bolsa de medio kg, rechaza duplicados, factores inválidos y dos preferencias",()=>{
 const {purchaseInventoryItemSchema}=compile("src/modules/supply/purchases/domain/purchase-schema.ts",require);
 const base={mode:"new_ingredient",categoryId:"",name:"Arroz",description:"",price:"",unit:"kg",presentationType:"unit",unitsPerPresentation:"1",minimumStock:"0"};
 const valid={presentationType:"bag",unitsPerPresentation:"0.5",isDefault:true};
 assert.equal(purchaseInventoryItemSchema.safeParse({...base,presentations:[valid]}).success,true);
 for(const presentations of [[],[valid,valid],[{...valid,unitsPerPresentation:"0"}],[{...valid,unitsPerPresentation:"1.0001"}],[{...valid,isDefault:false}],[valid,{...valid,presentationType:"sack"}]]){
  assert.equal(purchaseInventoryItemSchema.safeParse({...base,presentations}).success,false);
 }
});
test("transporte: combinaciones y conversiones se persisten; crear no genera stock",async()=>{
 const calls=[];
 const api=compile("src/modules/supply/purchases/infrastructure/purchases-api.ts",name=>{
  if(name==="@/shared/api/client")return{apiFetch:(path,options)=>{calls.push({path,options});return Promise.resolve({productId:null})}};
  if(name==="@/shared/api/product-image")return{};
  return require(name);
 });
 await api.listPurchaseCombinations("kg");await api.savePurchaseCombination({unit:"kg",code:"bag",name:"Bolsa",isDefault:true});
 await api.savePurchasePresentation("article",{presentationType:"bag",unitsPerPresentation:"0.5",isDefault:true});
 await api.createPurchaseInventoryItem({mode:"new_ingredient",name:"Arroz",unit:"kg",presentationType:"unit",unitsPerPresentation:"1",minimumStock:"0",presentations:[{presentationType:"bag",unitsPerPresentation:"0.5",isDefault:true}]});
 assert.equal(calls[0].path,"purchase-combinations?unit=kg");assert.equal(calls[1].options.method,"POST");
 assert.equal(JSON.parse(calls[2].options.body).unitsPerPresentation,.5);
 const body=JSON.parse(calls[3].options.body);assert.equal(body.presentations[0].unitsPerPresentation,.5);assert.equal("quantity" in body,false);
});
