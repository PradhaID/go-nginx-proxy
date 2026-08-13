import { writable } from 'svelte/store';
import { api } from './api.js';

export const realtimeStatus = writable(null);
export const realtimeSites = writable([]);
export const realtimeConnected = writable(false);
export const realtimeLoaded = writable(false);

let es = null;

export function connectRealtime() {
  if (es) return;
  es = new EventSource('/api/events');
  es.addEventListener('status', (e) => {
    realtimeStatus.set(JSON.parse(e.data));
    realtimeLoaded.set(true);
  });
  es.addEventListener('sites', (e) => {
    const data = JSON.parse(e.data);
    realtimeSites.set(Array.isArray(data) ? data : data.sites ?? []);
  });
  es.onopen = () => realtimeConnected.set(true);
  es.onerror = () => realtimeConnected.set(false);
}

export function disconnectRealtime() {
  if (es) {
    es.close();
    es = null;
  }
}

export async function refreshStatus() {
  try {
    realtimeStatus.set(await api.nginxStatus());
    realtimeLoaded.set(true);
  } catch {
    // SSE will deliver the next update; leave existing status in place
  }
}
