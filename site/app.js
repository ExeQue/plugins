// Renders the catalog from index.json, the same file lerd installs from.
const grid = document.getElementById('grid');
const count = document.getElementById('count');
const q = document.getElementById('q');
const cats = document.getElementById('categories');
const dialog = document.getElementById('detail');
const body = document.getElementById('d-body');
const sortSel = document.getElementById('sort');

// Each order sorts newest first on its own date, falling back to the name so
// listings approved the same day stay in a stable order.
const SORTS = {
  updated: (a, b) => (b.updated_at ?? '').localeCompare(a.updated_at ?? '') || a.name.localeCompare(b.name),
  stars: (a, b) => (b.stars ?? 0) - (a.stars ?? 0) || a.name.localeCompare(b.name),
  added: (a, b) => (b.added_at ?? '').localeCompare(a.added_at ?? '') || a.name.localeCompare(b.name),
  name: (a, b) => a.name.localeCompare(b.name)
};
try {
  const saved = localStorage.getItem('lerd-plugins-sort');
  if (saved in SORTS) sortSel.value = saved;
} catch {
  // Storage can be blocked; the default order still works.
}
const fmtDate = (iso) => (iso ? new Date(iso + 'T00:00:00').toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }) : '');

const CUBE = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"><path d="M12 3 4 7.5v9L12 21l8-4.5v-9L12 3Z"/><path d="M4 7.5 12 12l8-4.5M12 12v9"/></svg>';

const STAR = '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2.5l2.9 6.1 6.6.8-4.9 4.6 1.3 6.6L12 17.3l-5.9 3.3 1.3-6.6-4.9-4.6 6.6-.8z"/></svg>';

let plugins = [];
let category = '';

const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
const short = (sha) => sha.slice(0, 7);
const commitURL = (p) => (p.repo.startsWith('https://github.com/') ? `${p.repo}/tree/${p.commit}` : p.repo);

function preview(p) {
  return p.images?.length
    ? `<img class="preview" src="${esc(p.images[0])}" alt="" loading="lazy">`
    : `<div class="preview placeholder">${CUBE}</div>`;
}

function matches(p, term) {
  if (category && p.category !== category) return false;
  if (!term) return true;
  return [p.name, p.description, p.author, ...(p.tags ?? [])].join(' ').toLowerCase().includes(term);
}

function render() {
  const term = q.value.trim().toLowerCase();
  const shown = plugins.filter((p) => matches(p, term)).sort(SORTS[sortSel.value] ?? SORTS.updated);
  const dateLabel = sortSel.value === 'added' ? 'Added' : 'Updated';
  const dateOf = (p) => (sortSel.value === 'added' ? p.added_at : p.updated_at);
  count.textContent = `${shown.length} of ${plugins.length} plugins`;
  grid.innerHTML = shown.length
    ? shown
        .map(
          (p) => `
      <button class="card" data-name="${esc(p.name)}">
        ${preview(p)}
        <div class="card-body">
          <div class="card-title"><strong>${esc(p.name)}</strong><span class="muted">${esc(p.author)}</span></div>
          <p>${esc(p.description)}</p>
          <div class="tags">${(p.tags ?? []).map((t) => `<span class="tag">${esc(t)}</span>`).join('')}</div>
          <div class="foot">
            ${dateOf(p) ? `<span class="date">${dateLabel} ${esc(fmtDate(dateOf(p)))}</span>` : '<span></span>'}
            ${p.stars != null ? `<span class="stars" title="GitHub stars">${STAR} ${p.stars.toLocaleString()}</span>` : ''}
          </div>
        </div>
      </button>`
        )
        .join('')
    : plugins.length
      ? '<p class="empty">No plugin matches that.</p>'
      : '<p class="empty">No plugins are listed yet. <a href="https://github.com/lerd-env/plugins/blob/main/SUBMITTING.md">Submit the first one</a>.</p>';
}

function renderCategories() {
  const all = [...new Set(plugins.map((p) => p.category).filter(Boolean))].sort();
  cats.innerHTML = ['', ...all]
    .map((c) => `<button class="chip" data-cat="${esc(c)}" aria-pressed="${c === category}">${c ? esc(c) : 'All'}</button>`)
    .join('');
}

function open(name) {
  const p = plugins.find((x) => x.name === name);
  if (!p) return;
  const cmd = `lerd plugin install ${p.name}`;
  body.innerHTML = `
    <h2 id="d-name">${esc(p.name)}</h2>
    <div class="meta">
      <span>by ${esc(p.author)}</span>
      ${p.updated_at ? `<span>updated ${esc(fmtDate(p.updated_at))}</span>` : ''}
      ${p.category ? `<span>${esc(p.category)}</span>` : ''}
      <span>approved at <a href="${esc(commitURL(p))}"><code>${short(p.commit)}</code></a></span>
    </div>
    <p>${esc(p.description)}</p>
    ${p.requires?.length ? `<div class="reqs"><span>Requires</span><div class="tags">${p.requires.map((r) => `<span class="tag">${esc(r)}</span>`).join('')}</div></div>` : ''}
    ${p.images?.length ? `<div class="shots">${p.images.map((i) => `<img src="${esc(i)}" alt="" loading="lazy">`).join('')}</div>` : ''}
    <div class="install">
      <div class="cmd"><span>${esc(cmd)}</span><button class="btn" data-copy="${esc(cmd)}">Copy</button></div>
      <a class="btn primary" href="http://lerd.localhost/#system/plugins/browse/${encodeURIComponent(p.name)}">Install in Lerd</a>
    </div>
    <p class="note">Lerd installs commit <code>${short(p.commit)}</code> and nothing newer, even if the repository has moved on.</p>`;
  dialog.showModal();
  history.replaceState(null, '', '#' + p.name);
}

grid.addEventListener('click', (e) => {
  const card = e.target.closest('.card');
  if (card) open(card.dataset.name);
});
cats.addEventListener('click', (e) => {
  const chip = e.target.closest('.chip');
  if (!chip) return;
  category = chip.dataset.cat;
  renderCategories();
  render();
});
body.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-copy]');
  if (!btn) return;
  await navigator.clipboard.writeText(btn.dataset.copy);
  btn.textContent = 'Copied';
  setTimeout(() => (btn.textContent = 'Copy'), 1500);
});
dialog.addEventListener('close', () => history.replaceState(null, '', location.pathname));
q.addEventListener('input', render);
sortSel.addEventListener('change', () => {
  try {
    localStorage.setItem('lerd-plugins-sort', sortSel.value);
  } catch {
    // Not remembered, but still applied.
  }
  render();
});

fetch('index.json')
  .then((r) => r.json())
  .then((c) => {
    plugins = c.plugins ?? [];
    renderCategories();
    render();
    if (location.hash) open(decodeURIComponent(location.hash.slice(1)));
  })
  .catch(() => (grid.innerHTML = '<p class="empty">The catalog could not be loaded.</p>'));
