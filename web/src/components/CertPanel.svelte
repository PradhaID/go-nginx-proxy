<script>
  import { api } from '../lib/api.js';

  let { site, onclose, ontoast, onchanged } = $props();

  let cert = $state(null);
  let loading = $state(true);
  let busy = $state('');
  let output = $state('');

  async function load() {
    loading = true;
    try {
      cert = await api.certStatus(site.domain);
    } catch (e) {
      ontoast?.(e.message, 'error');
    } finally {
      loading = false;
    }
  }

  async function run(action, label, opts = {}) {
    busy = label;
    try {
      const res = await action(opts);
      output = res.output || '';
      ontoast?.(label + ' ok', 'success');
      onchanged?.();
      load();
    } catch (e) {
      output = e.data?.output || '';
      ontoast?.(`${e.message}${output ? `\n${output}` : ''}`, 'error');
    } finally {
      busy = '';
    }
  }

  const daysLeft = $derived(cert?.issued ? Math.max(cert.expiresInDays, 0) : null);
  const expiryPill = $derived.by(() => {
    if (!cert?.issued) return { cls: 'pill-muted', txt: 'no certificate' };
    if (daysLeft < 15) return { cls: 'pill-red', txt: `expires in ${daysLeft}d` };
    if (daysLeft < 45) return { cls: 'pill-amber', txt: `expires in ${daysLeft}d` };
    return { cls: 'pill-green', txt: `expires in ${daysLeft}d` };
  });

  $effect(() => {
    load();
  });
</script>

<div class="modal">
  <div class="modal-head">
    <div>
      <h2>Certificate — {site.domain}</h2>
      <p class="muted">Let's Encrypt via certbot{site.ssl ? '' : ' · site not yet configured for SSL'}</p>
    </div>
    <button class="btn" onclick={onclose}>Close</button>
  </div>

  <div class="cert-status card">
    <div>
      <div class="stat-label">Status</div>
      {#if loading}
        <span class="pill pill-muted">loading…</span>
      {:else}
        <span class="pill {expiryPill.cls}">{expiryPill.txt}</span>
      {/if}
    </div>
    <div>
      <div class="stat-label">Not after</div>
      <div class="stat-value">{cert?.issued ? cert.notAfter : '—'}</div>
    </div>
    <div>
      <div class="stat-label">Domains</div>
      <div class="domains mono">
        {#each (site.domains?.length ? [site.domain, ...site.domains] : [site.domain]) as d}
          <span>{d}</span>
        {/each}
      </div>
    </div>
  </div>

  <div class="actions">
    <button
      class="btn btn-primary"
      onclick={() => run((o) => api.certIssue(site.domain, o), 'Issue certificate', site)}
      disabled={!!busy}
    >
      {busy === 'Issue certificate' ? 'Issuing…' : 'Issue certificate'}
    </button>
    <button
      class="btn"
      onclick={() => run(() => api.certRenew(site.domain), 'Renew')}
      disabled={!!busy || !cert?.issued}
    >
      {busy === 'Renew' ? 'Renewing…' : 'Renew'}
    </button>
    <button
      class="btn btn-danger"
      onclick={() => run(() => api.certDelete(site.domain), 'Delete certificate')}
      disabled={!!busy || !cert?.issued}
    >
      {busy === 'Delete certificate' ? 'Deleting…' : 'Delete certificate'}
    </button>
  </div>

  {#if output}
    <pre class="output">{output}</pre>
  {/if}

  <div class="hint">
    <strong>DNS verification via Cloudflare</strong> is used when PROXY_CLOUDFLARE_DNS_TOKEN is set on the server.
    Otherwise certbot uses the webroot (<span class="mono">/var/www/html</span>).
  </div>
</div>

<style>
  .modal {
    width: 100%;
    max-width: 620px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 24px;
  }

  .modal-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 18px;
  }

  .modal-head .muted {
    margin: 6px 0 0;
    color: var(--muted);
  }

  .cert-status {
    display: flex;
    flex-wrap: wrap;
    gap: 28px;
    margin-bottom: 18px;
  }

  .stat-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--muted);
    margin-bottom: 6px;
  }

  .stat-value {
    font-size: 14px;
    font-weight: 600;
  }

  .domains {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 12px;
    color: var(--accent);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-bottom: 16px;
  }

  .output {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--muted);
    max-height: 220px;
    overflow-y: auto;
    margin-bottom: 16px;
  }

  .hint {
    font-size: 12px;
    color: var(--muted);
    border-top: 1px solid var(--border);
    padding-top: 14px;
    line-height: 1.6;
  }

  .hint strong {
    color: var(--text);
  }
</style>
