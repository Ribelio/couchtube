const YOUTUBE_BASE_VIDEO_URL = "https://www.youtube.com/watch?v=";
const IFRAME_API_URL = "https://www.youtube.com/iframe_api";
const BUFFERING_TIMEOUT = 3500;
const CHANNELS_ENDPOINT = "/api/channels";
const CURRENT_VIDEO_ENDPOINT = "/api/current-video";
const CURRENT_VIDEOS_ENDPOINT = "/api/current-videos";
const INVALIDATE_VIDEO_ENDPOINT = "/api/invalidate-video";
const LOAD_DEFAULTS_ENDPOINT = "/api/load-defaults";
const VOLUME_STEPS = 5;
const VOLUME_BAR_TIMEOUT = 2000;
const CHANNEL_NAME_TIMEOUT = 3000;
const INTERVAL_CHECK_MS = 1000;
const DIGIT_BUFFER_TIMEOUT = 1000;
const ERROR_THRESHOLD = 5;
const ERROR_WINDOW_MS = 60000;
const LAST_CHANNEL_KEY = "couchtube_last_channel_id";
const STATUS_TIMEOUT = 1800;
const DRIFT_THRESHOLD = 7;
const SETTINGS_KEY = "couchtube_settings";

const DEFAULT_SETTINGS = {
  showChannelName: false,
  showVideoTitle: true,
  showCaptions: false,
};

const ICONS = {
  power: "/assets/icons/power.svg",
  volume_muted: "/assets/icons/volume_muted.svg",
  volume_high: "/assets/icons/volume_high.svg",
  expand: "/assets/icons/expand.svg",
  contract: "/assets/icons/contract.svg",
};

let _statusTimer = null;
const showStatus = (msg) => {
  const el = document.querySelector("#status");
  if (!el) return;
  el.textContent = msg;
  el.classList.add("visible");
  if (_statusTimer) clearTimeout(_statusTimer);
  _statusTimer = setTimeout(
    () => el.classList.remove("visible"),
    STATUS_TIMEOUT,
  );
};

