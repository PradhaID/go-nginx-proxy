<script>
  import { api } from '../lib/api.js';
  import CertPanel from './CertPanel.svelte';

  let { sites, loading, oncreate, onedit, onchanged, ontoast } = $props();

  let busy = $state('');
  let showCert = $state(null);
  let pendingDelete = $state(null);

  async function toggle(site) {
    busy = site.domain;
    try {
      if (site.enabled) {
        await api.disableSite(site.domain);
        ontoast?.(`Disabled ${site.domain}`, 'info');
      } else {
        await api.enableSite(site.domain);
        ontoast?.(`Enabled ${site.domain}`, 'success');
      }
      onchanged?.();
    } catch (e) {
      ontoast?.(e.message, 'error');
    } finally {
      busy = '';
    }
  }

  async function test(site) {
    busy = site.domain;
    try {
      const res = await api.testSite(site.domain);
      ontoast?.(`Config test passed for ${site.domain}`, 'success');
      onchanged?.();
    } catch (e) {
      ontoast?.(`${e.message}${e.data?.output ? `\n${e.data.output}` : ''}`, 'error');
    } finally {
      busy = '';
    }
  }

  async function del(site) {
    try {
      await api.deleteSite(site.domain);
      ontoast?.(`Deleted ${site.domain}`, 'info');
      pendingDelete = null;
      onchanged?.();
    } catch (e) {
      ontoast?.(e.message, 'error');
    }
  }

  function certChanged() {
    onchanged?.();
  }
</script>

<div class="page-head">
  <div>
    <h1>Sites</h1>
    <p>Reverse proxy configurations under sites-available / sites-enabled.</p>
  </div>
  <button class="btn btn-primary" onclick={oncreate}>+ New site</button>
</div>

{#if loading}
  <p class="muted">Loading sites…</p>
{:else if !sites.length}
  <div class="card empty">
    <p>No sites yet. Create your first proxy config.</p>
    <button class="btn btn-primary" onclick={oncreate}>+ New site</button>
  </div>
{:else}
  <div class="card table-wrap">
    <table>
      <thead>
        <tr>
          <th>Domain</th>
          <th>Status</th>
          <th>SSL</th>
          <th>Upstreams</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each sites as site (site.domain)}
          <tr class:row-off={!site.enabled}>
            <td>
              <div class="domain mono">{site.domain}</div>
              {#if site.external}
                <div class="tag">external config</div>
              {/if}
            </td>
            <td>
              {#if site.enabled}
                <span class="pill pill-green">enabled</span>
              {:else}
                <span class="pill pill-red">disabled</span>
              {/if}
            </td>
            <td>
              {#if site.ssl}
                {#if site.hasCert}
                  <span class="pill pill-green">https</span>
                {:else}
                  <span class="pill pill-amber">no cert</span>
                {/if}
              {:else}
                <span class="pill pill-muted">http</span>
              {/if}
            </td>
            <td>
              <div class="upstreams mono">
                {#if site.locations?.length}
                  {site.locations.map((l) => `${l.path} → ${l.proxyPass}`).join(' · ')}
                {:else if site.webRoot}
                  <span class="muted">static: {site.webRoot}</span>
                {:else}
                  <span class="muted">—</span>
                {/if}
              </div>
            </td>
            <td>
              <div class="row-actions">
                {#if pendingDelete === site.domain}
                  <button class="btn btn-sm btn-danger" onclick={() => del(site)}>Confirm</button>
                  <button class="btn btn-sm" onclick={() => (pendingDelete = null)}>Cancel</button>
                {:else}
                  <button class="btn btn-sm" onclick={() => onedit(site)}>Edit</button>
                  <button class="btn btn-sm" onclick={() => toggle(site)} disabled={busy === site.domain}>
                    {site.enabled ? 'Disable' : 'Enable'}
                  </button>
                  <button class="btn btn-sm" onclick={() => test(site)} disabled={busy === site.domain}>Test</button>
                  <button class="btn btn-sm" onclick={() => (showCert = site)}>Cert</button>
                  <button class="btn btn-sm btn-danger" onclick={() => (pendingDelete = site.domain)}>Delete</button>
                {/if}
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

{#if showCert}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && (showCert = null)}>
    <CertPanel
      site={showCert}
      onclose={() => (showCert = null)}
      ontoast={ontoast}
      onchanged={certChanged}
    />
  </div>
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

  .table-wrap {
    padding: 0;
    overflow-x: auto;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th {
    text-align: left;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--muted);
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
  }

  td {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
    vertical-align: middle;
  }

  tbody tr:last-child td {
    border-bottom: none;
  }

  .row-off {
    opacity: 0.55;
  }

  .domain {
    font-weight: 600;
    font-size: 14px;
  }

  .tag {
    font-size: 10px;
    color: var(--amber);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-top: 2px;
  }

  .upstreams {
    font-size: 12px;
    color: var(--muted);
    max-width: 320px;
  }

  .row-actions {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
  }

  .empty {
    text-align: center;
    padding: 40px;
    color: var(--muted);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
  }

  .muted {
    color: var(--muted);
  }

  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(4, 8, 18, 0.7);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding: 40px 20px;
    overflow-y: auto;
    z-index: 50;
    backdrop-filter: blur(2px);
  }
</style>
