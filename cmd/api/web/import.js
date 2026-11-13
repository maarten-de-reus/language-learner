// Importeren in het beheer: een lijst plakken (hongaars;nederlands;context;voorbeeldzin),
// controleren, keuzes maken en dan alles in één keer toevoegen.

import { esc } from "./util.js";
import { api, toast } from "./api.js";
import { refreshStats } from "./quiz.js";
import { groupPickerHTML, groupState, pendingGroups } from "./groups.js";
import { accentBar, renderManage, view } from "./manage.js";

export const imp = {
  open: false,
  text: "",
  check: null, // laatste controle-uitkomst van de server
  choices: {}, // per regelnummer: { forward, context, reverse, mergeInto, existingHints, skip }
  busy: false,
  pending: false, // er is iets veranderd tijdens een controle: daarna nog een keer
  filter: null, // welke regels het voorbeeld toont; null = automatisch (eerst wat aandacht nodig heeft)
  touched: new Set(), // regels waar je iets aan veranderd hebt: blijven zichtbaar, ook als ze opgelost zijn
  anchor: null, // regel waar je mee bezig bent: die blijft na het bijwerken op dezelfde plek
};

const IMPORT_EXAMPLE = "olvas;leest;werkwoord · 3e persoon enkelvoud;Ő sokat olvas.\nolvasni;lezen;werkwoord · infinitief\npad,kanapé;bank;meubel";

export function importPanelHTML() {
  const c = imp.check;
  return `
    <form class="m-new card m-import" id="m-import">
      <h3>Importeren</h3>
      <p class="muted">Eén woord per regel: <code>hongaars;nederlands;context;voorbeeldzin</code>.
        Context en voorbeeldzin mogen leeg blijven; meerdere goede Hongaarse antwoorden scheid je met een komma.
        Nieuwe woorden komen via de wachtrij binnen, volgens je daglimiet.</p>
      <textarea id="imp-text" rows="8" spellcheck="false" data-hu
        placeholder="${esc(IMPORT_EXAMPLE)}">${esc(imp.text)}</textarea>
      ${accentBar()}
      ${groupPickerHTML("import", "Groepen voor alle woorden in deze import (optioneel)")}
      <div id="imp-preview">${c ? importPreviewHTML(c) : ""}</div>
      <div class="m-actions" id="imp-actions">${importActionsHTML(c)}</div>
    </form>`;
}

function importActionsHTML(c) {
  return `
    <button type="button" class="btn secondary" data-import-cancel>Annuleren</button>
    ${importReady(c)
      ? `<button type="submit" class="btn" id="imp-go">${importLabel(c)}</button>`
      : `<button type="submit" class="btn" id="imp-check" ${imp.text.trim() ? "" : "disabled"}>Controleren</button>`}`;
}

// Regels die nog iets van je vragen
const isOpen = (r) => r.status === "choice" || r.status === "error";

// Welke regels het voorbeeld toont. Automatisch: zijn er open regels, dan die (plus wat je net
// hebt aangepast), anders alles.
function importFilter(c) {
  return imp.filter ?? (c.choices + c.errors > 0 ? "open" : "all");
}

function importVisible(c, r) {
  const f = importFilter(c);
  if (f === "all") return true;
  if (f === "open") return isOpen(r) || imp.touched.has(r.no);
  return r.status === f;
}

// Er valt iets op te slaan: nieuwe woorden, of bestaande woorden die in een groep komen
const importReady = (c) => c && c.ok && (c.added > 0 || c.grouped > 0);

function importLabel(c) {
  const words = (n) => `${n} ${n === 1 ? "woord" : "woorden"}`;
  if (!c.added) return `${words(c.grouped)} in de groep zetten`;
  return `${words(c.added)} importeren${c.grouped ? ` en ${c.grouped} in de groep` : ""}`;
}

