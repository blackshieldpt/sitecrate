// url.js: maps references in bundled files to bundled files and shell routes.

const ORIGIN = 'http://sitecrate.invalid';
const shell = location.href.split('#')[0];
const enc = (path) => path.split('/').map(encodeURIComponent).join('/');
const at = (path) => ORIGIN + '/' + enc(path); // base URL for references in a bundled file

// resolve maps a URL reference found in a document at base to a bundled file
function resolve(ref, base) {
  let u, path;
  try {
    u = new URL(ref, base);
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
function route(ref, base) {
  if (ref.startsWith(shell + '#/')) return ref.slice(shell.length + 1);
  const r = resolve(ref, base);
  return r && manifest.files.get(r.path).type === 'text/html' ? '/' + enc(r.path) + r.search + r.hash : null;
}

