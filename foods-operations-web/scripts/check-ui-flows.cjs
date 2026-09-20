/* eslint-disable @typescript-eslint/no-require-imports -- Standalone browser audit. */
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const assert = require('node:assert/strict');
(async()=>{
 const browser = await chromium.launch({headless:true});
 try {
  const page = await browser.newPage({viewport:{width:390,height:844}});
  const base = process.env.AUDIT_URL || 'http://localhost:5174';
  await page.goto(base+'/turno');
  assert.equal(await page.locator('select').count(),2);
  await page.goto(base+'/pos');
  await page.getByRole('textbox',{name:'Buscar productos'}).fill('zzzzz');
  await page.getByText('Sin resultados',{exact:true}).waitFor();
  await page.getByRole('button',{name:'Limpiar búsqueda'}).click();
  assert.equal(await page.locator('.premium-product').count(),8);
  await page.goto(base+'/pos/personalizar');
  await page.getByLabel('Bien cocido',{exact:true}).check();
  assert.equal(await page.getByLabel('Bien cocido',{exact:true}).isChecked(),true);
  await page.getByRole('link',{name:'Guardar cambios'}).click();
  await page.waitForURL('**/pos');
  await page.goto(base+'/pos/pago');
  await page.getByRole('textbox',{name:'Monto recibido'}).fill('1');
  assert.equal(await page.getByRole('button',{name:'Confirmar pago'}).isDisabled(),true);
  await page.getByRole('button',{name:'S/ 50.00',exact:true}).click();
  await page.getByRole('link',{name:'Confirmar pago'}).click();
  await page.waitForURL('**/pos/comprobante');
  await page.getByRole('button',{name:/Factura/}).click();
  await page.getByPlaceholder('20XXXXXXXXX').waitFor();
  await page.goto(base+'/inventario');
  assert.equal(await page.getByRole('table').count(),1);
  await page.locator('.mobile-menu summary').click();
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('.mobile-menu').getAttribute('open'),null);
  console.log('PASS: selectors, search, customization, disabled payment, receipt selection, table and keyboard navigation. UI-only checks; no backend transaction assertions.');
 } finally {await browser.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});
