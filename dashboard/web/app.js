// State Management
const state = {
  activeTab: 'overview',
  overview: null,
  pods: [],
  rollouts: [],
  optimizer: [],
  autoscaler: null,
  events: [],
  selectedPatch: null,
};

// DOM Elements
const elements = {
  navItems: document.querySelectorAll('.nav-item'),
  tabPanes: document.querySelectorAll('.tab-pane'),
  pageTitle: document.getElementById('page-title'),
  pageSubtitle: document.getElementById('page-subtitle'),
  clusterIndicator: document.getElementById('cluster-indicator'),
  clusterName: document.getElementById('cluster-name'),
  modeToggle: document.getElementById('mode-toggle-checkbox'),
  modeLabel: document.getElementById('mode-label'),
  btnRefresh: document.getElementById('btn-refresh'),
  globalEventsList: document.getElementById('global-events-list'),
  btnClearFeed: document.getElementById('btn-clear-feed'),
  optimizerModal: document.getElementById('optimizer-modal'),
  modalPatchPreview: document.getElementById('modal-patch-preview'),
  btnCloseModal: document.getElementById('btn-close-modal'),
  btnCancelModal: document.getElementById('btn-cancel-modal'),
  btnConfirmApply: document.getElementById('btn-confirm-apply'),
  toastContainer: document.getElementById('toast-container'),
  navUnhealthyCount: document.getElementById('nav-unhealthy-count'),
  navStuckCount: document.getElementById('nav-stuck-count'),
};

const tabDescriptions = {
  overview: { title: 'Overview Radar Control Plane', subtitle: 'Unified observability and autonomous lifecycle controllers' },
  healer: { title: '01 — Pod Auto-Healer', subtitle: 'Owner-aware remediation with exponential backoff and deduplication' },
  rollouts: { title: '02 — Deployment Rollout Manager', subtitle: 'ProgressDeadlineExceeded detection & zero-downtime rollback' },
  optimizer: { title: '03 — Resource Optimizer', subtitle: 'Percentile usage profiling (p50/p90) and right-sizing recommendations' },
  scaler: { title: '04 — Horizontal Autoscaler', subtitle: 'Dynamic ratio-based scaling with asymmetric cooldown stabilization' },
};

// Initialize Application
document.addEventListener('DOMContentLoaded', () => {
  setupNavigation();
  setupEventSource();
  setupEventListeners();
  loadAllData();
  setInterval(loadAllData, 5000);
});

// Navigation Handling
function setupNavigation() {
  elements.navItems.forEach(item => {
    item.addEventListener('click', () => {
      const targetTab = item.dataset.tab;
      switchTab(targetTab);
    });
  });
}

function switchTab(tabId) {
  state.activeTab = tabId;
  
  elements.navItems.forEach(item => {
    item.classList.toggle('active', item.dataset.tab === tabId);
  });
  
  elements.tabPanes.forEach(pane => {
    pane.classList.toggle('active', pane.id === `pane-${tabId}`);
  });

  const info = tabDescriptions[tabId] || tabDescriptions.overview;
  elements.pageTitle.textContent = info.title;
  elements.pageSubtitle.textContent = info.subtitle;
}

// Event Listeners
function setupEventListeners() {
  elements.btnRefresh.addEventListener('click', () => {
    showToast('Refreshing cluster telemetry...', 'info');
    loadAllData();
  });

  elements.modeToggle.addEventListener('change', async (e) => {
    const isDemo = e.target.checked;
    try {
      const res = await fetch('/api/mode/toggle', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ demo: isDemo }),
      });
      const data = await res.json();
      showToast(data.message, 'success');
      loadAllData();
    } catch (err) {
      showToast('Failed to toggle mode: ' + err, 'danger');
    }
  });

  elements.btnClearFeed.addEventListener('click', () => {
    elements.globalEventsList.innerHTML = '';
  });

  elements.btnCloseModal.addEventListener('click', closeModal);
  elements.btnCancelModal.addEventListener('click', closeModal);

  elements.btnConfirmApply.addEventListener('click', async () => {
    if (!state.selectedPatch) return;
    try {
      const res = await fetch('/api/optimizer/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(state.selectedPatch),
      });
      const data = await res.json();
      closeModal();
      showToast(data.message, 'success');
      loadAllData();
    } catch (err) {
      showToast('Error applying recommendation: ' + err, 'danger');
    }
  });

  // Filter chips in healer table
  document.querySelectorAll('.filter-chip').forEach(chip => {
    chip.addEventListener('click', (e) => {
      document.querySelectorAll('.filter-chip').forEach(c => c.classList.remove('active'));
      chip.classList.add('active');
      renderHealerTable(chip.dataset.filter);
    });
  });
}

