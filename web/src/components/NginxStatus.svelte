<script>
  import { api } from '../lib/api.js';
  import { realtimeStatus, realtimeConnected, realtimeLoaded, refreshStatus } from '../lib/realtime.js';

  let { ontoast } = $props();

  let status = $derived($realtimeStatus);
  let connected = $derived($realtimeConnected);
  let loading = $derived(!$realtimeLoaded);
  let busy = $state('');
  let lastOutput = $state('');

  async function run(action, label) {
    busy = label;
    try {
      const res = await action();
      lastOutput = res.output || '';
      ontoast?.(label + ' ok', 'success');
    } catch (e) {
      lastOutput = e.data?.output || '';
      ontoast?.(`${e.message}${lastOutput ? `\n${lastOutput}` : ''}`, 'error');
    } finally {
      busy = '';
      refreshStatus();
    }
  }

  function fmtKB(kb) {
    if (!kb && kb !== 0) return '—';
    if (kb >= 1024 * 1024) return (kb / 1024 / 1024).toFixed(2) + ' GB';
    return (kb / 1024).toFixed(1) + ' MB';
  }
</script>

<div class="page-head">
  <div>
    <h1>Dashboard</h1>
    <p>Control and monitor the nginx service.</p>
  </div>
  <div class="head-actions">
    <span class="pill pill-muted" class:pill-green={connected} title={connected ? 'live updates connected' : 'realtime disconnected'}>
      {connected ? '● live' : '○ offline'}
    </span>
    <button class="btn" onclick={refreshStatus} disabled={loading}>
      {loading ? 'Refreshing…' : 'Refresh'}
    </button>
  </div>
</div>

{#if status}
  <div class="grid">
    <div class="card stat">
      <div class="stat-label">Service</div>
      <div class="stat-value">
        {#if status.running}
          <span class="pill pill-green">running</span>
        {:else}
          <span class="pill pill-red">stopped</span>
        {/if}
      </div>
      <div class="stat-sub">systemd: {status.active}</div>
    </div>

    <div class="card stat">
      <div class="stat-label">Version</div>
      <div class="stat-value mono">{status.version || 'unknown'}</div>
      <div class="stat-sub">nginx binary</div>
    </div>

    <div class="card stat">
      <div class="stat-label">Available</div>
      <div class="stat-value">{status.configs ?? 0}</div>
      <div class="stat-sub">configs in sites-available</div>
    </div>

    <div class="card stat">
      <div class="stat-label">Enabled</div>
      <div class="stat-value">{status.enabled ?? 0}</div>
      <div class="stat-sub">symlinks in sites-enabled</div>
    </div>
  </div>

  <div class="grid">
    <div class="card stat">
      <div class="stat-label">CPU</div>
      <div class="stat-value">{(status.metrics?.cpuPercent ?? 0).toFixed(1)}<span class="unit">%</span></div>
      <div class="meter"><div class="meter-fill" style="width:{Math.min(status.metrics?.cpuPercent ?? 0, 100)}%"></div></div>
      <div class="stat-sub">{status.metrics?.procs ?? 0} nginx process{status.metrics?.procs === 1 ? '' : 'es'}</div>
    </div>

    <div class="card stat">
      <div class="stat-label">Memory</div>
      <div class="stat-value">{fmtKB(status.metrics?.memKB)}</div>
      <div class="meter"><div class="meter-fill" style="width:{Math.min(status.metrics?.memPercent ?? 0, 100)}%"></div></div>
      <div class="stat-sub">
        {#if status.metrics?.memTotalKB}
          {(status.metrics?.memPercent ?? 0).toFixed(1)}% of {fmtKB(status.metrics.memTotalKB)}
        {:else}
          nginx resident set size
        {/if}
      </div>
    </div>
  </div>

  <div class="card">
    <h3 class="card-title">Actions</h3>
    <div class="actions">
      <button class="btn btn-primary" onclick={() => run(api.nginxStart, 'Start')} disabled={!!busy || status.running}>
        {busy === 'Start' ? '…' : 'Start'}
      </button>
      <button class="btn" onclick={() => run(api.nginxStop, 'Stop')} disabled={!!busy || !status.running}>
        {busy === 'Stop' ? '…' : 'Stop'}
      </button>
      <button class="btn" onclick={() => run(api.nginxRestart, 'Restart')} disabled={!!busy || !status.running}>
        {busy === 'Restart' ? '…' : 'Restart'}
      </button>
      <button class="btn" onclick={() => run(api.nginxReload, 'Reload')} disabled={!!busy || !status.running}>
        {busy === 'Reload' ? '…' : 'Reload'}
      </button>
      <button class="btn" onclick={() => run(api.nginxTest, 'Test config')} disabled={!!busy}>
        {busy === 'Test config' ? '…' : 'Test config'}
      </button>
    </div>

    {#if lastOutput}
      <pre class="output">{lastOutput}</pre>
    {/if}
  </div>
{:else if loading}
  <p class="muted">Loading…</p>
{:else}
  <p class="muted">Could not read nginx status.</p>
{/if}

<style>
  .page-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }

  .head-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .unit {
    font-size: 14px;
    font-weight: 500;
    color: var(--muted);
  }

  .meter {
    height: 6px;
    background: var(--bg);
    border-radius: 999px;
    overflow: hidden;
    margin: 6px 0;
  }

  .meter-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--accent), var(--accent-dim));
    border-radius: 999px;
    transition: width 0.4s ease;
  }

  .page-head p {
    color: var(--muted);
    margin: 6px 0 0;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
    margin-bottom: 20px;
  }

  .stat-value {
    font-size: 22px;
    font-weight: 700;
    margin: 8px 0 4px;
  }

  .stat-label {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--muted);
  }

  .stat-sub {
    font-size: 12px;
    color: var(--muted);
  }

  .card-title {
    margin-bottom: 14px;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .output {
    margin-top: 16px;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--muted);
  }

  .muted {
    color: var(--muted);
  }
</style>
