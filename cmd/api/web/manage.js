"use strict";

// Beheer: zoeken, bladeren, bewerken en toevoegen van concepten.
// Gebruikt de helpers uit app.js (esc, highlight, relative, days, LANG, ACCENTS, renderStats, act).

const manageEl = document.getElementById("manage");
const toastEl = document.getElementById("toast");

const view = {
  entries: [],
  query: "",
  direction: "all", // all | nl-hu | hu-nl
  onlyMeanings: false,
  onlyNoExample: false,
  onlyWrong: false, // alleen woorden die je ooit fout had (volgens het antwoordenlog)
  onlyNew: false, // alleen woorden in de wachtrij
  groupFilter: new Set(), // alleen woorden uit (een van) deze groepen
  newPerDay: null, // instelling, geladen bij het openen van het beheer
  openId: null, // concept dat nu bewerkt wordt
  adding: false,
  flash: new Set(), // net toegevoegde/bewerkte concepten, kort gemarkeerd
};

let lastHuInput = null; // invoerveld waar de accentknoppen in typen

// ---------- API ----------

async function api(method, path, body) {
  // Alle wijzigingen gaan als JSON: de server weigert andere verzoeken (bescherming tegen CSRF)
  const res = await fetch(path, method === "GET" ? { method } : {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body ?? {}),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = data.error || `Serverfout (${res.status})`;
    throw new Error(msg.charAt(0).toUpperCase() + msg.slice(1) + (/[.!?]$/.test(msg) ? "" : "."));
  }
  return data;
}

async function refreshStats() {
  try {
    const st = await api("GET", "/api/state");
    renderStats(st.stats);
  } catch { /* statistiek is bijzaak */ }
}

function toast(msg) {
  toastEl.textContent = msg;
  toastEl.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => (toastEl.hidden = true), 2600);
}

// ---------- Zoeken & filteren ----------

// fold: kleine letters zonder accenten, zodat "ido" ook "idő" vindt
const fold = (s) => String(s ?? "").normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().trim();

// markFold markeert de zoekterm in de tekst, ook als de accenten verschillen ("ido" in "idő")
function markFold(text, query) {
  const q = fold(query);
  text = String(text ?? "");
  if (!q) return esc(text);

  // Per teken: de gevouwen vorm en waar het in de originele tekst staat
  let folded = "";
  const origin = []; // folded index -> [start, end) in text
  for (let i = 0; i < text.length;) {
    const ch = String.fromCodePoint(text.codePointAt(i));
    for (const c of ch.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase()) {
      folded += c;
      origin.push([i, i + ch.length]);
    }
    i += ch.length;
  }

  let html = "";
  let pos = 0;
  for (let at = folded.indexOf(q); at !== -1; at = folded.indexOf(q, at + q.length)) {
    const start = origin[at][0];
    const end = origin[at + q.length - 1][1];
    html += esc(text.slice(pos, start)) + `<mark>${esc(text.slice(start, end))}</mark>`;
    pos = end;
  }
  return html + esc(text.slice(pos));
}

function matchesQuery(e, q) {
  if (!q) return true;
  return [e.prompt, e.context, e.example, ...e.answers, ...(e.groups || [])].some((f) => fold(f).includes(q));
}

function filtered() {
  const q = fold(view.query);
  // Een groep die niet meer bestaat (laatste woord eruit gehaald) kan ook niet meer filteren
  const existing = new Set(knownGroups().map((g) => g.name));
  for (const g of view.groupFilter) if (!existing.has(g)) view.groupFilter.delete(g);
  return view.entries.filter((e) =>
    matchesQuery(e, q) &&
    (view.direction === "all" || `${e.from}-${e.to}` === view.direction) &&
    (!view.onlyMeanings || e.meanings > 1) &&
    (!view.onlyNoExample || !e.example) &&
    (!view.onlyWrong || e.wrong > 0) &&
    (!view.onlyNew || e.new) &&
    (!view.groupFilter.size || (e.groups || []).some((g) => view.groupFilter.has(g))));
}

function count(pred) {
  return view.entries.filter(pred).length;
}

// ---------- Weergave ----------