// Server-Sent Events (SSE) Stream
function setupEventSource() {
  const eventSource = new EventSource('/api/events/stream');

  eventSource.addEventListener('automation-event', (e) => {
    try {
      const event = JSON.parse(e.data);
      appendLiveEvent(event);
      loadAllData();
    } catch (err) {
      console.error('Failed to parse SSE event:', err);
    }
  });

  eventSource.onerror = () => {
    document.getElementById('stream-status-text').textContent = 'SSE Reconnecting...';
  };

  eventSource.onopen = () => {
    document.getElementById('stream-status-text').textContent = 'SSE Live Stream Active';
  };
}

// Data Fetching
async function loadAllData() {
  try {
    const [overview, healer, rollouts, optimizer, autoscaler, events] = await Promise.all([
      fetch('/api/overview').then(r => r.json()),
      fetch('/api/healer').then(r => r.json()),
      fetch('/api/rollouts').then(r => r.json()),
      fetch('/api/optimizer').then(r => r.json()),
      fetch('/api/autoscaler').then(r => r.json()),
      fetch('/api/events').then(r => r.json()),
    ]);

    state.overview = overview;
    state.pods = healer.pods || [];
    state.rollouts = rollouts.rollouts || [];
    state.optimizer = optimizer.recommendations || [];
    state.autoscaler = autoscaler;
    state.events = events.events || [];

    renderOverview();
    renderHealerTable('all');
    renderRolloutsGrid();
    renderOptimizerTable();
    renderAutoscaler();
    renderEvents();
  } catch (err) {
    console.error('Error fetching cluster data:', err);
  }
}

// Render Overview
function renderOverview() {
  const ov = state.overview;
  if (!ov) return;

  elements.clusterName.textContent = ov.clusterName;
  elements.modeToggle.checked = ov.demoMode;
  elements.modeLabel.textContent = ov.demoMode ? 'Simulation Mode' : 'Live Cluster';

  document.getElementById('stat-total-pods').textContent = ov.totalPodsMonitored;
  document.getElementById('stat-healed-pods').textContent = ov.healedPodsCount;
  document.getElementById('stat-unhealthy-pods-meta').textContent = `${ov.unhealthyPodsCount} pods in failure backoff`;
  elements.navUnhealthyCount.textContent = ov.unhealthyPodsCount;

  document.getElementById('stat-deployments').textContent = ov.totalDeployments;
  document.getElementById('stat-rollbacks').textContent = ov.rollbacksCount;
  document.getElementById('stat-stuck-rollouts-meta').textContent = `${ov.stuckRolloutsCount} stuck rollouts`;
  elements.navStuckCount.textContent = ov.stuckRolloutsCount;

  document.getElementById('stat-waste-milli').textContent = ov.totalEstimatedWaste;
  document.getElementById('stat-containers-optimized').textContent = ov.containersOptimized;

  document.getElementById('stat-scale-events').textContent = ov.totalScaleEvents;
}

// Render 01 Pod Auto-Healer
function renderHealerTable(filter = 'all') {
  const tbody = document.getElementById('healer-tbody');
  tbody.innerHTML = '';

  const filteredPods = state.pods.filter(p => {
    if (filter === 'unhealthy') return p.isUnhealthy;
    return true;
  });

  if (filteredPods.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" style="text-align:center; padding: 24px;">No pods matching filter.</td></tr>';
    return;
  }

  filteredPods.forEach(p => {
    const tr = document.createElement('tr');
    const statusBadge = p.isUnhealthy
      ? `<span class="badge badge-danger">Unhealthy</span>`
      : `<span class="badge badge-success">${p.phase}</span>`;

    const failureReason = p.unhealthyReason
      ? `<span class="badge badge-warning">${p.unhealthyReason}</span> <small class="text-secondary">(${p.container || 'container'})</small>`
      : '<span class="text-secondary">—</span>';

    const ownerBadge = p.eligibleToHeal
      ? `<span class="badge badge-info">${p.ownerKind}</span> <code>${p.ownerName}</code>`
      : `<span class="badge badge-warning">${p.ownerKind} (Skip)</span> <code>${p.ownerName}</code>`;

    const backoffInfo = p.isUnhealthy
      ? `<span>Attempts: <strong>${p.healAttempts}</strong></span><br><small class="text-warning">${p.nextRetryIn || 'Queued'}</small>`
      : `<span class="text-success">Healthy</span>`;

    const actionBtn = p.isUnhealthy && p.eligibleToHeal
      ? `<button class="btn btn-xs btn-primary" onclick="triggerHeal('${p.namespace}', '${p.name}')">Heal Pod</button>`
      : `<button class="btn btn-xs btn-secondary" disabled>No Action</button>`;

    tr.innerHTML = `
      <td class="pod-name-col">
        <strong>${p.name}</strong>
        <span>Namespace: ${p.namespace} &bull; Node: ${p.node || 'unassigned'} &bull; Age: ${p.age}</span>
      </td>
      <td>${statusBadge}</td>
      <td>${failureReason}</td>
      <td>${ownerBadge}</td>
      <td><strong>${p.restarts}</strong></td>
      <td>${backoffInfo}</td>
      <td>${actionBtn}</td>
    `;
    tbody.appendChild(tr);
  });
}