const loadSettings = () => {
  try {
    const stored = localStorage.getItem(SETTINGS_KEY);
    return stored
      ? { ...DEFAULT_SETTINGS, ...JSON.parse(stored) }
      : { ...DEFAULT_SETTINGS };
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
};

const saveSettings = (settings) => {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
};

const STATE_LABELS = {
  "-1": "UNSTARTED",
  0: "ENDED",
  1: "PLAYING",
  2: "PAUSED",
  3: "BUFFERING",
  5: "CUED",
};

const updateStateReadout = (playerState) => {
  const el = document.querySelector("#state-readout");
  if (!el) return;
  el.textContent = STATE_LABELS[String(playerState)] ?? "";
};

const showSmpte = () =>
  document.querySelector("#smpte-bars")?.classList.add("active");
const hideSmpte = () =>
  document.querySelector("#smpte-bars")?.classList.remove("active");

const openSettingsPanel = () =>
  document.querySelector("#settings-modal-container")?.classList.add("active");

const closeSettingsPanel = () =>
  document
    .querySelector("#settings-modal-container")
    ?.classList.remove("active");

const loadYouTubeAPI = (onReady) => {
  const tag = document.createElement("script");
  tag.src = IFRAME_API_URL;
  document.head.appendChild(tag);
  window.onYouTubeIframeAPIReady = onReady;
};

const initializePlayer = (playerElementId, onReady, onStateChange, onError) => {
  return new YT.Player(playerElementId, {
    width: "100%",
    height: "100%",
    autoplay: 1,
    events: { onReady, onStateChange, onError },
    playerVars: {
      mute: 1,
      controls: 0,
      modestbranding: 1,
      disablekb: 1,
      fs: 0,
      iv_load_policy: 3,
      rel: 0,
      enablejsapi: 1,
      autoplay: 1,
      showinfo: 0,
    },
  });
};

const fetchChannels = async () => {
  const res = await fetch(CHANNELS_ENDPOINT);
  const data = await res.json();
  return data.channels || [];
};

const fetchCurrentVideo = async (channelId, videoId) => {
  const url = `${CURRENT_VIDEO_ENDPOINT}?channel-id=${channelId}${
    videoId ? `&video-id=${videoId}` : ""
  }`;
  const res = await fetch(url);
  const data = await res.json();
  return data.video || null;
};

const fetchCurrentVideos = async () => {
  const res = await fetch(CURRENT_VIDEOS_ENDPOINT);
  const data = await res.json();
  return data.channels || [];
};

const handleUnavailableVideo = async (state) => {
  const url = `${INVALIDATE_VIDEO_ENDPOINT}?video-id=${state.currentVideo.id}`;
  const res = await fetch(url, {
    method: "DELETE",
  });
  const data = await res.json();

  if (data.success) {
    const { newChannel, newVideo } = await changeChannel(state, 0);
    state.currentChannel = newChannel;
    state.currentVideo = newVideo;
  }
};

const showBuffering = () => {
  document.querySelector("#buffer-gif")?.classList.add("active");
};

const hideBuffering = () => {
  document.querySelector("#buffer-gif")?.classList.remove("active");
};

const deactiveBuffering = (state) => {
  const { player } = state;
  const shouldUnmute = state.isInteracted && !state.isMuted;

  // make sure that the player is muted so that it does not play sound
  // when buffering is active
  player.mute();

  setTimeout(() => {
    hideBuffering();
    if (shouldUnmute) {
      player.unMute();
      state.isMuted = false;
    }
    setControlIcon("control-power", ICONS.power, false);
  }, BUFFERING_TIMEOUT);
};

const setControlIcon = (iconId, iconSrc, isActive) => {
  const iconElement = document.querySelector(`#${iconId} .control-icon`);
  if (iconElement) {
    iconElement.src = iconSrc;
    iconElement.classList.toggle("red", isActive);
  }
};

const toggleMute = (player, isMuted) => {
  player[isMuted ? "unMute" : "mute"]();
  setControlIcon(
    "control-mute",
    isMuted ? ICONS.volume_high : ICONS.volume_muted,
    !isMuted,
  );
  showStatus(isMuted ? "UNMUTED" : "MUTED");
  return !isMuted;
};

const updateVideoLink = (state) => {
  const videoLinkContainer = document.querySelector("#video-link");
  const videoLinkTitle = document.querySelector("#video-link-title");

  if (state.currentVideoName) {
    videoLinkTitle.innerHTML = state.currentVideoName;
    videoLinkContainer.classList.add("active");
  } else {
    videoLinkContainer.classList.remove("active");
  }
};

const updateVolumeBar = (currentVolume) => {
  const volumeBar = document.querySelector("#volume-bar");
  if (!volumeBar) return;

  const maxBars = 100 / VOLUME_STEPS;
  const currentStep = Math.ceil(currentVolume / VOLUME_STEPS);

  volumeBar.classList.add("active");
  volumeBar.innerHTML = Array.from(
    { length: maxBars },
    (_, index) =>
      `<div class="volume-bar-step ${
        index < currentStep ? "active" : ""
      }"></div>`,
  ).join("");

  setTimeout(() => volumeBar.classList.remove("active"), VOLUME_BAR_TIMEOUT);
};

const saveLastChannel = (channelId) => {
  try {
    localStorage.setItem(LAST_CHANNEL_KEY, String(channelId));
  } catch (_) {}
};

const getLastChannel = (channels) => {
  try {
    const stored = localStorage.getItem(LAST_CHANNEL_KEY);
    if (stored) {
      const found = channels.find((c) => String(c.id) === stored);
      if (found) return found;
    }
  } catch (_) {}
  return null;
};

const updateChannelList = (state, channels) => {
  const channelList = document.querySelector("#channel-list");
  if (!channelList) return;

  const channelListItems = channels.map((channel) => {
    const channelListItem = document.createElement("div");
    channelListItem.classList.add("channel-list-item");

    if (channel.id === state.currentChannel.id) {
      channelListItem.classList.add("active");
    }

    channelListItem.innerHTML = `${channel.id.toString().padStart(2, "0")} - ${
      channel.name
    }`;
    channelListItem.addEventListener("click", async () => {
      const { newChannel, newVideo } = await jumpToChannel(state, channel.id);

      state.currentChannel = newChannel;
      state.currentVideo = newVideo;
      updateChannelName(newChannel);
    });
    return channelListItem;
  });

  channelList.innerHTML = "";
  channelList.append(...channelListItems);
};

const toggleFullscreen = (state) => {
  const playerElement = document.body;

  if (state.isFullscreen) {
    document.exitFullscreen();
    state.isFullscreen = false;
  } else {
    const requestFullScreen =
      playerElement.requestFullscreen ||
      playerElement.mozRequestFullScreen ||
      playerElement.webkitRequestFullScreen ||
      playerElement.msRequestFullscreen;
    if (requestFullScreen) {
      requestFullScreen.call(playerElement);
    }
    state.isFullscreen = true;
  }
};

const toggleControlGroup = (isMinimized) => {
  const controlGroup = document.querySelector("#controls");
  const minimizeIcon = document.querySelector(
    "#control-minimize .control-icon",
  );

  isMinimized = !isMinimized;
  controlGroup.classList.toggle("minimized", isMinimized);
  minimizeIcon.src = isMinimized ? ICONS.expand : ICONS.contract;

  return isMinimized;
};

const updateChannelName = (channel, settings) => {
  const channelId = channel.id.toString().padStart(2, "0");
  const channelName = `${channelId} - ${channel.name}`;
  const channelNameElement = document.querySelector("#channel-name");
  if (channelNameElement) {
    channelNameElement.innerHTML = channelName;
    channelNameElement.classList.add("active");

    if (!settings?.showChannelName) {
      setTimeout(() => {
        channelNameElement.classList.remove("active");
      }, CHANNEL_NAME_TIMEOUT);
    }
  }
};

const updateVideoTitle = (state) => {
  const el = document.querySelector("#video-title");
  if (!el) return;
  if (state.settings?.showVideoTitle && state.currentVideoName) {
    el.textContent = state.currentVideoName;
    el.classList.add("active");
  } else {
    el.classList.remove("active");
  }
};

const applySettings = (state) => {
  const { settings } = state;

  // Channel name persistence
  const channelNameEl = document.querySelector("#channel-name");
  if (channelNameEl && state.currentChannel) {
    if (settings.showChannelName) {
      const channelId = state.currentChannel.id.toString().padStart(2, "0");
      channelNameEl.innerHTML = `${channelId} - ${state.currentChannel.name}`;
      channelNameEl.classList.add("active");
    } else {
      channelNameEl.classList.remove("active");
    }
  }

  // Video title
  updateVideoTitle(state);

  // Captions
  if (state.player?.setOption) {
    state.player.setOption(
      "captions",
      "track",
      settings.showCaptions ? { languageCode: "en" } : {},
    );
  }

  // Sync checkboxes
  const cnCheck = document.querySelector("#setting-channel-name");
  const vtCheck = document.querySelector("#setting-video-title");
  const ccCheck = document.querySelector("#setting-captions");
  if (cnCheck) cnCheck.checked = settings.showChannelName;
  if (vtCheck) vtCheck.checked = settings.showVideoTitle;
  if (ccCheck) ccCheck.checked = settings.showCaptions;
};

const togglePlayPause = (player, isPlaying) => {
  isPlaying ? player.pauseVideo() : player.playVideo();
  setControlIcon("control-power", ICONS.power, !isPlaying);
  showStatus(isPlaying ? "⏸ PAUSED" : "▶ PLAYING");
  return !isPlaying;
};

const cueVideo = (state, video) => {
  const { player } = state;
  const videoId = video.id;
  if (!videoId) return;
  const shouldUnmute = state.isInteracted && !state.isMuted;
  player.cueVideoById({ videoId, startSeconds: video.sectionStart });
  player.setPlaybackRate(1);
  player.mute();
  player.playVideo();
  state.videoLoadedAt = Date.now() / 1000;
  if (shouldUnmute) {
    player.unMute();
    state.isMuted = false;
  }
};

// Switch to the next or previous channel
const changeChannel = async (state, offset) => {
  const { channels, currentChannel } = state;
  const currentIndex = channels.findIndex(
    (channel) => channel.id === currentChannel.id,
  );
  const newIndex = (currentIndex + offset + channels.length) % channels.length;
  const newChannel = channels[newIndex];
  const newVideo = await fetchCurrentVideo(newChannel.id);
  if (newVideo) {
    saveLastChannel(newChannel.id);
    cueVideo(state, newVideo);
    if (offset !== 0)
      showStatus((offset > 0 ? "▲ " : "▼ ") + newChannel.name.toUpperCase());
  }
  return { newChannel, newVideo };
};

const jumpToChannel = async (state, channelId) => {
  const newChannel = state.channels.find((channel) => channel.id === channelId);
  const newVideo = await fetchCurrentVideo(newChannel.id);
  if (newVideo) {
    saveLastChannel(newChannel.id);
    cueVideo(state, newVideo);
    showStatus("→ " + newChannel.name.toUpperCase());
  }
  return { newChannel, newVideo };
};

const closeChannelList = (state) => {
  document.querySelector("#channel-list-overlay").classList.remove("active");
  state.isOverlayOpen = false;
};

const renderOverlayChannelList = async (state) => {
  const listContainer = document.querySelector("#overlay-channel-list");
  if (!listContainer) return;

  const channelsWithVideo = await fetchCurrentVideos();
  const items = channelsWithVideo.map((entry, index) => {
    const item = document.createElement("div");
    item.classList.add("overlay-channel-item");
    if (entry.channel.id === state.currentChannel.id) {
      item.classList.add("active");
    }
    if (index === state.overlayHighlightIndex) {
      item.classList.add("highlighted");
    }
    item.dataset.index = index;
    item.dataset.channelId = entry.channel.id;
    item.innerHTML = `
      <span class="channel-id">${String(entry.channel.id).padStart(2, "0")}</span>
      <span class="channel-name">${entry.channel.name}</span>
      <span class="channel-now-playing">${entry.video ? entry.video.id : ""}</span>
    `;
    item.addEventListener("click", async () => {
      await selectOverlayChannel(state, entry.channel.id);
    });
    item.addEventListener("mouseenter", () => {
      document
        .querySelectorAll(".overlay-channel-item.highlighted")
        .forEach((el) => el.classList.remove("highlighted"));
      item.classList.add("highlighted");
      state.overlayHighlightIndex = index;
    });
    return item;
  });

  listContainer.innerHTML = "";
  listContainer.append(...items);
};

const selectOverlayChannel = async (state, channelId) => {
  const { newChannel, newVideo } = await jumpToChannel(state, channelId);
  if (newChannel) {
    state.currentChannel = newChannel;
    state.currentVideo = newVideo;
    updateChannelName(newChannel, state.settings);
  }
  closeChannelList(state);
};

const toggleChannelList = async (state) => {
  const overlay = document.querySelector("#channel-list-overlay");
  if (state.isOverlayOpen) {
    closeChannelList(state);
    return;
  }

  state.isOverlayOpen = true;
  const currentIndex = state.channels.findIndex(
    (c) => c.id === state.currentChannel.id,
  );
  state.overlayHighlightIndex = currentIndex >= 0 ? currentIndex : 0;
  overlay.classList.add("active");
  await renderOverlayChannelList(state);
  const highlighted = overlay.querySelector(
    ".overlay-channel-item.highlighted",
  );
  if (highlighted) highlighted.scrollIntoView({ block: "nearest" });
};

const closeInfoModal = () => {
  const infoPopup = document.querySelector("#info-modal-container");
  infoPopup.classList.remove("active");
};

const toggleInfoModal = () => {
  const infoPopup = document.querySelector("#info-modal-container");
  infoPopup.classList.toggle("active");
};

const fetchConfig = async () => {
  try {
    const response = await fetch("/api/config");
    const data = await response.json();

    return data;
  } catch (error) {
    console.error("Failed to fetch config:", error);
  }
};

const displayMessage = (message, title = " ") => {
  const messageModal = document.querySelector("#message-modal");
  const messageTitle = document.querySelector("#message-modal-title-content");

  if (title) {
    messageTitle.innerHTML = title;
  }

  const messageContent = document.querySelector("#message-modal-content");
  messageContent.innerHTML = message;

  messageModal.classList.add("active");
};

const showWelcomePopup = (editorMode) => {
  const container = document.querySelector("#welcome-modal-container");
  container.classList.add("active");

  const loadBtn = document.querySelector("#welcome-load-defaults");
  if (editorMode === "full") {
    loadBtn.addEventListener("click", async () => {
      loadBtn.disabled = true;
      loadBtn.textContent = "Loading...";
      const res = await fetch(LOAD_DEFAULTS_ENDPOINT, { method: "POST" });
      const data = await res.json();
      if (data.success) {
        location.reload();
      }
    });
  } else {
    loadBtn.disabled = true;
  }

  const editorBtn = document.querySelector("#welcome-go-editor");
  const editorNote = document.querySelector("#welcome-editor-note");
  if (editorMode !== "off") {
    editorBtn.addEventListener("click", () => {
      window.location.href = "/editor/";
    });
  } else {
    editorBtn.disabled = true;
    editorNote.style.display = "block";
  }
};

const closeMessageModal = () => {
  const messageModal = document.querySelector("#message-modal");
  messageModal.classList.remove("active");
};

const navigateOverlay = (state, direction) => {
  const items = document.querySelectorAll(".overlay-channel-item");
  if (!items.length) return;

  items.forEach((el) => el.classList.remove("highlighted"));
  state.overlayHighlightIndex =
    (state.overlayHighlightIndex + direction + items.length) % items.length;
  items[state.overlayHighlightIndex].classList.add("highlighted");
  items[state.overlayHighlightIndex].scrollIntoView({ block: "nearest" });
};

const handleDigitInput = (state, digit) => {
  if (state.isOverlayOpen) return;

  if (state.digitTimer) clearTimeout(state.digitTimer);
  state.digitBuffer += digit;
  showStatus("CH " + state.digitBuffer + "▌");

  const num = parseInt(state.digitBuffer, 10);
  const targetIndex = num - 1;

  if (targetIndex >= 0 && targetIndex < state.channels.length) {
    const targetChannel = state.channels[targetIndex];
    jumpToChannel(state, targetChannel.id).then(({ newChannel, newVideo }) => {
      if (newChannel) {
        state.currentChannel = newChannel;
        state.currentVideo = newVideo;
        updateChannelName(newChannel);
      }
    });
  }

  state.digitTimer = setTimeout(() => {
    state.digitBuffer = "";
    state.digitTimer = null;
  }, DIGIT_BUFFER_TIMEOUT);
};

const addEventListeners = (state) => {
  // Add Event Listeners for all buttons
  const controls = {
    power: () => {
      state.isPlaying = togglePlayPause(state.player, state.isPlaying);
    },
    mute: () => {
      state.isMuted = toggleMute(state.player, state.isMuted);
    },
    chup: async () => {
      const { newChannel, newVideo } = await changeChannel(state, 1);
      state.currentChannel = newChannel;
      state.currentVideo = newVideo;
      updateChannelName(newChannel, state.settings);
    },
    chdown: async () => {
      const { newChannel, newVideo } = await changeChannel(state, -1);
      state.currentChannel = newChannel;
      state.currentVideo = newVideo;
      updateChannelName(newChannel, state.settings);
    },
    volup: () => {
      const currentVolume = state.player.getVolume();
      const newVolume = Math.min(currentVolume + VOLUME_STEPS, 100);
      state.player.setVolume(newVolume);
      updateVolumeBar(newVolume);
      state.player.unMute();
      state.isMuted = false;
    },
    voldown: () => {
      const currentVolume = state.player.getVolume();
      const newVolume = Math.max(currentVolume - VOLUME_STEPS, 0);
      state.player.setVolume(newVolume);
      updateVolumeBar(newVolume);
      state.player.unMute();
      state.isMuted = false;
    },
    fullscreen: () => {
      toggleFullscreen(state);
    },
    minimize: () => {
      state.isControlGroupMinimized = toggleControlGroup(
        state.isControlGroupMinimized,
      );
    },
    info: () => {
      toggleInfoModal();
    },
    settings: () => {
      openSettingsPanel();
    },
  };

  for (const [control, handler] of Object.entries(controls)) {
    document
      .querySelector(`#control-${control}`)
      ?.addEventListener("click", () => {
        handler();
        state.isInteracted = true;
      });
  }

  // other events
  document
    .querySelector("#info-modal-close-button")
    .addEventListener("click", () => {
      closeInfoModal();
    });

  document
    .querySelector("#message-modal-close-button")
    .addEventListener("click", () => {
      closeMessageModal();
    });

  document.querySelector("#video-link").addEventListener("click", () => {
    window.open(YOUTUBE_BASE_VIDEO_URL + state.currentVideo.id, "_blank");
  });

  document
    .querySelector("#info-modal-container")
    .addEventListener("click", (event) => {
      if (event.target === event.currentTarget) closeInfoModal();
    });

  document
    .querySelector("#channel-list-overlay")
    .addEventListener("click", async (event) => {
      if (event.target === event.currentTarget) {
        closeChannelList(state);
      }
    });

  document.addEventListener("keydown", (event) => {
    if (state.isOverlayOpen) {
      switch (event.key) {
        case "ArrowUp":
          event.preventDefault();
          navigateOverlay(state, -1);
          break;
        case "ArrowDown":
          event.preventDefault();
          navigateOverlay(state, 1);
          break;
        case "Enter":
          event.preventDefault();
          const highlighted = document.querySelector(
            ".overlay-channel-item.highlighted",
          );
          if (highlighted) {
            const channelId = parseInt(highlighted.dataset.channelId, 10);
            selectOverlayChannel(state, channelId);
          }
          break;
        case "Escape":
        case " ":
          event.preventDefault();
          closeChannelList(state);
          break;
      }
      return;
    }

    switch (event.key) {
      case "ArrowLeft":
        changeChannel(state, -1).then(({ newChannel, newVideo }) => {
          state.currentChannel = newChannel;
          state.currentVideo = newVideo;
          updateChannelName(newChannel, state.settings);
        });
        break;

      case "ArrowRight":
        changeChannel(state, 1).then(({ newChannel, newVideo }) => {
          state.currentChannel = newChannel;
          state.currentVideo = newVideo;
          updateChannelName(newChannel, state.settings);
        });
        break;

      case "ArrowUp":
        event.preventDefault();
        const volUp = state.player.getVolume();
        const newVolUp = Math.min(volUp + VOLUME_STEPS, 100);
        state.player.setVolume(newVolUp);
        updateVolumeBar(newVolUp);
        state.player.unMute();
        state.isMuted = false;
        break;

      case "ArrowDown":
        event.preventDefault();
        const volDown = state.player.getVolume();
        const newVolDown = Math.max(volDown - VOLUME_STEPS, 0);
        state.player.setVolume(newVolDown);
        updateVolumeBar(newVolDown);
        state.player.unMute();
        state.isMuted = false;
        break;

      case " ":
        event.preventDefault();
        toggleChannelList(state);
        break;

      case "p":
        state.isPlaying = togglePlayPause(state.player, state.isPlaying);
        break;

      case "m":
        state.isMuted = toggleMute(state.player, state.isMuted);
        break;

      default:
        if (event.key >= "0" && event.key <= "9") {
          handleDigitInput(state, event.key);
        }
        break;
    }
  });

  document.addEventListener("DOMContentLoaded", fetchConfig);

  // Settings panel
  document
    .querySelector("#settings-modal-close-button")
    ?.addEventListener("click", closeSettingsPanel);
  document
    .querySelector("#settings-modal-container")
    ?.addEventListener("click", (e) => {
      if (e.target === e.currentTarget) closeSettingsPanel();
    });

  document
    .querySelector("#setting-channel-name")
    ?.addEventListener("change", (e) => {
      state.settings.showChannelName = e.target.checked;
      saveSettings(state.settings);
      applySettings(state);
    });
  document
    .querySelector("#setting-video-title")
    ?.addEventListener("change", (e) => {
      state.settings.showVideoTitle = e.target.checked;
      saveSettings(state.settings);
      applySettings(state);
    });
  document
    .querySelector("#setting-captions")
    ?.addEventListener("change", (e) => {
      state.settings.showCaptions = e.target.checked;
      saveSettings(state.settings);
      applySettings(state);
    });
  document
    .querySelector("#settings-editor-link")
    ?.addEventListener("click", () => {
      window.location.href = "/editor/";
    });
};

const initApp = async (playerElementId) => {
  const channels = await fetchChannels();

  if (channels.length === 0) {
    showSmpte();
    const config = await fetchConfig();
    if (config?.editorMode && config.editorMode !== "off") {
      const settingsBtn = document.querySelector("#control-settings");
      settingsBtn.addEventListener("click", () => {
        window.location.href = "/editor/";
      });
    }
    showWelcomePopup(config?.editorMode || "off");
    return;
  }

  const initialChannel =
    getLastChannel(channels) ||
    channels[Math.floor(Math.random() * channels.length)];

  const state = {
    player: null,
    isPlaying: false,
    isMuted: true,
    isFullscreen: false,
    isControlGroupMinimized: false,
    currentChannel: initialChannel,
    currentVideo: null,
    channels,
    isInteracted: false,
    currentVideoName: "",
    videoCheckIntervalId: null,
    isOverlayOpen: false,
    overlayHighlightIndex: 0,
    digitBuffer: "",
    digitTimer: null,
    errorTimestamps: [],
    settings: loadSettings(),
    videoLoadedAt: null,
  };

  addEventListeners(state);
  showSmpte();
  fetchConfig().then((config) => {
    state.editorMode = config.editorMode;
    if (config.editorMode && config.editorMode !== "off") {
      const editorLink = document.querySelector("#settings-editor-link");
      if (editorLink) editorLink.style.display = "";
    }
  });

  const onReady = async () => {
    const initialVideo = await fetchCurrentVideo(state.currentChannel.id);
    if (initialVideo) {
      state.currentVideo = initialVideo;
      cueVideo(state, initialVideo);
    }
    applySettings(state);
  };

  const onStateChange = ({ target, data }) => {
    state.isPlaying = data === YT.PlayerState.PLAYING;
    state.isMuted = target.isMuted();
    state.currentVideoName = target.getVideoData().title;

    updateStateReadout(data);
    updateVideoTitle(state);
    updateVideoLink(state);
    updateChannelList(state, channels);

    if (state.isMuted) {
      setControlIcon("control-mute", ICONS.volume_muted, true);
    }

    if (data === YT.PlayerState.PLAYING) {
      state.errorTimestamps = [];
      hideSmpte();
    }

    if (
      (data === YT.PlayerState.PLAYING || data === YT.PlayerState.ENDED) &&
      state.currentVideo
    ) {
      deactiveBuffering(state);

      if (state.videoCheckIntervalId) {
        clearInterval(state.videoCheckIntervalId);
      }
      state.videoCheckIntervalId = setInterval(async () => {
        const currentTime = state.player.getCurrentTime();

        // Drift re-sync: if player time deviates more than DRIFT_THRESHOLD seconds
        // from the expected wall-clock-based position, seek back into sync.
        if (state.videoLoadedAt && state.currentVideo) {
          const elapsed = Date.now() / 1000 - state.videoLoadedAt;
          const expectedTime = state.currentVideo.sectionStart + elapsed;
          if (Math.abs(currentTime - expectedTime) > DRIFT_THRESHOLD) {
            state.player.seekTo(expectedTime, true);
            showStatus("↻ RESYNCED");
          }
        }

        if (
          currentTime >= state.currentVideo.sectionEnd ||
          data === YT.PlayerState.ENDED
        ) {
          clearInterval(state.videoCheckIntervalId);
          state.videoCheckIntervalId = null;
          const { newChannel, newVideo } = await changeChannel(state, 0);
          state.currentChannel = newChannel;
          state.currentVideo = newVideo;
        }
      }, INTERVAL_CHECK_MS);
    } else if (
      data === YT.PlayerState.CUED ||
      data === YT.PlayerState.UNSTARTED
    ) {
      state.player.playVideo();
    } else {
      showBuffering();

      setControlIcon("control-power", ICONS.power, true);
    }
  };

  const onError = ({ data: errorCode }) => {
    const now = Date.now();
    state.errorTimestamps = state.errorTimestamps.filter(
      (t) => now - t < ERROR_WINDOW_MS,
    );
    state.errorTimestamps.push(now);

    if (state.errorTimestamps.length > ERROR_THRESHOLD) {
      console.warn(
        "Too many video errors — likely YouTube rate limiting. Skipping deletion.",
      );
      displayMessage(
        "YouTube appears to be rate-limiting playback. Videos are being skipped<br>temporarily to prevent data loss. Try again later.",
        "⚠ Rate Limit Detected",
      );
      return;
    }

    switch (errorCode) {
      case 100:
        console.error(
          "Error code:",
          errorCode,
          "Video is unavailable: removed or marked as private.",
        );
        handleUnavailableVideo(state);
        break;
      case 101:
      case 150:
        console.error("Error code:", errorCode, "Video cannot be embedded.");
        handleUnavailableVideo(state);
        break;
      default:
        console.error(
          "Error code:",
          errorCode,
          "An unknown error occurred with the video.",
        );
    }
  };

  loadYouTubeAPI(() => {
    state.player = initializePlayer(
      playerElementId,
      onReady,
      onStateChange,
      onError,
    );
  });
};

initApp("player");