function renderManage() {
  const list = filtered();
  const chip = (key, value, label, n) =>
    `<button type="button" class="fchip" data-filter="${key}" data-value="${value}"
       aria-pressed="${view[key] === value}">${label} <span>${n}</span></button>`;
  const toggle = (key, label, n) =>
    `<button type="button" class="fchip" data-toggle="${key}" aria-pressed="${view[key]}">${label} <span>${n}</span></button>`;

  manageEl.innerHTML = `
    <form class="m-settings" id="m-settings">
      <label for="s-new">Nieuwe woorden per dag</label>
      <input id="s-new" type="number" min="0" max="500" inputmode="numeric" value="${view.newPerDay ?? ""}">
      <small class="muted">0 = geen limiet</small>
      <button class="btn secondary" type="submit">Opslaan</button>
    </form>

    <div class="m-toolbar">
      <label class="search">
        <span aria-hidden="true">⌕</span>
        <input id="m-search" type="search" placeholder="Zoek op woord, antwoord, hint of zin…"
               value="${esc(view.query)}" autocomplete="off" spellcheck="false" aria-label="Zoeken">
      </label>
      <button type="button" class="btn secondary" id="m-import-open" ${view.adding || imp.open ? "hidden" : ""}>Importeren</button>
      <button type="button" class="btn" id="m-add" ${view.adding || imp.open ? "hidden" : ""}>+ Nieuw</button>
    </div>

    <div class="m-filters">
      ${chip("direction", "all", "Alle", view.entries.length)}
      ${chip("direction", "nl-hu", "NL → HU", count((e) => e.from === "nl"))}
      ${chip("direction", "hu-nl", "HU → NL", count((e) => e.from === "hu"))}
      <span class="sep" aria-hidden="true"></span>
      ${toggle("onlyMeanings", "Meerdere betekenissen", count((e) => e.meanings > 1))}
      ${toggle("onlyNoExample", "Zonder voorbeeldzin", count((e) => !e.example))}
      ${toggle("onlyWrong", "Fout gehad", count((e) => e.wrong > 0))}
      ${toggle("onlyNew", "In wachtrij", count((e) => e.new))}
    </div>
    ${groupFilterHTML()}
    ${groupDatalistHTML()}

    <div id="m-new-slot">${view.adding ? newFormHTML() : imp.open ? importPanelHTML() : ""}</div>

    <p class="m-count muted">${list.length === view.entries.length
      ? `${list.length} concepten`
      : `${list.length} van ${view.entries.length} concepten`}</p>

    <ul class="m-list">
      ${list.map(rowHTML).join("") || `<li class="m-empty">Niets gevonden${view.query ? ` voor “${esc(view.query)}”` : ""}.</li>`}
    </ul>`;

  bindManage();
}

// groupFilterHTML: een rij met alle groepen om op te filteren (meerdere tegelijk: woorden uit één ervan)
function groupFilterHTML() {
  const groups = knownGroups();
  if (!groups.length) return "";
  return `<div class="m-filters m-groupfilter" aria-label="Filter op groep">
    <span class="muted">Groepen:</span>
    ${groups.map((g) => `<button type="button" class="fchip" data-group-filter="${esc(g.name)}"
      aria-pressed="${view.groupFilter.has(g.name)}">${esc(g.name)} <span>${g.count}</span></button>`).join("")}
  </div>`;
}

function rowHTML(e) {
  const q = view.query;
  const due = new Date(e.nextReview) <= new Date();
  const open = e.id === view.openId;
  return `
    <li class="m-row ${open ? "open" : ""} ${view.flash.has(e.id) ? "flash" : ""}" data-id="${esc(e.id)}">
      <button type="button" class="m-row-head" aria-expanded="${open}">
        <span class="m-main">
          <span class="m-prompt" lang="${esc(e.from)}">${markFold(e.prompt, q)}</span>
          ${e.context ? `<span class="chip">${markFold(e.context, q)}</span>` : ""}
          <span class="m-arrow" aria-hidden="true">→</span>
          <span class="m-answers" lang="${esc(e.to)}">${markFold(e.answers.join(", "), q)}</span>
          ${(e.groups || []).map((g) => `<span class="m-group">${markFold(g, q)}</span>`).join("")}
        </span>
        <span class="m-meta">
          <span class="m-dir">${e.from.toUpperCase()} → ${e.to.toUpperCase()}</span>
          ${e.meanings > 1 ? `<span class="m-badge">${e.meanings} betekenissen</span>` : ""}
          ${e.wrong ? `<span class="m-wrong" title="${e.wrong} van ${e.attempts} keer fout bij de eerste poging">✕ ${e.wrong}/${e.attempts}</span>` : ""}
          ${e.new
            ? `<span class="m-queued">nieuw · in wachtrij</span>`
            : `<span class="m-due"><span class="dot" style="background:var(--s${e.step})"></span>${due ? "nu te herhalen" : esc(relative(e.nextReview))}</span>`}
        </span>
        ${e.example ? `<span class="m-example" lang="hu">“${markFold(e.example, q)}”</span>` : ""}
      </button>
      ${open ? editFormHTML(e) : ""}
    </li>`;
}

