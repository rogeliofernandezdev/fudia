import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";

const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");

test("Cobros lista solo lo listo para cobrar con la regla del backend",()=>{
  assert.match(read("src/modules/operations/pos/infrastructure/pos-api.ts"),/ready:"true"/);
});

test("un delivery propio no sale «En camino» hasta que caja deje el saldo en cero",()=>{
  const actions=read("src/modules/operations/orders/domain/order-actions.ts");
  assert.match(actions,/if\(order\.channel==="delivery"\)return awaitingPayment\(order\)\?null:\{status:"en_camino"/);
  assert.match(actions,/export function awaitingDispatchPayment/);
  assert.match(read("src/modules/operations/orders/presentation/order-detail-dialog.tsx"),/awaitingDispatchPayment\(order\)&&<p className="order-detail-notice" role="note"><Icon name="cash" size=\{18\}\/><span>Falta cobrar en caja/);
});
