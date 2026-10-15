// frame.js: shows pages in the shell's iframe and routes their navigation
// through the shell's address.

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

// message replaces the frame with a line of text
function message(title, text) {
  const next = document.createElement('pre');
  next.textContent = text;
  frame.replaceWith(next);
  frame = next;
  current = undefined;
  document.title = title;
}

// render writes page p into the empty frame and routes its link clicks, GET
// form submits and fetches of bundled files through the shell
function render(p, search, hash) {
  const w = frame.contentWindow;
  w.document.open(); // the document takes the shell's URL, so it can be given a query string
  if (search) w.history.replaceState(null, '', search);

  // on adds a listener that runs after the page's own listeners on window,
  // so that the page can cancel the event first
  const on = (type, fn) => w.addEventListener(type, () => w.addEventListener(type, fn, { once: true }), true);
  on('click', function (e) {
    const a = e.target.closest && e.target.closest('a[href], area[href]');
    const target = a && a.getAttribute('target');
    if (!a || e.defaultPrevented || e.button || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey ||
        (target && target !== '_self')) return;
    const to = route(a.getAttribute('href'), p.base);
    if (to) {
      e.preventDefault();
      go(to);
    }
  });
  on('submit', function (e) {
    const form = e.target, button = e.submitter;
    const attr = (name) => button && button.hasAttribute('form' + name) ? button.getAttribute('form' + name) : form.getAttribute(name);
    const to = route(attr('action') || '', p.base);
    if (!to || e.defaultPrevented || (attr('method') || 'get').toLowerCase() !== 'get') return;
    e.preventDefault();
    const m = /^([^?#]*)[^#]*(.*)$/.exec(to);
    const url = m[1] + '?' + new URLSearchParams(new FormData(form, button)) + m[2];
    const target = attr('target');
    if (!target || /^_(self|parent|top)$/i.test(target)) go(url);
    else open(shell + '#' + url, target);
  });
  const fetch = w.fetch;
  w.fetch = function (input, init) {
    const r = typeof input === 'string' && resolve(input, p.base);
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
  const r = m ? resolve(m[1], ORIGIN + '/') : { path: manifest.entry };
  const search = (m && m[2]) || '', hash = (m && m[3]) || '';
  if (r && r.path + search === current) {
    jump(hash);
    return;
  }
  if (!r || manifest.files.get(r.path).type !== 'text/html') {
    message('Not found', 'Not found: ' + location.hash.slice(1));
    return;
  }
  const p = page(r.path, search);
  if (p.redirect && !p.delay) {
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
  if (p.redirect) {
    setTimeout(() => {
      if (frame === next) location.replace(shell + '#' + p.redirect);
    }, p.delay * 1000);
  }
}

