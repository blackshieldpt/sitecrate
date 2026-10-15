// main.js: unpacks the site and shows the page named by the shell's address.

unpack().then(() => {
  addEventListener('hashchange', show);
  show();
}, (e) => message('Cannot open', 'Cannot open this file: ' + e));