// Render 02 Rollout Manager
function renderRolloutsGrid() {
  const grid = document.getElementById('rollouts-grid');
  grid.innerHTML = '';

  state.rollouts.forEach(r => {
    const card = document.createElement('div');
    card.className = `deployment-card ${r.isStuck ? 'stuck' : ''}`;

    const progressPct = r.replicasDesired > 0 ? (r.replicasAvailable / r.replicasDesired) * 100 : 0;
    const isStuckBadge = r.isStuck
      ? `<span class="badge badge-danger">Stuck (Deadline Exceeded)</span>`
      : `<span class="badge badge-success">Progressing (OK)</span>`;

    const rollbackBtn = r.canRollback
      ? `<button class="btn btn-sm btn-primary" onclick="triggerRollback('${r.namespace}', '${r.name}')">Rollback to Rev ${r.previousRevision || '1'}</button>`
      : `<button class="btn btn-sm btn-secondary" disabled>No Rollback Needed</button>`;

    card.innerHTML = `
      <div class="dep-header">
        <div class="dep-title">
          <h4>${r.name}</h4>
          <span>Namespace: ${r.namespace} &bull; Rev: <strong>${r.currentRevision}</strong></span>
        </div>
        ${isStuckBadge}
      </div>

      <div class="progress-bar-wrap">
        <div class="progress-bar-fill ${r.isStuck ? 'stuck' : ''}" style="width: ${progressPct}%"></div>
      </div>

      <div class="metric-row" style="margin-bottom: 8px;">
        <span>Replicas Ready / Desired</span>
        <strong>${r.replicasReady} / ${r.replicasDesired} (${r.replicasUpdated} updated)</strong>
      </div>

      <div class="metric-row" style="margin-bottom: 16px;">
        <span>Rollback Target</span>
        <strong>Revision ${r.previousRevision || '1'}</strong>
      </div>

      <div style="display: flex; justify-content: space-between; align-items: center;">
        <span style="font-size: 0.75rem; color: var(--text-muted);">Rollbacks triggered: ${r.rollbackCount}</span>
        ${rollbackBtn}
      </div>
    `;
    grid.appendChild(card);
  });
}

// Render 03 Resource Optimizer
function renderOptimizerTable() {
  const tbody = document.getElementById('optimizer-tbody');
  tbody.innerHTML = '';

  state.optimizer.forEach(rec => {
    const tr = document.createElement('tr');

    const cpuDriftBadge = rec.cpuDrift === 'over'
      ? `<span class="badge badge-warning">Over-provisioned</span>`
      : (rec.cpuDrift === 'under' ? `<span class="badge badge-danger">Under-provisioned</span>` : `<span class="badge badge-success">Optimal</span>`);

    const memDriftBadge = rec.memDrift === 'over'
      ? `<span class="badge badge-warning">Over-provisioned</span>`
      : (rec.memDrift === 'under' ? `<span class="badge badge-danger">Under-provisioned</span>` : `<span class="badge badge-success">Optimal</span>`);

    const actionBtn = rec.cpuDrift !== 'optimal' || rec.memDrift !== 'optimal'
      ? `<button class="btn btn-xs btn-primary" onclick="openOptimizerModal('${rec.key}')">Preview & Apply</button>`
      : `<button class="btn btn-xs btn-secondary" disabled>Optimal</button>`;

    tr.innerHTML = `
      <td class="pod-name-col">
        <strong>${rec.container}</strong>
        <span>${rec.namespace}/${rec.pod} &bull; Samples: ${rec.samplesCount}</span>
      </td>
      <td>
        <div>CPU: <code>${rec.currentCpuReqMilli}m / ${rec.currentCpuLimMilli}m</code></div>
        <div>Mem: <code>${Math.round(rec.currentMemReqBytes / (1024*1024))}Mi / ${Math.round(rec.currentMemLimBytes / (1024*1024))}Mi</code></div>
      </td>
      <td>
        <div>CPU: <strong>${rec.recCpuReqMilli}m / ${rec.recCpuLimMilli}m</strong></div>
        <div>Mem: <strong>${Math.round(rec.recMemReqBytes / (1024*1024))}Mi / ${Math.round(rec.recMemLimBytes / (1024*1024))}Mi</strong></div>
      </td>
      <td>${cpuDriftBadge}</td>
      <td>${memDriftBadge}</td>
      <td><small class="text-secondary">${rec.estimatedWaste}</small></td>
      <td>${actionBtn}</td>
    `;
    tbody.appendChild(tr);
  });
}

