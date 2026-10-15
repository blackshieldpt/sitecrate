// rewrite.js: rewrites bundled HTML and CSS so that their references point to
// blob URLs and shell routes.

const urls = new Map(); // path -> blob URL
const pages = new Map(); // path and query -> { title, base, redirect, delay, html }

function blobURL(path) {
  let url = urls.get(path);
  if (url === undefined) {
    urls.set(path, ''); // breaks @import cycles
    const f = manifest.files.get(path);
    let data = f.data, type = f.type;
    if (type === 'text/css') {
      data = rewriteCSS(new TextDecoder().decode(data), at(path));
      type += ';charset=utf-8';
    }
    url = URL.createObjectURL(new Blob([data], { type }));
    urls.set(path, url);
  }
  return url;
}

// asset returns the blob URL for ref, or ref itself if it is not bundled
function asset(ref, base) {
  const r = /^\s*(#|$)/.test(ref) ? null : resolve(ref, base);
  return r ? blobURL(r.path) + r.hash : ref;
}

function rewriteCSS(css, base) {
  return css
    .replace(/url\(\s*(['"]?)(.*?)\1\s*\)/gi, (m, q, ref) => 'url(' + q + asset(ref, base) + q + ')')
    .replace(/@import\s+(['"])(.*?)\1/gi, (m, q, ref) => '@import ' + q + asset(ref, base) + q)
    .replace(/image-set\(((?:[^()]|\([^()]*\))*)\)/gi, (m, args) => 'image-set(' +
      args.replace(/(^|,)(\s*)(['"])(.*?)\3/g, (m, sep, space, q, ref) => sep + space + q + asset(ref, base) + q) + ')');
}

// rewriteSrcset parses srcset as the HTML spec does: a URL may contain
// commas, and candidates are separated by a comma after the URL or after
// its descriptors
function rewriteSrcset(srcset, base) {
  const out = [];
  const re = /[\s,]*([^\s,]\S*)/g;
  let m;
  while ((m = re.exec(srcset))) {
    let ref = m[1], descriptors = '';
    if (ref.endsWith(',')) {
      ref = ref.replace(/,+$/, '');
    } else {
      const d = /^(?:[^,(]|\([^)]*\))*/.exec(srcset.slice(re.lastIndex))[0];
      re.lastIndex += d.length;
      descriptors = d.trim();
    }
    out.push(asset(ref, base) + (descriptors ? ' ' + descriptors : ''));
  }
  return out.join(', ');
}

// parseRefresh parses the content of a <meta http-equiv="refresh"> into its
// delay in seconds and its URL ('' to reload the page), or returns null
function parseRefresh(content) {
  const m = /^\s*(\d*)(\.[\d.]*)?[\s;,]*(?:url\s*=\s*)?(['"]?)(.*?)\3\s*$/i.exec(content);
  return m && (m[1] || m[2]) ? { delay: Number(m[1]), url: m[4] } : null;
}

// page returns the rewritten HTML document at path, opened with query search
function page(path, search) {
  let p = pages.get(path + search);
  if (!p) pages.set(path + search, p = rewriteHTML(new TextDecoder().decode(manifest.files.get(path).data), path, search));
  return p;
}

function rewriteHTML(html, path, search) {
  const doc = new DOMParser().parseFromString(html, 'text/html');
  const page = { title: doc.title };

  let base = at(path) + search;
  const b = doc.querySelector('base[href]');
  if (b) {
    let u;
    try {
      u = new URL(b.getAttribute('href'), base);
    } catch (e) {
      // an invalid base is ignored, as browsers do
    }
    if (u && u.origin === ORIGIN) {
      base = u.href;
      b.remove();
    }
  }
  page.base = base;

  const refresh = doc.querySelector('meta[http-equiv=refresh i]');
  const r = refresh && parseRefresh(refresh.getAttribute('content'));
  if (r) {
    const self = '/' + enc(path) + search;
    const to = r.url ? route(r.url, base) : self;
    if (to) {
      // reloads are dropped: reloading the frame would load the shell into it
      if (to.split('#')[0] !== self) {
        page.redirect = to;
        page.delay = r.delay;
      }
      refresh.remove();
    }
  }

  const attrs = [
    ['[src]', 'src'],
    ['link[href]', 'href'],
    ['[poster]', 'poster'],
    ['object[data]', 'data'],
  ];
  for (const [selector, attr] of attrs) {
    for (const el of doc.querySelectorAll(selector)) {
      el.setAttribute(attr, asset(el.getAttribute(attr), base));
      el.removeAttribute('integrity');
    }
  }
  for (const el of doc.querySelectorAll('[srcset]')) el.setAttribute('srcset', rewriteSrcset(el.getAttribute('srcset'), base));
  for (const el of doc.querySelectorAll('style')) el.textContent = rewriteCSS(el.textContent, base);
  for (const el of doc.querySelectorAll('[style]')) el.setAttribute('style', rewriteCSS(el.getAttribute('style'), base));

  for (const a of doc.querySelectorAll('a[href], area[href]')) {
    const href = a.getAttribute('href');
    const to = route(href, base);
    if (to) {
      a.setAttribute('href', shell + '#' + to);
    } else if (/^(https?:)?\/\//i.test(href)) {
      if (!a.hasAttribute('target')) a.setAttribute('target', '_top');
    } else {
      a.setAttribute('href', asset(href, base));
    }
  }

  page.html = (doc.doctype ? new XMLSerializer().serializeToString(doc.doctype) : '') + doc.documentElement.outerHTML;
  return page;
}

