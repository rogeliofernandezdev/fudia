import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
function compile(path,resolve,extra={}){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,...extra});
 return exports;
}
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
function mount({items=[{id:"u1",code:"saco-25",name:"Saco de 25 kg"}],loading=false,error=false,manage=true,pending=false,organizationId="org-a"}={}){
 const state=[],refs=[],queries=[],changes=[],writes=[],mutations=[];let cursor=0,refCursor=0;
 const query={data:{items},isLoading:loading,isError:error,isFetching:false,error:error?new Error("Servicio no disponible"):null,refetch:()=>writes.push("refetch")};
 const exports=compile("src/modules/supply/purchases/presentation/purchase-unit-field.tsx",name=>{
  if(name==="react")return{useState:initial=>{const index=cursor++;if(!(index in state))state[index]=initial;return[state[index],value=>{state[index]=value}]},useRef:initial=>{const index=refCursor++;return refs[index]??(refs[index]={current:initial})}};
  if(name==="react-dom")return{createPortal:tree=>tree};
  if(name==="@tanstack/react-query")return{useQuery:options=>{queries.push(options);return query},useQueryClient:()=>({setQueryData:(key,fn)=>writes.push(fn({items})),invalidateQueries:()=>writes.push("invalidate")}),useMutation:options=>({isPending:pending,isError:false,mutate:draft=>{mutations.push(draft);options.onSuccess?.({...draft,id:"new-unit"})}})};
  if(name==="@/providers/session-context")return{useSession:()=>({organization:{id:organizationId},can:()=>manage})};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","FormField","Icon","IconButton","Input","Select","Dialog"].map(key=>[key,key]));
  if(name==="../infrastructure/inventory-units-api")return{listInventoryUnits:()=>{},createInventoryUnit:()=>{}};
  return require(name);
 },{document:{body:{}}});
 const render=()=>{cursor=0;return nodes(exports.PurchaseUnitField({value:"",onChange:value=>changes.push(value),disabled:false,onBusyChange:value=>writes.push(value)}))};
 const dialog=()=>{cursor=0;refCursor=0;return nodes(exports.InventoryUnitDialog({close:()=>writes.push("close"),save:value=>changes.push(value)}))};
 return{render,dialog,queries,changes,writes,mutations,state};
}
test("unidades: opciones remotas sin lista fija y caché aislada por empresa",()=>{
 const a=mount(),b=mount({organizationId:"org-b"});
 const tree=a.render();b.render();
 assert.equal(tree.find(node=>node.type==="FormField").props.label,"Unidad de inventario");
 assert.equal(tree.find(node=>node.type==="Select").props["aria-label"],"Unidad de inventario");
 assert.ok(tree.some(node=>node.type==="option"&&node.props.value==="saco-25"&&node.props.children==="Saco de 25 kg"));
 assert.equal(tree.some(node=>node.type==="option"&&node.props.value==="und"),false);
 assert.equal(a.queries[0].queryKey[1],"org-a");assert.equal(b.queries[0].queryKey[1],"org-b");
 tree.find(node=>node.type==="Select").props.onChange({target:{value:"saco-25"}});
 assert.deepEqual(a.changes,["saco-25"]);
 const source=read("src/modules/supply/purchases/presentation/purchase-item-dialog.tsx");
 assert.ok(source.includes("<PurchaseUnitField"));assert.equal(/<option value="(?:und|kg|botella|lata|caja|l)">/.test(source),false);
});
test("unidades: skeleton, error con reintento y vacío sin opciones inventadas",()=>{
 assert.ok(mount({loading:true}).render().some(node=>node.props["aria-label"]==="Cargando unidades"));
 const error=mount({error:true}),tree=error.render();
 assert.equal(tree.find(node=>node.type==="Select").props.disabled,true);
 assert.equal(tree.find(node=>node.type==="FormField").props.error,"Servicio no disponible");
 tree.find(node=>node.type==="Button"&&node.props.children==="Reintentar").props.onClick();
 assert.ok(error.writes.includes("refetch"));
 const empty=mount({items:[]}).render();
 assert.ok(empty.some(node=>node.type==="option"&&node.props.children==="No hay unidades registradas"));
 assert.equal(empty.find(node=>node.type==="Select").props.disabled,true);
});
test("unidades: crear selecciona el resultado confirmado y lectura no ofrece alta",()=>{
 assert.equal(mount({manage:false}).render().some(node=>node.props.label==="Nueva unidad"),false);
 const ui=mount();ui.render().find(node=>node.type==="IconButton"&&node.props.label==="Nueva unidad").props.onClick();
 const dialog=ui.render().find(node=>typeof node.type==="function");assert.ok(dialog);
 dialog.props.save({id:"u2",code:"bolsa-grande",name:"Bolsa grande"});
 assert.deepEqual(ui.changes,["bolsa-grande"]);
 assert.ok(ui.writes.some(value=>value?.items?.some(item=>item.code==="bolsa-grande")));
 assert.ok(ui.writes.includes("invalidate"));
});

