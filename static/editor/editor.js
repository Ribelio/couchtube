const state = { channels: [], selectedChannelId: null };
const channelListEl = document.getElementById("channel-list");
const videoListEl = document.getElementById("video-list");
const emptyStateEl = document.getElementById("empty-state");
const channelEditorEl = document.getElementById("channel-editor");
const channelNameInput = document.getElementById("channel-name-input");

async function init() {
  await fetchChannels();
  renderChannelList();
  bindGlobalEvents();
}

async function api(url, options = {}) {
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, ...options });
  if (!res.ok) { const text = await res.text(); throw new Error(text || res.statusText); }
  if (res.status === 204) return null;
  return res.json();
}

async function fetchChannels() {
  const data = await api("/api/editor/channels");
  state.channels = data.channels || [];
}

function renderChannelList() {
  channelListEl.innerHTML = "";
  state.channels.forEach((ch) => {
    const el = document.createElement("div");
    el.className = "channel-item" + (ch.id === state.selectedChannelId ? " selected" : "") + (ch.videos.length === 0 ? " empty" : "");
    el.draggable = true;
    el.dataset.channelId = ch.id;
    el.innerHTML = `<div class="ch-drag-handle">&#x2807;</div><div><div class="ch-name">${esc(ch.name)}</div><div class="ch-count">${ch.videos.length} videos</div></div>`;
    el.onclick = () => selectChannel(ch.id);
    channelListEl.appendChild(el);
  });
  bindChannelDragEvents();
}

function bindChannelDragEvents() {
  let draggedEl = null;
  channelListEl.querySelectorAll(".channel-item").forEach((item) => {
    item.addEventListener("dragstart", (e) => { draggedEl = item; item.classList.add("dragging"); e.dataTransfer.effectAllowed = "move"; });
    item.addEventListener("dragend", () => { item.classList.remove("dragging"); channelListEl.querySelectorAll(".channel-item").forEach((i) => i.classList.remove("drag-over")); draggedEl = null; });
    item.addEventListener("dragover", (e) => { e.preventDefault(); if (draggedEl && draggedEl !== item) item.classList.add("drag-over"); });
    item.addEventListener("dragleave", () => { item.classList.remove("drag-over"); });
    item.addEventListener("drop", (e) => {
      e.preventDefault(); item.classList.remove("drag-over");
      if (!draggedEl || draggedEl === item) return;
      const all = [...channelListEl.querySelectorAll(".channel-item")];
      if (all.indexOf(draggedEl) < all.indexOf(item)) item.after(draggedEl); else item.before(draggedEl);
      reorderChannels([...channelListEl.querySelectorAll(".channel-item")].map((el) => parseInt(el.dataset.channelId)));
    });
  });
}

async function reorderChannels(channelIds) {
  try { await api("/api/editor/channels/reorder", { method: "PUT", body: JSON.stringify({ channelIds }) }); await fetchChannels(); }
  catch (e) { alert("Reorder failed: " + e.message); }
}

function selectChannel(id) {
  state.selectedChannelId = id;
  renderChannelList();
  renderChannelEditor();
}

function renderChannelEditor() {
  const ch = state.channels.find((c) => c.id === state.selectedChannelId);
  if (!ch) { emptyStateEl.style.display = "flex"; channelEditorEl.style.display = "none"; return; }
  emptyStateEl.style.display = "none";
  channelEditorEl.style.display = "flex";
  channelNameInput.value = ch.name;
  renderVideoList(ch.videos);
}

