/**
 * LED Strip Animation Editor
 * Connects to the Go server via WebSocket for live preview.
 * All strip state, timeline, and CRUD are managed here in plain JS.
 */

// ─── State ────────────────────────────────────────────────────────────────────

const app = document.getElementById('app');
const NUM_LEDS = parseInt(app.dataset.numLeds, 10) || 99;

let currentColor = { h: 0, s: 100, v: 100 };  // HSV 0-360, 0-100, 0-100
let activeTool = 'paint';
let power = true;
let brightness = 128;
let serverMode = 'idle';

// Timeline: array of { leds: Uint8Array(NUM_LEDS*3), duration_ms }
let frames = [];
let activeFrameIdx = 0;

// Working copy of the currently displayed frame (Uint8Array RGB flat)
let workingLEDs = new Uint8Array(NUM_LEDS * 3);

// Gradient selection state
let gradientStart = -1;

// ─── WebSocket ────────────────────────────────────────────────────────────────

let ws;
let wsReconnectTimer;

function connectWS() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${proto}://${location.host}/ws`);

  ws.onopen = () => {
    setConnStatus(true);
    clearTimeout(wsReconnectTimer);
  };

  ws.onclose = () => {
    setConnStatus(false);
    wsReconnectTimer = setTimeout(connectWS, 3000);
  };

  ws.onerror = () => ws.close();

  ws.onmessage = (ev) => {
    let msg;
    try { msg = JSON.parse(ev.data); } catch { return; }

    if (msg.type === 'frame') {
      // Server pushed a frame — update strip visualizer (but don't re-push)
      const leds = msg.leds; // [[r,g,b], ...]
      for (let i = 0; i < NUM_LEDS && i < leds.length; i++) {
        setStripLED(i, leds[i][0], leds[i][1], leds[i][2]);
      }
    } else if (msg.type === 'status') {
      serverMode = msg.mode;
      power = msg.power;
      brightness = msg.brightness;
      syncUIFromStatus();
    }
  };
}

function sendWS(obj) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(obj));
  }
}

// Push current working frame to server (live mode)
function pushFrame() {
  const leds = [];
  for (let i = 0; i < NUM_LEDS; i++) {
    leds.push([workingLEDs[i*3], workingLEDs[i*3+1], workingLEDs[i*3+2]]);
  }
  sendWS({ type: 'setFrame', leds });
}

// ─── Strip Visualizer ────────────────────────────────────────────────────────

const stripEl = document.getElementById('strip');
const ledEls = [];

function buildStrip() {
  for (let i = 0; i < NUM_LEDS; i++) {
    const d = document.createElement('div');
    d.className = 'led';
    d.dataset.idx = i;
    d.style.cssText = `
      width: 18px; height: 18px; border-radius: 50%;
      background: #111; border: 1px solid #333;
      cursor: crosshair; flex-shrink: 0;
    `;
    stripEl.appendChild(d);
    ledEls.push(d);
  }

  // Paint on mousedown + drag
  let painting = false;
  stripEl.addEventListener('mousedown', (e) => {
    const idx = ledIdxFromEvent(e);
    if (idx !== -1) {
      painting = true;
      applyTool(idx);
    }
  });
  stripEl.addEventListener('mousemove', (e) => {
    if (!painting) return;
    const idx = ledIdxFromEvent(e);
    if (idx !== -1 && activeTool === 'paint') applyTool(idx);
  });
  window.addEventListener('mouseup', () => { painting = false; });
}

function ledIdxFromEvent(e) {
  const t = e.target.closest('[data-idx]');
  return t ? parseInt(t.dataset.idx, 10) : -1;
}

function setStripLED(i, r, g, b) {
  workingLEDs[i*3] = r;
  workingLEDs[i*3+1] = g;
  workingLEDs[i*3+2] = b;
  if (ledEls[i]) {
    ledEls[i].style.background = `rgb(${r},${g},${b})`;
    ledEls[i].style.borderColor = `rgb(${Math.min(r+30,255)},${Math.min(g+30,255)},${Math.min(b+30,255)})`;
  }
}

