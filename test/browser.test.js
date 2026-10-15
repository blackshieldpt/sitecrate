// Browser tests for the runtime: packs testdata/site and drives the
// output in Chromium, Firefox and WebKit, opened from file:// and over HTTP.
// BROWSERS=chromium,firefox limits the engines.
const { test, before, after } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const http = require('node:http');
const os = require('node:os');
const path = require('node:path');
const pw = require('playwright');

const browsers = (process.env.BROWSERS || 'chromium,firefox,webkit').split(',');
const repo = path.join(__dirname, '..');
let file, server;

before(async () => {
  file = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'sitecrate-')), 'site.html');
  execFileSync('go', ['run', './cmd/sitecrate', 'testdata/site', '-o', file], { cwd: repo, stdio: 'inherit' });
  server = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'text/html');
    res.end(fs.readFileSync(file));
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
});

after(() => {
  server.close();
  fs.rmSync(path.dirname(file), { recursive: true });
});

for (const name of browsers) {
  for (const scheme of ['file', 'http']) {
    test(`${name} ${scheme}`, async () => {
      const browser = await pw[name].launch();
      const context = await browser.newContext();
      try {
        const base = scheme === 'file' ? 'file://' + file : `http://127.0.0.1:${server.address().port}/site.html`;
        const failures = await run(context, base);
        assert.deepStrictEqual(failures, []);
      } finally {
        await browser.close();
      }
    });
  }
}

