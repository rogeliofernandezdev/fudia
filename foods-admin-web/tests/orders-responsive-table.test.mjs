import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";

const root=new URL("../",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");
const orders=read("src/modules/operations/orders/presentation/orders-manager.tsx");
const styles=read("src/modules/operations/styles/orders.css");

function ruleInMobile(selector){
  const ast=postcss.parse(styles);
  const rules=[];
  ast.walkAtRules("media",atRule=>{
    if(atRule.params!=="(max-width:700px)")return;
    atRule.walkRules(rule=>{
      if(rule.selectors.includes(selector)){
        rules.push(Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value])));
      }
    });
  });
  assert.equal(rules.length,1,`Expected one mobile rule for ${selector}`);
  return rules[0];
}

test("Pedidos usa el DataTable compartido y su región desplazable",()=>{
  assert.match(orders,/\bDataTable\b/);
  assert.match(orders,/<DataTable label="Listado de pedidos" containerClassName="hover-scroll" className="orders-table">/);
  assert.doesNotMatch(orders,/<div className="table-wrap hover-scroll">/);
  assert.match(orders,/<\/DataTable>/);
  assert.equal(ruleInMobile(".orders-panel .table-wrap")["overflow-x"],"auto");
  assert.equal(ruleInMobile(".orders-panel .table-wrap")["overflow-y"],"hidden");
});

test("Pedidos móvil no aplasta las cuatro columnas informativas",()=>{
  const table=ruleInMobile(".orders-panel .orders-table");
  assert.equal(table["min-width"],"var(--size-500)");
  assert.equal(table.width,"100%");
  assert.equal(table["table-layout"],"fixed");
  for(const [selector,width] of [
    [".orders-panel .orders-table th:nth-child(1)","var(--size-208)"],
    [".orders-panel .orders-table th:nth-child(3)","var(--size-106)"],
    [".orders-panel .orders-table th:nth-child(5)","var(--size-100)"],
    [".orders-panel .orders-table th:nth-child(6)","var(--size-86)"]
  ])assert.equal(ruleInMobile(selector).width,width);
  for(const col of [2,4]){
    assert.equal(ruleInMobile(`.orders-table th:nth-child(${col})`).display,"none");
    assert.equal(ruleInMobile(`.orders-table td:nth-child(${col})`).display,"none");
  }
});

test("la fecha del pedido no fuerza cortes de palabra en pantallas estrechas",()=>{
  const date=ruleInMobile(".orders-table td:first-child .order-mobile-registered");
  const time=ruleInMobile(".orders-table .order-mobile-registered time");
  assert.equal(date["overflow-wrap"],"normal");
  assert.equal(date["word-break"],"normal");
  assert.equal(time["overflow-wrap"],"normal");
  assert.equal(time["white-space"],"normal");
});
