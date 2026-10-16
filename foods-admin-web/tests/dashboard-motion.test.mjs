import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import postcss from "postcss";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL("../"+file,import.meta.url),"utf8");
const chartsSource=read("src/modules/dashboard/presentation/dashboard-charts.tsx");
const css=postcss.parse(read("src/modules/dashboard/presentation/dashboard.css"));

function charts(){
 const exports={};
 const resolve=name=>name==="@/design-system/icons"?{Icon:"Icon"}:require(name);
 vm.runInNewContext(ts.transpileModule(chartsSource,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const walk=(node,visit)=>{if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(child=>walk(child,visit));return}visit(node);walk(node.props?.children,visit)};
const all=(node,predicate)=>{const found=[];walk(node,item=>{if(predicate(item))found.push(item)});return found};

test("el anillo de medios de pago reparte la circunferencia, agrupa desde el sexto medio y usa la rampa índigo",()=>{
 const {DonutChart}=charts();
 const items=[["card",50],["cash",20],["yape",10],["plin",10],["transfer",6],["other",4]].map(([key,value])=>({key,label:key,value,detail:"1 cobro"}));
 const view=DonutChart({label:"Cobros",items,format:value=>`S/ ${value}`,empty:{icon:"payment",title:"Vacío",text:""}});
 const legend=all(view,node=>node.type==="li");
 assert.deepEqual(legend.map(node=>all(node,item=>item.type==="b")[0].props.children),["card","cash","yape","plin","Otros 2"]);
 const arcs=all(view,node=>node.type==="circle"&&node.props.className?.startsWith("dash-donut-arc"));
 assert.deepEqual(arcs.map(node=>node.props.className),["dash-donut-arc s0","dash-donut-arc s1","dash-donut-arc s2","dash-donut-arc s3","dash-donut-arc s4"]);
 const total=2*Math.PI*46;
 const drawn=arcs.reduce((sum,node)=>sum+Number(node.props.strokeDasharray.split(" ")[0]),0);
 assert.ok(Math.abs(drawn+arcs.length*2-total)<.001,"Los arcos más sus separaciones cubren el anillo completo");
 assert.equal(arcs[1].props.strokeDashoffset,-(total*.5),"Cada arco empieza donde termina el anterior");
 assert.equal(all(view,node=>node.props?.["aria-label"]==="Cobros").length,1,"La leyenda es la lectura accesible");
 const empty=DonutChart({label:"Cobros",items:[],format:String,empty:{icon:"payment",title:"Sin cobros registrados",text:"t"}});
 assert.ok(all(empty,node=>node.type==="b"&&node.props.children==="Sin cobros registrados").length);
 for(const tone of ["s0","s1","s2","s3","s4"])assert.ok(css.toString().includes(`.dash-donut-arc.${tone},.dash-donut-swatch.${tone}`),tone);
});

test("las barras del dashboard se desplazan con transform y nunca animan el ancho",()=>{
 const {BarList,RankTrack}=charts();
 const list=BarList({label:"Canal",items:[{key:"a",label:"A",value:80},{key:"b",label:"B",value:40},{key:"c",label:"C",value:0}],format:String,empty:{icon:"orders",title:"",text:""}});
 const bars=all(list,node=>node.type==="i");
 assert.deepEqual(bars.map(node=>node.props.style.transform),["translateX(0%)","translateX(-50%)","translateX(-100%)"]);
 assert.deepEqual(bars.map(node=>node.props.style.animationDelay),["calc(var(--duration-slow) + 0ms)","calc(var(--duration-slow) + 60ms)","calc(var(--duration-slow) + 120ms)"]);
 assert.equal(all(RankTrack({ratio:.25,index:0}),node=>node.type==="i")[0].props.style.transform,"translateX(-75%)");
 assert.equal(/width:`\$\{/.test(chartsSource),false,"Ninguna barra calcula su ancho en línea");
});

test("el movimiento del dashboard solo usa transform, opacidad y recorte, y se apaga con movimiento reducido",()=>{
 const allowed=new Set(["opacity","transform","clip-path","background-position"]);
 css.walkAtRules("keyframes",rule=>rule.walkDecls(decl=>assert.ok(allowed.has(decl.prop),`${rule.params}: ${decl.prop}`)));
 const reduced=[];css.walkAtRules("media",rule=>{if(rule.params.includes("prefers-reduced-motion"))rule.walkRules(item=>reduced.push(...item.selectors))});
 for(const selector of [".dash-enter",".dash-trend-live",".dash-trend-live::after",".dash-peak",".dash-donut-ring svg",".dash-bar-track i",".top-track i",".dash-live i::after"])assert.ok(reduced.includes(selector),selector);
 assert.ok(chartsSource.includes('matchMedia?.("(prefers-reduced-motion: reduce)")'),"El conteo y las líneas respetan el movimiento reducido");
 assert.ok(chartsSource.includes("const wave=useTweenedSeries(points.map(point=>point.value),0)"),"La onda crece desde la base y se transforma entre series");
 assert.ok(chartsSource.includes("const spark=useTweenedSeries(values,min)"),"Los minigráficos se transforman como la onda");
 assert.equal(/<svg key=/.test(chartsSource.slice(0,chartsSource.indexOf("export function DonutChart"))),false,"Las líneas no se remontan al cambiar de datos");
 assert.ok(chartsSource.includes('<span className="sr-only">{children}</span>'),"El conteo anuncia solo el valor final");
 let transitionAll=false;css.walkDecls("transition",decl=>{if(/\ball\b/.test(decl.value))transitionAll=true});
 assert.equal(transitionAll,false,"Sin transition: all");
});
