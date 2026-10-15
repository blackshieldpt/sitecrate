const query = new URLSearchParams(location.search).get('q') || '';
document.getElementById('query').textContent = query;
fetch('search-index.json').then((r) => r.json()).then((index) => {
  for (const entry of index) {
    if (!query || !entry.text.includes(query.toLowerCase())) continue;
    const a = document.createElement('a');
    a.href = entry.url;
    a.textContent = entry.title;
    document.getElementById('results').appendChild(document.createElement('li')).appendChild(a);
  }
});
