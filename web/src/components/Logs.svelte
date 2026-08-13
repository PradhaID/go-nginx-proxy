<script>
  import { api } from '../lib/api.js';

  let type = $state('access'); // access | error
  let domain = $state(''); // '' = all domains
  let domains = $state([]); // [{domain, accessLog, errorLog}]
  let lines = $state([]); // [{text, domain, kind}]
  let filter = $state('');
  let showKind = $state('all'); // all | human | bot | other
  let follow = $state(true);
  let live = $state(false);
  let es = null;

  $effect(() => {
    api
      .logDomains()
      .then((d) => (domains = d.domains || []))
      .catch(() => {});
  });

  let counts = $derived.by(() => {
    const c = { human: 0, bot: 0, other: 0 };
    for (const l of lines) if (l.kind) c[l.kind] = (c[l.kind] || 0) + 1;
    return c;
  });

  let visible = $derived(
    lines.filter((l) => {
      if (type === 'access' && showKind !== 'all' && l.kind !== showKind) return false;
      if (filter && !l.text.toLowerCase().includes(filter.toLowerCase())) return false;
      return true;
    })
  );

  let activeLog = $derived.by(() => {
    if (!domain) return { access: null, error: null };
    const d = domains.find((x) => x.domain === domain);
    return d ? { access: d.accessLog, error: d.errorLog } : null;
  });

  $effect(() => {
    const k = type;
    const d = domain;
    if (es) es.close();
    es = null;
    lines = [];
    live = false;
    const q = new URLSearchParams({ type: k });
    if (d) q.set('domain', d);
    es = new EventSource(`/api/logs/stream?${q}`);
    es.onopen = () => (live = true);
    es.onerror = () => (live = false);
    es.addEventListener('snapshot', (e) => {
      lines = JSON.parse(e.data);
    });
    es.addEventListener('log', (e) => {
      const l = JSON.parse(e.data);
      lines = [...lines, l].slice(-5000);
    });
    return () => {
      es.close();
      es = null;
    };
  });

  let logEl;
  $effect(() => {
    visible.length;
    if (follow && logEl) logEl.scrollTop = logEl.scrollHeight;
  });

  function fmt(n) {
    return n ? n.toLocaleString() : '0';
  }

  function lineClass(text) {
    const t = text.toLowerCase();
    if (/(emerg|alert|crit|error)/.test(t)) return 'err';
    if (/warn/.test(t)) return 'warn';
    if (/notice|info/.test(t)) return 'info';
    return '';
  }
</script>

<div class="page-head">
  <div>
    <h1>Logs</h1>
    <p>Live nginx access and error logs.</p>
  </div>
  <div class="head-actions">
    <span class="pill" class:pill-green={live} class:pill-muted={!live} title={live ? 'streaming live' : 'reconnecting…'}>
      {live ? '● live' : '○ offline'}
    </span>
    <button class="btn btn-sm" onclick={() => (follow = !follow)}>
      {follow ? 'Following' : 'Paused'}
    </button>
    <button class="btn btn-sm" onclick={() => (lines = [])}>Clear</button>
  </div>
</div>

<div class="tabs">
  <button class="tab" class:active={type === 'access'} onclick={() => (type = 'access')}>Access</button>
  <button class="tab" class:active={type === 'error'} onclick={() => (type = 'error')}>Error</button>
</div>