function loadFrameIntoStrip(frameIdx) {
  if (!frames[frameIdx]) return;
  const src = frames[frameIdx].leds;
  for (let i = 0; i < NUM_LEDS; i++) {
    setStripLED(i, src[i*3], src[i*3+1], src[i*3+2]);
  }
  pushFrame();
  renderTimeline();
}

// ─── Tools ───────────────────────────────────────────────────────────────────

function applyTool(idx) {
  const [r, g, b] = hsvToRgb(currentColor.h, currentColor.s, currentColor.v);

  switch (activeTool) {
    case 'paint':
      setStripLED(idx, r, g, b);
      saveWorkingToActiveFrame();
      pushFrame();
      break;

    case 'fill':
      for (let i = 0; i < NUM_LEDS; i++) setStripLED(i, r, g, b);
      saveWorkingToActiveFrame();
      pushFrame();
      break;

    case 'gradient':
      if (gradientStart === -1) {
        gradientStart = idx;
        ledEls[idx].style.outline = '2px solid white';
        document.getElementById('gradient-hint').textContent = `Start: LED ${idx} — now click end LED`;
      } else {
        applyGradient(gradientStart, idx);
        ledEls[gradientStart].style.outline = '';
        gradientStart = -1;
        document.getElementById('gradient-hint').textContent = 'Click start LED, then end LED';
        saveWorkingToActiveFrame();
        pushFrame();
      }
      break;

    case 'clear':
      for (let i = 0; i < NUM_LEDS; i++) setStripLED(i, 0, 0, 0);
      saveWorkingToActiveFrame();
      pushFrame();
      break;
  }
}

function applyGradient(startIdx, endIdx) {
  const [r1, g1, b1] = hsvToRgb(currentColor.h, currentColor.s, currentColor.v);
  // Gradient goes from current color to black; for two-color gradient use a second color picker later
  const [r2, g2, b2] = [0, 0, 0]; // end color = off/black as default
  const lo = Math.min(startIdx, endIdx);
  const hi = Math.max(startIdx, endIdx);
  const steps = hi - lo || 1;
  for (let i = lo; i <= hi; i++) {
    const t = (i - lo) / steps;
    setStripLED(i,
      Math.round(r1 + (r2 - r1) * t),
      Math.round(g1 + (g2 - g1) * t),
      Math.round(b1 + (b2 - b1) * t)
    );
  }
}

// ─── Frame Timeline ──────────────────────────────────────────────────────────

const timelineEl = document.getElementById('timeline');

function saveWorkingToActiveFrame() {
  if (!frames[activeFrameIdx]) return;
  frames[activeFrameIdx].leds = workingLEDs.slice();
}

function renderTimeline() {
  timelineEl.innerHTML = '';
  frames.forEach((f, i) => {
    const thumb = document.createElement('div');
    thumb.className = 'frame-thumb';
    thumb.style.cssText = `
      position: relative; width: 48px; height: 48px; border-radius: 4px;
      border: 2px solid ${i === activeFrameIdx ? '#818cf8' : '#374151'};
      background: #111; cursor: pointer; flex-shrink: 0; overflow: hidden;
    `;

    // Mini strip preview (5-pixel blocks)
    const canvas = document.createElement('canvas');
    canvas.width = 48; canvas.height = 48;
    const ctx = canvas.getContext('2d');
    const step = 48 / NUM_LEDS;
    for (let j = 0; j < NUM_LEDS; j++) {
      ctx.fillStyle = `rgb(${f.leds[j*3]},${f.leds[j*3+1]},${f.leds[j*3+2]})`;
      ctx.fillRect(j * step, 0, Math.ceil(step), 48);
    }
    thumb.appendChild(canvas);

    // Frame number label
    const label = document.createElement('div');
    label.textContent = i + 1;
    label.style.cssText = `
      position: absolute; bottom: 1px; right: 3px;
      font-size: 10px; color: rgba(255,255,255,0.7); pointer-events: none;
    `;
    thumb.appendChild(label);

    // Delete button
    const del = document.createElement('button');
    del.textContent = '×';
    del.style.cssText = `
      position: absolute; top: 1px; right: 2px;
      font-size: 11px; color: rgba(255,255,255,0.6);
      background: none; border: none; cursor: pointer; padding: 0; line-height: 1;
    `;
    del.addEventListener('click', (e) => { e.stopPropagation(); deleteFrame(i); });
    thumb.appendChild(del);

    thumb.addEventListener('click', () => {
      saveWorkingToActiveFrame();
      activeFrameIdx = i;
      loadFrameIntoStrip(i);
      document.getElementById('inp-duration').value = f.duration_ms;
    });

    timelineEl.appendChild(thumb);
  });
}

