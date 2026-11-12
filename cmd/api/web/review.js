"use strict";

// Controleren: iemand anders kijkt de woorden na voordat ze in de les komen. Goedkeuren, een
// opmerking geven (dan blijft het woord wachten) of in Beheer verbeteren.
// Gebruikt helpers uit app.js (esc, LANG) en manage.js (api, view, refreshStats, toast).

const reviewEl = document.getElementById("review");
const REVIEW_PAGE = 30;

const rv = {
  entries: [],
  filter: null, // "queue" | "notes" | "all"; null = automatisch (eerst de wachtrij)
  limit: REVIEW_PAGE,
  noteFor: null, // woord waarbij het opmerkingenveld open staat
  query: "", // zoeken, zonder op accenten te letten
};

async function showReview() {
  reviewEl.innerHTML = `<div class="skeleton"></div>`;
  refreshStats();
  try {
    rv.entries = (await api("GET", "/api/concepts")).entries;
  } catch (err) {
    reviewEl.innerHTML = `<p class="error">Kon de woorden niet laden: ${esc(err.message)}</p>`;
    return;
  }
  renderReview();
}

// Een woord is een kaart met zijn omgekeerde richting(en), met de Nederlandse kant voorop.
// Alleen woorden waarvan nog iets niet goedgekeurd is.
function reviewItems() {
  const byId = new Map(rv.entries.map((e) => [e.id, e]));
  const used = new Set();
  const items = [];
  for (const e of rv.entries) {
    if (used.has(e.id)) continue;
    // Vanuit de Nederlandse kant, met al haar omgekeerde richtingen (ook Hongaarse synoniemen)
    const reverses = (x) => (x.reverse || []).map((id) => byId.get(id)).filter(Boolean);
    const main = e.from === "nl" ? e : reverses(e).find((g) => g.from === "nl") ?? e;
    const cards = [main, ...reverses(main).filter((g) => g !== main)];
    if (!cards.includes(e)) cards.push(e);
    cards.forEach((g) => used.add(g.id));
    if (cards.every((g) => g.verified)) continue;
    // In de wachtrij = het blokkeert: een nieuwe kaart die nog niet goedgekeurd is
    items.push({ main, cards, queue: cards.some((g) => g.new && !g.verified), note: cards.find((g) => g.reviewNote)?.reviewNote ?? "" });
  }
  // Wat de wachtrij blokkeert eerst
  return items.sort((a, b) => b.queue - a.queue);
}

function renderReview() {
  const all = reviewItems();
  const counts = { queue: all.filter((i) => i.queue).length, notes: all.filter((i) => i.note).length, all: all.length };
  const filter = rv.filter ?? (counts.queue ? "queue" : "all");
  const q = fold(rv.query.trim());
  const items = all.filter((i) => (filter === "all" || (filter === "queue" ? i.queue : i.note)) &&
    (!q || i.cards.some((e) => fold([e.prompt, ...e.answers, e.context].join(" ")).includes(q))));
  const shown = items.slice(0, rv.limit);
  const chip = (key, label) => `<button type="button" class="fchip" data-rv-filter="${key}" aria-pressed="${filter === key}">
    ${label} <span>${counts[key]}</span></button>`;

  reviewEl.innerHTML = `
    <article class="card rv-head">
      <h2>Controleren</h2>
      <p class="muted">Kijk de woorden na voordat ze in de les komen. Een nieuw woord komt pas bij <b>Nieuwe woorden</b>
        als het hier is goedgekeurd; woorden die al in de herhalingen zitten lopen gewoon door.
        Klopt er iets niet, verbeter het dan in Beheer of laat een opmerking achter.</p>
      <label class="search rv-search">
        <span aria-hidden="true">⌕</span>
        <input id="rv-search" type="search" placeholder="Zoek een woord…" value="${esc(rv.query)}"
               autocomplete="off" spellcheck="false" aria-label="Zoeken">
      </label>
      <div class="m-filters">
        ${chip("queue", "In de wachtrij")}
        ${chip("notes", "Met opmerking")}
        ${chip("all", "Alles")}
      </div>
    </article>
    ${shown.length ? `<div class="rv-list">${shown.map(reviewItemHTML).join("")}</div>`
      : `<p class="sx-empty rv-empty">${all.length ? "Niets te controleren in deze selectie." : "Alles is gecontroleerd. 🎉"}</p>`}
    <div class="rv-more">
      ${shown.length > 1 ? `<button type="button" class="btn secondary" data-rv-approve-shown>Deze ${shown.length} goedkeuren</button>` : ""}
      ${items.length > shown.length ? `<button type="button" class="btn secondary" data-rv-more>Meer tonen (nog ${items.length - shown.length})</button>` : ""}
    </div>`;
}