<div class="domain-row">
  <label class="dlabel">Domain</label>
  <select class="dselect" bind:value={domain}>
    <option value="">All domains</option>
    {#each domains as d (d.domain)}
      <option value={d.domain}>{d.domain}</option>
    {/each}
  </select>
  <span class="dpath" title="log file for the selected view">
    {#if type === 'access'}
      {activeLog?.access ?? 'default access log'}
    {:else}
      {activeLog?.error ?? 'default error log'}
    {/if}
  </span>
</div>

{#if type === 'access'}
  <div class="filter-row">
    <div class="kind-filters">
      <button class="chip" class:active={showKind === 'all'} onclick={() => (showKind = 'all')}>All · {fmt(lines.length)}</button>
      <button class="chip" class:active={showKind === 'human'} onclick={() => (showKind = 'human')}>
        <span class="kdot k-human"></span> Humans · {fmt(counts.human)}
      </button>
      <button class="chip" class:active={showKind === 'bot'} onclick={() => (showKind = 'bot')}>
        <span class="kdot k-bot"></span> Bots · {fmt(counts.bot)}
      </button>
      <button class="chip" class:active={showKind === 'other'} onclick={() => (showKind = 'other')}>
        <span class="kdot k-other"></span> No UA · {fmt(counts.other)}
      </button>
    </div>
    <input class="filter-input" bind:value={filter} placeholder="Filter lines…" />
  </div>
{/if}

<div class="log-shell" bind:this={logEl}>
  {#if visible.length === 0}
    <div class="empty">No matching log lines yet.</div>
  {:else}
    {#each visible as l, i (i)}
      <div class="lrow">
        {#if type === 'access'}
          <span class="kbadge" class:human={l.kind === 'human'} class:bot={l.kind === 'bot'} class:other={l.kind === 'other'}>
            {l.kind || 'other'}
          </span>
        {/if}
        <span class="ltext {lineClass(l.text)}">
          {#if type === 'access' && l.domain && l.domain !== domain}
            <span class="ldomain">{l.domain}</span>
          {/if}
          {l.text}
        </span>
      </div>
    {/each}
  {/if}
</div>

<style>
  .page-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  .page-head p {
    color: var(--muted);
    margin: 6px 0 0;
  }

  .head-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .tabs {
    display: flex;
    gap: 6px;
    margin-bottom: 14px;
  }

  .domain-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  .dlabel {
    margin-bottom: 0;
    font-size: 11px;
  }

  .dselect {
    width: auto;
    min-width: 200px;
    padding: 6px 10px;
    font-size: 13px;
  }

  .dpath {
    font-family: 'SF Mono', ui-monospace, Menlo, monospace;
    font-size: 11px;
    color: var(--muted);
    opacity: 0.85;
  }

  .tab {
    background: var(--bg-soft);
    border: 1px solid var(--border);
    color: var(--muted);
    border-radius: 8px;
    padding: 7px 16px;
    font-size: 13px;
  }

  .tab.active {
    background: var(--panel);
    border-color: var(--accent);
    color: var(--text);
  }

  .filter-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  .kind-filters {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-soft);
    border: 1px solid var(--border);
    color: var(--muted);
    border-radius: 999px;
    padding: 4px 12px;
    font-size: 12px;
  }

  .chip.active {
    border-color: var(--accent);
    color: var(--text);
  }

  .filter-input {
    max-width: 260px;
    margin-left: auto;
    padding: 5px 10px;
    font-size: 12px;
  }

  .kdot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    display: inline-block;
  }

  .k-human { background: var(--green); }
  .k-bot { background: var(--amber); }
  .k-other { background: var(--muted); }

  .log-shell {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 10px 0;
    height: calc(100vh - 300px);
    min-height: 320px;
    overflow-y: auto;
    font-family: 'SF Mono', ui-monospace, Menlo, monospace;
  }

  .lrow {
    display: flex;
    gap: 10px;
    padding: 2px 14px;
    align-items: flex-start;
  }

  .lrow:hover {
    background: rgba(255, 255, 255, 0.02);
  }

  .kbadge {
    flex-shrink: 0;
    margin-top: 3px;
    width: 44px;
    text-align: center;
    font-size: 9px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-radius: 4px;
    padding: 1px 0;
  }

  .kbadge.human { background: rgba(52, 211, 153, 0.15); color: var(--green); }
  .kbadge.bot { background: rgba(251, 191, 36, 0.15); color: var(--amber); }
  .kbadge.other { background: rgba(148, 163, 184, 0.15); color: var(--muted); }

  .ltext {
    display: block;
    flex: 1;
    font-size: 11.5px;
    line-height: 1.55;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--muted);
  }

  .ldomain {
    display: inline-block;
    background: rgba(45, 212, 191, 0.12);
    color: var(--accent);
    border-radius: 4px;
    padding: 0 6px;
    margin-right: 6px;
    font-size: 10px;
    font-weight: 600;
  }

  .ltext.err { color: var(--red); }
  .ltext.warn { color: var(--amber); }
  .ltext.info { color: var(--accent); }

  .empty {
    color: var(--muted);
    padding: 30px;
    text-align: center;
    font-size: 13px;
  }
</style>
