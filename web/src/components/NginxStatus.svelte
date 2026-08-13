<script>
  import { api } from '../lib/api.js';

  let { ontoast } = $props();

  let status = $state(null);
  let loading = $state(true);
  let busy = $state('');
  let lastOutput = $state('');

  async function load() {
    loading = true;
    try {
      status = await api.nginxStatus();
    } catch (e) {
      ontoast?.(e.message, 'error');
    } finally {
      loading = false;
    }
  }

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
      load();
    }
  }

  $effect(() => {
    load();
    const id = setInterval(load, 10000);
    return () => clearInterval(id);
  });
</script>

<div class="page-head">
  <div>
    <h1>Dashboard</h1>
    <p>Control and monitor the nginx service.</p>
  </div>
  <button class="btn" onclick={load} disabled={loading}>
    {loading ? 'Refreshing…' : 'Refresh'}
  </button>
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
