// Run with PLAYWRIGHT_MODULE pointing to the available Playwright installation.
/* eslint-disable @typescript-eslint/no-require-imports -- Standalone Node CJS audit with an externally supplied runtime. */
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs/promises');
const path = require('node:path');
const routes = ['login','turno','inicio','mesas','pos','pos/personalizar','pos/pago','pos/comprobante','cocina','pedidos','caja','clientes','inventario','menu','personal','reportes','reservas'];
(async () => {
  const browser = await chromium.launch({headless:true});
  const results = [];
  const output = path.resolve('artifacts/layout-audit');
  await fs.mkdir(output,{recursive:true});
  for (const width of (process.env.AUDIT_WIDTHS || '390,1024,1440').split(',').map(Number)) {
    const page = await browser.newPage({viewport:{width,height:900},deviceScaleFactor:1});
    for (const route of routes) {
      const response = await page.goto(`${process.env.AUDIT_URL || 'http://localhost:5173'}/${route}`,{waitUntil:'networkidle'});
      if (!response?.ok()) throw new Error(`${route}: HTTP ${response?.status()}`);
      await page.locator('h1,h2').first().waitFor();
      const result = await page.evaluate(() => {
        const visible = e => e.getBoundingClientRect().width && e.getBoundingClientRect().height && getComputedStyle(e).visibility !== 'hidden';
        const controls = [...document.querySelectorAll('button, a.button, .save-order, .send-kitchen, .detail-actions a, .payment-summary>a, .success-card>a, .option-panel footer>a')].filter(visible);
        return {overflow:document.documentElement.scrollWidth>innerWidth,
          small:controls.filter(e=>e.getBoundingClientRect().height<43.5 || e.getBoundingClientRect().width<43.5).map(e=>({text:e.textContent.trim().slice(0,50),class:e.className,w:Math.round(e.getBoundingClientRect().width),h:Math.round(e.getBoundingClientRect().height)}))};
      });
      await page.screenshot({path:path.join(output,`${route.replaceAll('/','-')}-${width}.png`),fullPage:true});
      results.push({route,width,...result});
      console.log(JSON.stringify(results.at(-1)));
    }
    if(width===390){
      await page.locator('.mobile-menu summary').click();
      const menu = await page.locator('.mobile-menu-panel').evaluate(e=>({scroll:e.scrollHeight>e.clientHeight,bottom:e.getBoundingClientRect().bottom}));
      await page.screenshot({path:path.join(output,'mobile-menu.png')});
      console.log(JSON.stringify({menu}));
      await page.keyboard.press('Escape');
    }
    await page.close();
  }
  await fs.writeFile(path.join(output,'report.json'),JSON.stringify(results,null,2));
  await browser.close();
  process.exitCode = results.some(r=>r.overflow || r.small.length) ? 1 : 0;
})().catch(e=>{console.error(e);process.exitCode=1});
