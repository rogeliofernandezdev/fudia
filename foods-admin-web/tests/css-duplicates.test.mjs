import assert from "node:assert/strict";
import {readFile,readdir,stat} from "node:fs/promises";
import test from "node:test";
import postcss from "postcss";

const root=new URL("../",import.meta.url);
const sharedRoots=["src/design-system/styles","src/shell/styles"];

async function files(dir){
  const absolute=new URL(dir+"/",root);
  const names=await readdir(absolute);
  const out=[];
  for(const name of names){
    const url=new URL(name,absolute);
    const info=await stat(url);
    if(info.isDirectory())out.push(...await files(dir+"/"+name));
    else if(name.endsWith(".css"))out.push(dir+"/"+name);
  }
  return out;
}

test("cada hoja compartida especializada declara un selector una sola vez por contexto",async()=>{
  for(const sheet of (await Promise.all(sharedRoots.map(files))).flat()){
    const ast=postcss.parse(await readFile(new URL(sheet,root),"utf8"));
    const seen=new Set(),duplicates=[];
    ast.walkRules(rule=>{
      const context=rule.parent.type==="atrule"?`@${rule.parent.name} ${rule.parent.params}`:"root";
      for(const selector of rule.selectors){
        const key=`${context}|${selector}`;
        if(seen.has(key))duplicates.push(key);
        seen.add(key);
      }
    });
    assert.deepEqual(duplicates,[],sheet);
  }
});

test("las hojas compartidas especializadas no compiten por el mismo selector raiz",async()=>{
  const owners=new Map();
  for(const sheet of (await Promise.all(sharedRoots.map(files))).flat()){
    const ast=postcss.parse(await readFile(new URL(sheet,root),"utf8"));
    ast.walkRules(rule=>{
      if(rule.parent.type!=="root")return;
      for(const selector of rule.selectors){
        if(!owners.has(selector))owners.set(selector,new Set());
        owners.get(selector).add(sheet);
      }
    });
  }
  const shared=[...owners].filter(([,sheets])=>sheets.size>1).map(([selector,sheets])=>`${selector} -> ${[...sheets].join(" + ")}`);
  assert.deepEqual(shared,[]);
});


test("globals conserva un inventario acotado de duplicados heredados",async()=>{
  const sheet="src/styles/globals.css";
  const ast=postcss.parse(await readFile(new URL(sheet,root),"utf8"));
  const seen=new Set(),duplicates=new Set();
  ast.walkRules(rule=>{
    const parent=rule.parent.type==="atrule"?`@${rule.parent.name} ${rule.parent.params}`:"root";
    for(const selector of rule.selectors){
      const key=`${parent}|${selector}`;
      if(seen.has(key))duplicates.add(key);
      seen.add(key);
    }
  });
  assert.ok(duplicates.size<=47,`globals.css aumentó duplicados heredados: ${duplicates.size}`);
});