function renderVideoList(videos) {
  videoListEl.innerHTML = "";
  videos.sort((a, b) => a.position - b.position).forEach((v) => {
    const el = document.createElement("div");
    el.className = "video-item";
    el.draggable = true;
    el.dataset.videoId = v.videoId;
    el.innerHTML = `
      <div class="drag-handle">&#x2807;</div>
      <img class="video-thumb" src="https://img.youtube.com/vi/${esc(v.videoId)}/mqdefault.jpg" onerror="this.style.display='none'" />
      <div class="video-details">
        <div class="video-title" data-video-id="${esc(v.videoId)}">${esc(v.videoId)}</div>
        <div class="video-id-small">${esc(v.videoId)}</div>
        <div class="video-controls">
          <label>Start:</label>
          <input type="text" class="input-start time-input" value="${formatTime(v.sectionStart)}" data-video-id="${esc(v.videoId)}" placeholder="0:00" />
          <label>End:</label>
          <input type="text" class="input-end time-input" value="${formatTime(v.sectionEnd)}" data-video-id="${esc(v.videoId)}" placeholder="0:00" />
          <button class="btn btn-ghost btn-sm btn-preview" data-video-id="${esc(v.videoId)}">Preview</button>
          <button class="btn btn-ghost btn-sm btn-update-times" data-video-id="${esc(v.videoId)}">Save</button>
        </div>
      </div>
      <button class="btn-remove" data-video-id="${esc(v.videoId)}">&times;</button>`;
    videoListEl.appendChild(el);
  });
  bindVideoEvents();
  bindDragEvents();
  videos.forEach((v) => {
    fetchVideoTitle(v.videoId).then((title) => {
      if (!title) return;
      const el = videoListEl.querySelector(`.video-title[data-video-id="${v.videoId}"]`);
      if (el) el.textContent = title;
    });
  });
}

function bindGlobalEvents() {
  document.getElementById("btn-new-channel").onclick = createChannel;
  document.getElementById("btn-save-name").onclick = saveChannelName;
  document.getElementById("btn-delete-channel").onclick = deleteChannel;
  document.getElementById("btn-add-video").onclick = addVideo;
  document.getElementById("btn-export").onclick = () => { window.location.href = "/api/editor/export"; };
  document.getElementById("btn-import").onclick = () => document.getElementById("import-file-input").click();
  document.getElementById("import-file-input").onchange = importJSON;
  document.getElementById("btn-import-playlist").onclick = importPlaylist;
  document.getElementById("btn-close-playlist-modal").onclick = () => { document.getElementById("playlist-modal").style.display = "none"; };
  document.getElementById("btn-close-preview").onclick = closePreview;
  document.getElementById("btn-mark-start").onclick = () => markTime("start");
  document.getElementById("btn-mark-end").onclick = () => markTime("end");
  document.getElementById("btn-confirm-playlist").onclick = confirmPlaylistImport;
}

function bindVideoEvents() {
  videoListEl.querySelectorAll(".btn-remove").forEach((b) => { b.onclick = () => removeVideo(b.dataset.videoId); });
  videoListEl.querySelectorAll(".btn-preview").forEach((b) => { b.onclick = () => {
    const vid = b.dataset.videoId;
    const startEl = videoListEl.querySelector(`.input-start[data-video-id="${vid}"]`);
    const endEl = videoListEl.querySelector(`.input-end[data-video-id="${vid}"]`);
    openPreview(vid, startEl ? parseTime(startEl.value) : 0, endEl ? parseTime(endEl.value) : 0);
  }; });
  videoListEl.querySelectorAll(".btn-update-times").forEach((b) => { b.onclick = () => updateVideoTimes(b.dataset.videoId); });
}

function bindDragEvents() {
  let draggedEl = null;
  videoListEl.querySelectorAll(".video-item").forEach((item) => {
    item.addEventListener("dragstart", (e) => { draggedEl = item; item.classList.add("dragging"); e.dataTransfer.effectAllowed = "move"; });
    item.addEventListener("dragend", () => { item.classList.remove("dragging"); videoListEl.querySelectorAll(".video-item").forEach((i) => i.classList.remove("drag-over")); draggedEl = null; });
    item.addEventListener("dragover", (e) => { e.preventDefault(); if (draggedEl && draggedEl !== item) item.classList.add("drag-over"); });
    item.addEventListener("dragleave", () => { item.classList.remove("drag-over"); });
    item.addEventListener("drop", (e) => {
      e.preventDefault(); item.classList.remove("drag-over");
      if (!draggedEl || draggedEl === item) return;
      const all = [...videoListEl.querySelectorAll(".video-item")];
      if (all.indexOf(draggedEl) < all.indexOf(item)) item.after(draggedEl); else item.before(draggedEl);
      reorderVideos([...videoListEl.querySelectorAll(".video-item")].map((el) => el.dataset.videoId));
    });
  });
}

