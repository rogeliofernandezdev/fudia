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
  await new Promise((r) => setTimeout(r, 1500));
  const offenders = await page.evaluate(() => {
    const vw = window.innerWidth;
    const out = [];
    document.querySelectorAll("*").forEach((el) => {
      const r = el.getBoundingClientRect();
      if (r.right > vw + 1) out.push(`${el.tagName}.${String(el.className).slice(0, 50)} right=${r.right.toFixed(0)}`);
    });
    return { vw, scrollW: document.documentElement.scrollWidth, list: out.slice(0, 10) };
  });
  console.log(JSON.stringify(offenders, null, 1));
  await browser.close();
})().catch((e) => { console.error("FATAL:", e.message); process.exit(1); });