// run walks through the fixture site and returns the failed checks
async function run(context, base) {
  const failures = [];
  const check = (label, ok, detail) => {
    if (!ok) failures.push(label + ': ' + JSON.stringify(detail));
  };
  const errors = [];
  const open = async (hash, init) => {
    const page = await context.newPage();
    page.on('pageerror', (e) => errors.push(String(e)));
    page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
    if (init) await page.addInitScript(init);
    await page.goto(base + hash);
    return page;
  };

  let page = await open('');
  const frame = () => page.locator('iframe').contentFrame();
  const inFrame = (fn) => frame().locator('html').evaluate(fn);
  const hash = () => page.evaluate(() => location.hash);
  const title = async (want) => {
    await page.waitForFunction((t) => document.title === t, want, { timeout: 5000 }).catch(() => {});
    return page.title();
  };
  const scrolled = async (pred) => {
    for (let i = 0; i < 50; i++) {
      const y = await inFrame(() => scrollY).catch(() => -1);
      if (pred(y)) return y;
      await page.waitForTimeout(100);
    }
    return inFrame(() => scrollY);
  };

  check('entry title', await title('Home') === 'Home', await page.title());
  check('script ran', await frame().locator('#js').textContent() === 'js ok');
  const s = await inFrame(async () => {
    const img = (id) => document.getElementById(id);
    await Promise.all(['dot', 'srcset', 'srcset-data'].map((id) => img(id).decode().catch(() => {})));
    const bg = (sel) => getComputedStyle(document.querySelector(sel)).backgroundImage;
    return {
      color: getComputedStyle(document.body).color,
      bg: bg('h1'),
      imageSet: bg('#image-set'),
      upper: bg('#upper'),
      img: img('dot').naturalWidth,
      srcset: img('srcset').getAttribute('srcset'),
      srcsetData: img('srcset-data').naturalWidth,
      ext: img('to-ext').getAttribute('target'),
      href: img('to-about').href,
    };
  });
  check('@import css applied', s.color === 'rgb(0, 0, 128)', s.color);
  check('css url() rewritten', s.bg.startsWith('url("blob:'), s.bg);
  check('css image-set() rewritten', s.imageSet.includes('blob:'), s.imageSet);
  check('css URL() with query and hash rewritten', s.upper.startsWith('url("blob:'), s.upper);
  check('img loaded', s.img === 1, s.img);
  check('srcset URL with comma', /^blob:\S+ 1x, blob:\S+ 2x$/.test(s.srcset), s.srcset);
  check('srcset data URI', s.srcsetData === 1, s.srcsetData);
  check('external link targets _top', s.ext === '_top', s.ext);
  check('internal href is shell route', s.href === base + '#/about/index.html', s.href);

  await frame().locator('#cancelled').click();
  await page.waitForTimeout(300);
  check('page script cancels link', await page.title() === 'Home' && await hash() === '', await hash());

  await frame().locator('#to-about').click();
  check('directory link', await title('About') === 'About' && await hash() === '#/about/index.html', await hash());
  check('root-relative css', await inFrame(() => getComputedStyle(document.body).color) === 'rgb(0, 0, 128)');

  await frame().locator('#to-docs').click();
  check('extensionless root-relative link', await title('Docs') === 'Docs' && await hash() === '#/docs/page.html', await hash());
  check('inline <style> url()', (await inFrame(() => getComputedStyle(document.querySelector('.spacer')).backgroundImage)).startsWith('url("blob:'));

  await frame().locator('#to-second').click();
  let y = await scrolled((v) => v > 2000);
  check('same-page anchor scrolls + shell hash', y > 2000 && await hash() === '#/docs/page.html#second', [y, await hash()]);

  await page.goBack();
  y = await scrolled((v) => v === 0);
  check('back restores top', y === 0 && await hash() === '#/docs/page.html', [y, await hash()]);
  await page.goBack();
  check('back to About', await title('About') === 'About', await page.title());
  await page.goBack();
  check('back to Home', await title('Home') === 'Home', await page.title());

  await frame().locator('#to-second').click();
  await title('Docs');
  y = await scrolled((v) => v > 2000);
  check('cross-page anchor', y > 2000 && await hash() === '#/docs/page.html#second', [y, await hash()]);

  await frame().locator('#to-home').click();
  await title('Home');
  await frame().locator('#to-old').click();
  check('meta refresh redirect', await title('About') === 'About' && await hash() === '#/about/index.html', await hash());
  await page.goBack();
  check('redirect leaves no history entry', await title('Home') === 'Home', await page.title());

  await frame().locator('#form-about').click();
  check('formaction', await title('About') === 'About' && await hash() === '#/about/index.html?q=second+section', await hash());
  await page.goBack();
  await title('Home');

  const results = async (n) => {
    await frame().locator('#results a').nth(n - 1).waitFor({ timeout: 5000 }).catch(() => {});
    return frame().locator('#results a').allTextContents();
  };
  await frame().locator('#search button[name=go]').click();
  check('GET form navigates with query and submitter', await title('Search') === 'Search' && await hash() === '#/search.html?q=second+section&go=yes', await hash());
  check('page sees location.search', await frame().locator('#query').textContent() === 'second section');
  let found = await results(1);
  check('fetch of bundled file + dynamic results', found.length === 1 && found[0] === 'Docs, second section', found);
  await frame().locator('#to-results').click();
  await page.waitForTimeout(300);
  check('same-page anchor keeps query', await hash() === '#/search.html?q=second+section&go=yes#results' &&
    await frame().locator('#query').textContent() === 'second section', await hash());
  await frame().locator('#results a').first().click();
  await title('Docs');
  y = await scrolled((v) => v > 2000);
  check('dynamic link navigates', y > 2000 && await hash() === '#/docs/page.html#second', [y, await hash()]);
  await page.goBack();
  check('back to results', await title('Search') === 'Search' && (await results(1)).length === 1, await hash());
  await frame().locator('#search input').fill('about');
  await frame().locator('#search input').press('Enter');
  await page.waitForFunction(() => location.hash === '#/search.html?q=about', null, { timeout: 5000 }).catch(() => {});
  await frame().locator('#query', { hasText: 'about' }).waitFor({ timeout: 5000 }).catch(() => {});
  found = await results(1);
  check('same-page form (action="#") re-renders', await hash() === '#/search.html?q=about' && found.length === 1 && found[0] === 'About', [await hash(), found]);
  await frame().locator('#results a').first().click();
  check('dynamic directory link', await title('About') === 'About' && await hash() === '#/about/index.html', await hash());

  page = await open('#/docs/page.html#second');
  await title('Docs');
  y = await scrolled((v) => v > 2000);
  check('deep link with anchor', y > 2000, y);

  page = await open('#/later.html');
  check('delayed refresh shows the page first', await title('Later') === 'Later', await page.title());
  check('delayed refresh redirects', await title('About') === 'About' && await hash() === '#/about/index.html', await hash());

  page = await open('#/reload.html');
  await title('Reload');
  await page.waitForTimeout(1500);
  check('reload is dropped', await inFrame(() => document.title) === 'Reload' && await inFrame(() => !document.querySelector('iframe')),
    await inFrame(() => document.title));

  page = await open('#/badbase.html');
  check('invalid <base> ignored', await title('Bad base') === 'Bad base', await page.title());
  await frame().locator('#to-home').click();
  check('link on invalid-base page', await title('Home') === 'Home', await hash());

  page = await open('#/nope.html');
  check('not found', await page.locator('pre').textContent() === 'Not found: /nope.html' && await page.title() === 'Not found',
    [await page.locator('pre').textContent(), await page.title()]);

  page = await open('', () => { delete window.DecompressionStream; });
  check('unpack failure is reported', (await page.locator('pre').textContent()).startsWith('Cannot open this file:'),
    await page.locator('pre').textContent());

  check('no page errors', errors.length === 0, errors);
  return failures;
}
