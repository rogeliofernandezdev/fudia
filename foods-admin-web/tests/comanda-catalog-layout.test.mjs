import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
import {resolveTokens} from "./helpers/design-tokens.mjs";
const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
const source=read("src/modules/operations/salon/presentation/comanda-catalog.tsx");
const exports={};
vm.runInNewContext(ts.transpileModule(source+"\nexport {CategoryFilters,MenuItem,ComboItem,categoryTone};",{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:name=>{
 if(name==="@/design-system")return Object.fromEntries(["Button","EmptyState","ErrorState","Icon","Modal","ModalBody","ModalFooter","NumberStepper","SearchField","Skeleton"].map(name=>[name,name]));
 if(name.startsWith(".")||name.startsWith("@/"))return {};
 return require(name);
}});
function nodes(tree,result=[]){if(Array.isArray(tree))tree.forEach(node=>nodes(node,result));else if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result);}return result;}
const categories=["Entradas","Segundos","Bebidas","Postres"].map((name,index)=>({id:`c${index}`,name}));
const base={categories,category:"",loading:false,error:false,retrying:false,onRetry:()=>{},onChange:()=>{}};
test("categorías deslizables incluyen menús y acceso persistente a la lista completa",()=>{
 const changes=[];const view=nodes(exports.CategoryFilters({...base,onChange:id=>changes.push(id)}));
 const chips=view.find(node=>node.props.className==="cmd-chips");
 assert.equal(nodes(chips).filter(node=>node.type==="button").length,6);
 const combo=view.find(node=>node.props.className?.includes("cmd-chip-combos"));
 assert.equal(nodes(chips).includes(combo),true,"menús precede a las categorías del API");
 assert.ok(view.some(node=>node.props["aria-label"]==="Ver todas las categorías"));
 for(const button of view.filter(node=>node.type==="button"))button.props.onClick();
 assert.deepEqual(changes,["","__combos",...categories.map(item=>item.id)]);
 for(const category of ["",...categories.map(item=>item.id),"__combos"]){
  const selected=nodes(exports.CategoryFilters({...base,category})).filter(node=>node.type==="button"&&node.props["aria-pressed"]);
  assert.equal(selected.length,1);
 }
});
test("lista completa muestra todas las categorías y cierra explícitamente",()=>{
 let opened=0,closed=0;const changes=[];
 const view=nodes(exports.CategoryFilters({...base,expanded:true,onExpand:()=>opened++,onDismiss:()=>closed++,onChange:id=>changes.push(id)}));
 view.find(node=>node.props["aria-label"]==="Ver todas las categorías").props.onClick();assert.equal(opened,1);
 const modal=view.find(node=>node.type==="Modal");assert.equal(modal.props.title,"Categorías de la carta");modal.props.onClose();assert.equal(closed,1);
 const options=nodes(modal).filter(node=>node.props.className==="cmd-category-option");assert.equal(options.length,6);
 for(const option of options)option.props.onClick();assert.deepEqual(changes,["","__combos",...categories.map(item=>item.id)]);
});
test("categorías conservan carga, error recuperable y menús aun cuando el catálogo está vacío",()=>{
 const loading=nodes(exports.CategoryFilters({...base,loading:true,categories:[]}));
 assert.ok(loading.some(node=>node.props["aria-label"]==="Cargando categorías"));
 let retries=0;const error=nodes(exports.CategoryFilters({...base,error:true,categories:[],onRetry:()=>retries++}));
 assert.ok(error.some(node=>node.props.role==="alert"));error.find(node=>node.type==="Button"&&node.props.children==="Reintentar").props.onClick();assert.equal(retries,1);
 assert.equal(error.filter(node=>node.type==="button").length,2);
});
test("productos conservan precio, controles de cantidad y bloqueo; foto rota deja visible el icono",()=>{
 const money=value=>`S/ ${value}`;const picks=[];
 const p={id:"p",name:"Plato de prueba",price:"18",imageUrl:"/broken-image"};
 const decreases=[];const view=nodes(exports.MenuItem({p,qty:2,money,onPick:item=>picks.push(item),onDecrease:id=>decreases.push(id)}));
 assert.ok(view.some(node=>node.props.className==="cmd-item-price"&&node.props.children==="S/ 18"));
 assert.ok(view.some(node=>node.type==="Icon"&&node.props.name==="dish"));
 const image={hidden:false};view.find(node=>node.type==="img").props.onError({currentTarget:image});assert.equal(image.hidden,true);
 assert.match(view.find(node=>node.type==="article").props.className,/picked/);
 const stepper=view.find(node=>node.type==="NumberStepper");assert.equal(stepper.props.appearance,"soft");stepper.props.onChange(3);assert.equal(picks[0],p);stepper.props.onChange(1);stepper.props.onRemove();assert.deepEqual(decreases,["p","p"]);
 for(const availabilityStatus of ["sold_out","unavailable"]){
  const props={p:{...p,availabilityStatus},money,onPick:()=>{},onDecrease:()=>{}};
  assert.equal(nodes(exports.MenuItem({...props,qty:0})).find(node=>node.type==="button").props.disabled,true);
  const selected=nodes(exports.MenuItem({...props,qty:2})).find(node=>node.type==="NumberStepper");assert.equal(selected.props.max,2);assert.equal(selected.props.disabled,false);
 }
 let configured=0;const combo=nodes(exports.ComboItem({combo:{id:"combo",name:"Menú",price:"25",groupCount:2},qty:0,money,onConfigure:()=>configured++}));combo.find(node=>node.type==="button").props.onClick();assert.equal(configured,1);
});
test("categorías y carta se deslizan sin barras visibles ni reglas duplicadas",()=>{
 const css=postcss.parse(read("src/modules/operations/styles/salon-comanda.css"));const seen=new Set();
 css.walkRules(rule=>{for(const selector of rule.selectors){const key=`${rule.parent.type==="atrule"?rule.parent.params:"root"}|${selector}`;assert.equal(seen.has(key),false,key);seen.add(key);}});
 const chips=css.nodes.find(node=>node.type==="rule"&&node.selector===".cmd-chips");const declarations=Object.fromEntries(chips.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 assert.equal(declarations["flex-wrap"],undefined);assert.equal(declarations["overflow-y"],"hidden");assert.equal(declarations["scrollbar-width"],"none");assert.equal(declarations["overflow-x"],"auto");
 assert.ok(css.nodes.some(node=>node.selectors?.includes(".cmd-chips::-webkit-scrollbar")));
 for(const selector of [".cmd-list-scroll",".cmd-summary-scroll"]){const rule=css.nodes.find(node=>node.selector===selector);assert.equal(rule.nodes.find(node=>node.prop==="scrollbar-width").value,"none");assert.equal(rule.nodes.find(node=>node.prop==="overflow-y").value,"auto");}
 const selected=css.nodes.find(node=>node.selector===".cmd-chip[aria-pressed=true]");assert.equal(selected.nodes.find(node=>node.prop==="background").value,"var(--primary-700)");
 assert.match(source,/queryFn:listOrderCategories/);
});

test("cada categoría conserva su acento al renombrar o reordenar; selección incluye una marca",()=>{
 assert.equal(exports.categoryTone(""),"indigo");assert.equal(exports.categoryTone("__combos"),"violet");
 const original=new Map(categories.map(item=>[item.id,exports.categoryTone(item.id)]));
 assert.equal(new Set(original.values()).size,4);
 const reordered=categories.toReversed().map(item=>({...item,name:"Nombre actualizado"}));
 for(const item of reordered)assert.equal(exports.categoryTone(item.id),original.get(item.id));
 for(const category of ["","__combos",...categories.map(item=>item.id)]){
  const view=nodes(exports.CategoryFilters({...base,category,categories:reordered,expanded:true}));
  const selected=view.find(node=>node.type==="button"&&node.props.className?.startsWith("cmd-chip ")&&node.props["aria-pressed"]);
  assert.ok(selected.props.className.includes(`cmd-tone-${exports.categoryTone(category)}`));
  assert.ok(nodes(selected).some(node=>node.type==="Icon"&&node.props.name==="check"));
  for(const option of view.filter(node=>node.props.className==="cmd-category-option"))assert.ok(nodes(option).some(node=>node.props.className==="cmd-category-dot"&&node.props["aria-hidden"]));
 }
});

test("acentos de categorías cumplen contraste AA en fondos suaves y seleccionados",()=>{
 const css=postcss.parse(read("src/modules/operations/styles/salon-comanda.css"));
 const declarations=selector=>Object.fromEntries(css.nodes.find(node=>node.selector===selector).nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 function luminance(value){const hex=resolveTokens(value).replace("#","");return [0,2,4].map(index=>parseInt(hex.slice(index,index+2),16)/255).map(channel=>channel<=.04045?channel/12.92:((channel+.055)/1.055)**2.4).reduce((sum,channel,index)=>sum+channel*[.2126,.7152,.0722][index],0);}
 function contrast(foreground,background){const values=[luminance(foreground),luminance(background)].sort((a,b)=>b-a);return (values[0]+.05)/(values[1]+.05);}
 for(const tone of ["indigo","sky","violet","teal"]){
  const base=declarations(tone==="indigo"?".cmd-chip":`.cmd-chip.cmd-tone-${tone}`);
  const selected=declarations(tone==="indigo"?".cmd-chip[aria-pressed=true]":`.cmd-chip.cmd-tone-${tone}[aria-pressed=true]`);
  assert.ok(contrast(base.color,base.background)>=4.5,`${tone}: fondo tenue`);
  assert.ok(contrast(selected.color,selected.background)>=4.5,`${tone}: selección`);
 }
});