function accentBar() {
  return `<div class="accents m-accents" aria-label="Hongaarse tekens">
    ${ACCENTS.map((a) => `<button type="button" data-char="${a}" tabindex="-1">${a}</button>`).join("")}
  </div>`;
}

function field(id, label, value, { hu = false, placeholder = "", hint = "" } = {}) {
  return `<label class="m-field" for="${id}">
    <span>${label}</span>
    <input id="${id}" value="${esc(value)}" placeholder="${esc(placeholder)}" autocomplete="off"
           spellcheck="false" ${hu ? `data-hu lang="hu"` : ""}>
    ${hint ? `<small class="muted">${hint}</small>` : ""}
  </label>`;
}

function editFormHTML(e) {
  const toHu = e.to === "hu";
  return `
    <form class="m-edit" data-id="${esc(e.id)}">
      ${field("e-context", "Hint", e.context, { placeholder: "bijv. buiten ☀️", hint: "Wijst aan welke betekenis bedoeld is." })}
      ${field("e-answers", `Goede antwoorden (${esc(LANG[e.to] || e.to)})`, e.answers.join(", "),
        { hu: toHu, hint: "Meerdere antwoorden scheiden met een komma." })}
      ${field("e-example", "Voorbeeldzin (Hongaars)", e.example || "", { hu: true, placeholder: "bijv. Milyen lesz az idő holnap?" })}
      ${groupPickerHTML("edit", "Groepen (geldt ook voor de andere richting)")}
      ${accentBar()}
      <p class="m-error" hidden></p>
      <div class="m-actions">
        <button type="button" class="btn danger-ghost" data-delete>Verwijderen…</button>
        <span class="m-spacer"></span>
        <button type="button" class="btn secondary" data-cancel>Annuleren</button>
        <button type="submit" class="btn">Opslaan</button>
      </div>
      <div class="m-confirm" hidden>${confirmDeleteHTML(e)}</div>
    </form>`;
}

// Bevestiging in de pagina zelf; bij een omgekeerde richting kun je kiezen of die ook weg moet
function confirmDeleteHTML(e) {
  const reverse = (e.reverse || []).map((id) => view.entries.find((x) => x.id === id)).filter(Boolean);
  return `
    <p><b>${esc(e.prompt)}</b>${e.context ? ` (${esc(e.context)})` : ""} → ${esc(e.answers.join(", "))} verwijderen?
    Je voortgang voor dit concept gaat ook verloren.</p>
    ${reverse.length ? `<label class="m-radio">
      <input type="checkbox" id="d-reverse" checked>
      <span>Ook de andere richting: ${reverse.map((r) => `<b>${esc(r.prompt)}</b> → ${esc(r.answers.join(", "))}`).join("; ")}
      <small class="muted">Hoort die ook bij andere woorden, dan gaat alleen „${esc(e.prompt)}” eruit als antwoord.</small></span>
    </label>` : ""}
    <div class="m-actions">
      <button type="button" class="btn secondary" data-delete-cancel>Toch niet</button>
      <button type="button" class="btn danger" data-delete-confirm>Verwijderen</button>
    </div>`;
}

function newFormHTML() {
  return `
    <form class="m-new card" id="m-new">
      <h3>Nieuw concept</h3>
      <p class="muted">Maakt Nederlands → Hongaars én Hongaars → Nederlands aan.</p>
      <div class="m-grid">
        ${field("n-dutch", "Nederlands woord", "", { placeholder: "bijv. tijd" })}
        ${field("n-context", "Hint", "", { placeholder: "bijv. klok ⏰" })}
        ${field("n-hungarian", "Hongaarse vertaling(en)", "", { hu: true, placeholder: "bijv. idő", hint: "Meerdere: scheiden met een komma." })}
        ${field("n-example", "Voorbeeldzin (Hongaars, optioneel)", "", { hu: true, placeholder: "bijv. Nincs elég időm." })}
      </div>
      ${groupPickerHTML("new", "Groepen (optioneel)")}
      ${accentBar()}
      <div id="n-meaning"></div>
      <p class="m-error" hidden></p>
      <div class="m-actions">
        <button type="button" class="btn secondary" data-cancel>Annuleren</button>
        <button type="submit" class="btn" id="n-submit">Toevoegen</button>
      </div>
    </form>`;
}

