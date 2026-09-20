import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import postcss from "postcss";

test("globals no reescribe selectores dentro del mismo contexto", async () => {
  const css = await readFile(new URL("../src/app/globals.css", import.meta.url), "utf8");
  const root = postcss.parse(css);
  const duplicates = [];
  const inspect = container => {
    const selectors = new Set();
    for (const node of container.nodes ?? []) {
      if (node.type === "rule") {
        if (selectors.has(node.selector)) duplicates.push(node.selector);
        selectors.add(node.selector);
      }
      if (node.nodes) inspect(node);
    }
  };
  inspect(root);
  assert.deepEqual(duplicates, []);
});