test("unidad y presentación: mismo botón global, contenedor y alineación de columnas",()=>{
 const unit=read("src/modules/supply/purchases/presentation/purchase-unit-field.tsx");
 const presentation=read("src/modules/supply/purchases/presentation/purchase-presentations-field.tsx");
 for(const source of [unit,presentation]){
  assert.match(source,/<IconButton icon="plus" label="Nueva (?:unidad|presentación)"/);
  assert.ok(source.includes('className="purchase-reference-controls"'));
  assert.equal(/<Button[^>]*icon="plus"/.test(source),false);
 }
 const css=read("src/modules/supply/purchases/presentation/purchases.css");
 const grid=css.match(/\.purchase-item-section \.form-grid\{([^}]+)\}/)[1];
 const row=css.match(/\.purchase-presentation-row\{([^}]+)\}/)[1];
 assert.equal(grid.match(/gap:([^;]+)/)[1],row.match(/gap:([^;]+)/)[1]);
 assert.equal(/\.purchase-presentations\{[^}]*margin-top/.test(css),false);
});
test("unidad nueva: guardado explícito validado, sin cerrar durante envío",()=>{
 const ui=mount();let tree=ui.dialog();
 tree.find(node=>node.type==="Button"&&node.props.children==="Guardar").props.onClick();
 assert.equal(ui.mutations.length,0);
 tree=ui.dialog();const inputs=tree.filter(node=>node.type==="Input");
 inputs[0].props.onChange({target:{value:"Bolsa grande"}});
 tree=ui.dialog();tree.filter(node=>node.type==="Input")[1].props.onChange({target:{value:" BOLSA-GRANDE "}});
 tree=ui.dialog();tree.find(node=>node.type==="Button"&&node.props.children==="Guardar").props.onClick();
 assert.equal(ui.mutations[0].code,"bolsa-grande");assert.equal(ui.changes[0].name,"Bolsa grande");
 tree.find(node=>node.type==="Button"&&node.props.children==="Guardar").props.onClick();assert.equal(ui.mutations.length,1,"No repetir el POST antes de renderizar el estado ocupado");
 const pending=mount({pending:true}).dialog();
 assert.ok(pending.filter(node=>node.type==="Button"||node.type==="Input"||node.props["aria-label"]==="Cerrar").every(node=>node.props.disabled));
 let prevented=false,stopped=false;
 tree.find(node=>node.type==="form").props.onSubmit({preventDefault:()=>{prevented=true},stopPropagation:()=>{stopped=true}});
 assert.ok(prevented&&stopped);
});
test("unidades: transporte guarda en el catálogo remoto y no solo en memoria",async()=>{
 const calls=[];
 const api=compile("src/modules/supply/purchases/infrastructure/inventory-units-api.ts",name=>name==="@/shared/api/client"?{apiFetch:(path,options)=>{calls.push({path,options});return Promise.resolve({})}}:require(name));
 await api.listInventoryUnits();await api.createInventoryUnit({code:" BOLSA-GRANDE ",name:" Bolsa grande "});
 assert.equal(calls[0].path,"purchase-units");assert.equal(calls[1].options.method,"POST");
 assert.deepEqual(JSON.parse(calls[1].options.body),{code:"bolsa-grande",name:"Bolsa grande"});
});