// ---------- Interactie ----------

function bindManage() {
  document.getElementById("m-settings").addEventListener("submit", async (e) => {
    e.preventDefault();
    const n = Number(document.getElementById("s-new").value);
    try {
      const res = await api("PUT", "/api/settings", { newPerDay: n });
      view.newPerDay = res.newPerDay;
      toast(res.newPerDay === 0 ? "Geen limiet meer op nieuwe woorden" : `Voortaan ${res.newPerDay} nieuwe woorden per dag`);
      refreshStats();
    } catch (err) {
      toast(err.message);
    }
  });

  const search = document.getElementById("m-search");
  search.addEventListener("input", () => {
    view.query = search.value;
    const pos = search.selectionStart;
    renderManage();
    const again = document.getElementById("m-search");
    again.focus();
    again.setSelectionRange(pos, pos);
  });

  document.getElementById("m-import-open").addEventListener("click", openImport);
  bindImport();
  for (const id of ["new", "edit", "import"]) bindGroupPicker(id);
  manageEl.querySelectorAll("[data-group-filter]").forEach((b) =>
    b.addEventListener("click", () => {
      const g = b.dataset.groupFilter;
      view.groupFilter.has(g) ? view.groupFilter.delete(g) : view.groupFilter.add(g);
      renderManage();
    }));

  document.getElementById("m-add").addEventListener("click", () => {
    imp.open = false;
    view.adding = true;
    groupState.new = [];
    view.openId = null;
    renderManage();
    document.getElementById("n-dutch").focus();
  });

  manageEl.querySelectorAll("[data-filter]").forEach((b) =>
    b.addEventListener("click", () => {
      view[b.dataset.filter] = b.dataset.value;
      renderManage();
    }));
  manageEl.querySelectorAll("[data-toggle]").forEach((b) =>
    b.addEventListener("click", () => {
      view[b.dataset.toggle] = !view[b.dataset.toggle];
      renderManage();
    }));

  manageEl.querySelectorAll(".m-row-head").forEach((head) =>
    head.addEventListener("click", () => {
      const id = head.closest(".m-row").dataset.id;
      view.openId = view.openId === id ? null : id;
      groupState.edit = [...(view.entries.find((x) => x.id === id)?.groups || [])];
      view.adding = false;
      renderManage();
      document.getElementById("e-context")?.focus();
    }));

  manageEl.querySelectorAll("[data-hu]").forEach((input) =>
    input.addEventListener("focus", () => (lastHuInput = input)));
  manageEl.querySelectorAll(".m-accents button").forEach((b) => {
    b.addEventListener("mousedown", (e) => e.preventDefault());
    b.addEventListener("click", () => {
      const form = b.closest("form");
      const target = lastHuInput && form.contains(lastHuInput) ? lastHuInput : form.querySelector("[data-hu]");
      insertAtCursor(target, b.dataset.char);
    });
  });

  manageEl.querySelectorAll("[data-cancel]").forEach((b) =>
    b.addEventListener("click", () => {
      view.openId = null;
      view.adding = false;
      renderManage();
    }));

  const edit = manageEl.querySelector(".m-edit");
  if (edit) {
    edit.addEventListener("submit", (e) => { e.preventDefault(); saveEdit(edit); });
    const confirmBox = edit.querySelector(".m-confirm");
    const actions = edit.querySelector(".m-actions");
    edit.querySelector("[data-delete]").addEventListener("click", () => {
      confirmBox.hidden = false;
      actions.hidden = true;
      confirmBox.querySelector("[data-delete-confirm]").focus();
    });
    confirmBox.querySelector("[data-delete-cancel]").addEventListener("click", () => {
      confirmBox.hidden = true;
      actions.hidden = false;
    });
    confirmBox.querySelector("[data-delete-confirm]").addEventListener("click", () => deleteConcept(edit));
  }

  const add = document.getElementById("m-new");
  if (add) {
    add.addEventListener("submit", (e) => { e.preventDefault(); submitNew(add); });
    // Andere invoer = opnieuw controleren op betekenissen
    ["n-dutch", "n-context", "n-hungarian"].forEach((id) =>
      document.getElementById(id).addEventListener("input", () => {
        add.dataset.checked = "";
        document.getElementById("n-meaning").innerHTML = "";
      }));
  }
}

