# sitecrate

Packs a complete static site into a single HTML file.

The output is one self-contained `.html` file holding every page, stylesheet,
script, image and font of the site. It opens straight from disk (`file://`) or
from any web server, with no extraction step and no server-side support.
Navigation between pages, back/forward, deep links and site search keep working
inside the file.

## Install

Requires Go 1.23 or later.

```sh
go build -o sitecrate .
```

## Usage

```sh
sitecrate <site-dir> [-o out.html] [-entry index.html]
```

| Flag     | Default      | Meaning                                         |
|----------|--------------|-------------------------------------------------|
| `-o`     | `out.html`   | Output file.                                    |
| `-entry` | `index.html` | Page shown first, relative to `<site-dir>`.     |

Flags may come before or after `<site-dir>`.

```sh
sitecrate ./public -o docs.html
```

`<site-dir>` is treated as the site root, so root-relative URLs such as
`/css/site.css` resolve inside it.

What is packed:

- every regular file under `<site-dir>`;
- symlinks to files are followed; symlinked directories are not;
- files and directories whose name starts with `.` are skipped;
- the output file itself is skipped if it lives inside `<site-dir>`.

Warnings about features that may not work in the packed output go to stderr,
one line per feature. The exit status is 0 on success, 1 on error and 2 on
bad usage.

## Linking into a packed site

Each page has an address inside the file, carried in the URL fragment:

```
out.html#/docs/guide.html
out.html#/docs/guide.html#install
out.html#/search.html?q=term
```

These can be bookmarked, shared and opened in a new tab.

## How it works

**Packing.** The CLI walks the site, concatenates the files, appends a JSON
manifest (path, MIME type, size), gzips the result and embeds it as base64 in a
`<script type="application/octet-stream">` element, next to a small JavaScript
runtime. The site's files are stored unmodified.

**Viewing.** The runtime inflates the payload with `DecompressionStream`, then
renders the requested page in a full-window iframe. All rewriting happens in the
browser, when a page is first shown:

- assets (`src`, `link href`, `srcset`, `poster`, `object data`) become blob URLs;
- `url()` and `@import` in stylesheets, `<style>` elements and `style`
  attributes are rewritten the same way;
- links to other pages become routes of the form shown above.

## What works

- Multi-page navigation, back/forward, deep links, anchors and page titles.
- Relative, root-relative and directory URLs (`about/` resolves to
  `about/index.html`, `/guide` to `guide.html` or `guide/index.html`).
- Query strings and fragments on asset URLs (`style.css?v=3`, `font.woff2#iefix`).
- `<base href>` and `<meta http-equiv="refresh">` redirects.
- Links created by scripts after the page loads.
- `GET` forms that submit to a page of the site.
- Query strings on pages (`location.search`).
- `fetch()` of files of the site, when called with a string URL.
- Search on Sphinx-generated sites.

## Limitations

- `XMLHttpRequest`, web workers and `fetch()` called with a `Request` or `URL`
  object are not redirected to the packed files.
- Relative ES module imports (`import x from "./x.js"`) do not resolve.
- `POST` forms and navigation done from scripts (`location.href = ...`) are not
  handled.
- Pages embedded in another page with `<iframe>` are shown unmodified, so their
  own assets and links do not resolve.
- External SVG references (`<use href="sprite.svg#icon">`) are not rewritten.
- HTML and CSS are read as UTF-8.
- Inside a page, `location` is the address of the packed file, not of the
  original page.
- Resources loaded from other hosts (CDNs, web fonts, analytics) are not
  embedded. They are still fetched at view time and need a network connection.
- `integrity` attributes are removed from rewritten elements.
- The whole site is held in memory while the file is open.

The viewer needs a browser with `DecompressionStream`, which all major engines
have shipped since 2023.

## Development

```sh
go test ./...
```

- `main.go`: command line.
- `pack/pack.go`: walking the site and writing the output file.
- `pack/runtime.js`: the in-browser runtime, embedded with `go:embed`.
- `pack/testdata/site`: fixture site used by the tests.

## Security

See [SECURITY.md](SECURITY.md) for how to report a vulnerability and for the
trust model of packed files.

## License

Apache License 2.0. See [LICENSE](LICENSE).
