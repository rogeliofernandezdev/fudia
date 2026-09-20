const puppeteer = require("puppeteer-core");
const TOKEN = process.env.SESSION_TOKEN;
const BASE = "http://localhost:5175";
(async () => {
  const browser = await puppeteer.launch({
    executablePath: "C:/Program Files/Google/Chrome/Application/chrome.exe",
    headless: "new",
    args: ["--no-sandbox"],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 390, height: 844 });
  await page.setCookie({ name: "foods_session", value: TOKEN, url: BASE });
  await page.goto(BASE + "/salon", { waitUntil: "networkidle2", timeout: 45000 });
  await page.waitForSelector(".salon-table", { timeout: 30000 }).catch(() => {});
  await new Promise((r) => setTimeout(r, 1000));
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll(".salon-table"));
    const free = btns.find((b) => (b.textContent || "").includes("Abrir mesa")) || btns[0];
    if (free) free.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  await page.waitForSelector(".comanda-listrow", { timeout: 30000 }).catch(() => console.log("NO ROWS"));
  await new Promise((r) => setTimeout(r, 4000));
  // pick 3 items
  await page.evaluate(() => {
    const rows = Array.from(document.querySelectorAll(".comanda-listrow"));
    rows.slice(0, 3).forEach((row) => row.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  });
  await new Promise((r) => setTimeout(r, 500));
  // open the ticket sheet
  await page.evaluate(() => {
    const btn = document.querySelector(".comanda-bar-info");
    if (btn) btn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  await new Promise((r) => setTimeout(r, 600));
  const sheetOpen = await page.evaluate(() => document.querySelector(".comanda-ticket").classList.contains("open"));
  const closeVisible = await page.evaluate(() => {
    const c = document.querySelector(".comanda-ticket-close");
    return c ? getComputedStyle(c).display : "missing";
  });
  await page.screenshot({ path: "__mob-sheet.png" });
  console.log("sheetOpen:", sheetOpen, "| close btn:", closeVisible);
  // close it
  await page.evaluate(() => {
    const c = document.querySelector(".comanda-ticket-close");
    if (c) c.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  await new Promise((r) => setTimeout(r, 600));
  const sheetClosed = await page.evaluate(() => !document.querySelector(".comanda-ticket").classList.contains("open"));
  const rowsVisible = await page.evaluate(() => document.querySelectorAll(".comanda-listrow").length);
  console.log("after close — sheet closed:", sheetClosed, "| rows visible:", rowsVisible);
  await browser.close();
})().catch((e) => { console.error("FATAL:", e.message); process.exit(1); });