function importPreviewHTML(c) {
  const f = importFilter(c);
  // De tellers zijn ook filters: klik om alleen die regels te zien
  const chip = (n, cls, label, filter) => (n ? `<button type="button" class="imp-chip ${cls}" data-imp-filter="${filter}"
    aria-pressed="${f === filter}">${n} ${label}</button>` : "");
  const open = c.choices + c.errors;
  const rows = c.results.filter((r) => importVisible(c, r));
  return `
    <div class="imp-summary">
      ${chip(open, "choice", open === 1 ? "heeft aandacht nodig" : "hebben aandacht nodig", "open")}
      ${chip(c.added, "ok", "nieuw", "added")}
      ${chip(c.duplicates, "dup", c.duplicates === 1 ? "bestaat al" : "bestaan al", "duplicate")}
      ${chip(c.skipped, "dup", "overgeslagen", "skipped")}
      ${c.results.length ? `<button type="button" class="imp-chip all" data-imp-filter="all" aria-pressed="${f === "all"}">alle ${c.results.length}</button>` : ""}
      ${!c.results.length ? `<span class="muted">Geen regels gevonden.</span>` : ""}
      ${open > 1 ? `<button type="button" class="linklike imp-skip-open" data-imp-skip-open>alle ${open} open regels overslaan</button>` : ""}
    </div>
    ${c.grouped ? `<p class="muted imp-note">${c.grouped} bestaande ${c.grouped === 1 ? "woord komt" : "woorden komen"} in de groep van deze import.</p>` : ""}
    ${rows.length
      ? `<ol class="imp-list">${rows.map(importRowHTML).join("")}</ol>`
      : c.results.length ? `<p class="sx-empty">${f === "open" ? "Alles opgelost: niets meer dat aandacht nodig heeft." : "Geen regels in deze selectie."}</p>` : ""}`;
}

const IMP_ICON = { added: "✓", duplicate: "↷", choice: "?", error: "✕", skipped: "–" };

function importRowHTML(r) {
  const words = r.dutch || r.hungarian
    ? `<b lang="hu">${esc(r.hungarian)}</b> → ${esc(r.dutch)}${r.context ? ` <span class="chip">${esc(r.context)}</span>` : ""}`
    : "";
  return `
    <li class="imp-row ${r.status}" data-row="${r.no}">
      <span class="imp-icon" aria-hidden="true">${IMP_ICON[r.status]}</span>
      <span class="imp-no muted">r${r.no}</span>
      <div class="imp-body">
        <div>${words}</div>
        ${r.example ? `<div class="imp-example" lang="hu">“${esc(r.example)}”</div>` : ""}
        ${r.status === "duplicate" ? `<div class="imp-msg">Bestaat al als <b>${esc(r.existing)}</b>${r.grouped
          ? ` · komt in ${esc(pendingGroups("import").join(", "))}` : ""}</div>`
          : r.message && r.status !== "added" && r.status !== "skipped" ? `<div class="imp-msg">${esc(r.message)}</div>` : ""}
        ${r.extraReverse?.length ? `<div class="imp-extra">Ook als vraag: ${r.extraReverse.map((h) => `<b lang="hu">${esc(h)}</b> → ${esc(r.dutch)}`).join(", ")}</div>` : ""}
        ${r.skippedReverse?.length ? `<div class="imp-msg">${r.skippedReverse.map((h) => `<b lang="hu">${esc(h)}</b>`).join(", ")}
          ${r.skippedReverse.length === 1 ? "heeft" : "hebben"} al een andere betekenis, dus er komt geen kaart
          ${r.skippedReverse.map((h) => `${esc(h)} → ${esc(r.dutch)}`).join(", ")}. Betekent het hier hetzelfde, voeg die dan na de import zelf toe.</div>` : ""}
        ${r.forwardCandidates?.length ? forwardChoiceHTML(r) : ""}
        ${r.candidates?.length || imp.choices[r.no]?.reverse ? importChoiceHTML(r) : ""}
      </div>
      ${r.status === "skipped"
        ? `<button type="button" class="linklike imp-skip" data-imp-unskip="${r.no}">toch meenemen</button>`
        : `<button type="button" class="linklike imp-skip" data-imp-skip="${r.no}" title="Deze regel niet importeren">overslaan</button>`}
    </li>`;
}

// Het Nederlandse woord bestaat al: extra goed antwoord bij een bestaande betekenis, of een nieuwe betekenis
function forwardChoiceHTML(r) {
  const ch = imp.choices[r.no]?.forward;
  const name = `imp-fwd-${r.no}`;
  const dutch = esc(r.dutch);
  return `
    <fieldset class="imp-choice" data-fwd-no="${r.no}">
      <legend class="imp-legend"><b>${dutch}</b> bestaat al in het Nederlands:</legend>
      ${r.forwardCandidates.map((c) => `
        <label class="m-radio"><input type="radio" name="${name}" value="${esc(c.id)}" ${ch?.extend === c.id ? "checked" : ""}>
          <span>Extra goed antwoord bij <b>${dutch}</b>${c.context ? ` (${esc(c.context)})` : ""} → ${esc(c.answers.join(", "))}
          <small class="muted">alleen voor echte synoniemen in het Hongaars</small></span></label>`).join("")}
      <label class="m-radio"><input type="radio" name="${name}" value="" ${ch && !ch.extend ? "checked" : ""}>
        <span>Nieuwe betekenis, met een eigen hint</span></label>
      ${ch && !ch.extend ? `
        <label class="m-field imp-hint">
          <span>Hint voor deze nieuwe betekenis: ${dutch} → ${esc(r.hungarian)}</span>
          <input data-line-hint value="${esc(imp.choices[r.no]?.context ?? r.context)}" placeholder="bijv. spreektaal">
          <small class="muted">Vervangt de context-kolom van deze regel.</small>
        </label>` : ""}
      ${ch && !ch.extend ? r.forwardCandidates.filter((c) => !c.context).map((c) => `
        <label class="m-field imp-hint">
          <span>Hint voor het bestaande ${dutch} → ${esc(c.answers.join(", "))}</span>
          <input data-fwd-hint="${esc(c.id)}" value="${esc(ch.hints?.[c.id] ?? "")}" placeholder="bijv. meubel">
        </label>`).join("") : ""}
    </fieldset>`;
}

