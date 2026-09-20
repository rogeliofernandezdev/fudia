import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import postcss from "postcss";

// Hojas importadas por src/app/layout.tsx, en orden de cascada.
const sheets = ["globals.css", "navigation-state.css", "loading.css", "table-actions.css"];

const parse = async name => postcss.parse(await readFile(new URL(`../src/app/${name}`, import.meta.url), "utf8"));

test("globals no reescribe selectores dentro del mismo contexto", async () => {
  const root = await parse("globals.css");
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

test("un selector no se declara en dos hojas del mismo contexto", async () => {
  const owners = new Map();
  for (const sheet of sheets) {
    const root = await parse(sheet);
    root.walkRules(rule => {
      const context = rule.parent.type === "atrule" ? `@${rule.parent.name} ${rule.parent.params}` : "root";
      for (const selector of rule.selectors) {
        const key = `${context}|${selector}`;
        if (!owners.has(key)) owners.set(key, new Set());
        owners.get(key).add(sheet);
      }
    });
  }
  const shared = [...owners]
    .filter(([, files]) => files.size > 1)
    .map(([key, files]) => `${key} -> ${[...files].join(" + ")}`);
  assert.deepEqual(shared, []);
});