async function createChannel() {
  const name = prompt("Channel name:"); if (!name) return;
  try { const ch = await api("/api/editor/channels", { method: "POST", body: JSON.stringify({ name }) }); await fetchChannels(); selectChannel(ch.id); renderChannelList(); }
  catch (e) { alert("Failed: " + e.message); }
}
async function saveChannelName() {
  const name = channelNameInput.value.trim(); if (!name || !state.selectedChannelId) return;
  try { await api(`/api/editor/channels?channel-id=${state.selectedChannelId}`, { method: "PUT", body: JSON.stringify({ name }) }); await fetchChannels(); renderChannelList(); }
  catch (e) { alert("Failed: " + e.message); }
}
async function deleteChannel() {
  if (!state.selectedChannelId || !confirm("Delete this channel?")) return;
  try { await api(`/api/editor/channels?channel-id=${state.selectedChannelId}`, { method: "DELETE" }); state.selectedChannelId = null; await fetchChannels(); renderChannelList(); renderChannelEditor(); }
  catch (e) { alert("Failed: " + e.message); }
}
function getVideoDuration(videoId) {
  return new Promise((resolve) => {
    const host = document.getElementById("hidden-player-host");
    const div = document.createElement("div");
    div.id = "hidden-yt-player";
    host.appendChild(div);

    let resolved = false;
    const finish = (dur) => {
      if (resolved) return;
      resolved = true;
      p.destroy();
      if (host.contains(div)) host.removeChild(div);
      resolve(dur);
    };

    const tryGetDuration = () => {
      if (resolved) return;
      const dur = Math.floor(p.getDuration());
      if (dur > 0) finish(dur);
    };

    const p = new YT.Player("hidden-yt-player", {
      videoId: videoId,
      playerVars: { autoplay: 1, mute: 1 },
      events: {
        onReady: tryGetDuration,
        onStateChange: tryGetDuration,
      }
    });

    // Timeout fallback
    setTimeout(() => finish(0), 10000);
  });
}

async function addVideo() {
  if (!state.selectedChannelId) return;
  const input = document.getElementById("add-video-input").value.trim();
  const videoId = extractVideoId(input);
  if (!videoId) { alert("Could not extract video ID"); return; }

  const statusEl = document.getElementById("add-video-status");
  const btn = document.getElementById("btn-add-video");
  btn.disabled = true;
  statusEl.textContent = "Fetching duration...";
  statusEl.style.display = "inline";

  let sectionEnd = videoDurations[videoId];
  if (!sectionEnd) {
    sectionEnd = await getVideoDuration(videoId);
    if (sectionEnd > 0) videoDurations[videoId] = sectionEnd;
  }

  btn.disabled = false;
  statusEl.style.display = "none";

  if (!sectionEnd) { alert("Could not determine video duration. Add it manually and set the end time via preview."); return; }

  try {
    await api(`/api/editor/channels/videos?channel-id=${state.selectedChannelId}`, { method: "POST", body: JSON.stringify({ videoId, sectionStart: 0, sectionEnd }) });
    document.getElementById("add-video-input").value = "";
    await fetchChannels(); renderChannelList(); renderChannelEditor();
  } catch (e) { alert("Failed: " + e.message); }
}
function parseTime(str) {
  str = str.trim();
  if (str.includes(":")) {
    const parts = str.split(":").map(p => parseInt(p, 10) || 0);
    if (parts.length === 3) return parts[0] * 3600 + parts[1] * 60 + parts[2];
    return parts[0] * 60 + parts[1];
  }
  return parseInt(str, 10) || 0;
}

