# Security Policy

## Reporting a vulnerability

Please report security issues privately to **security@blackshield.pt**. Do not
open public issues for vulnerabilities.

Include the sitecrate version or commit, the browser and how the file was opened
(`file://` or HTTP) if the issue is in a packed file, and a minimal site that
reproduces it.

## Supported versions

Only the latest release receives security fixes.

## Trust model

sitecrate is a packaging tool, not a sandbox. Keep the following in mind when
packing a site or opening a packed file.

**A packed file runs the site's scripts.** Opening it is equivalent to opening
the site. Scripts run with the privileges of the packed file's own address:
when it is served over HTTP they share the origin of the host that serves it,
including that origin's cookies and storage. Only pack sites you trust, and only
host a packed file where you would host the site itself.

**Pages are not isolated from each other.** Every page of the site is rendered
in a same-origin iframe by the embedded runtime.

**External resources are not embedded.** Scripts, stylesheets, fonts and images
loaded from other hosts are still fetched from those hosts when the file is
viewed.

**Subresource Integrity is not preserved.** `integrity` attributes are removed
from elements whose URLs are rewritten. The integrity of the site is that of the
packed file as a whole; distribute it with a checksum or signature if that
matters.

**The packer follows symlinks to files.** A symlink inside the site directory
that points to a file outside it causes that file's contents to be embedded. Do
not pack a directory whose contents you have not reviewed.

**The packer embeds everything it finds.** All non-hidden files under the site
directory end up in the output, readable by anyone who has the file. Check for
secrets, backups and source maps before distributing it.

The packer itself makes no network connections and writes only the output file.
