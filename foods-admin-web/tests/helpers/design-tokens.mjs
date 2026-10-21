import {readFileSync} from "node:fs";
import postcss from "postcss";

export const tokenSource=readFileSync(new URL("../../src/styles/variables.css",import.meta.url),"utf8");
export const tokenAst=postcss.parse(tokenSource);
export const tokens=new Map();
tokenAst.walkDecls(d=>{
  if(d.parent.parent.type==="root"&&[":root","body"].includes(d.parent.selector))tokens.set(d.prop,d.value.trim());
});

export function resolveTokens(value,definitions=tokens,stack=[],preserveUnknown=false){
  let result="",cursor=0;
  while(true){
    const start=value.indexOf("var(",cursor);
    if(start<0)return result+value.slice(cursor);
    result+=value.slice(cursor,start);
    let depth=1,end=start+4,comma=-1;
    for(;end<value.length&&depth;end++){
      if(value[end]==="(")depth++;
      else if(value[end]===")")depth--;
      else if(value[end]===","&&depth===1&&comma<0)comma=end;
    }
    if(depth)throw new Error(`Unclosed variable: ${value}`);
    const name=value.slice(start+4,comma<0?end-1:comma).trim();
    if(stack.includes(name))throw new Error(`Circular token: ${[...stack,name].join(" -> ")}`);
    const replacement=definitions.get(name)??(preserveUnknown||comma<0?undefined:value.slice(comma+1,end-1).trim());
    result+=replacement===undefined?value.slice(start,end):resolveTokens(replacement,definitions,[...stack,name],preserveUnknown);
    cursor=end;
  }
}

// Existing layout contracts inspect geometry, while keeping semantic colors as vars.
export function resolveDesignScales(source){
  const scales=new Map([...tokens].filter(([name])=>/^--(?:size|space|radius|stroke|blur|font-size|font-weight|line-height|tracking|duration)-/.test(name)));
  return resolveTokens(source,scales,[],true).replace(/calc\((\d+(?:\.\d+)?px) \* -1\)/g,"-$1");
}