async function updateVideoTimes(videoId) {
  if (!state.selectedChannelId) return;
  const startEl = videoListEl.querySelector(`.input-start[data-video-id="${videoId}"]`);
  const endEl = videoListEl.querySelector(`.input-end[data-video-id="${videoId}"]`);
  if (!startEl || !endEl) return;
  const s = parseTime(startEl.value);
  const e = parseTime(endEl.value);
  if (e < s + 1) { alert("End must be at least 1 second after start"); return; }
  const dur = videoDurations[videoId];
  if (dur && e > dur) { alert("End time (" + formatTime(e) + ") exceeds video duration (" + formatTime(dur) + ")"); return; }
  if (dur && s >= dur) { alert("Start time (" + formatTime(s) + ") exceeds video duration (" + formatTime(dur) + ")"); return; }
  try { await api(`/api/editor/channels/videos?channel-id=${state.selectedChannelId}&video-id=${videoId}`, { method: "PUT", body: JSON.stringify({ sectionStart: s, sectionEnd: e }) }); await fetchChannels(); renderChannelEditor(); }
  catch (err) { alert("Failed: " + err.message); }
}
async function removeVideo(videoId) {
  if (!state.selectedChannelId) return;
  try { await api(`/api/editor/channels/videos?channel-id=${state.selectedChannelId}&video-id=${videoId}`, { method: "DELETE" }); await fetchChannels(); renderChannelList(); renderChannelEditor(); }
  catch (e) { alert("Failed: " + e.message); }
}
async function reorderVideos(videoIds) {
  if (!state.selectedChannelId) return;
  try { await api(`/api/editor/channels/videos/reorder?channel-id=${state.selectedChannelId}`, { method: "PUT", body: JSON.stringify({ videoIds }) }); await fetchChannels(); }
  catch (e) { alert("Reorder failed: " + e.message); }
}

async function importJSON(e) {
  const file = e.target.files[0]; if (!file) return;
  try { const text = await file.text(); await api("/api/editor/import", { method: "POST", body: text }); state.selectedChannelId = null; await fetchChannels(); renderChannelList(); renderChannelEditor(); }
  catch (err) { alert("Import failed: " + err.message); }
  e.target.value = "";
}

let pendingPlaylistVideos = [];
async function importPlaylist() {
  const url = document.getElementById("playlist-url-input").value.trim(); if (!url) return;
  try {
    const data = await api("/api/editor/import-playlist", { method: "POST", body: JSON.stringify({ url }) });
    if (!data.videos || data.videos.length === 0) { alert("No videos found"); return; }
    pendingPlaylistVideos = data.videos;
    document.getElementById("playlist-modal-body").innerHTML = data.videos.map((v, i) => `
      <div class="playlist-video-item">
        <input type="checkbox" checked data-index="${i}" />
        <img src="https://img.youtube.com/vi/${esc(v.id)}/default.jpg" />
        <span class="pv-id">${esc(v.id)}</span>
        <label>End (s):</label>
        <input type="number" class="pv-end" data-index="${i}" min="1" placeholder="required" />
      </div>`).join("");
    document.getElementById("playlist-modal").style.display = "flex";
  } catch (e) { alert("Failed: " + e.message); }
}
async function confirmPlaylistImport() {
  if (!state.selectedChannelId) return;
  const toAdd = [];
  document.querySelectorAll(".playlist-video-item").forEach((item, i) => {
    if (item.querySelector('input[type="checkbox"]').checked) {
      const end = parseInt(item.querySelector(".pv-end").value);
      if (end > 0) toAdd.push({ videoId: pendingPlaylistVideos[i].id, sectionStart: 0, sectionEnd: end });
    }
  });
  if (toAdd.length === 0) { alert("No videos selected or end times not set"); return; }
  try {
    for (const v of toAdd) await api(`/api/editor/channels/videos?channel-id=${state.selectedChannelId}`, { method: "POST", body: JSON.stringify(v) });
    document.getElementById("playlist-modal").style.display = "none"; document.getElementById("playlist-url-input").value = "";
    await fetchChannels(); renderChannelList(); renderChannelEditor();
  } catch (e) { alert("Failed: " + e.message); }
}