// Het Hongaarse woord bestaat al: zelfde of andere betekenis, zoals in het formulier voor één woord
function importChoiceHTML(r) {
  const cands = r.candidates || (imp.check._cands || {})[r.no] || [];
  if (!cands.length) return "";
  (imp.check._cands ??= {})[r.no] = cands; // bewaren: na de keuze komen ze niet meer mee
  const ch = imp.choices[r.no];
  const decided = ch?.reverse;
  const name = `imp-choice-${r.no}`;
  const hu = esc(r.reverse || r.hungarian.split(",")[0].trim());
  return `
    <fieldset class="imp-choice" data-no="${r.no}">
      <legend class="imp-legend"><b lang="hu">${hu}</b> bestaat al in het Hongaars:</legend>
      ${cands.map((c) => `
        <label class="m-radio"><input type="radio" name="${name}" value="${esc(c.id)}" ${decided && ch.mergeInto === c.id ? "checked" : ""}>
          <span>Zelfde betekenis als <b lang="hu">${hu}</b>${c.context ? ` (${esc(c.context)})` : ""} → ${esc(c.answers.join(", "))}
          <small class="muted">alleen voor echte synoniemen, zoals szia → hoi / hallo</small></span></label>`).join("")}
      <label class="m-radio"><input type="radio" name="${name}" value="" ${decided && !ch.mergeInto ? "checked" : ""}>
        <span>Andere betekenis, met de hint van deze regel</span></label>
      ${decided && !ch.mergeInto ? cands.filter((c) => !c.context).map((c) => `
        <label class="m-field imp-hint">
          <span>Hint voor het bestaande ${hu} → ${esc(c.answers.join(", "))}</span>
          <input data-existing-hint="${esc(c.id)}" value="${esc(ch.existingHints?.[c.id] ?? "")}" placeholder="bijv. buiten ☀️">
        </label>`).join("") : ""}
    </fieldset>`;
}

// Koppelt het formulier één keer: keuzes, hints, filters en overslaan gaan via event delegation,
// zodat het voorbeeld opnieuw opgebouwd kan worden zonder alles opnieuw te koppelen
export function bindImport() {
  const form = document.getElementById("m-import");
  if (!form) return;
  const text = document.getElementById("imp-text");

  text.addEventListener("input", () => {
    imp.text = text.value;
    if (imp.check) { // andere tekst: opnieuw controleren
      Object.assign(imp, { check: null, choices: {}, filter: null, anchor: null });
      imp.touched.clear();
    }
    renderImportPreview();
  });

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    importReady(imp.check) ? runImport() : checkImport();
  });

  form.addEventListener("click", (e) => {
    const b = e.target.closest("[data-import-cancel], [data-imp-filter], [data-imp-skip], [data-imp-unskip], [data-imp-skip-open]");
    if (!b) return;
    if (b.matches("[data-import-cancel]")) return closeImport();
    if (b.dataset.impFilter) {
      imp.filter = b.dataset.impFilter;
      imp.anchor = null;
      return renderImportPreview();
    }
    if (b.matches("[data-imp-skip-open]")) {
      for (const r of imp.check.results.filter(isOpen)) {
        (imp.choices[r.no] ??= {}).skip = true;
        imp.touched.add(r.no);
      }
      return checkImport();
    }
    const no = Number(b.dataset.impSkip ?? b.dataset.impUnskip);
    (imp.choices[no] ??= {}).skip = b.matches("[data-imp-skip]");
    imp.touched.add(no);
    checkImport(no);
  });

  form.addEventListener("change", (e) => {
    const el = e.target;
    const row = el.closest(".imp-row");
    if (!row) return;
    const no = Number(row.dataset.row);
    const choice = (imp.choices[no] ??= {});
    imp.touched.add(no);
    if (el.type === "radio" && el.name.startsWith("imp-fwd-")) {
      // Het Nederlandse woord: aanvullen of nieuwe betekenis. Een andere keuze kan een ander
      // Hongaars woord opleveren: de keuze daarvoor opnieuw laten maken
      choice.forward = { extend: el.value, hints: choice.forward?.hints ?? {} };
      delete choice.reverse;
      if (imp.check?._cands) delete imp.check._cands[no];
    } else if (el.type === "radio" && el.name.startsWith("imp-choice-")) {
      // Het Hongaarse woord: zelfde of andere betekenis
      Object.assign(choice, { reverse: true, mergeInto: el.value, existingHints: choice.existingHints ?? {} });
    } else if (el.dataset.fwdHint) {
      choice.forward.hints[el.dataset.fwdHint] = el.value;
    } else if (el.matches("[data-line-hint]")) {
      choice.context = el.value;
    } else if (el.dataset.existingHint) {
      choice.existingHints[el.dataset.existingHint] = el.value;
    } else {
      return;
    }
    checkImport(no, el.type === "radio");
  });
}

