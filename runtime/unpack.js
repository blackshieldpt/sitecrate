// unpack.js: the site embedded in the shell.

const manifest = { files: new Map() }; // path -> { type, data }

// unpack inflates the payload into manifest.
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

