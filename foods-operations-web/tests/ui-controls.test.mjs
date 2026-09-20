import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { readdir } from "node:fs/promises";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

const require = createRequire(import.meta.url);
const source = await readFile(new URL("../src/components/ui/controls.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS } }).outputText;
const exports = {};
vm.runInNewContext(compiled, { exports, require: name => name.endsWith(".css") ? new Proxy({}, { get: (_, key) => String(key) }) : require(name) });

test("Button preserves native disabled semantics and safe default type", () => {
  const html = renderToStaticMarkup(createElement(exports.Button, { disabled: true, tone: "primary" }, "Guardar"));
  assert.match(html, /type="button"/);
  assert.match(html, /disabled=""/);
  assert.match(html, /data-tone="primary"/);
  assert.match(html, /data-layout="action"/);
});

test("Input forwards accessible error associations", () => {
  const html = renderToStaticMarkup(createElement(exports.Input, { id: "email", "aria-invalid": true, "aria-describedby": "email-help", type: "email" }));
  assert.match(html, /id="email"/);
  assert.match(html, /aria-invalid="true"/);
  assert.match(html, /aria-describedby="email-help"/);
  assert.match(html, /data-ui="input"/);
});

async function sourceFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = await Promise.all(entries.map(entry => entry.isDirectory()
    ? sourceFiles(new URL(`${entry.name}/`, directory))
    : [new URL(entry.name, directory)]));
  return files.flat().filter(file => /\.(tsx|jsx)$/.test(file.pathname));
}

test("application screens do not bypass shared form controls", async () => {
  const roots = [new URL("../src/app/", import.meta.url), new URL("../src/components/", import.meta.url)];
  const files = (await Promise.all(roots.map(sourceFiles))).flat().filter(file => !file.pathname.endsWith("/ui/controls.tsx"));
  for (const file of files) {
    const content = await readFile(file, "utf8");
    assert.doesNotMatch(content, /<(input|select|textarea|label)(?:\s|>)/, `${file.pathname} bypasses controls.tsx`);
  }
});

test("Select and Table retain native HTML semantics", () => {
  assert.match(renderToStaticMarkup(createElement(exports.Select, { "aria-label": "Local" }, createElement("option", { value: "1" }, "Local actual"))), /<select/);
  const html = renderToStaticMarkup(createElement(exports.Table, { caption: "Existencias" }, createElement("tbody")));
  assert.match(html, /<caption>Existencias<\/caption>/);
  assert.match(html, /role="region"/);
  assert.match(html, /tabindex="0"/);
});
