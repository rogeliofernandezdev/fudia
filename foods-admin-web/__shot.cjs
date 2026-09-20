const puppeteer = require("puppeteer-core");
const TOKEN = process.env.SESSION_TOKEN;
const BASE = "http://localhost:5175";
async function shoot(browser, width, height, out) {
  const page = await browser.newPage();
  await page.setViewport({ width, height });
  await page.setCookie({ name: "foods_session", value: TOKEN, url: BASE });
  await page.goto(BASE + "/salon", { waitUntil: "networkidle2", timeout: 45000 });
  await page.waitForSelector(".salon-table", { timeout: 30000 }).catch(() => {});
  await new Promise((r) => setTimeout(r, 1000));
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll(".salon-table"));
    const free = btns.find((b) => (b.textContent || "").includes("Abrir mesa")) || btns[0];
    if (free) free.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  await page.waitForSelector(".comanda-listrow", { timeout: 30000 }).catch(() => console.log(out + ": NO ROWS"));
  await new Promise((r) => setTimeout(r, 5000));
  const info = await page.evaluate(() => {
    return {
      rows: document.querySelectorAll(".comanda-listrow").length,
      overflowX: document.documentElement.scrollWidth > window.innerWidth + 1 ? "OVERFLOW" : "ok",
    };
  });
  console.log(out, JSON.stringify(info));
  await page.screenshot({ path: out });
  await page.close();
}
(async () => {
  const browser = await puppeteer.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: "new",
    args: ["--no-sandbox"],
  });
  await shoot(browser, 1440, 900, "__comanda-desk2.png");
  await shoot(browser, 390, 844, "__comanda-mob2.png");
  await browser.close();
})().catch((e) => { console.error("FATAL:", e.message); process.exit(1); });
