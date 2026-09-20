import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("navigation keeps an independently scrollable region", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  assert.match(css, /\.side-scroll\{[^}]*min-height:0;[^}]*overflow-y:auto/);
  assert.match(css, /\.mobile-menu-panel\{[^}]*100dvh[^}]*overflow-y:auto/);
  assert.match(css, /svg:last-child\{grid-column:4;grid-row:1/);
});

test("menu supports keyboard dismissal and resets on navigation", async () => {
  const shell = await readFile(new URL("../src/components/app-shell.tsx", import.meta.url), "utf8");
  assert.match(shell, /details key=\{pathname\}/);
  assert.match(shell, /event.key === "Escape"/);
  assert.match(shell, /querySelector\("summary"\)\?\.focus\(\)/);
  assert.match(shell, /aria-current=/);
});

test("defines one action-control geometry and responsive module grids", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  assert.match(css, /--control-height:44px/);
  assert.match(css, /--control-icon:20px/);
  assert.match(css, /height:var\(--control-height\)/);
  assert.match(css, /\.inv-page[^}]*grid-template-columns:minmax\(0,1fr\)/);
  assert.match(css, /prefers-reduced-motion:reduce/);
});