function showError(form, msg) {
  const el = form.querySelector(".m-error");
  el.textContent = msg;
  el.hidden = !msg;
}

const splitList = (s) => s.split(",").map((x) => x.trim()).filter(Boolean);

async function saveEdit(form) {
  const id = form.dataset.id;
  try {
    const res = await api("PUT", `/api/concepts/${encodeURIComponent(id)}`, {
      context: form.querySelector("#e-context").value,
      answers: splitList(form.querySelector("#e-answers").value),
      groups: pendingGroups("edit"),
      example: form.querySelector("#e-example").value,
    });
    view.entries = res.entries;
    view.openId = null;
    flash([id]);
    toast("Opgeslagen");
  } catch (err) {
    showError(form, err.message);
  }
}

async function deleteConcept(form) {
  const id = form.dataset.id;
  const entry = view.entries.find((e) => e.id === id);
  const withReverse = form.querySelector("#d-reverse")?.checked ?? false;
  try {
    const res = await api("DELETE", `/api/concepts/${encodeURIComponent(id)}${withReverse ? "?reverse=1" : ""}`);
    view.entries = res.entries;
    view.openId = null;
    renderManage();
    toast(`“${entry?.prompt ?? id}” verwijderd`);
    refreshStats();
  } catch (err) {
    showError(form, err.message);
  }
}

// Toevoegen gaat in twee stappen: eerst controleren (duplicaat? bestaat het Hongaarse woord al
// met een andere betekenis?), dan pas opslaan met de gemaakte keuze.
async function submitNew(form) {
  const val = (id) => document.getElementById(id).value;
  const req = { dutch: val("n-dutch"), context: val("n-context"), hungarian: val("n-hungarian"), example: val("n-example"), groups: pendingGroups("new") };
  showError(form, "");

  try {
    if (!form.dataset.checked) {
      const check = await api("POST", "/api/concepts/check", req);
      if (check.duplicate) {
        showError(form, `“${req.dutch.trim()}”${req.context.trim() ? ` (${req.context.trim()})` : ""} → ${req.hungarian.trim()} bestaat al.`);
        return;
      }
      if (check.forwardCandidates.length > 0 || check.candidates.length > 0) {
        form.dataset.checked = "1";
        renderMeaningChoice(check, req);
        return;
      }
    }

    Object.assign(req, forwardChoice(), meaningChoice());
    const res = await api("POST", "/api/concepts", req);
    view.entries = res.entries;
    view.adding = false;
    const hu = splitList(req.hungarian);
    flash(view.entries.filter((e) =>
      (fold(e.prompt) === fold(req.dutch) && e.answers.some((a) => hu.some((h) => fold(h) === fold(a)))) ||
      (hu.some((h) => fold(h) === fold(e.prompt)) && e.answers.some((a) => fold(a) === fold(req.dutch)))).map((e) => e.id));
    toast(req.extendForward
      ? `${hu.join(", ")} toegevoegd als extra goed antwoord bij “${req.dutch.trim()}”`
      : `“${req.dutch.trim()}” toegevoegd; het komt via de wachtrij binnen`);
    refreshStats();
  } catch (err) {
    showError(form, err.message);
  }
}

// De keuzes als een woord al bestaat: eerst het Nederlandse woord (aanvullen of nieuwe betekenis),
// dan het Hongaarse woord (zelfde of andere betekenis)
function renderMeaningChoice(check, req) {
  const slot = document.getElementById("n-meaning");
  slot.innerHTML = `<div id="n-forward-choice"></div><div id="n-reverse-choice"></div>`;
  renderForwardChoice(check, req);
  renderReverseChoice(check, req);
  slot.querySelector("input[type=radio]")?.focus();
  document.getElementById("n-submit").textContent = "Bevestigen en toevoegen";
}