// Render 04 Horizontal Autoscaler
function renderAutoscaler() {
  const auto = state.autoscaler;
  if (!auto) return;

  document.getElementById('scaler-cpu-val').textContent = `${auto.currentCpuPercent}%`;
  document.getElementById('scaler-target-val').textContent = `${auto.targetCpuPercent}%`;
  document.getElementById('scaler-current-rep').textContent = auto.currentReplicas;
  document.getElementById('scaler-desired-rep').textContent = auto.desiredReplicas;
  document.getElementById('scaler-ns').textContent = auto.namespace;
  document.getElementById('scaler-dep').textContent = auto.deployment;
  document.getElementById('scaler-bounds').textContent = `${auto.minReplicas} / ${auto.maxReplicas}`;
  document.getElementById('scaler-total-evts').textContent = auto.scaleEventsCount;
  document.getElementById('scaler-down-cooldown').textContent = `Stabilizing (${auto.nextScaleDownAllowedIn || '300s'})`;

  // Update radial gauge gradient
  const pct = Math.min(Math.max(auto.currentCpuPercent, 0), 100);
  const color = pct > 80 ? 'var(--accent-rose)' : (pct > 60 ? 'var(--accent-amber)' : 'var(--accent-cyan)');
  document.getElementById('scaler-gauge').style.background = `conic-gradient(${color} 0% ${pct}%, var(--bg-surface-elevated) ${pct}% 100%)`;
}

// Render Real-time Events
function renderEvents() {
  elements.globalEventsList.innerHTML = '';
  state.events.forEach(evt => appendLiveEvent(evt, false));
}

function appendLiveEvent(evt, prepend = true) {
  const item = document.createElement('div');
  item.className = `event-item severity-${evt.severity || 'info'}`;
  
  const timeStr = new Date(evt.timestamp).toLocaleTimeString();
  item.innerHTML = `
    <span class="event-time">${timeStr}</span>
    <div class="event-content">
      <strong>[${(evt.module || 'engine').toUpperCase()}] ${evt.title}</strong>
      <p>${evt.message}</p>
    </div>
  `;

  if (prepend && elements.globalEventsList.firstChild) {
    elements.globalEventsList.insertBefore(item, elements.globalEventsList.firstChild);
  } else {
    elements.globalEventsList.appendChild(item);
  }
}

// Modal Handlers
function openOptimizerModal(key) {
  const rec = state.optimizer.find(o => o.key === key);
  if (!rec) return;

  state.selectedPatch = {
    namespace: rec.namespace,
    deployment: rec.pod,
    container: rec.container,
    reqCpuMilli: rec.recCpuReqMilli,
    limCpuMilli: rec.recCpuLimMilli,
    reqMemBytes: rec.recMemReqBytes,
    limMemBytes: rec.recMemLimBytes,
  };

  const yamlPreview = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${rec.pod}
  namespace: ${rec.namespace}
spec:
  template:
    spec:
      containers:
      - name: ${rec.container}
        resources:
          requests:
            cpu: "${rec.recCpuReqMilli}m"
            memory: "${Math.round(rec.recMemReqBytes / (1024*1024))}Mi"
          limits:
            cpu: "${rec.recCpuLimMilli}m"
            memory: "${Math.round(rec.recMemLimBytes / (1024*1024))}Mi"`;

  elements.modalPatchPreview.textContent = yamlPreview;
  elements.optimizerModal.classList.add('active');
}

function closeModal() {
  elements.optimizerModal.classList.remove('active');
  state.selectedPatch = null;
}

// Action Triggers
async function triggerHeal(namespace, pod) {
  showToast(`Triggering auto-healer for pod ${pod}...`, 'info');
  try {
    const res = await fetch('/api/healer/remediate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ namespace, pod }),
    });
    const data = await res.json();
    showToast(data.message, 'success');
    loadAllData();
  } catch (err) {
    showToast('Failed to heal pod: ' + err, 'danger');
  }
}

async function triggerRollback(namespace, deployment) {
  showToast(`Initiating rollback for deployment ${deployment}...`, 'warning');
  try {
    const res = await fetch('/api/rollouts/rollback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ namespace, deployment }),
    });
    const data = await res.json();
    showToast(data.message, 'success');
    loadAllData();
  } catch (err) {
    showToast('Rollback failed: ' + err, 'danger');
  }
}

// Toast Notifications
function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  toast.className = `toast text-${type}`;
  toast.textContent = message;
  elements.toastContainer.appendChild(toast);
  setTimeout(() => toast.remove(), 4000);
}
