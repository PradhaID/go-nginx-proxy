async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body && typeof options.body !== 'string') {
    headers['Content-Type'] = 'application/json';
    options.body = JSON.stringify(options.body);
  }
  const res = await fetch('/api' + path, { ...options, headers });
  let data = {};
  try {
    data = await res.json();
  } catch {
    /* empty body */
  }
  if (!res.ok) {
    const err = new Error(data.error || res.statusText);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

export const api = {
  health: () => request('/health'),
  nginxStatus: () => request('/nginx/status'),
  nginxStart: () => request('/nginx/start', { method: 'POST' }),
  nginxStop: () => request('/nginx/stop', { method: 'POST' }),
  nginxRestart: () => request('/nginx/restart', { method: 'POST' }),
  nginxReload: () => request('/nginx/reload', { method: 'POST' }),
  nginxTest: () => request('/nginx/test', { method: 'POST' }),

  listSites: () => request('/sites'),
  getSite: (domain) => request(`/sites/${encodeURIComponent(domain)}`),
  createSite: (site) => request('/sites', { method: 'POST', body: site }),
  updateSite: (domain, site) => request(`/sites/${encodeURIComponent(domain)}`, { method: 'PUT', body: site }),
  deleteSite: (domain) => request(`/sites/${encodeURIComponent(domain)}`, { method: 'DELETE' }),
  enableSite: (domain) => request(`/sites/${encodeURIComponent(domain)}/enable`, { method: 'POST' }),
  disableSite: (domain) => request(`/sites/${encodeURIComponent(domain)}/disable`, { method: 'POST' }),
  testSite: (domain) => request(`/sites/${encodeURIComponent(domain)}/test`, { method: 'POST' }),

  certStatus: (domain) => request(`/sites/${encodeURIComponent(domain)}/cert`),
  certIssue: (domain, body) => request(`/sites/${encodeURIComponent(domain)}/cert`, { method: 'POST', body: body || {} }),
  certRenew: (domain) => request(`/sites/${encodeURIComponent(domain)}/cert/renew`, { method: 'POST' }),
  certDelete: (domain) => request(`/sites/${encodeURIComponent(domain)}/cert`, { method: 'DELETE' })
};
