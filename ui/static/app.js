(function () {
  const FOLD_KEY = "rackwire.home.folds";
  const foldEls = document.querySelectorAll("details.fold[data-fold]");

  function persistFolds() {
    const state = {};
    foldEls.forEach((el) => {
      state[el.getAttribute("data-fold")] = el.open;
    });
    try {
      localStorage.setItem(FOLD_KEY, JSON.stringify(state));
    } catch (_) {}
  }

  function restoreFolds() {
    let state = null;
    try {
      state = JSON.parse(localStorage.getItem(FOLD_KEY) || "null");
    } catch (_) {}
    if (!state || typeof state !== "object") return;
    foldEls.forEach((el) => {
      const id = el.getAttribute("data-fold");
      if (Object.prototype.hasOwnProperty.call(state, id)) {
        el.open = !!state[id];
      }
    });
  }

  if (foldEls.length) {
    restoreFolds();
    foldEls.forEach((el) => el.addEventListener("toggle", persistFolds));
    // Keep open panels across POST redirects on the home page (e.g. new link).
    document.querySelectorAll("details.fold[data-fold] form").forEach((form) => {
      form.addEventListener("submit", persistFolds);
    });
  } else {
    // Left the home page — next visit uses default open/closed panels.
    try {
      localStorage.removeItem(FOLD_KEY);
    } catch (_) {}
  }

  function applyColorSwatch(select) {
    const row = select.closest("tr");
    if (!row) return;
    const swatch = row.querySelector(".swatch");
    if (!swatch) return;
    const opt = select.selectedOptions[0];
    if (!opt) return;
    const solid = !opt.value || opt.dataset.solid !== "false";
    if (solid) {
      swatch.className = "swatch swatch-solid";
      swatch.style.background = opt.dataset.hex || "#888888";
      swatch.style.removeProperty("--base");
      swatch.style.removeProperty("--stripe");
    } else {
      swatch.className = "swatch swatch-stripe";
      swatch.style.background = "";
      swatch.style.setProperty("--base", opt.dataset.base || "#f5f5f5");
      swatch.style.setProperty("--stripe", opt.dataset.stripe || "#888888");
    }
    const title = opt.value ? opt.value + " " + (opt.textContent || "").trim() : "";
    if (title) swatch.title = title;
  }

  document.addEventListener("change", (e) => {
    const el = e.target;
    if (!(el instanceof HTMLSelectElement)) return;
    if (!el.name || el.name.indexOf("_color") === -1) return;
    applyColorSwatch(el);
  });

  const mapEl = document.getElementById("port-map");
  if (mapEl) {
    let map = {};
    try {
      map = JSON.parse(mapEl.textContent);
    } catch (err) {
      console.error("port-map parse failed", err);
    }

    function fillPorts(deviceSelect, portSelect) {
      const deviceId = deviceSelect.value;
      portSelect.innerHTML = "";
      const placeholder = document.createElement("option");
      placeholder.value = "";
      placeholder.textContent = deviceId ? "Port wählen…" : "Gerät wählen…";
      portSelect.appendChild(placeholder);
      const ports = map[deviceId] || [];
      for (const p of ports) {
        const opt = document.createElement("option");
        opt.value = p.id;
        opt.textContent = p.label || p.id;
        portSelect.appendChild(opt);
      }
    }

    const aDev = document.getElementById("aDeviceId");
    const aPort = document.getElementById("aPortId");
    const bDev = document.getElementById("bDeviceId");
    const bPort = document.getElementById("bPortId");
    if (aDev && aPort) {
      aDev.addEventListener("change", () => fillPorts(aDev, aPort));
    }
    if (bDev && bPort) {
      bDev.addEventListener("change", () => fillPorts(bDev, bPort));
    }
  }

  const patchMap = document.getElementById("patch-map");
  if (!patchMap) return;
  const stage = patchMap.querySelector(".map-stage");
  if (!stage) return;

  function clearActive() {
    stage.classList.remove("is-hovering");
    stage.querySelectorAll(".is-active").forEach((el) => el.classList.remove("is-active"));
  }

  function activate(keys) {
    stage.classList.add("is-hovering");
    keys.forEach((key) => {
      stage.querySelectorAll(`[data-port-key="${CSS.escape(key)}"]`).forEach((el) => {
        el.classList.add("is-active");
      });
      stage.querySelectorAll(".map-curve").forEach((path) => {
        const ends = (path.getAttribute("data-link-ends") || "").split(/\s+/);
        if (ends.includes(key)) path.classList.add("is-active");
      });
    });
  }

  stage.querySelectorAll(".map-port").forEach((port) => {
    port.addEventListener("mouseenter", () => {
      clearActive();
      const key = port.getAttribute("data-port-key");
      if (!key) return;
      const keys = new Set([key]);
      stage.querySelectorAll(".map-curve").forEach((path) => {
        const ends = (path.getAttribute("data-link-ends") || "").split(/\s+/);
        if (ends.includes(key)) ends.forEach((k) => keys.add(k));
      });
      activate([...keys]);
    });
    port.addEventListener("mouseleave", clearActive);
  });

  stage.querySelectorAll(".map-curve").forEach((path) => {
    path.style.pointerEvents = "stroke";
    path.addEventListener("mouseenter", () => {
      clearActive();
      const ends = (path.getAttribute("data-link-ends") || "").split(/\s+/).filter(Boolean);
      activate(ends);
    });
    path.addEventListener("mouseleave", clearActive);
  });
})();
