// sitecrate runtime: unpacks the embedded site and renders its pages in an iframe,
// with assets served from blob URLs.
(async function () {
  'use strict';

  const ORIGIN = 'http://sitecrate.invalid';
  const shell = location.href.split('#')[0];
  const enc = (path) => path.split('/').map(encodeURIComponent).join('/');

  const manifest = { files: new Map() }; // path -> { type, data }
  const urls = new Map(); // path -> blob URL
  const pages = new Map(); // path -> { title, redirect, from, html }

  // payload: gzip(file bytes... | manifest JSON | uint32 manifest length)
  async function unpack() {
    const bin = atob(document.getElementById('sitecrate-data').textContent);
    const gz = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) gz[i] = bin.charCodeAt(i);
    const buf = new Uint8Array(await new Response(
      new Blob([gz]).stream().pipeThrough(new DecompressionStream('gzip'))).arrayBuffer());
    const end = buf.length - 4;
    const len = new DataView(buf.buffer).getUint32(end);
    const m = JSON.parse(new TextDecoder().decode(buf.subarray(end - len, end)));
    manifest.entry = m.entry;
    let off = 0;
    for (const f of m.files) {
      manifest.files.set(f.path, { type: f.type, data: buf.subarray(off, off + f.size) });
      off += f.size;
    }
  }

  // resolve maps a URL reference found in the file at `from` to a bundled file
  function resolve(ref, from) {
    let u, path;
    try {
      u = new URL(ref, ORIGIN + '/' + enc(from));
      path = decodeURIComponent(u.pathname.slice(1));
    } catch (e) {
      return null;
    }
    if (u.origin !== ORIGIN) return null;
    const tries = path === '' || path.endsWith('/')
      ? [path + 'index.html']
      : [path, path + '.html', path + '/index.html'];
    path = tries.find((p) => manifest.files.has(p));
    return path === undefined ? null : { path, search: u.search, hash: u.hash };
  }

  // route returns the shell route (/path?query#anchor) for a reference to a
  // bundled page, or null
  function route(ref, from) {
    if (ref.startsWith(shell + '#/')) return ref.slice(shell.length + 1);
    const r = resolve(ref, from);
    return r && manifest.files.get(r.path).type === 'text/html' ? '/' + enc(r.path) + r.search + r.hash : null;
  }

  function blobURL(path) {
    let url = urls.get(path);
    if (url === undefined) {
      urls.set(path, ''); // breaks @import cycles
      const f = manifest.files.get(path);
      let data = f.data, type = f.type;
      if (type === 'text/css') {
        data = rewriteCSS(new TextDecoder().decode(data), path);
        type += ';charset=utf-8';
      }
      url = URL.createObjectURL(new Blob([data], { type }));
      urls.set(path, url);
    }
    return url;
  }

  // asset returns the blob URL for ref, or ref itself if it is not bundled
  function asset(ref, from) {
    const r = /^\s*(#|$)/.test(ref) ? null : resolve(ref, from);
    return r ? blobURL(r.path) + r.hash : ref;
  }

  function rewriteCSS(css, from) {
    return css
      .replace(/url\(\s*(['"]?)(.*?)\1\s*\)/g, (m, q, ref) => 'url(' + q + asset(ref, from) + q + ')')
      .replace(/@import\s+(['"])(.*?)\1/g, (m, q, ref) => '@import ' + q + asset(ref, from) + q);
  }

  // page returns the rewritten HTML document at path
  function page(path) {
    let p = pages.get(path);
    if (!p) pages.set(path, p = rewriteHTML(new TextDecoder().decode(manifest.files.get(path).data), path));
    return p;
  }

  function rewriteHTML(html, path) {
    const doc = new DOMParser().parseFromString(html, 'text/html');
    const page = { title: doc.title };

    let from = path;
    const base = doc.querySelector('base[href]');
    if (base) {
      const u = new URL(base.getAttribute('href'), ORIGIN + '/' + enc(path));
      if (u.origin === ORIGIN) {
        from = decodeURIComponent(u.pathname.slice(1));
        base.remove();
      }
    }
    page.from = from;

    const refresh = doc.querySelector('meta[http-equiv=refresh i]');
    const target = refresh && /^\s*\d+\s*[;,]\s*(?:url\s*=\s*)?['"]?(.*?)['"]?\s*$/i.exec(refresh.getAttribute('content'));
    const to = target && route(target[1], from);
    if (to) {
      page.redirect = to;
      refresh.remove();
    }

    const attrs = [
      ['[src]', 'src'],
      ['link[href]', 'href'],
      ['[poster]', 'poster'],
      ['object[data]', 'data'],
    ];
    for (const [selector, attr] of attrs) {
      for (const el of doc.querySelectorAll(selector)) {
        el.setAttribute(attr, asset(el.getAttribute(attr), from));
        el.removeAttribute('integrity');
      }
    }
    for (const el of doc.querySelectorAll('[srcset]')) {
      el.setAttribute('srcset', el.getAttribute('srcset').split(',').map((candidate) => {
        const [ref, ...descriptors] = candidate.trim().split(/\s+/);
        return [asset(ref, from), ...descriptors].join(' ');
      }).join(', '));
    }
    for (const el of doc.querySelectorAll('style')) el.textContent = rewriteCSS(el.textContent, from);
    for (const el of doc.querySelectorAll('[style]')) el.setAttribute('style', rewriteCSS(el.getAttribute('style'), from));

    for (const a of doc.querySelectorAll('a[href], area[href]')) {
      const href = a.getAttribute('href');
      const to = route(href, from);
      if (to) {
        a.setAttribute('href', shell + '#' + to);
      } else if (/^(https?:)?\/\//i.test(href)) {
        if (!a.hasAttribute('target')) a.setAttribute('target', '_top');
      } else {
        a.setAttribute('href', asset(href, from));
      }
    }

    page.html = (doc.doctype ? new XMLSerializer().serializeToString(doc.doctype) : '') + doc.documentElement.outerHTML;
    return page;
  }

  let frame = document.body.appendChild(document.createElement('iframe'));
  let current; // path and query of the page in frame

  // go navigates the shell to a route
  function go(to) {
    if (location.hash === '#' + to) show();
    else location.hash = to;
  }

  // jump scrolls the current page to hash ('#anchor', or '' for the top)
  function jump(hash) {
    const loc = frame.contentWindow.location;
    loc.replace(loc.href.split('#')[0] + (hash || '#'));
  }

  // render writes page p into the empty frame and routes its link clicks, GET
  // form submits and fetches of bundled files through the shell
  function render(p, search, hash) {
    const w = frame.contentWindow;
    w.document.open(); // the document takes the shell's URL, so it can be given a query string
    if (search) w.history.replaceState(null, '', search);

    w.addEventListener('click', function (e) {
      const a = e.target.closest('a[href], area[href]');
      const target = a && a.getAttribute('target');
      if (!a || e.defaultPrevented || e.button || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey ||
          (target && target !== '_self')) return;
      const to = route(a.getAttribute('href'), p.from);
      if (to) {
        e.preventDefault();
        go(to);
      }
    });
    w.addEventListener('submit', function (e) {
      const form = e.target;
      const to = route(form.getAttribute('action') || '', p.from);
      if (!to || e.defaultPrevented || (form.getAttribute('method') || 'get').toLowerCase() !== 'get') return;
      e.preventDefault();
      const m = /^([^?#]*)[^#]*(.*)$/.exec(to);
      go(m[1] + '?' + new URLSearchParams(new FormData(form)) + m[2]);
    });
    const fetch = w.fetch;
    w.fetch = function (input, init) {
      const r = typeof input === 'string' && resolve(input, p.from);
      if (!r) return fetch.call(w, input, init);
      const f = manifest.files.get(r.path);
      return Promise.resolve(new w.Response(f.data, { headers: { 'Content-Type': f.type } }));
    };
    if (hash) w.addEventListener('load', () => jump(hash));

    w.document.write(p.html);
    w.document.close();
  }

  // show renders the page named by the shell's hash: #/path/page.html?query#anchor
  function show() {
    const m = /^#(\/[^?#]*)(\?[^#]*)?(#.*)?$/.exec(location.hash);
    const r = m ? resolve(m[1], '') : { path: manifest.entry };
    const search = (m && m[2]) || '', hash = (m && m[3]) || '';
    if (r && r.path + search === current) {
      jump(hash);
      return;
    }
    if (!r || manifest.files.get(r.path).type !== 'text/html') {
      const next = document.createElement('pre');
      next.textContent = 'Not found: ' + location.hash.slice(1);
      frame.replaceWith(next);
      frame = next;
      current = undefined;
      return;
    }
    const p = page(r.path);
    if (p.redirect) {
      location.replace(shell + '#' + p.redirect);
      return;
    }
    const next = document.createElement('iframe');
    next.allowFullscreen = true;
    frame.replaceWith(next);
    frame = next;
    current = r.path + search;
    document.title = p.title;
    render(p, search, hash);
  }

  await unpack();
  addEventListener('hashchange', show);
  show();
})();
