# Changelog

All notable changes to sitecrate are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

First version.

### Added

- `sitecrate <site-dir> [-o out.html] [-entry index.html]` packs a static site
  into a single HTML file that opens from disk or from a web server.
- Gzip-compressed payload embedded in the output, inflated in the browser.
- Every regular file under the site directory is packed. Symlinks to files are
  followed; hidden files and the output file itself are skipped.
- In-browser runtime that renders pages in an iframe and serves assets from
  blob URLs: `src`, `link href`, `srcset`, `poster`, `object data`, and `url()`
  and `@import` in stylesheets, `<style>` elements and `style` attributes.
- Navigation between pages with back/forward, deep links, anchors and page
  titles, addressed as `out.html#/path/page.html?query#anchor`.
- Relative, root-relative, directory (`about/`) and extensionless (`/guide`)
  URLs.
- `<base href>` and `<meta http-equiv="refresh">` redirects.
- Links created by scripts after the page loads.
- `GET` forms that submit to a page of the site, and query strings on pages.
- `fetch()` of files of the site.
- Warnings for features that may not work in the packed output:
  `XMLHttpRequest`, relative ES module imports and `POST` forms.