function addFrame() {
  saveWorkingToActiveFrame();
  // New frame inherits current working state (so you can build on it)
  frames.push({ leds: workingLEDs.slice(), duration_ms: parseInt(document.getElementById('inp-duration').value) || 100 });
  activeFrameIdx = frames.length - 1;
  renderTimeline();
}

function deleteFrame(idx) {
  if (frames.length <= 1) {
    // Don't delete the last frame — just clear it
    frames[0].leds = new Uint8Array(NUM_LEDS * 3);
    activeFrameIdx = 0;
    loadFrameIntoStrip(0);
    return;
  }
  frames.splice(idx, 1);
  if (activeFrameIdx >= frames.length) activeFrameIdx = frames.length - 1;
  loadFrameIntoStrip(activeFrameIdx);
}

function clearCurrentFrame() {
  for (let i = 0; i < NUM_LEDS; i++) setStripLED(i, 0, 0, 0);
  saveWorkingToActiveFrame();
  pushFrame();
  renderTimeline();
}

// ─── Local Playback ──────────────────────────────────────────────────────────

let playbackTimer = null;

function startLocalPlayback() {
  stopPlayback();
  const fps = parseInt(document.getElementById('inp-fps').value) || 30;
  const loop = document.getElementById('chk-loop').checked;
  let idx = 0;

  function tick() {
    if (!frames[idx]) { stopPlayback(); return; }
    const f = frames[idx];
    for (let i = 0; i < NUM_LEDS; i++) {
      setStripLED(i, f.leds[i*3], f.leds[i*3+1], f.leds[i*3+2]);
    }
    workingLEDs = f.leds.slice();
    renderTimeline();
    activeFrameIdx = idx;

    // Push to real strip
    const leds = [];
    for (let i = 0; i < NUM_LEDS; i++) {
      leds.push([f.leds[i*3], f.leds[i*3+1], f.leds[i*3+2]]);
    }
    sendWS({ type: 'setFrame', leds });

    const delay = f.duration_ms > 0 ? f.duration_ms : Math.round(1000 / fps);
    idx++;
    if (idx >= frames.length) {
      if (loop) idx = 0; else { stopPlayback(); return; }
    }
    playbackTimer = setTimeout(tick, delay);
  }
  tick();
}

function stopPlayback() {
  clearTimeout(playbackTimer);
  playbackTimer = null;
  // Also stop server-side playback
  fetch('/api/stop', { method: 'POST' }).catch(() => {});
}

// ─── Saved Animations ────────────────────────────────────────────────────────

const animListEl = document.getElementById('anim-list');

async function loadAnimationList() {
  try {
    const res = await fetch('/api/animations');
    const list = await res.json();
    renderAnimationList(list);
  } catch (e) {
    console.error('Failed to load animations', e);
  }
}