function renderForwardChoice(check, req) {
  const box = document.getElementById("n-forward-choice");
  if (!check.forwardCandidates.length) return;
  const dutch = esc(req.dutch.trim());
  const hu = esc(splitList(req.hungarian).join(", "));
  box.innerHTML = `
    <fieldset class="m-meaning">
      <legend><b>${dutch}</b> bestaat al in het Nederlands. Hoort <b lang="hu">${hu}</b> bij een bestaande betekenis, of is het een nieuwe betekenis?</legend>
      ${check.forwardCandidates.map((c) => `
        <label class="m-radio">
          <input type="radio" name="forward" value="${esc(c.id)}">
          <span>Extra goed antwoord bij <b>${dutch}</b>${c.context ? ` (${esc(c.context)})` : ""} → ${esc(c.answers.join(", "))}
          <small class="muted">Alleen voor echte synoniemen in het Hongaars, zoals bank (meubel) → pad / kanapé</small></span>
        </label>`).join("")}
      <label class="m-radio">
        <input type="radio" name="forward" value="">
        <span>Nieuwe betekenis
        <small class="muted">Apart concept; de hint hierboven moet de betekenis onderscheiden, zoals bank (meubel) / bank (geld)</small></span>
      </label>
      <div id="n-forward-hints" hidden>
        ${req.context.trim() ? "" : `<p class="m-warn">Vul hierboven een hint in die deze nieuwe betekenis onderscheidt.</p>`}
        ${check.forwardCandidates.filter((c) => !c.context).map((c) =>
          `<label class="m-field">
            <span>Hint voor het bestaande ${dutch} → ${esc(c.answers.join(", "))}</span>
            <input data-forward-hint="${esc(c.id)}" placeholder="bijv. meubel" autocomplete="off">
            <small class="muted">Dat concept heeft nog geen hint; zonder hint zijn de betekenissen niet uit elkaar te houden.</small>
          </label>`).join("")}
      </div>
    </fieldset>`;

  box.querySelectorAll('input[name="forward"]').forEach((r) =>
    r.addEventListener("change", async () => {
      document.getElementById("n-forward-hints").hidden = r.value !== "";
      // Bij aanvullen kan een ander Hongaars woord de omgekeerde richting krijgen: opnieuw controleren
      try {
        const again = await api("POST", "/api/concepts/check", { ...req, extendForward: r.value });
        renderReverseChoice(again, req);
        showError(document.getElementById("m-new"), again.duplicate && r.value
          ? "Deze Hongaarse woorden zijn al goede antwoorden bij die betekenis: er is niets nieuws om toe te voegen." : "");
      } catch (err) {
        showError(document.getElementById("m-new"), err.message);
      }
    }));
}

function renderReverseChoice(check, req) {
  const box = document.getElementById("n-reverse-choice");
  if (!check.candidates.length) {
    box.innerHTML = "";
    return;
  }
  const dutch = esc(req.dutch.trim());
  const hu = esc(check.reverse);
  const extended = check.forwardCandidates.find((c) => c.id === forwardExtend());
  const defaultHint = req.context.trim() || extended?.context || "";
  box.innerHTML = `
    <fieldset class="m-meaning">
      <legend><b lang="hu">${hu}</b> bestaat al als Hongaars → Nederlands. Is <b>${dutch}</b> dezelfde betekenis of een andere?</legend>
      ${check.candidates.map((c) => `
        <label class="m-radio">
          <input type="radio" name="meaning" value="${esc(c.id)}">
          <span>Zelfde betekenis als <b lang="hu">${hu}</b>${c.context ? ` (${esc(c.context)})` : ""} → ${esc(c.answers.join(", "))}
          <small class="muted">${dutch} wordt een extra goed antwoord. Alleen voor echte synoniemen, zoals szia → hoi / hallo</small></span>
        </label>`).join("")}
      <label class="m-radio">
        <input type="radio" name="meaning" value="" checked>
        <span>Andere betekenis
        <small class="muted">Apart concept met een eigen hint, zoals idő → weer / tijd</small></span>
      </label>
      <div id="n-hints">
        ${check.candidates.filter((c) => !c.context).map((c) =>
          field(`n-existing-${esc(c.id)}`, `Hint voor het bestaande ${hu} → ${esc(c.answers.join(", "))}`, "",
            { placeholder: "bijv. buiten ☀️", hint: "Dat concept heeft nog geen hint; zonder hint zijn de betekenissen niet uit elkaar te houden." })
        ).join("")}
        ${field("n-reverse", `Hint voor ${hu} → ${dutch}`, defaultHint, { hint: "Standaard dezelfde hint als hierboven." })}
      </div>
    </fieldset>`;

  const hints = document.getElementById("n-hints");
  box.querySelectorAll('input[name="meaning"]').forEach((r) =>
    r.addEventListener("change", () => (hints.hidden = r.value !== "" && r.checked)));
  box.querySelectorAll('input[id^="n-existing-"]').forEach((i) => i.setAttribute("data-existing", i.id.slice("n-existing-".length)));
}

