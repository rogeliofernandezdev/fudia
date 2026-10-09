import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
import {resolveTokens} from "./helpers/design-tokens.mjs";

const require=createRequire(import.meta.url);
const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file,globals={},resolve=require){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,...globals});
  return exports;
}
const optionQuantity=compile("src/modules/menu/combos/domain/option-quantity.ts");
const validation=compile("src/modules/menu/combos/domain/wizard-validation.ts",{},name=>name==="./option-quantity"?optionQuantity:require(name));
const products=[{id:"lomo",name:"Lomo saltado",active:true,availableQuantity:10}];
const draft=()=>({name:"Menú ejecutivo",description:"",price:"25.00",groups:[{name:"Segundo",required:true,minSelections:1,maxSelections:1,options:[{productId:"lomo",quota:"5",surcharge:""}]}],availableFrom:"",availableUntil:"",availableDays:[]});

test("el wizard identifica y enfoca por id el primer dato inválido de cada paso",()=>{
  assert.equal(validation.validateComboStep({...draft(),name:""},products,1).field,"combo-name");
  assert.equal(validation.validateComboStep({...draft(),price:"-1"},products,1).field,"combo-price");
  assert.equal(validation.validateComboStep({...draft(),groups:[]},products,2).field,"combo-add-group");
  const value=draft();value.groups[0].options=[];
  assert.equal(validation.validateComboStep(value,products,2).field,"combo-group-0-product");
});
test("cupos, recargos y fechas se validan antes de avanzar o guardar",()=>{
  const value=draft();value.groups[0].options[0].quota="11";
  assert.equal(validation.validateComboDraft(value,products).field,"combo-group-0-quota-0");
  value.groups[0].options[0].quota="5";value.groups[0].options[0].surcharge="abc";
  assert.equal(validation.validateComboDraft(value,products).field,"combo-group-0-surcharge-0");
  value.groups[0].options[0].surcharge="";value.availableFrom="2026-10-05T12:00";value.availableUntil="2026-10-05T11:00";
  assert.equal(validation.validateComboDraft(value,products).field,"combo-until");
  value.availableUntil="2026-10-05T14:00";
  const before=JSON.stringify(value);
  assert.equal(validation.validateComboDraft(value,products),null);
  assert.equal(JSON.stringify(value),before,"validar o retroceder no modifica datos");
});
test("Cocina actualiza los paneles sin modal exitoso y conserva el error",()=>{
  const source=read("src/modules/operations/kitchen/presentation/kitchen-board.tsx");
  const success=source.slice(source.indexOf("onSuccess:"),source.indexOf("onError:"));
  assert.equal(success.includes("notify("),false);
  assert.ok(success.includes("invalidateQueries"));
  assert.match(source,/onError:error=>notify\(\{tone:"danger"/);
});
test("la acción primaria tiene una sola definición y comparte geometría",()=>{
  const css=postcss.parse(read("src/styles/globals.css"));
  const primary=[];css.walkRules(rule=>{if(rule.parent.type==="root"&&rule.selectors.includes(".button.primary"))primary.push(rule);});
  assert.equal(primary.length,1);
  assert.equal(primary[0].nodes.find(node=>node.prop==="background").value,"var(--primary-600)");
  assert.ok(primary[0].nodes.every(node=>!["height","min-height","border-radius","font-size"].includes(node.prop)));
});
function luminance(hex){const rgb=hex.replace("#","").match(/../g).map(value=>parseInt(value,16)/255).map(value=>value<=.04045?value/12.92:((value+.055)/1.055)**2.4);return rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722;}
test("el botón de éxito conserva menta con texto de contraste legible",()=>{
  const css=postcss.parse(read("src/providers/styles/feedback.css"));
  const rule=css.nodes.find(node=>node.selector===".feedback-accept");
  const values=Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
  const foreground=luminance(resolveTokens(values.color)),background=luminance(resolveTokens(values.background));
  assert.ok((Math.max(foreground,background)+.05)/(Math.min(foreground,background)+.05)>=4.5);
  assert.equal(values.background,"var(--brand-600)");
});
test("todas las ventanas modales consumen el comportamiento compartido",()=>{
  const files=dir=>readdirSync(new URL(dir+"/",root),{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(`${dir}/${entry.name}`):entry.name.endsWith(".tsx")?[`${dir}/${entry.name}`]:[]);
  for(const file of files("src")){
    if(file==="src/design-system/dialog.tsx")continue;
    const ast=ts.createSourceFile(file,read(file),ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
    function visit(node){if(ts.isJsxOpeningElement(node)&&node.attributes.properties.some(prop=>ts.isJsxAttribute(prop)&&prop.name.getText(ast)==="aria-modal"&&prop.initializer?.getText(ast)==='"true"'))assert.equal(node.tagName.getText(ast),"Dialog",file);ts.forEachChild(node,visit);}
    visit(ast);
  }
});
test("éxito conserva autocierre; errores y confirmaciones no cierran con Escape o fondo",()=>{
  const feedback=read("src/providers/feedback-provider.tsx"),dialog=read("src/design-system/dialog.tsx");
  assert.ok(feedback.includes('if(item.tone!=="success")return;'));
  assert.ok(feedback.includes("item.duration??4200"));
  assert.equal(feedback.includes("onMouseDown="),false);
  assert.equal(dialog.includes('event.key==="Escape"'),false);
  assert.equal(read("src/modules/operations/salon/presentation/comanda-catalog.tsx").includes("onMouseDown="),false);
});

function dialogHarness(){
  const listeners=new Map(),effects=[];
  const document={body:{style:{overflow:"auto"}},activeElement:null,addEventListener:(type,fn)=>{if(!listeners.has(type))listeners.set(type,new Set());listeners.get(type).add(fn);},removeEventListener:(type,fn)=>listeners.get(type)?.delete(fn),getElementById:()=>null};
  class Element{
    constructor(children=[]){this.children=children;this.isConnected=true;this.initial=false;}
    contains(element){return this===element||this.children.some(child=>child.contains(element));}
    getClientRects(){return [1];}
    closest(){return null;}
    focus(){document.activeElement=this;for(const fn of listeners.get("focusin")??[])fn({target:this});}
    querySelector(){return this.children.find(child=>child.initial)??null;}
    querySelectorAll(selector){return selector==='[aria-controls]'?[]:this.children;}
  }
  const {Dialog,closeDialogsForFeedback}=compile("src/design-system/dialog.tsx",{document,HTMLElement:Element,Node:Element},name=>name==="react"?{useRef:value=>({current:value}),useEffect:effect=>effects.push(effect)}:require(name));
  function mount(panel,props={}){const tree=Dialog({children:[],...props});tree.props.ref(panel);const cleanups=effects.splice(0).map(effect=>effect()).filter(Boolean);return()=>cleanups.forEach(cleanup=>cleanup());}
  function key(key,shiftKey=false){let prevented=false;for(const fn of listeners.get("keydown")??[])fn({key,shiftKey,defaultPrevented:false,preventDefault(){prevented=true;}});return prevented;}
  return{document,Element,mount,key,closeDialogsForFeedback};
}

test("una respuesta cierra todos los modales de acción de hijo a padre, sin cerrar el aviso",()=>{
  const{Element,mount,closeDialogsForFeedback}=dialogHarness();
  const closed=[];
  const parent=mount(new Element(),{onResponseClose:()=>closed.push("formulario")});
  const child=mount(new Element(),{onResponseClose:()=>closed.push("confirmación")});
  const feedback=mount(new Element());
  closeDialogsForFeedback();
  assert.deepEqual(closed,["confirmación","formulario"]);
  child();parent();closeDialogsForFeedback();
  assert.equal(closed.length,2,"no quedan callbacks de modales desmontados");
  feedback();
});

test("todos los modales de origen registran su cierre de respuesta en Dialog",()=>{
  const files=dir=>readdirSync(new URL(dir+"/",root),{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(`${dir}/${entry.name}`):entry.name.endsWith(".tsx")?[`${dir}/${entry.name}`]:[]);
  for(const file of files("src")){
    if(file==="src/providers/feedback-provider.tsx")continue;
    const ast=ts.createSourceFile(file,read(file),ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
    function visit(node){
      if((ts.isJsxOpeningElement(node)||ts.isJsxSelfClosingElement(node))&&node.tagName.getText(ast)==="Dialog")assert.ok(node.attributes.properties.some(prop=>ts.isJsxAttribute(prop)&&prop.name.getText(ast)==="onResponseClose"),file);
      ts.forEachChild(node,visit);
    }
    visit(ast);
  }
});

test("Feedback cierra el origen antes de publicar éxito o error y conserva el mensaje remoto",()=>{
  const events=[];
  const{FeedbackProvider}=compile("src/providers/feedback-provider.tsx",{window:{clearTimeout(){}}},name=>{
    if(name==="react")return{createContext:()=>({Provider:"Provider"}),useCallback:fn=>fn,useContext:()=>null,useEffect:()=>{},useMemo:fn=>fn(),useRef:value=>({current:value}),useState:initial=>[initial,value=>events.push(value)]};
    if(name==="@/design-system/dialog")return{Dialog:"Dialog",closeDialogsForFeedback:()=>events.push("cerrado")};
    if(name==="@/design-system/icons")return{Icon:"Icon"};
    if(name.endsWith(".css"))return{};
    return require(name);
  });
  const{notify}=FeedbackProvider({children:[]}).props.value;
  for(const tone of ["success","danger"]){
    events.length=0;notify({tone,title:"Respuesta",message:"Mensaje del API"});
    assert.equal(events[0],"cerrado");
    assert.equal(events.at(-1).tone,tone);assert.equal(events.at(-1).message,"Mensaje del API");
  }
});
test("Dialog retiene Tab y devuelve el foco al abrirse desde un botón",()=>{
  const {document,Element,mount,key}=dialogHarness();
  const opener=new Element(),first=new Element(),last=new Element(),panel=new Element([first,last]);
  opener.focus();first.initial=true;const cleanup=mount(panel);
  assert.equal(document.activeElement,first);assert.equal(document.body.style.overflow,"hidden");
  last.focus();assert.equal(key("Tab"),true);assert.equal(document.activeElement,first);
  assert.equal(key("Tab",true),true);assert.equal(document.activeElement,last);
  assert.equal(key("Escape"),false);assert.equal(document.activeElement,last);
  cleanup();assert.equal(document.activeElement,opener);assert.equal(document.body.style.overflow,"auto");
});
test("Dialog conserva el foco y el scroll al cerrar una confirmación superpuesta",()=>{
  const {document,Element,mount}=dialogHarness();
  const opener=new Element(),field=new Element(),panel=new Element([field]);opener.focus();const closeFirst=mount(panel);
  field.focus();const cancel=new Element(),confirm=new Element([cancel]);const closeConfirm=mount(confirm);
  assert.equal(document.activeElement,cancel);closeConfirm();assert.equal(document.activeElement,field);assert.equal(document.body.style.overflow,"hidden");
  closeFirst();assert.equal(document.activeElement,opener);assert.equal(document.body.style.overflow,"auto");
});
test("descartar el wizard y su confirmación restaura el botón de apertura",()=>{
  const {document,Element,mount}=dialogHarness();
  const opener=new Element(),field=new Element(),panel=new Element([field]);opener.focus();const closeWizard=mount(panel);
  const cancel=new Element(),confirm=new Element([cancel]);const closeConfirm=mount(confirm);
  field.isConnected=false;panel.isConnected=false;closeWizard();closeConfirm();
  assert.equal(document.activeElement,opener);assert.equal(document.body.style.overflow,"auto");
});
test("el selector de composición no limita las alternativas a la primera página",async()=>{
  const calls=[];
  const api=compile("src/modules/menu/combos/infrastructure/combos-api.ts",{},name=>name==="@/shared/api/client"?{apiFetch:async path=>{calls.push(path);const page=new URLSearchParams(path.split("?")[1]).get("page");return page==="1"?{items:Array.from({length:100},(_,index)=>({id:String(index)})),total:101}:{items:[{id:"100"}],total:101};}}:require(name));
  const result=await api.listComboProducts();
  assert.equal(result.items.length,101);assert.equal(result.items.at(-1).id,"100");assert.equal(calls.length,2);
});