function renderAnimationList(list) {
  animListEl.innerHTML = '';
  if (!list || list.length === 0) {
    animListEl.innerHTML = '<p class="text-xs text-gray-600 italic">No saved animations yet.</p>';
    return;
  }
  list.forEach(anim => {
    const row = document.createElement('div');
    row.className = 'flex items-center gap-2 py-1 border-b border-gray-800';
    row.innerHTML = `
      <span class="flex-1 text-sm text-gray-300 truncate">${escHtml(anim.name)}</span>
      <span class="text-xs text-gray-600">${anim.fps}fps${anim.loop ? ' loop' : ''}</span>
      <button data-id="${anim.id}" class="btn-play-saved text-xs px-2 py-0.5 rounded bg-green-800 hover:bg-green-700 text-white">▶ Play</button>
      <button data-id="${anim.id}" class="btn-load-saved text-xs px-2 py-0.5 rounded bg-gray-700 hover:bg-gray-600 text-white">Load</button>
      <button data-id="${anim.id}" class="btn-del-saved text-xs px-2 py-0.5 rounded bg-red-900 hover:bg-red-800 text-white">✕</button>
    `;
    animListEl.appendChild(row);
  });

  animListEl.querySelectorAll('.btn-play-saved').forEach(btn => {
    btn.addEventListener('click', () => playSavedAnimation(parseInt(btn.dataset.id)));
  });
  animListEl.querySelectorAll('.btn-load-saved').forEach(btn => {
    btn.addEventListener('click', () => loadSavedAnimation(parseInt(btn.dataset.id)));
  });
  animListEl.querySelectorAll('.btn-del-saved').forEach(btn => {
    btn.addEventListener('click', () => deleteSavedAnimation(parseInt(btn.dataset.id)));
  });
}

async function saveAnimation() {
  const name = document.getElementById('inp-anim-name').value.trim();
  if (!name) { alert('Enter a name for the animation.'); return; }
  if (frames.length === 0) { alert('Add at least one frame first.'); return; }
  saveWorkingToActiveFrame();

  const fps = parseInt(document.getElementById('inp-fps').value) || 30;
  const loop = document.getElementById('chk-loop').checked;

  const framesPayload = frames.map(f => ({
    leds: ledsToArray(f.leds),
    duration_ms: f.duration_ms,
  }));

  try {
    const res = await fetch('/api/animations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, fps, loop, frames: framesPayload }),
    });
    if (!res.ok) throw new Error(await res.text());
    document.getElementById('inp-anim-name').value = '';
    loadAnimationList();
  } catch (e) {
    alert('Save failed: ' + e.message);
  }
}

async function playSavedAnimation(id) {
  try {
    await fetch(`/api/play/${id}`, { method: 'POST' });
  } catch (e) {
    console.error('Play failed', e);
  }
}

async function loadSavedAnimation(id) {
  try {
    const res = await fetch(`/api/animations/${id}`);
    const anim = await res.json();
    if (!anim.frames || anim.frames.length === 0) return;

    frames = anim.frames.map(f => ({
      leds: arrayToLEDs(f.leds),
      duration_ms: f.duration_ms,
    }));
    document.getElementById('inp-fps').value = anim.fps;
    document.getElementById('chk-loop').checked = anim.loop;
    document.getElementById('inp-anim-name').value = anim.name;
    activeFrameIdx = 0;
    loadFrameIntoStrip(0);
  } catch (e) {
    console.error('Load failed', e);
  }
}

async function deleteSavedAnimation(id) {
  if (!confirm('Delete this animation?')) return;
  try {
    await fetch(`/api/animations/${id}`, { method: 'DELETE' });
    loadAnimationList();
  } catch (e) {
    console.error('Delete failed', e);
  }
}

// ─── Color Picker ────────────────────────────────────────────────────────────

const slHue = document.getElementById('sl-hue');
const slSat = document.getElementById('sl-sat');
const slVal = document.getElementById('sl-val');
const inpHex = document.getElementById('inp-hex');
const colorPreview = document.getElementById('color-preview');

