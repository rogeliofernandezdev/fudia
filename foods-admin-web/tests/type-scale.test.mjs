import assert from "node:assert/strict";
import {readFile,readdir,stat} from "node:fs/promises";
import test from "node:test";
import {resolveDesignScales} from "./helpers/design-tokens.mjs";

const root=new URL("../",import.meta.url);
const MIN_PX=10;

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

test(`ninguna hoja de estilos declara texto menor a ${MIN_PX}px`,async()=>{
  const offenders=[];
  const sheets=[...await files("src/modules"),...await files("src/styles"),...await files("src/design-system")];
  for(const sheet of sheets){
    const source=resolveDesignScales(await readFile(new URL(sheet,root),"utf8"));
    for(const match of source.matchAll(/font-size:\s*([0-9.]+)px/g)){
      if(Number(match[1])<MIN_PX)offenders.push(`${sheet}: ${match[0]}`);
    }
    for(const match of source.matchAll(/\bfont:\s*(?:[a-z0-9 ]+\s)?([0-9.]+)px/g)){
      if(Number(match[1])<MIN_PX)offenders.push(`${sheet}: ${match[0]}`);
    }
  }
  assert.deepEqual(offenders,[],`texto por debajo de ${MIN_PX}px:\n${offenders.join("\n")}`);
});
