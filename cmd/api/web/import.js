"use strict";

// Importeren in het beheer: een lijst plakken (hongaars;nederlands;context;voorbeeldzin),
// controleren, keuzes maken en dan alles in één keer toevoegen.
// Gebruikt helpers uit app.js (esc) en manage.js (api, view, renderManage, toast, refreshStats, field).

const imp = {
  open: false,
  text: "",
  check: null, // laatste controle-uitkomst van de server
  choices: {}, // per regelnummer: { mergeInto, existingHints }
  busy: false,
};

const IMPORT_EXAMPLE = "olvas;leest;werkwoord · 3e persoon enkelvoud;Ő sokat olvas.\nolvasni;lezen;werkwoord · infinitief\npad,kanapé;bank;meubel";

function importPanelHTML() {
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
      ${c ? importPreviewHTML(c) : ""}
      <div class="m-actions">
        <button type="button" class="btn secondary" data-import-cancel>Annuleren</button>
        ${c && c.ok && c.added > 0
          ? `<button type="submit" class="btn" id="imp-go">${c.added} ${c.added === 1 ? "woord" : "woorden"} importeren</button>`
          : `<button type="submit" class="btn" id="imp-check" ${imp.text.trim() ? "" : "disabled"}>Controleren</button>`}
      </div>
    </form>`;
}

function importPreviewHTML(c) {
  const chip = (n, cls, label) => (n ? `<span class="imp-chip ${cls}">${n} ${label}</span>` : "");
  return `
    <div class="imp-summary">
      ${chip(c.added, "ok", "nieuw")}
      ${chip(c.duplicates, "dup", "bestaan al (worden overgeslagen)")}
      ${chip(c.choices, "choice", c.choices === 1 ? "keuze nodig" : "keuzes nodig")}
      ${chip(c.errors, "err", c.errors === 1 ? "fout" : "fouten")}
      ${!c.results.length ? `<span class="muted">Geen regels gevonden.</span>` : ""}
    </div>
    <ol class="imp-list">${c.results.map(importRowHTML).join("")}</ol>`;
}

const IMP_ICON = { added: "✓", duplicate: "↷", choice: "?", error: "✕" };

function importRowHTML(r) {
  const words = r.dutch || r.hungarian
    ? `<b lang="hu">${esc(r.hungarian)}</b> → ${esc(r.dutch)}${r.context ? ` <span class="chip">${esc(r.context)}</span>` : ""}`
    : "";
  return `
    <li class="imp-row ${r.status}">
      <span class="imp-icon" aria-hidden="true">${IMP_ICON[r.status]}</span>
      <span class="imp-no muted">r${r.no}</span>
      <div class="imp-body">
        <div>${words}</div>
        ${r.example ? `<div class="imp-example" lang="hu">“${esc(r.example)}”</div>` : ""}
        ${r.message && r.status !== "added" ? `<div class="imp-msg">${esc(r.message)}</div>` : ""}
        ${r.forwardCandidates?.length ? forwardChoiceHTML(r) : ""}
        ${r.candidates?.length || imp.choices[r.no]?.reverse ? importChoiceHTML(r) : ""}
      </div>
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

function bindImport() {
  const form = document.getElementById("m-import");
  if (!form) return;
  const text = document.getElementById("imp-text");

  text.addEventListener("input", () => {
    imp.text = text.value;
    if (imp.check) { // andere tekst: opnieuw controleren
      imp.check = null;
      imp.choices = {};
      renderManage();
      const again = document.getElementById("imp-text");
      again.focus();
      again.setSelectionRange(again.value.length, again.value.length);
    } else {
      document.getElementById("imp-check").disabled = !text.value.trim();
    }
  });

  form.querySelector("[data-import-cancel]").addEventListener("click", closeImport);
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    imp.check?.ok && imp.check.added > 0 ? runImport() : checkImport();
  });

  // Keuzes voor het Nederlandse woord
  form.querySelectorAll("[data-fwd-no]").forEach((fs) => {
    const no = Number(fs.dataset.fwdNo);
    const choice = () => (imp.choices[no] ??= {});
    fs.querySelectorAll('input[type="radio"]').forEach((radio) =>
      radio.addEventListener("change", () => {
        choice().forward = { extend: radio.value, hints: choice().forward?.hints ?? {} };
        // Een andere keuze kan een ander Hongaars woord opleveren: de keuze daarvoor opnieuw laten maken
        delete choice().reverse;
        if (imp.check?._cands) delete imp.check._cands[no];
        checkImport();
      }));
    fs.querySelectorAll("[data-fwd-hint]").forEach((input) =>
      input.addEventListener("change", () => {
        choice().forward.hints[input.dataset.fwdHint] = input.value;
        checkImport();
      }));
    fs.querySelectorAll("[data-line-hint]").forEach((input) =>
      input.addEventListener("change", () => {
        choice().context = input.value;
        checkImport();
      }));
  });

  // Keuzes voor het Hongaarse woord
  form.querySelectorAll("[data-no]").forEach((fs) => {
    const no = Number(fs.dataset.no);
    const choice = () => (imp.choices[no] ??= {});
    fs.querySelectorAll('input[type="radio"]').forEach((radio) =>
      radio.addEventListener("change", () => {
        Object.assign(choice(), { reverse: true, mergeInto: radio.value, existingHints: choice().existingHints ?? {} });
        checkImport();
      }));
    fs.querySelectorAll("[data-existing-hint]").forEach((input) =>
      input.addEventListener("change", () => {
        choice().existingHints[input.dataset.existingHint] = input.value;
        checkImport();
      }));
  });
}

async function checkImport() {
  if (imp.busy) return;
  imp.busy = true;
  try {
    const cands = imp.check?._cands;
    imp.check = await api("POST", "/api/import/check", { text: imp.text, choices: imp.choices, groups: pendingGroups("import") });
    if (cands) imp.check._cands = cands;
    renderManage();
  } catch (err) {
    toast(err.message);
  } finally {
    imp.busy = false;
  }
}

async function runImport() {
  if (imp.busy) return;
  imp.busy = true;
  try {
    const res = await api("POST", "/api/import", { text: imp.text, choices: imp.choices, groups: pendingGroups("import") });
    view.entries = res.entries ?? view.entries;
    closeImport();
    toast(`${res.added} ${res.added === 1 ? "woord" : "woorden"} toegevoegd${res.duplicates ? `, ${res.duplicates} overgeslagen` : ""}. Ze komen via de wachtrij binnen.`);
    refreshStats();
  } catch (err) {
    toast(err.message); // bijv. iets veranderd sinds het controleren: opnieuw controleren
    imp.check = null;
    renderManage();
  } finally {
    imp.busy = false;
  }
}

function openImport() {
  imp.open = true;
  groupState.import = [];
  view.adding = false;
  view.openId = null;
  renderManage();
  document.getElementById("imp-text").focus();
}

function closeImport() {
  Object.assign(imp, { open: false, text: "", check: null, choices: {} });
  renderManage();
}
