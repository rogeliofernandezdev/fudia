import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";

const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");

test("con el módulo costos, el egreso del turno es un gasto con categoría",()=>{
 const page=read("src/modules/operations/cash/presentation/cash-page.tsx");
 assert.match(page,/const cashExpenses=Boolean\(modules\?\.costos\)/);
 assert.match(page,/\{cashExpenses\?"Gasto":"Egreso"\}/,"el botón del turno dice Gasto solo con el módulo");
 assert.match(page,/asExpense=\{cashExpenses\}/);
 const dialogs=read("src/modules/operations/cash/presentation/cash-dialogs.tsx");
 assert.match(dialogs,/function CashExpenseDialog/);
 assert.match(dialogs,/queryKey:\["cash-expense-categories"\]/);
 for(const fragment of ["Cargando categorías","No pudimos cargar las categorías","Aún no hay categorías de gastos"])assert.ok(dialogs.includes(fragment),fragment);
 const schema=read("src/modules/operations/cash/domain/cash-schema.ts");
 assert.match(schema,/categoryId:z\.string\(\)\.min\(1,"Selecciona la categoría del gasto\."\)/);
 const api=read("src/modules/operations/cash/infrastructure/cash-api.ts");
 assert.match(api,/\.\.\.\(draft\.categoryId\?\{categoryId:draft\.categoryId\}:\{\}\)/);
 assert.match(api,/"cash-expense-categories"/);
});

test("la pestaña Gastos no registra efectivo y anula según el turno de origen",()=>{
 const panel=read("src/modules/expenses/presentation/expenses-page.tsx");
 assert.match(panel,/paymentOptions\.filter\(option=>!cashMethods\.includes\(option\.value\)\)/);
 assert.ok(panel.includes("Los gastos en efectivo se registran desde el turno, en Caja › Cajas."));
 assert.match(panel,/\(!expense\.cashRegisterName\|\|expense\.cashShiftOpen\)/,"sin anular con el turno cerrado");
 assert.ok(panel.includes("el importe volverá al efectivo del turno de"));
 const labels=read("src/modules/operations/cash/domain/report-format.ts");
 assert.ok(labels.includes('expense:"Gasto",expense_void:"Gasto anulado"'));
});