// Werkt alleen het voorbeeld en de knoppen bij. De lijst heeft een eigen scrollbalk: de regel waar
// je mee bezig bent blijft op dezelfde plek staan, anders blijft de scrollpositie gewoon bewaard.
function renderImportPreview(focusField = false) {
  const preview = document.getElementById("imp-preview");
  if (!preview) return;
  const list = preview.querySelector(".imp-list");
  const scroll = list?.scrollTop ?? 0;
  const anchorRow = list && imp.anchor != null ? list.querySelector(`[data-row="${imp.anchor}"]`) : null;
  const offset = anchorRow ? anchorRow.offsetTop - list.scrollTop : null;

  preview.innerHTML = imp.check ? importPreviewHTML(imp.check) : "";
  document.getElementById("imp-actions").innerHTML = importActionsHTML(imp.check);

  const newList = preview.querySelector(".imp-list");
  if (!newList) return;
  const row = imp.anchor != null ? newList.querySelector(`[data-row="${imp.anchor}"]`) : null;
  newList.scrollTop = row && offset != null ? row.offsetTop - offset : scroll;
  // Net "nieuwe betekenis" gekozen? Dan meteen het lege hintveld in die regel
  if (focusField && row) row.querySelector(".imp-hint input:placeholder-shown")?.focus({ preventScroll: true });
}

// Controleert de import op de server. Verandert er iets terwijl een controle loopt, dan volgt er
// direct nog een, zodat geen enkele keuze verloren gaat.
async function checkImport(anchor, focusField = false) {
  if (anchor !== undefined) imp.anchor = anchor;
  imp.focusField = imp.focusField || focusField;
  if (imp.busy) {
    imp.pending = true;
    return;
  }
  imp.busy = true;
  document.getElementById("imp-preview")?.classList.add("busy");
  try {
    do {
      imp.pending = false;
      const cands = imp.check?._cands;
      imp.check = await api("POST", "/api/import/check", { text: imp.text, choices: imp.choices, groups: pendingGroups("import") });
      if (cands) imp.check._cands = cands;
    } while (imp.pending);
    renderImportPreview(imp.focusField);
  } catch (err) {
    toast(err.message);
  } finally {
    imp.busy = false;
    imp.focusField = false;
    document.getElementById("imp-preview")?.classList.remove("busy");
  }
}

async function runImport() {
  if (imp.busy) return;
  imp.busy = true;
  try {
    const res = await api("POST", "/api/import", { text: imp.text, choices: imp.choices, groups: pendingGroups("import") });
    view.entries = res.entries ?? view.entries;
    closeImport();
    const parts = [];
    if (res.added) parts.push(`${res.added} ${res.added === 1 ? "woord" : "woorden"} toegevoegd`);
    if (res.grouped) parts.push(`${res.grouped} bestaande in de groep gezet`);
    if (res.duplicates - res.grouped) parts.push(`${res.duplicates - res.grouped} bestonden al`);
    if (res.skipped) parts.push(`${res.skipped} overgeslagen`);
    toast(`${parts.join(", ")}.${res.added ? " Nieuwe woorden komen via de wachtrij binnen; oefen ze vooraf onder Kaarten." : ""}`);
    refreshStats();
  } catch (err) {
    toast(err.message); // bijv. iets veranderd sinds het controleren: opnieuw controleren
    imp.check = null;
    renderImportPreview();
  } finally {
    imp.busy = false;
  }
}

export function openImport() {
  imp.open = true;
  groupState.import = [];
  view.adding = false;
  view.openId = null;
  renderManage();
  document.getElementById("imp-text").focus();
}

export function closeImport() {
  Object.assign(imp, { open: false, text: "", check: null, choices: {}, filter: null, anchor: null });
  imp.touched.clear();
  renderManage();
}
