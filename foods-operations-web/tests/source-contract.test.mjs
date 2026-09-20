import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import test from "node:test";

test("no fija un locale regional de Peru en la UI", async () => {
  const forbiddenRegionalLocale=["es","PE"].join("-");
  const sourceRoot=new URL("../src/",import.meta.url);
  const files=(await readdir(sourceRoot,{recursive:true})).filter(path=>/\.(ts|tsx|js|jsx)$/.test(path));
  const bad=[];
  for(const path of files){
    const content=await readFile(new URL(path,sourceRoot),"utf8");
    if(content.includes(forbiddenRegionalLocale))bad.push(path);
  }
  assert.deepEqual(bad,[]);
});

test("declares installable application metadata", async () => {
  const manifest = await readFile(new URL("../src/app/manifest.ts", import.meta.url), "utf8");
  assert.match(manifest, /display:\s*"standalone"/);
  assert.match(manifest, /theme_color:\s*"#176B52"/);
});

test("keeps required visual tokens and mobile breakpoint", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  assert.match(css, /--green:#16875f/);
  assert.match(css, /--blue:#3946b8/);
  assert.match(css, /--violet:#5421a8/);
  assert.match(css, /max-width:390px/);
});

test("menu interactions use stable color transitions and reduced motion", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  assert.match(css, /\.side\{[^}]*border-right:1px solid #ffffff14/);
  assert.match(css, /\.mobile-menu\[open\] \.mobile-menu-panel\{animation:menu-panel-in/);
  assert.match(css, /@media\(prefers-reduced-motion:reduce\)/);
});

test("primary action uses the taller 48px geometry", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  assert.match(css, /\.button\.primary,.button\.primary\.green,.button\.green,.open-table,.send-kitchen,.login-submit\{(?=[^}]*min-height:48px)(?=[^}]*height:48px)(?=[^}]*padding-inline:20px)/);
});

test("primary actions and table headers use the requested blue", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  assert.match(css, /--primary-600:#4654cd/);
  assert.match(css, /\.button\.primary,.button\.primary\.green,.button\.green,.open-table,.send-kitchen,.login-submit\{(?=[^}]*color:#fff)(?=[^}]*background:var\(--primary-600\))/);
  assert.match(css, /body :is\(\.inv-table-head,.table-head\)\{[^}]*background:var\(--primary-600\)/);
  const controls = await readFile(new URL("../src/components/ui/controls.module.css", import.meta.url), "utf8");
  assert.match(controls, /\.button\[data-tone="primary"\]\{--action-bg:#4654cd/);
  assert.match(controls, /\.table th\{padding:14px 20px;background:#4654cd;color:#fff/);
});

test("sidebar colapsable y carga remota conservan sus contratos", async () => {
  const shell = await readFile(new URL("../src/components/app-shell.tsx", import.meta.url), "utf8");
  const navigation = await readFile(new URL("../src/app/navigation-state.css", import.meta.url), "utf8");
  const loading = await readFile(new URL("../src/app/loading.tsx", import.meta.url), "utf8");
  for (const contract of ["collapsed", "side-toggle", "data-sidebar", "Contraer navegación", "Expandir navegación"]) assert.match(shell, new RegExp(contract));
  assert.match(navigation, /data-sidebar="collapsed"/);
  assert.match(loading, /aria-busy="true"/);
});

test("las acciones de tabla comparten icono accesible y tooltip", async () => {
  const controls = await readFile(new URL("../src/components/ui/controls.tsx", import.meta.url), "utf8");
  const css = await readFile(new URL("../src/components/ui/controls.module.css", import.meta.url), "utf8");
  assert.match(controls, /export function TableAction/);
  assert.match(controls, /data-tooltip=\{label\}/);
  assert.match(css, /content:attr\(data-tooltip\)/);
});

test("cerrar sesión vive en el perfil superior y no en la navegación", async () => {
  const shell = await readFile(new URL("../src/components/app-shell.tsx", import.meta.url), "utf8");
  assert.match(shell, /className="profile-menu"/);
  assert.match(shell, /className="profile-signout"/);
  assert.match(shell, /fetch\("\/api\/session", \{ method: "DELETE" \}\)/);
  assert.doesNotMatch(shell, /className="logout"/);
  assert.doesNotMatch(shell, /className="mobile-logout"/);
});
