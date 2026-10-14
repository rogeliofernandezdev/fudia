import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const source = readFileSync(new URL("../src/app/uploads/products/[file]/route.ts", import.meta.url), "utf8");
const code = ts.transpileModule(source, {compilerOptions: {module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022}}).outputText;
const image = "12345678-1234-1234-1234-123456789abc.png";
function routes(fetch) {
  const exports = {};
  vm.runInNewContext(code, {exports, fetch, Response, Request, URL, AbortSignal, Set, process: {env: {FOODS_API_URL: "https://api.example.com"}}});
  return exports;
}
const context = file => ({params: Promise.resolve({file})});

test("serves image bytes through the BFF without upstream cookies, credentials or cache pinning", async () => {
  const api = routes(async (url, options) => {
    assert.equal(String(url), `https://api.example.com/uploads/products/${image}`);
    assert.equal(options.headers, undefined);
    assert.equal(options.cache, "no-store"); assert.equal(options.redirect, "error");
    return new Response(new Uint8Array([137, 80, 78, 71]), {headers: {"Content-Type": "image/png", "Set-Cookie": "ignored"}});
  });
  const response = await api.GET(new Request(`https://app.example.com/uploads/products/${image}`, {headers: {Cookie: "foods_session=private"}}), context(image));
  assert.equal(response.status, 200);
  assert.equal(response.headers.get("Content-Type"), "image/png");
  assert.equal(response.headers.get("Set-Cookie"), null);
  assert.equal(response.headers.get("Cache-Control"), "no-cache");
  assert.equal(response.headers.get("X-Content-Type-Options"), "nosniff");
  assert.deepEqual([...new Uint8Array(await response.arrayBuffer())], [137, 80, 78, 71]);
});

test("HEAD preserves image type but does not return a body", async () => {
  const api = routes(async (_, options) => {
    assert.equal(options.method, "HEAD"); return new Response(null, {headers: {"Content-Type": "image/webp"}});
  });
  const response = await api.HEAD(new Request("https://app.example.com", {method: "HEAD"}), context(image.replace("png", "webp")));
  assert.equal(response.status, 200); assert.equal(await response.text(), "");
});

test("only product UUID filenames and supported raster formats reach the backend", async () => {
  const api = routes(async () => {throw new Error("must not fetch");});
  for (const name of ["../secret.png", "a.svg", "../../.env", "123.png", `${image}?x=1`, "https://evil.example.com/a.png"]) {
    assert.equal((await api.GET(new Request("https://app.example.com"), context(name))).status, 404);
  }
});

test("errors, redirects, and non-image content cannot become successful image responses", async () => {
  for (const [upstream, expected] of [[new Response(null, {status: 404}), 404], [new Response("error", {status: 500}), 502], [new Response("<script>", {headers: {"Content-Type": "text/html"}}), 502]]) {
    const api = routes(async () => upstream);
    assert.equal((await api.GET(new Request("https://app.example.com"), context(image))).status, expected);
  }
  const unavailable = routes(async () => {throw new Error("connection failed");});
  assert.equal((await unavailable.GET(new Request("https://app.example.com"), context(image))).status, 503);
});
