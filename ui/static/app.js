(function () {
  const mapEl = document.getElementById("port-map");
  if (!mapEl) return;
  let map = {};
  try {
    map = JSON.parse(mapEl.textContent);
  } catch (e) {
    console.error("port-map parse failed", e);
    return;
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
})();
