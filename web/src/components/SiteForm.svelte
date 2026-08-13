<script>
  import { api } from '../lib/api.js';

  let { site, onsave, oncancel, ondeleted, ontoast } = $props();

  let domain = $state(site?.domain || '');
  let domainsText = $state(site?.domains?.join(', ') || '');
  let ssl = $state(site?.ssl ?? false);
  let redirect = $state(site?.redirectToHttps ?? false);
  let clientMaxBodySize = $state(site?.clientMaxBodySize || '');
  let webRoot = $state(site?.webRoot || '');
  let locations = $state(
    site?.locations?.length
      ? site.locations.map((l) => ({ ...l }))
      : [{ path: '/', proxyPass: '', websocket: false, extra: '' }]
  );
  let extraConfig = $state(site?.extraConfig || '');
  let saving = $state(false);

  function addLocation() {
    locations = [...locations, { path: '/', proxyPass: '', websocket: false, extra: '' }];
  }

  function removeLocation(i) {
    locations = locations.filter((_, idx) => idx !== i);
    if (!locations.length) {
      locations = [{ path: '/', proxyPass: '', websocket: false, extra: '' }];
    }
  }

  function payload() {
    return {
      domain: domain.trim(),
      domains: domainsText
        .split(/[,\s]+/)
        .map((d) => d.trim())
        .filter(Boolean),
      ssl,
      redirectToHttps: redirect,
      clientMaxBodySize: clientMaxBodySize.trim(),
      webRoot: webRoot.trim(),
      locations: locations.filter((l) => l.path.trim() || l.proxyPass.trim()),
      extraConfig
    };
  }

  async function save() {
    if (!domain.trim()) {
      ontoast?.('Domain is required', 'error');
      return;
    }
    saving = true;
    try {
      if (site?.domain) {
        await api.updateSite(site.domain, payload());
      } else {
        await api.createSite(payload());
      }
      onsave?.();
    } catch (e) {
      ontoast?.(e.message, 'error');
    } finally {
      saving = false;
    }
  }

  async function del() {
    if (!confirm(`Delete site ${site.domain}? This removes the config file and disables it.`)) return;
    saving = true;
    try {
      await api.deleteSite(site.domain);
      ondeleted?.();
    } catch (e) {
      ontoast?.(e.message, 'error');
      saving = false;
    }
  }
</script>

<div class="modal">
  <div class="modal-head">
    <div>
      <h2>{site ? `Edit ${site.domain}` : 'New site'}</h2>
      <p class="muted">Creates {domain ? `${domain}.conf` : '&lt;domain&gt;.conf'} in sites-available</p>
    </div>
    <div class="head-actions">
      {#if site}
        <button class="btn btn-sm btn-danger" onclick={del} disabled={saving}>Delete</button>
      {/if}
      <button class="btn" onclick={oncancel} disabled={saving}>Cancel</button>
    </div>
  </div>

  <div class="fields">
    <div class="grid-2">
      <div>
        <label for="domain">Primary domain *</label>
        <input id="domain" bind:value={domain} placeholder="example.com" />
      </div>
      <div>
        <label for="domains">Additional domains</label>
        <input id="domains" bind:value={domainsText} placeholder="www.example.com, api.example.com" />
      </div>
    </div>

    <div class="checks">
      <label class="check">
        <input type="checkbox" bind:checked={ssl} />
        <span>Enable SSL (port 443)</span>
      </label>
      <label class="check">
        <input type="checkbox" bind:checked={redirect} disabled={!ssl} />
        <span>Redirect HTTP → HTTPS</span>
      </label>
    </div>

    <div class="grid-2">
      <div>
        <label for="maxbody">Client max body size</label>
        <input id="maxbody" bind:value={clientMaxBodySize} placeholder="10m" />
      </div>
      <div>
        <label for="webroot">Web root (static files, optional)</label>
        <input id="webroot" bind:value={webRoot} placeholder="/var/www/example.com" />
      </div>
    </div>

    <div class="locations">
      <div class="locations-head">
        <label>Proxy locations</label>
        <button class="btn btn-sm" onclick={addLocation}>+ Add location</button>
      </div>

      {#each locations as loc, i (i)}
        <div class="location">
          <div class="loc-grid">
            <div>
              <label>Path</label>
              <input bind:value={loc.path} placeholder="/" />
            </div>
            <div>
              <label>Proxy pass</label>
              <input bind:value={loc.proxyPass} placeholder="http://127.0.0.1:3000" />
            </div>
            <div>
              <label>Extra directives</label>
              <input bind:value={loc.extra} placeholder="proxy_read_timeout 60s;" />
            </div>
            <label class="check loc-check">
              <input type="checkbox" bind:checked={loc.websocket} />
              <span>WebSocket</span>
            </label>
            <button class="btn btn-sm btn-danger" onclick={() => removeLocation(i)}>✕</button>
          </div>
        </div>
      {/each}
    </div>

    <div>
      <label for="extra">Extra config (inside server block)</label>
      <textarea
        id="extra"
        bind:value={extraConfig}
        rows="4"
        placeholder="add_header X-Frame-Options SAMEORIGIN;"
      ></textarea>
    </div>
  </div>

  <div class="modal-foot">
    <span class="muted">
      {#if ssl && !site?.hasCert}
        After saving, issue a certificate from the site's Cert panel.
      {:else}
        Configs are written to disk on save; run "Test" to validate before reload.
      {/if}
    </span>
    <button class="btn btn-primary" onclick={save} disabled={saving}>
      {saving ? 'Saving…' : 'Save'}
    </button>
  </div>
</div>

<style>
  .modal {
    width: 100%;
    max-width: 720px;
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
    margin-bottom: 20px;
  }

  .modal-head .muted {
    margin: 6px 0 0;
    color: var(--muted);
    font-size: 12px;
  }

  .head-actions {
    display: flex;
    gap: 8px;
  }

  .fields {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }

  .checks {
    display: flex;
    gap: 22px;
  }

  .check {
    display: flex;
    align-items: center;
    gap: 8px;
    text-transform: none;
    letter-spacing: 0;
    font-size: 13px;
    color: var(--text);
    cursor: pointer;
    margin: 0;
  }

  .check input {
    width: auto;
    accent-color: var(--accent-dim);
  }

  .locations {
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 14px;
    background: var(--bg-soft);
  }

  .locations-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }

  .locations-head label {
    margin: 0;
  }

  .location {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 10px;
  }

  .location:last-child {
    margin-bottom: 0;
  }

  .loc-grid {
    display: grid;
    grid-template-columns: 1fr 1.4fr 1.2fr auto auto;
    gap: 10px;
    align-items: end;
  }

  .loc-check {
    white-space: nowrap;
  }

  textarea {
    resize: vertical;
    font-family: 'SF Mono', ui-monospace, Menlo, monospace;
  }

  .modal-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid var(--border);
  }

  .muted {
    color: var(--muted);
    font-size: 12px;
  }

  @media (max-width: 700px) {
    .grid-2,
    .loc-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
