'use strict';
const byId = id => document.getElementById(id);
let token = '';
let config;
let busy = false;
function controls() { for (const id of ['apply', 'restore', 'save']) byId(id).disabled = !token || busy; }
async function request(path, body) {
  const options = body === undefined ? {} : {method: 'POST', headers: {'Content-Type': 'application/json', Authorization: `Bearer ${token}`}, body: JSON.stringify(body)};
  const response = await fetch(path, options);
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
  return data;
}
function render(state) {
  byId('policy').textContent = state.policy;
  byId('cpu').textContent = state.device.resources.cpu;
  byId('density').textContent = `${state.device.resources.density} dpi`;
  byId('temperature').textContent = `${state.device.temperature_c}°C`;
  byId('state-note').textContent = `${state.throttled ? 'Thermal protection active. ' : ''}Baseline: ${state.baseline.cpu}, ${state.baseline.density} dpi. Commands attempted: ${state.commands}.`;
  byId('error').textContent = state.error || '';
  byId('processes').replaceChildren(...state.device.apps.map(app => {
    const process = state.device.processes.find(p => p.package === app.package);
    const row = document.createElement('tr');
    for (const value of [app.label, process?.pid ?? '—', process?.rss_mib ?? '—']) {
      const cell = document.createElement('td'); cell.textContent = String(value); row.append(cell);
    }
    return row;
  }));
}
async function loadConfig() {
  config = await request('/api/v1/config');
  byId('revision').textContent = `Configuration revision ${config.revision}`;
  byId('limit').value = config.thermal_limit_c; byId('cool').value = config.thermal_cool_c;
  byId('rules').replaceChildren(...config.rules.map(rule => {
    const item = document.createElement('li'); item.textContent = `${rule.package}: ${rule.resources.cpu}, ${rule.resources.density} dpi`; return item;
  }));
}
async function action(path, body) {
  busy = true; controls(); byId('error').textContent = '';
  try { render(await request(path, body)); await loadConfig(); }
  catch (err) { byId('error').textContent = err.message; if (path.endsWith('/config')) await loadConfig().catch(() => {}); }
  finally { busy = false; controls(); }
}
byId('unlock').addEventListener('click', () => {
  token = byId('token').value.trim(); byId('token').value = '';
  byId('session-note').textContent = 'Token loaded in memory. The server verifies it on each mutation.'; controls();
});
byId('apply').addEventListener('click', () => action('/api/v1/simulation', {foreground: byId('foreground').value, temperature_c: Number(byId('temp').value)}));
byId('restore').addEventListener('click', () => action('/api/v1/actions/restore', {}));
byId('save').addEventListener('click', () => action('/api/v1/config', {...config, thermal_limit_c: Number(byId('limit').value), thermal_cool_c: Number(byId('cool').value)}));
Promise.all([request('/api/v1/state').then(render), loadConfig()]).catch(err => { byId('error').textContent = err.message; });