const videoTitleCache = {};
async function fetchVideoTitle(videoId) {
  if (videoTitleCache[videoId]) return videoTitleCache[videoId];
  try {
    const res = await fetch(`https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=${videoId}&format=json`);
    if (!res.ok) return null;
    const data = await res.json();
    videoTitleCache[videoId] = data.title;
    return data.title;
  } catch { return null; }
}

let previewPlayer = null;
let previewVideoId = null;
let previewDuration = 0;
const videoDurations = {}; // videoId -> duration in seconds

function onYouTubeIframeAPIReady() {}

function formatTime(seconds) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return h + ":" + String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
  return m + ":" + String(s).padStart(2, "0");
}

function openPreview(videoId, startSec, endSec) {
  startSec = startSec || 0;
  endSec = endSec || 0;
  previewVideoId = videoId;

  if (previewPlayer) {
    previewPlayer.destroy();
    previewPlayer = null;
  }

  document.getElementById("mark-start-time").textContent = formatTime(startSec);
  document.getElementById("mark-end-time").textContent = endSec > 0 ? formatTime(endSec) : "--:--";

  previewDuration = 0;

  previewPlayer = new YT.Player("preview-player", {
    videoId: videoId,
    playerVars: { start: startSec, autoplay: 1, rel: 0 },
    events: {
      onReady: function () {
        const dur = Math.floor(previewPlayer.getDuration());
        if (dur > 0) {
          previewDuration = dur;
          videoDurations[videoId] = dur;
        }
      },
      onStateChange: function () {
        // Duration may not be available until playback starts
        if (!previewDuration && previewPlayer.getDuration) {
          const dur = Math.floor(previewPlayer.getDuration());
          if (dur > 0) {
            previewDuration = dur;
            videoDurations[videoId] = dur;
          }
        }
      }
    }
  });

  document.getElementById("preview-modal").style.display = "flex";
}

function closePreview() {
  if (previewPlayer) { previewPlayer.destroy(); previewPlayer = null; }
  const container = document.getElementById("preview-player-container");
  container.textContent = "";
  const div = document.createElement("div");
  div.id = "preview-player";
  container.appendChild(div);
  document.getElementById("preview-modal").style.display = "none";
}

function markTime(which) {
  if (!previewPlayer || typeof previewPlayer.getCurrentTime !== "function") return;
  if (!previewVideoId) return;
  let t = Math.floor(previewPlayer.getCurrentTime());

  if (which === "end" && previewDuration > 0 && t > previewDuration) t = previewDuration;

  const startEl = videoListEl.querySelector(`.input-start[data-video-id="${previewVideoId}"]`);
  const endEl = videoListEl.querySelector(`.input-end[data-video-id="${previewVideoId}"]`);
  if (which === "end" && startEl) {
    const s = parseTime(startEl.value);
    if (t < s + 1) t = s + 1;
  } else if (which === "start" && endEl) {
    const e = parseTime(endEl.value);
    if (e > 0 && t > e - 1) t = e - 1;
  }

  document.getElementById(`mark-${which}-time`).textContent = formatTime(t);

  const el = videoListEl.querySelector(`.input-${which}[data-video-id="${previewVideoId}"]`);
  if (el) el.value = formatTime(t);
}

function extractVideoId(input) {
  input = input.trim();
  if (/^[a-zA-Z0-9_-]{11}$/.test(input)) return input;
  for (const re of [/(?:youtube\.com\/watch\?.*v=)([a-zA-Z0-9_-]{11})/, /(?:youtu\.be\/)([a-zA-Z0-9_-]{11})/, /(?:youtube\.com\/embed\/)([a-zA-Z0-9_-]{11})/, /(?:youtube\.com\/shorts\/)([a-zA-Z0-9_-]{11})/]) {
    const m = input.match(re); if (m) return m[1];
  }
  return null;
}

function esc(s) { const d = document.createElement("div"); d.textContent = s; return d.innerHTML; }

init();