function reviewItemHTML(item) {
  const card = (e) => `${esc(e.prompt)}${e.context ? ` <span class="chip">${esc(e.context)}</span>` : ""} →
    <b lang="${esc(e.to)}">${esc(e.answers.join(", "))}</b>`;
  const [main, ...rest] = item.cards;
  const example = item.cards.find((e) => e.example)?.example;
  const groups = [...new Set(item.cards.flatMap((e) => e.groups || []))];
  return `
    <article class="card rv-item" data-rv-id="${esc(main.id)}">
      <div class="rv-words">
        <div class="rv-main" lang="${esc(main.from)}">${card(main)}</div>
        ${rest.map((e) => `<div class="rv-rev muted">${card(e)}</div>`).join("")}
        ${example ? `<p class="example" lang="hu">“${esc(example)}”</p>` : ""}
        <div class="rv-tags">
          ${item.queue ? `<span class="m-queued">in de wachtrij</span>` : `<span class="m-group">al in de herhalingen</span>`}
          ${groups.map((g) => `<span class="m-group">${esc(g)}</span>`).join("")}
        </div>
        ${item.note ? `<p class="m-warn rv-note">Opmerking: ${esc(item.note)}</p>` : ""}
      </div>
      <div class="rv-actions">
        <button type="button" class="btn" data-rv-approve>Goedkeuren</button>
        <button type="button" class="btn secondary" data-rv-note>Opmerking</button>
        <button type="button" class="linklike" data-rv-fix>Verbeteren in Beheer</button>
      </div>
      ${rv.noteFor === main.id ? `
        <form class="rv-note-form" data-rv-note-form>
          <textarea rows="2" placeholder="Wat klopt er niet? Bijv. 'dit is óra, niet idő'">${esc(item.note)}</textarea>
          <div class="m-actions">
            <button type="button" class="btn secondary" data-rv-note-cancel>Annuleren</button>
            <button type="submit" class="btn">Opslaan, nog niet goedkeuren</button>
          </div>
        </form>` : ""}
    </article>`;
}

// Legt de controle vast en werkt de lijst bij; de server geeft terug welke kaarten geraakt zijn
async function reviewWord(id, verified, note = "") {
  const res = await api("POST", `/api/concepts/${encodeURIComponent(id)}/review`, { verified, note });
  for (const e of rv.entries) {
    if (res.ids.includes(e.id)) Object.assign(e, { verified, reviewNote: verified ? "" : note.trim() });
  }
}

// Bijwerken zonder dat de pagina naar boven springt
function rerenderReview() {
  const y = window.scrollY;
  renderReview();
  window.scrollTo(0, y);
}

reviewEl.addEventListener("click", async (e) => {
  const b = e.target.closest("button");
  if (!b) return;
  const id = b.closest("[data-rv-id]")?.dataset.rvId;
  try {
    if (b.dataset.rvFilter) {
      Object.assign(rv, { filter: b.dataset.rvFilter, limit: REVIEW_PAGE, noteFor: null });
      renderReview();
    } else if (b.matches("[data-rv-more]")) {
      rv.limit += REVIEW_PAGE;
      rerenderReview();
    } else if (b.matches("[data-rv-approve]")) {
      b.disabled = true;
      await reviewWord(id, true);
      rerenderReview();
      refreshStats();
    } else if (b.matches("[data-rv-approve-shown]")) {
      const ids = [...reviewEl.querySelectorAll("[data-rv-id]")].map((el) => el.dataset.rvId);
      b.disabled = true;
      for (const one of ids) await reviewWord(one, true);
      toast(`${ids.length} woorden goedgekeurd`);
      rerenderReview();
      refreshStats();
    } else if (b.matches("[data-rv-note]")) {
      rv.noteFor = id;
      rerenderReview();
      reviewEl.querySelector("[data-rv-note-form] textarea")?.focus();
    } else if (b.matches("[data-rv-note-cancel]")) {
      rv.noteFor = null;
      rerenderReview();
    } else if (b.matches("[data-rv-fix]")) {
      const main = rv.entries.find((x) => x.id === id);
      view.query = main.prompt; // zoekterm voor het beheer
      view.openId = id;
      location.hash = "#beheer";
    }
  } catch (err) {
    toast(err.message);
    b.disabled = false;
  }
});

reviewEl.addEventListener("submit", async (e) => {
  const form = e.target.closest("[data-rv-note-form]");
  if (!form) return;
  e.preventDefault();
  const id = form.closest("[data-rv-id]").dataset.rvId;
  const note = form.querySelector("textarea").value;
  if (!note.trim()) return form.querySelector("textarea").focus();
  try {
    await reviewWord(id, false, note);
    rv.noteFor = null;
    toast("Opmerking opgeslagen; het woord wacht tot het verbeterd en goedgekeurd is");
    rerenderReview();
  } catch (err) {
    toast(err.message);
  }
});

reviewEl.addEventListener("input", (e) => {
  if (e.target.id !== "rv-search") return;
  const pos = e.target.selectionStart;
  Object.assign(rv, { query: e.target.value, limit: REVIEW_PAGE, noteFor: null });
  renderReview();
  const again = document.getElementById("rv-search");
  again.focus();
  again.setSelectionRange(pos, pos);
});
