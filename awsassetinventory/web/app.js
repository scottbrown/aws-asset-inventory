(() => {
  'use strict';

  const PALETTE = [
    '#ff9f1c', '#4cc9f0', '#f72585', '#80ed99', '#b388ff', '#ffd166', '#06d6a0', '#ef476f',
    '#118ab2', '#f4a261', '#90be6d', '#e76f51', '#c77dff', '#43aa8b', '#f9c74f', '#577590',
  ];
  const OTHER_COLOR = '#9aa3b5';
  const STUB_COLOR = '#3c4354';
  const DIM_COLOR = '#333a4a';
  const LINK_COLOR = 'rgba(160, 175, 210, 0.35)';
  const LINK_HIGHLIGHT = '#ffb547';

  const HELP = {
    fly: 'W/S forward and back · A/D left and right · R/F up and down · Q/E roll · arrow keys turn · drag to look. Click a resource to fly to it.',
    orbit: 'Drag to rotate · right-drag to pan · scroll to zoom. Click a resource to fly to it.',
    trackball: 'Drag to rotate · right-drag to pan · scroll to zoom. Click a resource to fly to it.',
  };

  const $ = (id) => document.getElementById(id);
  const el = (tag, props = {}, children = []) => {
    const e = Object.assign(document.createElement(tag), props);
    for (const c of [].concat(children)) e.append(c);
    return e;
  };

  const params = new URLSearchParams(location.search);
  const controlType = HELP[params.get('controls')] ? params.get('controls') : 'fly';

  const state = {
    nodes: [],
    links: [],             // { source, target, label } with string IDs
    byId: new Map(),
    neighbours: new Map(), // node ID -> Set of node IDs
    colors: new Map(),     // service -> colour
    regions: new Set(),
    services: new Set(),
    showStubs: true,
    showIsolated: true,
    selected: null,
  };

  const graph = new ForceGraph3D($('graph'), { controlType })
    .backgroundColor('#05070d')
    .showNavInfo(false)
    .nodeId('id')
    .nodeLabel(nodeLabel)
    .nodeColor(nodeColor)
    .nodeVal((n) => (n.stub ? 0.5 : 1 + Math.sqrt(n.degree)))
    .nodeOpacity(0.95)
    .nodeResolution(10)
    .linkLabel((l) => escapeHTML(l.label || ''))
    .linkColor((l) => (isHighlighted(l) ? LINK_HIGHLIGHT : LINK_COLOR))
    .linkWidth((l) => (isHighlighted(l) ? 0.6 : 0))
    .linkOpacity(0.6)
    .onNodeClick((n) => select(n))
    .onBackgroundClick(() => select(null))
    .warmupTicks(60)
    .cooldownTime(8000);

  if (controlType === 'fly') {
    graph.controls().movementSpeed = 250;
  }

  window.addEventListener('resize', () => graph.width(innerWidth).height(innerHeight));

  // ---------- data ----------

  async function load() {
    const [summary, data] = await Promise.all([
      fetch('api/summary').then(checkResponse),
      fetch('api/graph').then(checkResponse),
    ]);

    state.nodes = data.nodes;
    state.links = data.links;
    for (const n of state.nodes) {
      state.byId.set(n.id, n);
      state.neighbours.set(n.id, new Set());
    }
    for (const l of state.links) {
      state.neighbours.get(l.source).add(l.target);
      state.neighbours.get(l.target).add(l.source);
    }

    assignColors();
    renderSummary(summary);
    renderFilters();
    renderSearchOptions();
    applyFilters();
  }

  function checkResponse(res) {
    if (!res.ok) throw new Error(`${res.url}: ${res.status} ${res.statusText}`);
    return res.json();
  }

  function assignColors() {
    const counts = countBy(state.nodes.filter((n) => !n.stub), (n) => n.service);
    [...counts.entries()]
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .forEach(([service], i) => state.colors.set(service, PALETTE[i] || OTHER_COLOR));
  }

  function countBy(items, key) {
    const counts = new Map();
    for (const item of items) counts.set(key(item), (counts.get(key(item)) || 0) + 1);
    return counts;
  }

  // ---------- filters ----------

  function renderSummary(summary) {
    const when = new Date(summary.collectedAt).toLocaleString();
    const profile = summary.profile ? ` · profile ${summary.profile}` : '';
    $('summary').textContent =
      `${summary.resources.toLocaleString()} resources · ${summary.links.toLocaleString()} relationships · collected ${when}${profile}`;

    if (summary.links === 0) {
      showNotice('This inventory has no relationship data. Collect it again with this version of aws-asset-inventory to see how resources connect.');
    }
  }

  function showNotice(text) {
    $('notice').textContent = text;
    $('notice').hidden = false;
  }

  function renderFilters() {
    const real = state.nodes.filter((n) => !n.stub);
    renderChecklist('regions', countBy(state.nodes, (n) => n.region || 'global'), state.regions, null);
    renderChecklist('services', countBy(real, (n) => n.service), state.services, (s) => state.colors.get(s));
  }

  function renderChecklist(containerId, counts, selected, colorOf) {
    const container = $(containerId);
    container.replaceChildren();
    const entries = [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
    for (const [value, count] of entries) {
      selected.add(value);
      const box = el('input', { type: 'checkbox', checked: true, value });
      box.addEventListener('change', () => {
        box.checked ? selected.add(value) : selected.delete(value);
        applyFilters();
      });
      const children = [box];
      if (colorOf) {
        const sw = el('span', { className: 'swatch' });
        sw.style.background = colorOf(value);
        children.push(sw);
      }
      children.push(value, el('span', { className: 'count', textContent: count.toLocaleString() }));
      container.append(el('label', { className: 'check' }, children));
    }
  }

  document.querySelectorAll('[data-all],[data-none]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const id = btn.dataset.all || btn.dataset.none;
      const on = Boolean(btn.dataset.all);
      const set = id === 'regions' ? state.regions : state.services;
      $(id).querySelectorAll('input').forEach((box) => {
        box.checked = on;
        on ? set.add(box.value) : set.delete(box.value);
      });
      applyFilters();
    });
  });

  $('show-stubs').addEventListener('change', (e) => { state.showStubs = e.target.checked; applyFilters(); });
  $('show-isolated').addEventListener('change', (e) => { state.showIsolated = e.target.checked; applyFilters(); });

  function nodeVisible(n) {
    if (n.stub && !state.showStubs) return false;
    if (!n.stub && !state.services.has(n.service)) return false;
    if (!state.regions.has(n.region || 'global')) return false;
    return true;
  }

  function applyFilters() {
    let nodes = state.nodes.filter(nodeVisible);
    const visible = new Set(nodes.map((n) => n.id));
    const links = state.links.filter((l) => visible.has(l.source) && visible.has(l.target));

    if (!state.showIsolated) {
      const linked = new Set();
      for (const l of links) { linked.add(l.source); linked.add(l.target); }
      nodes = nodes.filter((n) => linked.has(n.id));
    }

    if (state.selected && !nodes.includes(state.selected)) select(null);

    // Pass copies of the links: the graph replaces source and target with node objects.
    graph.graphData({ nodes, links: links.map((l) => ({ ...l })) });
  }

  // ---------- search ----------

  function displayName(n) {
    return n.name && n.name !== n.resourceId ? `${n.name} (${n.resourceId})` : n.resourceId;
  }

  function searchKey(n) {
    return `${displayName(n)} — ${n.type}${n.region ? ' — ' + n.region : ''}`;
  }

  function renderSearchOptions() {
    const list = $('search-options');
    const items = state.nodes.filter((n) => !n.stub).slice(0, 5000);
    list.replaceChildren(...items.map((n) => el('option', { value: searchKey(n) })));
  }

  const search = $('search');
  // Stop the fly controls from moving the camera while the user types.
  search.addEventListener('keydown', (e) => e.stopPropagation());
  search.addEventListener('keyup', (e) => e.stopPropagation());
  search.addEventListener('change', () => {
    const q = search.value.trim().toLowerCase();
    if (!q) return;
    const n = state.nodes.find((x) => searchKey(x).toLowerCase() === q) ||
      state.nodes.find((x) => searchKey(x).toLowerCase().includes(q));
    if (!n) return;
    if (!graph.graphData().nodes.includes(n)) {
      showNotice(`${displayName(n)} is hidden by the current filters.`);
      return;
    }
    select(n);
  });

  // ---------- camera ----------

  const controlsSelect = $('controls');
  controlsSelect.value = controlType;
  $('controls-help').textContent = HELP[controlType];
  controlsSelect.addEventListener('change', () => {
    params.set('controls', controlsSelect.value);
    location.search = params.toString();
  });

  const FLY_MS = 1200;
  let flight = 0;

  function flyTo(n) {
    const distance = 90;
    const len = Math.hypot(n.x, n.y, n.z) || 1;
    const ratio = 1 + distance / len;
    const to = { x: n.x * ratio, y: n.y * ratio, z: n.z * ratio };

    if (controlType !== 'fly') {
      graph.cameraPosition(to, n, FLY_MS);
      return;
    }

    // Fly controls have no target, so the built-in transition stops turning the
    // camera part of the way through. Move and turn the camera ourselves.
    const camera = graph.camera();
    const from = camera.position.clone();
    const fromQ = camera.quaternion.clone();
    const start = performance.now();
    cancelAnimationFrame(flight);

    const step = (now) => {
      const t = Math.min(1, (now - start) / FLY_MS);
      const ease = 1 - (1 - t) * (1 - t);
      camera.position.set(
        from.x + (to.x - from.x) * ease,
        from.y + (to.y - from.y) * ease,
        from.z + (to.z - from.z) * ease,
      );
      camera.lookAt(n.x, n.y, n.z);
      const toQ = camera.quaternion.clone();
      camera.quaternion.copy(fromQ).slerp(toQ, Math.min(1, ease * 2));
      if (t < 1) flight = requestAnimationFrame(step);
    };
    flight = requestAnimationFrame(step);
  }

  // ---------- selection ----------

  function isHighlighted(l) {
    if (!state.selected) return false;
    const s = typeof l.source === 'object' ? l.source.id : l.source;
    const t = typeof l.target === 'object' ? l.target.id : l.target;
    return s === state.selected.id || t === state.selected.id;
  }

  function nodeColor(n) {
    if (state.selected) {
      const near = n === state.selected || state.neighbours.get(state.selected.id).has(n.id);
      if (!near) return DIM_COLOR;
      if (n === state.selected) return '#ffffff';
    }
    return n.stub ? STUB_COLOR : state.colors.get(n.service) || OTHER_COLOR;
  }

  function nodeLabel(n) {
    const lines = [
      `<b>${escapeHTML(displayName(n))}</b>`,
      escapeHTML(n.type),
      escapeHTML(n.region || 'global'),
    ];
    if (n.stub) lines.push('<i>Not in inventory</i>');
    return lines.join('<br>');
  }

  function refreshStyles() {
    graph.nodeColor(graph.nodeColor()).linkColor(graph.linkColor()).linkWidth(graph.linkWidth());
  }

  function select(n) {
    state.selected = n;
    refreshStyles();
    if (!n) {
      $('details').hidden = true;
      return;
    }
    flyTo(n);
    showDetails(n);
  }

  async function showDetails(n) {
    $('details').hidden = false;
    $('details-title').textContent = displayName(n);

    const fields = [
      ['Type', n.type],
      ['ID', n.resourceId],
      ['Name', n.name],
      ['Region', n.region],
      ['Account', n.accountId],
      ['Links', String(n.degree)],
    ];
    renderFields(fields);
    $('details-tags').replaceChildren();
    $('details-config').textContent = '';
    $('details-config-wrap').hidden = true;
    renderRelationships(n);

    if (n.stub) {
      renderFields([...fields, ['Status', 'Referenced by another resource but not in the inventory']]);
      return;
    }

    try {
      const res = await fetch('api/resource?id=' + encodeURIComponent(n.id)).then(checkResponse);
      if (state.selected !== n) return;
      renderFields([...fields, ['ARN', res.arn], ['Zone', res.availabilityZone]]);
      renderTags(res.tags);
      if (res.configuration) {
        $('details-config').textContent = JSON.stringify(res.configuration, null, 2);
        $('details-config-wrap').hidden = false;
      }
    } catch (err) {
      renderFields([...fields, ['Error', err.message]]);
    }
  }

  function renderFields(fields) {
    const dl = $('details-fields');
    dl.replaceChildren();
    for (const [k, v] of fields) {
      if (!v) continue;
      dl.append(el('dt', { textContent: k }), el('dd', { textContent: v }));
    }
  }

  function renderTags(tags) {
    const container = $('details-tags');
    const entries = Object.entries(tags || {}).sort((a, b) => a[0].localeCompare(b[0]));
    if (!entries.length) return;
    const rows = entries.map(([k, v]) => el('tr', {}, [el('td', { textContent: k }), el('td', { textContent: v })]));
    container.replaceChildren(el('h3', { textContent: 'Tags' }), el('table', {}, rows));
  }

  function renderRelationships(n) {
    const container = $('details-relationships');
    container.replaceChildren();
    const related = state.links
      .filter((l) => l.source === n.id || l.target === n.id)
      .map((l) => ({ label: l.label, other: state.byId.get(l.source === n.id ? l.target : l.source) }))
      .sort((a, b) => (a.label || '').localeCompare(b.label || ''));
    if (!related.length) return;

    const visible = new Set(graph.graphData().nodes);
    const items = related.map(({ label, other }) => {
      const btn = el('button', { type: 'button', textContent: displayName(other), disabled: !visible.has(other) });
      btn.title = visible.has(other) ? other.type : `${other.type} (hidden by filters)`;
      btn.addEventListener('click', () => select(other));
      return el('li', {}, [
        el('div', { className: 'muted small', textContent: `${label || 'Related to'} · ${other.type}` }),
        btn,
      ]);
    });
    container.append(el('h3', { textContent: `Relationships (${related.length})` }), el('ul', { className: 'rels' }, items));
  }

  $('details-close').addEventListener('click', () => select(null));

  function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  }

  load().catch((err) => {
    $('summary').textContent = 'Could not load the inventory.';
    showNotice(err.message);
  });
})();
