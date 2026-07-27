(function () {
  const mapEl = document.getElementById("port-map");
  if (mapEl) {
    let map = {};
    try {
      map = JSON.parse(mapEl.textContent);
    } catch (e) {
      console.error("port-map parse failed", e);
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