function syncColorUI() {
  const [r, g, b] = hsvToRgb(currentColor.h, currentColor.s, currentColor.v);
  const hex = rgbToHex(r, g, b);
  colorPreview.style.background = hex;
  inpHex.value = hex;

  // Update saturation slider gradient
  const [rl, gl, bl] = hsvToRgb(currentColor.h, 0, currentColor.v);
  const [rr, gr, br] = hsvToRgb(currentColor.h, 100, currentColor.v);
  slSat.style.background = `linear-gradient(to right, ${rgbToHex(rl,gl,bl)}, ${rgbToHex(rr,gr,br)})`;

  // Update value slider gradient
  const [rv, gv, bv] = hsvToRgb(currentColor.h, currentColor.s, 100);
  slVal.style.background = `linear-gradient(to right, #000, ${rgbToHex(rv,gv,bv)})`;
}

slHue.addEventListener('input', () => {
  currentColor.h = parseInt(slHue.value);
  syncColorUI();
});
slSat.addEventListener('input', () => {
  currentColor.s = parseInt(slSat.value);
  syncColorUI();
});
slVal.addEventListener('input', () => {
  currentColor.v = parseInt(slVal.value);
  syncColorUI();
});
inpHex.addEventListener('change', () => {
  const hex = inpHex.value.trim();
  if (/^#[0-9a-fA-F]{6}$/.test(hex)) {
    const [r, g, b] = hexToRgb(hex);
    const [h, s, v] = rgbToHsv(r, g, b);
    currentColor = { h, s, v };
    slHue.value = h; slSat.value = s; slVal.value = v;
    syncColorUI();
  }
});

// ─── Header controls ─────────────────────────────────────────────────────────

const btnPower = document.getElementById('btn-power');
const sliderBrightness = document.getElementById('slider-brightness');
const brightnessVal = document.getElementById('brightness-val');

btnPower.addEventListener('click', async () => {
  power = !power;
  try {
    await fetch('/api/power', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ power: power ? 'on' : 'off' }),
    });
  } catch (e) { console.error(e); }
  syncUIFromStatus();
});

sliderBrightness.addEventListener('input', () => {
  brightness = parseInt(sliderBrightness.value);
  brightnessVal.textContent = brightness;
});
sliderBrightness.addEventListener('change', async () => {
  try {
    await fetch('/api/brightness', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ brightness }),
    });
  } catch (e) { console.error(e); }
});

function syncUIFromStatus() {
  btnPower.textContent = power ? 'Power Off' : 'Power On';
  btnPower.className = power
    ? 'px-3 py-1 rounded text-sm font-medium bg-red-800 hover:bg-red-700 text-white'
    : 'px-3 py-1 rounded text-sm font-medium bg-green-800 hover:bg-green-700 text-white';
  sliderBrightness.value = brightness;
  brightnessVal.textContent = brightness;
}

function setConnStatus(connected) {
  const el = document.getElementById('conn-status');
  el.innerHTML = connected
    ? '<span class="w-2 h-2 rounded-full bg-green-400 inline-block"></span> connected'
    : '<span class="w-2 h-2 rounded-full bg-red-400 inline-block"></span> disconnected';
  el.className = connected
    ? 'flex items-center gap-1 text-xs text-green-400'
    : 'flex items-center gap-1 text-xs text-red-400';
}

// ─── Tool buttons ────────────────────────────────────────────────────────────

document.querySelectorAll('[data-tool]').forEach(btn => {
  btn.className = 'tool-btn px-2 py-1 rounded text-xs font-medium bg-gray-700 hover:bg-gray-600 text-white';
  btn.addEventListener('click', () => {
    activeTool = btn.dataset.tool;
    gradientStart = -1;
    document.querySelectorAll('[data-tool]').forEach(b => {
      b.className = 'tool-btn px-2 py-1 rounded text-xs font-medium bg-gray-700 hover:bg-gray-600 text-white';
    });
    btn.className = 'tool-btn px-2 py-1 rounded text-xs font-medium bg-indigo-600 text-white';

    const hint = document.getElementById('gradient-hint');
    if (activeTool === 'gradient') {
      hint.classList.remove('hidden');
      hint.textContent = 'Click start LED, then end LED';
    } else {
      hint.classList.add('hidden');
    }

    // Fill and clear execute immediately on click
    if (activeTool === 'fill' || activeTool === 'clear') {
      applyTool(0);
    }
  });
});