// forwardExtend: het gekozen bestaande concept om aan te vullen ("" = nieuwe betekenis of nog niets gekozen)
function forwardExtend() {
  return document.querySelector('input[name="forward"]:checked')?.value ?? "";
}

function forwardChoice() {
  if (!document.getElementById("n-forward-choice")?.innerHTML) return {};
  const picked = document.querySelector('input[name="forward"]:checked');
  if (!picked) throw new Error("Kies of het een extra goed antwoord bij een bestaande betekenis is, of een nieuwe betekenis.");
  if (picked.value) return { extendForward: picked.value };

  const forwardHints = {};
  document.querySelectorAll("[data-forward-hint]").forEach((i) => (forwardHints[i.dataset.forwardHint] = i.value));
  if (Object.values(forwardHints).some((v) => !v.trim())) {
    throw new Error("Geef de bestaande betekenis ook een hint, anders zijn de betekenissen niet uit elkaar te houden.");
  }
  return { forwardHints };
}

function meaningChoice() {
  const picked = document.querySelector('input[name="meaning"]:checked');
  if (!picked) return {};
  if (picked.value) return { mergeReverseInto: picked.value };

  const existingHints = {};
  document.querySelectorAll("[data-existing]").forEach((i) => (existingHints[i.dataset.existing] = i.value));
  const missing = Object.entries(existingHints).find(([, v]) => !v.trim());
  if (missing) throw new Error("Geef het bestaande concept ook een hint, anders zijn de betekenissen niet uit elkaar te houden.");
  return { existingHints, reverseContext: document.getElementById("n-reverse").value };
}

function flash(ids) {
  view.flash = new Set(ids);
  renderManage();
  manageEl.querySelector(".m-row.flash")?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  setTimeout(() => {
    view.flash.clear();
    manageEl.querySelectorAll(".m-row.flash").forEach((r) => r.classList.remove("flash"));
  }, 2400);
}

document.addEventListener("keydown", (e) => {
  if (e.key !== "Escape" || manageEl.hidden || (!view.openId && !view.adding && !imp.open)) return;
  if (imp.open) return closeImport();
  view.openId = null;
  view.adding = false;
  renderManage();
});

// ---------- Tabs ----------

async function showTab() {
  const tab = { "#beheer": "beheer", "#statistiek": "statistiek", "#kaarten": "kaarten" }[location.hash] ?? "oefenen";
  manageEl.hidden = tab !== "beheer";
  statsEl.hidden = tab !== "statistiek";
  cardsEl.hidden = tab !== "kaarten";
  app.hidden = tab !== "oefenen";
  document.getElementById("quiz-foot").hidden = tab !== "oefenen";
  document.body.classList.toggle("mode-manage", tab !== "oefenen");
  for (const t of ["oefenen", "kaarten", "statistiek", "beheer"]) {
    document.getElementById(`tab-${t}`).setAttribute("aria-current", t === tab ? "page" : "false");
  }

  if (tab === "beheer") {
    manageEl.innerHTML = `<div class="skeleton"></div>`;
    try {
      const [concepts, settings] = await Promise.all([api("GET", "/api/concepts"), api("GET", "/api/settings")]);
      view.entries = concepts.entries;
      view.newPerDay = settings.newPerDay;
      renderManage();
      refreshStats();
      // Komend vanuit de statistiek: het gekozen woord openklappen
      manageEl.querySelector(".m-row.open")?.scrollIntoView({ block: "center" });
    } catch (err) {
      manageEl.innerHTML = `<p class="error">Kon de concepten niet laden: ${esc(err.message)}</p>`;
    }
  } else if (tab === "statistiek") {
    showStatistics();
  } else if (tab === "kaarten") {
    showCards();
  } else {
    renderedCardId = null; // kaart opnieuw opbouwen, er kunnen concepten bijgekomen zijn
    act("/api/state");
  }
}

window.addEventListener("hashchange", showTab);
showTab();
