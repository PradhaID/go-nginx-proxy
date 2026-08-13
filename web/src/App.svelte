<script>
  import NginxStatus from './components/NginxStatus.svelte';
  import SiteList from './components/SiteList.svelte';
  import SiteForm from './components/SiteForm.svelte';
  import { api } from './lib/api.js';

  let view = $state('dashboard');
  let sites = $state([]);
  let showForm = $state(false);
  let editing = $state(null);
  let toasts = $state([]);
  let loading = $state(false);

  async function loadSites() {
    loading = true;
    try {
      const data = await api.listSites();
      sites = data.sites;
    } catch (e) {
      toast(e.message, 'error');
    } finally {
      loading = false;
    }
  }

  function toast(msg, type = 'info') {
    const id = crypto.randomUUID();
    toasts = [...toasts, { id, msg, type }];
    setTimeout(() => {
      toasts = toasts.filter((t) => t.id !== id);
    }, 4500);
  }

  function openCreate() {
    editing = null;
    showForm = true;
  }

  function openEdit(site) {
    editing = site;
    showForm = true;
  }

  async function onSaved() {
    showForm = false;
    await loadSites();
    toast('Site saved', 'success');
  }

  function onDeleted() {
    showForm = false;
    loadSites();
    toast('Site deleted', 'info');
  }

  $effect(() => {
    loadSites();
  });
</script>

<svelte:head>
  <title>go-nginx-proxy</title>
</svelte:head>

<div class="layout">
  <aside class="sidebar">
    <div class="brand">
      <span class="logo">NGX</span>
      <div>
        <div class="brand-title">go-nginx-proxy</div>
        <div class="brand-sub">reverse proxy manager</div>
      </div>
    </div>

    <nav>
      <button
        class="nav-item"
        class:active={view === 'dashboard'}
        onclick={() => (view = 'dashboard')}
      >
        <span class="dot" style:background="var(--accent)"></span>
        Dashboard
      </button>
      <button
        class="nav-item"
        class:active={view === 'sites'}
        onclick={() => (view = 'sites')}
      >
        <span class="dot" style:background="var(--green)"></span>
        Sites
        <span class="count">{sites.length}</span>
      </button>
    </nav>

    <div class="sidebar-footer">
      <span class="pill pill-muted">v0.1.0</span>
    </div>
  </aside>

  <main class="content">
    {#if view === 'dashboard'}
      <NginxStatus />
    {:else if view === 'sites'}
      <SiteList
        sites={sites}
        loading={loading}
        oncreate={openCreate}
        onedit={openEdit}
        onchanged={loadSites}
        ontoast={toast}
      />
    {/if}
  </main>
</div>

{#if showForm}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && (showForm = false)}>
    <SiteForm
      site={editing}
      onsave={onSaved}
      oncancel={() => (showForm = false)}
      ondeleted={onDeleted}
      ontoast={toast}
    />
  </div>
{/if}

<div class="toasts">
  {#each toasts as t (t.id)}
    <div class="toast toast-{t.type}">{t.msg}</div>
  {/each}
</div>

<style>
  .layout {
    display: flex;
    min-height: 100vh;
  }

  .sidebar {
    width: 230px;
    flex-shrink: 0;
    background: var(--bg-soft);
    border-right: 1px solid var(--border);
    padding: 20px 14px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: 10px;
    background: linear-gradient(135deg, var(--accent), var(--accent-dim));
    color: #04211c;
    font-weight: 800;
    font-size: 12px;
    letter-spacing: 0.05em;
  }

  .brand-title {
    font-weight: 600;
  }

  .brand-sub {
    font-size: 11px;
    color: var(--muted);
  }

  nav {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    text-align: left;
    background: transparent;
    border: none;
    color: var(--muted);
    padding: 10px 12px;
    border-radius: 8px;
    font-size: 14px;
  }

  .nav-item:hover {
    background: var(--bg);
    color: var(--text);
  }

  .nav-item.active {
    background: var(--bg);
    color: var(--text);
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    display: inline-block;
  }

  .count {
    margin-left: auto;
    background: var(--bg);
    color: var(--muted);
    border-radius: 999px;
    padding: 1px 8px;
    font-size: 11px;
  }

  .sidebar-footer {
    margin-top: auto;
  }

  .content {
    flex: 1;
    padding: 28px;
    max-width: 1100px;
    width: 100%;
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

  .toasts {
    position: fixed;
    top: 18px;
    right: 18px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    z-index: 100;
  }

  .toast {
    padding: 10px 16px;
    border-radius: 8px;
    font-size: 13px;
    border: 1px solid var(--border);
    background: var(--panel);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    min-width: 220px;
    max-width: 360px;
  }

  .toast-success { border-color: var(--green); color: var(--green); }
  .toast-error { border-color: var(--red); color: var(--red); }
  .toast-info { border-color: var(--accent); color: var(--accent); }

  @media (max-width: 720px) {
    .layout {
      flex-direction: column;
    }
    .sidebar {
      width: 100%;
      flex-direction: row;
      align-items: center;
      border-right: none;
      border-bottom: 1px solid var(--border);
    }
    nav {
      flex-direction: row;
      margin-left: auto;
    }
    .sidebar-footer {
      display: none;
    }
  }
</style>
