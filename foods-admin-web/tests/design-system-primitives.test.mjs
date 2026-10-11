import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";

const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");

test("primitivas homologadas: se exportan desde el design system y providers",()=>{
 const index=read("src/design-system/index.ts");
 for(const name of ["EmptyState","ErrorState","Skeleton","TableSkeleton","SearchField","Tabs","TabsBar"])assert.match(index,new RegExp(`\\b${name}\\b`),name);
 assert.doesNotMatch(index,/\bToolbar\b/,"El Toolbar sin comportamiento fue retirado");
 assert.match(read("src/providers/index.ts"),/useMoney/);
});

test("estados: vacío con estado y error recuperable con reintento secundario",()=>{
 const states=read("src/design-system/states.tsx");
 assert.match(states,/className=\{`catalog-state \$\{className\}`\.trim\(\)\} role="status"/);
 assert.match(states,/className=\{`catalog-state error \$\{className\}`\.trim\(\)\} role="alert"/);
 assert.match(states,/<Button kind="secondary" icon="refresh"/);
});

test("skeleton: un único brillo y tabla con las columnas reales",()=>{
 const skeleton=read("src/design-system/skeleton.tsx");
 const css=read("src/design-system/styles/skeleton.css");
 assert.match(skeleton,/columns\.map\(column=><th key=\{column\}>\{column\}<\/th>\)/);
 assert.match(skeleton,/aria-busy="true"/);
 assert.equal((css.match(/@keyframes/g)??[]).length,1);
 assert.match(css,/prefers-reduced-motion:reduce/);
});

test("pestañas: tablist accesible con foco itinerante y teclado",()=>{
 const tabs=read("src/design-system/tabs.tsx");
 assert.match(tabs,/role="tablist"/);
 assert.match(tabs,/role="tab" aria-selected=\{selected\} tabIndex=\{selected\?0:-1\}/);
 for(const key of ["ArrowRight","ArrowLeft","Home","End"])assert.ok(tabs.includes(key),key);
 assert.match(read("src/design-system/styles/tabs.css"),/min-height:var\(--size-44\)/,"44 px en móvil");
});

test("status y dinero: tonos rojo/violeta y formato regional con signo",()=>{
 const header=read("src/design-system/page-header.tsx");
 const tones=header.match(/tone\?:([^;]+);/)?.[1]??"";
 for(const tone of ["green","blue","sky","orange","red","violet","gray","teal"])assert.ok(tones.includes(`"${tone}"`),tone);
 const globals=read("src/styles/globals.css");
 assert.match(globals,/\.status\.red\{/);
 assert.match(globals,/\.status\.violet\{/);
 const money=read("src/providers/use-money.ts");
 assert.match(money,/formatRegionalNumber\(Math\.abs\(amount\)/);
 assert.match(money,/const sign=amount<0\?"-":""/);
});

test("fundamentos: foco visible y una sola animación spin",()=>{
 const globals=read("src/styles/globals.css");
 assert.ok(globals.includes(":where(button,a,summary,[role=tab],[role=button],[role=switch],[tabindex]):focus-visible{outline:var(--stroke-2) solid var(--primary-600);outline-offset:var(--space-0-5)}"),"Un solo foco visible para todo control interactivo");
 assert.equal((globals.match(/@keyframes spin\{/g)??[]).length,1);
 assert.match(read("src/shell/admin-shell.tsx"),/currentItem\?\.href===item\.href\?"active":""/,"Una sola opción activa en el menú");
});
