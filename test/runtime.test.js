// Unit tests for the runtime's URL and rewriting helpers, run in Node without
// a browser.
const { test } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const SHELL = 'file:///out.html';

// load evaluates the runtime parts that do not need a DOM, with a manifest
// holding files (path -> MIME type)
function load(files) {
  const code = ['unpack.js', 'url.js', 'rewrite.js']
    .map((f) => fs.readFileSync(path.join(__dirname, '..', 'runtime', f), 'utf8')).join('\n');
  const context = vm.createContext({ location: { href: SHELL + '#/index.html' }, URL, Blob, TextDecoder });
  const rt = vm.runInContext(`'use strict';\n${code}\n({ manifest, at, resolve, route, blobURL, rewriteCSS, rewriteSrcset, parseRefresh })`, context);
  for (const [p, type] of Object.entries(files)) rt.manifest.files.set(p, { type, data: new Uint8Array() });
  return rt;
}

const rt = load({
  'index.html': 'text/html',
  'about/index.html': 'text/html',
  'docs/page.html': 'text/html',
  'search.html': 'text/html',
  'a b.html': 'text/html',
  'img/dot.png': 'image/png',
  'img/a,b.png': 'image/png',
  'css/base.css': 'text/css',
});
// plain copies, so that deepStrictEqual does not compare prototypes across realms
const plain = (v) => JSON.parse(JSON.stringify(v));

test('resolve', () => {
  const cases = [
    ['about/', 'index.html', { path: 'about/index.html', search: '', hash: '' }],
    ['/docs/page', 'about/index.html', { path: 'docs/page.html', search: '', hash: '' }],
    ['../img/dot.png?v=1#x', 'docs/page.html', { path: 'img/dot.png', search: '?v=1', hash: '#x' }],
    ['a%20b.html', 'index.html', { path: 'a b.html', search: '', hash: '' }],
    ['./', 'about/index.html', { path: 'about/index.html', search: '', hash: '' }],
    ['missing.png', 'index.html', null],
    ['https://example.com/index.html', 'index.html', null],
    ['data:image/png;base64,AAAA', 'index.html', null],
    ['%', 'index.html', null],
  ];
  for (const [ref, from, want] of cases) assert.deepStrictEqual(plain(rt.resolve(ref, rt.at(from))), want, ref);
});

test('route', () => {
  const cases = [
    ['docs/page.html#second', rt.at('index.html'), '/docs/page.html#second'],
    ['#results', rt.at('search.html') + '?q=x', '/search.html?q=x#results'],
    ['?q=y', rt.at('search.html') + '?q=x', '/search.html?q=y'],
    ['index.html', rt.at('search.html') + '?q=x', '/index.html'],
    ['a b.html', rt.at('index.html'), '/a%20b.html'],
    [SHELL + '#/about/index.html', rt.at('index.html'), '/about/index.html'],
    ['img/dot.png', rt.at('index.html'), null],
    ['https://example.com/', rt.at('index.html'), null],
  ];
  for (const [ref, base, want] of cases) assert.strictEqual(rt.route(ref, base), want, ref);
});

test('rewriteCSS', () => {
  const base = rt.at('css/style.css');
  const dot = rt.blobURL('img/dot.png'), css = rt.blobURL('css/base.css');
  const cases = [
    ['a{background:url(../img/dot.png)}', `a{background:url(${dot})}`],
    ['a{background:url( "../img/dot.png?v=2#x" )}', `a{background:url("${dot}#x")}`],
    ['a{background:URL(\'/img/dot.png\')}', `a{background:url('${dot}')}`],
    ['@import "base.css";', `@import "${css}";`],
    ['@IMPORT url(base.css);', `@IMPORT url(${css});`],
    ['a{background:image-set("../img/dot.png" 1x, \'../img/dot.png\' type("image/png") 2x)}',
      `a{background:image-set("${dot}" 1x, '${dot}' type("image/png") 2x)}`],
    ['a{background:url(data:image/png;base64,AAAA)}', 'a{background:url(data:image/png;base64,AAAA)}'],
    ['a{background:url(missing.png)}', 'a{background:url(missing.png)}'],
    ['a{background:url(#grad)}', 'a{background:url(#grad)}'],
  ];
  for (const [input, want] of cases) assert.strictEqual(rt.rewriteCSS(input, base), want, input);
});

test('rewriteSrcset', () => {
  const base = rt.at('index.html');
  const dot = rt.blobURL('img/dot.png'), comma = rt.blobURL('img/a,b.png');
  const cases = [
    ['img/dot.png', dot],
    ['img/dot.png 1x, img/a,b.png 2x', `${dot} 1x, ${comma} 2x`],
    ['img/a,b.png 100w,img/dot.png 200w', `${comma} 100w, ${dot} 200w`],
    ['img/dot.png, img/a,b.png 2x', `${dot}, ${comma} 2x`],
    [' img/dot.png  1x ,, ', `${dot} 1x`],
    ['data:image/png;base64,AA,BB 1x, missing.png 2x', 'data:image/png;base64,AA,BB 1x, missing.png 2x'],
  ];
  for (const [input, want] of cases) assert.strictEqual(rt.rewriteSrcset(input, base), want, input);
});

test('parseRefresh', () => {
  const cases = [
    ['0; url=about/', { delay: 0, url: 'about/' }],
    ['5', { delay: 5, url: '' }],
    ['0; url=', { delay: 0, url: '' }],
    ['3,URL="a b.html"', { delay: 3, url: 'a b.html' }],
    ["1.5; url='x.html'", { delay: 1, url: 'x.html' }],
    ['.5 x.html', { delay: 0, url: 'x.html' }],
    ['url=x.html', null],
    ['', null],
    [null, null],
  ];
  for (const [content, want] of cases) assert.deepStrictEqual(plain(rt.parseRefresh(content)), want, String(content));
});
