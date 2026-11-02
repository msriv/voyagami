// Renders each diagram HTML in this folder to a PNG next to it. Run: node docs/diagrams/render.js
const { chromium } = require("playwright");
const fs = require("fs");
const path = require("path");
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 800 }, deviceScaleFactor: 2 });
  for (const f of fs.readdirSync(__dirname).filter((f) => f.endsWith(".html"))) {
    await page.goto("file://" + path.join(__dirname, f));
    await page.locator(".wrap").screenshot({ path: path.join(__dirname, f.replace(/\.html$/, ".png")) });
  }
  await browser.close();
})();