document.getElementById('btn-add-frame').addEventListener('click', () => {
  addFrame();
  document.getElementById('inp-duration').value = frames[activeFrameIdx]?.duration_ms ?? 100;
});

document.getElementById('btn-clear-frame').addEventListener('click', clearCurrentFrame);

document.getElementById('inp-duration').addEventListener('change', () => {
  if (frames[activeFrameIdx]) {
    frames[activeFrameIdx].duration_ms = parseInt(document.getElementById('inp-duration').value) || 100;
    renderTimeline();
  }
});

document.getElementById('btn-play-local').addEventListener('click', startLocalPlayback);
document.getElementById('btn-stop').addEventListener('click', stopPlayback);
document.getElementById('btn-save-anim').addEventListener('click', saveAnimation);

// ─── Colour math ─────────────────────────────────────────────────────────────

function hsvToRgb(h, s, v) {
  s /= 100; v /= 100;
  if (s === 0) { const vv = Math.round(v * 255); return [vv, vv, vv]; }
  h /= 60;
  const i = Math.floor(h);
  const f = h - i;
  const p = v * (1 - s);
  const q = v * (1 - s * f);
  const t = v * (1 - s * (1 - f));
  const tbl = [[v,t,p],[q,v,p],[p,v,t],[p,q,v],[t,p,v],[v,p,q]];
  const [r, g, b] = tbl[i % 6];
  return [Math.round(r*255), Math.round(g*255), Math.round(b*255)];
}

function rgbToHsv(r, g, b) {
  r /= 255; g /= 255; b /= 255;
  const max = Math.max(r, g, b), min = Math.min(r, g, b), d = max - min;
  let h = 0;
  if (d !== 0) {
    if (max === r) h = ((g - b) / d + 6) % 6;
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    h *= 60;
  }
  return [Math.round(h), Math.round(max === 0 ? 0 : (d / max) * 100), Math.round(max * 100)];
}

function rgbToHex(r, g, b) {
  return '#' + [r, g, b].map(x => x.toString(16).padStart(2, '0')).join('');
}

function hexToRgb(hex) {
  const v = parseInt(hex.slice(1), 16);
  return [(v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff];
}

// ─── LED data helpers ────────────────────────────────────────────────────────

// Convert Uint8Array(numLEDs*3) to [[r,g,b], ...] for JSON
function ledsToArray(flat) {
  const arr = [];
  for (let i = 0; i < NUM_LEDS; i++) {
    arr.push([flat[i*3], flat[i*3+1], flat[i*3+2]]);
  }
  return arr;
}

// Convert [[r,g,b], ...] from API back to Uint8Array
function arrayToLEDs(arr) {
  const flat = new Uint8Array(NUM_LEDS * 3);
  for (let i = 0; i < NUM_LEDS && i < arr.length; i++) {
    flat[i*3]   = arr[i][0];
    flat[i*3+1] = arr[i][1];
    flat[i*3+2] = arr[i][2];
  }
  return flat;
}

function escHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ─── Boot ────────────────────────────────────────────────────────────────────

function init() {
  buildStrip();

  // Start with one empty frame
  frames.push({ leds: new Uint8Array(NUM_LEDS * 3), duration_ms: 100 });
  activeFrameIdx = 0;
  renderTimeline();

  // Activate paint tool
  const paintBtn = document.querySelector('[data-tool="paint"]');
  if (paintBtn) paintBtn.click();

  syncColorUI();
  syncUIFromStatus();
  connectWS();
  loadAnimationList();
}

init();
