"use strict";

// Groepenkiezer voor het beheer: gekozen groepen als chips, plus een invoerveld met suggesties
// van bestaande groepen. Enter of komma voegt de getypte naam toe (spaties mogen), × haalt weg.
// Een naam die nog niet bestaat wordt bij het opslaan een nieuwe groep.
// Gebruikt esc uit app.js en view uit manage.js.

const groupState = {}; // per kiezer ("new", "edit", "import"): de gekozen namen

// knownGroups telt de groepen in de woordenlijst, voor suggesties en het filter
function knownGroups() {
  const counts = new Map();
  for (const e of view.entries) for (const g of e.groups || []) counts.set(g, (counts.get(g) || 0) + 1);
  return [...counts].map(([name, count]) => ({ name, count }))
    .sort((a, b) => a.name.localeCompare(b.name, "nl", { sensitivity: "base" }));
}

function groupDatalistHTML() {
  return `<datalist id="g-list">${knownGroups().map((g) => `<option value="${esc(g.name)}">`).join("")}</datalist>`;
}

function groupPickerHTML(id, label = "Groepen") {
  return `
    <div class="m-field">
      <span>${label}</span>
      <div class="g-picker" data-picker="${id}">
        <span class="g-chips">${groupChipsHTML(id)}</span>
        <input list="g-list" data-group-input placeholder="Groep toevoegen…" autocomplete="off"
               aria-label="${esc(label)}: typ een naam en druk op Enter">
      </div>
      <small class="muted">Kies een bestaande groep of typ een nieuwe naam; Enter of komma voegt toe.</small>
    </div>`;
}

function groupChipsHTML(id) {
  return (groupState[id] || []).map((g) => `
    <span class="g-chip">${esc(g)}<button type="button" data-remove-group="${esc(g)}" aria-label="${esc(g)} verwijderen">×</button></span>`).join("");
}

function bindGroupPicker(id) {
  const picker = manageEl.querySelector(`[data-picker="${id}"]`);
  if (!picker) return;
  const input = picker.querySelector("[data-group-input]");
  const chips = picker.querySelector(".g-chips");

  const redraw = () => {
    chips.innerHTML = groupChipsHTML(id);
    chips.querySelectorAll("[data-remove-group]").forEach((b) =>
      b.addEventListener("click", () => {
        groupState[id] = groupState[id].filter((g) => g !== b.dataset.removeGroup);
        redraw();
        input.focus();
      }));
  };
  const add = (raw) => {
    for (const part of raw.split(",")) {
      let name = part.trim().replace(/\s+/g, " ");
      if (!name) continue;
      // Bestaande groep met andere hoofdletters: die gebruiken
      name = knownGroups().find((g) => g.name.toLowerCase() === name.toLowerCase())?.name ?? name;
      const list = (groupState[id] ??= []);
      if (!list.some((g) => g.toLowerCase() === name.toLowerCase())) list.push(name);
    }
    input.value = "";
    redraw();
  };

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault(); // niet het formulier versturen
      add(input.value);
    } else if (e.key === "Backspace" && !input.value && groupState[id]?.length) {
      groupState[id].pop();
      redraw();
    }
  });
  // Gekozen uit de suggestielijst
  input.addEventListener("change", () => {
    if (knownGroups().some((g) => g.name === input.value)) add(input.value);
  });
  // Getypt maar niet bevestigd: bij het verlaten van het veld toch meenemen
  input.addEventListener("blur", () => input.value.trim() && add(input.value));
  redraw();
}

// pendingGroups neemt ook een nog niet bevestigde getypte naam mee
function pendingGroups(id) {
  const input = manageEl.querySelector(`[data-picker="${id}"] [data-group-input]`);
  const typed = input?.value.trim();
  return [...(groupState[id] || []), ...(typed ? typed.split(",").map((s) => s.trim()).filter(Boolean) : [])];
}
