// Controleren: iemand anders kijkt de woorden na voordat ze in de les komen. Goedkeuren, een
// opmerking geven (dan blijft het woord wachten) of in Beheer verbeteren.

import { esc, fold } from "./util.js";
import { api, toast } from "./api.js";
import { refreshStats } from "./quiz.js";
import { view } from "./manage.js";

const reviewEl = document.getElementById("review");
const REVIEW_PAGE = 30;

const rv = {
  entries: [],
  filter: null, // "queue" | "notes" | "all"; null = automatisch (eerst de wachtrij)
  limit: REVIEW_PAGE,
  noteFor: null, // woord waarbij het opmerkingenveld open staat
  query: "", // zoeken, zonder op accenten te letten
  current: null, // woord waar de sneltoetsen op werken
  forms: {}, // per kaart: de vormen die de regels van het woord maken (POST /api/grammar/forms)
};

export async function showReview() {
  reviewEl.innerHTML = `<div class="skeleton"></div>`;
  refreshStats();
  try {
    rv.entries = (await api("GET", "/api/concepts")).entries;
  } catch (err) {
    reviewEl.innerHTML = `<p class="error">Kon de woorden niet laden: ${esc(err.message)}</p>`;
    return;
  }
  rv.forms = {};
  renderReview();
}

// loadForms haalt de vormen op van de getoonde woorden die we nog niet hebben, en tekent dan opnieuw
async function loadForms() {
  const ids = [...reviewEl.querySelectorAll("[data-rv-id]")].map((el) => el.dataset.rvId).filter((id) => !(id in rv.forms));
  if (!ids.length) return;
  try {
    const res = await api("POST", "/api/grammar/forms", { ids });
    for (const id of ids) rv.forms[id] = res.forms[id] ?? null;
    rerenderReview();
  } catch {
    for (const id of ids) rv.forms[id] = null; // geen vormen: dan alleen het woord
  }
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
    // Twijfels over vormen ("Klopt niet" in Regels): ook bij goedgekeurde woorden
    const doubts = Object.assign({}, ...cards.map((g) => g.doubts || {}));
    if (cards.every((g) => g.verified) && !Object.keys(doubts).length) continue;
    // In de wachtrij = het blokkeert: een nieuwe kaart die nog niet goedgekeurd is
    // released: alvast in de les gezet, nog niet goedgekeurd (blijft hier staan)
    items.push({ main, cards, queue: cards.some((g) => g.new && !g.verified), released: cards.some((g) => g.released && !g.verified),
      note: cards.find((g) => g.reviewNote)?.reviewNote ?? "", doubts, verified: cards.every((g) => g.verified),
      doubtId: cards.find((g) => g.doubts && Object.keys(g.doubts).length)?.id ?? main.id });
  }
  // Twijfels en wat de wachtrij blokkeert eerst
  const hasDoubts = (i) => Object.keys(i.doubts).length > 0;
  return items.sort((a, b) => hasDoubts(b) - hasDoubts(a) || b.queue - a.queue);
}

function renderReview() {
  const all = reviewItems();
  const doubted = (i) => Object.keys(i.doubts).length > 0;
  const counts = { queue: all.filter((i) => i.queue).length, notes: all.filter((i) => i.note).length,
    doubts: all.filter(doubted).length, all: all.length };
  // Eerst de twijfels over vormen (die houden een vorm uit de oefening), dan de wachtrij
  const filter = rv.filter ?? (counts.doubts ? "doubts" : counts.queue ? "queue" : "all");
  const q = fold(rv.query.trim());
  const items = all.filter((i) => (filter === "all" || (filter === "queue" ? i.queue : filter === "doubts" ? doubted(i) : i.note)) &&
    (!q || i.cards.some((e) => fold([e.prompt, ...e.answers, e.context, ...(e.groups || [])].join(" ")).includes(q))));
  // Wat met "alvast leren" in de les kan: nieuw, nog niet goedgekeurd en nog niet vrijgegeven
  const releasable = items.filter((i) => i.queue && !i.released);
  const shown = items.slice(0, rv.limit);
  // Het huidige woord voor de sneltoetsen: blijft staan, of schuift door als het weg is
  if (!shown.some((i) => i.main.id === rv.current)) rv.current = shown[Math.min(rv.currentIndex ?? 0, shown.length - 1)]?.main.id ?? null;
  rv.currentIndex = shown.findIndex((i) => i.main.id === rv.current);
  const chip = (key, label) => `<button type="button" class="fchip" data-rv-filter="${key}" aria-pressed="${filter === key}">
    ${label} <span>${counts[key]}</span></button>`;

  reviewEl.innerHTML = `
    <article class="card rv-head">
      <h2>Controleren</h2>
      <p class="muted">Een nieuw woord komt pas in de les als het hier is goedgekeurd.</p>
      <details class="rv-help">
        <summary>Hoe werkt het?</summary>
        <p class="muted">Klopt een woord, keur het dan goed: beide richtingen tegelijk. Klopt er iets niet, verbeter het
          in Beheer of laat een opmerking achter; dan wacht het woord. Woorden die al in de herhalingen zitten lopen gewoon door.</p>
        <p class="muted">Sneltoetsen: <kbd>J</kbd>/<kbd>K</kbd> volgende/vorige · <kbd>A</kbd> goedkeuren ·
          <kbd>O</kbd> opmerking · <kbd>B</kbd> verbeteren in Beheer</p>
        <p class="muted"><b>Vormen</b>: onder elk woord staan de vormen die de app er met de regels van maakt
          (<i>ban</i> = -ban/-ben, <i>t</i> = lijdend voorwerp, <i>én</i> = ik-vorm …). Klopt er een niet, klik hem dan aan:
          die vorm wordt niet meer geoefend tot de twijfel is opgelost. Een <i>?</i> is een vorm die de regels niet zeker weten
          en dus niet oefenen.</p>
        <p class="muted"><b>Alvast leren</b> zet een woord al in de les voordat het is goedgekeurd; het blijft hier staan
          tot iemand het goedkeurt. Zoek op een groep (bijv. <i>2.8</i>) om een hele les in één keer vrij te geven.</p>
      </details>
      <label class="search rv-search">
        <span aria-hidden="true">⌕</span>
        <input id="rv-search" type="search" placeholder="Zoek een woord of groep…" value="${esc(rv.query)}"
               autocomplete="off" spellcheck="false" aria-label="Zoeken">
      </label>
      <div class="m-filters">
        ${chip("queue", "In de wachtrij")}
        ${chip("notes", "Met opmerking")}
        ${counts.doubts ? chip("doubts", "Twijfel over vormen") : ""}
        ${chip("all", "Alles")}
      </div>
    </article>
    ${shown.length ? `<div class="rv-list">${shown.map(reviewItemHTML).join("")}</div>`
      : `<p class="sx-empty rv-empty">${all.length ? "Niets te controleren in deze selectie." : "Alles is gecontroleerd. 🎉"}</p>`}
    <div class="rv-more">
      ${shown.length > 1 ? `<button type="button" class="btn secondary" data-rv-approve-shown>Deze ${shown.length} goedkeuren</button>` : ""}
      ${releasable.length > 1 ? `<button type="button" class="btn secondary" data-rv-release-all="${releasable.map((i) => esc(i.main.id)).join(" ")}"
        title="Alle woorden in deze selectie alvast in de les zetten; ze blijven hier staan tot ze zijn goedgekeurd">Alle ${releasable.length} alvast leren</button>` : ""}
      ${items.length > shown.length ? `<button type="button" class="btn secondary" data-rv-more>Meer tonen (nog ${items.length - shown.length})</button>` : ""}
    </div>`;
  loadForms();
}

// formsHTML: de vormen van het woord als kleine knoppen; aanklikken = "klopt niet"
function formsHTML(item) {
  const f = rv.forms[item.main.id];
  if (!f) return "";
  if (f.note) return `<p class="rv-forms muted">${esc(f.note)}</p>`;
  const groups = [];
  for (const r of f.rows) {
    const last = groups[groups.length - 1];
    if (last?.rule === r.rule) last.rows.push(r);
    else groups.push({ rule: r.rule, name: r.ruleName, rows: [r] });
  }
  const formHTML = (r) => {
    if (r.doubt != null) return `<span class="rv-form doubt" title="${esc(r.label)}: twijfel, zie hierboven"><small>${esc(r.key)}</small> ${esc(r.doubt || "–")}</span>`;
    if (!r.text) return `<span class="rv-form unsure" title="${esc(r.label)}: ${r.fixed ? "niet oefenen (bij het woord)" : "de regels weten deze vorm niet zeker; wordt niet geoefend"}"><small>${esc(r.key)}</small> ?</span>`;
    return `<button type="button" class="rv-form ${r.fixed ? "fixed" : ""}" data-rv-doubt="${esc(r.key)}" data-rv-form="${esc(r.text)}"
      title="${esc(r.label)}${r.fixed ? " (uitzondering bij het woord)" : ""} · klik als deze vorm niet klopt"><small>${esc(r.key)}</small> <span lang="hu">${esc(r.text)}</span></button>`;
  };
  // Werkwoorden per regel op een regel; zelfstandige naamwoorden alles bij elkaar
  const verb = groups.some((g) => g.rule.startsWith("ww-"));
  return `<details class="rv-forms" ${item.verified ? "" : "open"}>
    <summary>Vormen van <b lang="hu">${esc(f.word)}</b></summary>
    ${verb ? groups.map((g) => `<div class="rv-form-row"><span class="muted">${esc(g.name)}:</span> ${g.rows.map(formHTML).join("")}</div>`).join("")
      : `<div class="rv-form-row">${f.rows.map(formHTML).join("")}</div>`}
  </details>`;
}

// doubtsHTML: de twijfels over vormen van dit woord, met de drie manieren om ze op te lossen
function doubtsHTML(item) {
  const keys = Object.keys(item.doubts);
  if (!keys.length) return "";
  const label = (k) => rv.forms[item.main.id]?.rows?.find((r) => r.key === k)?.label ?? k;
  return `<div class="rv-doubts">
    <p class="m-warn">Twijfel over ${keys.length === 1 ? "een vorm" : `${keys.length} vormen`}: wordt niet geoefend tot het is opgelost.</p>
    ${keys.map((k) => `
      <form class="rv-doubt" data-rv-resolve="${esc(k)}" data-rv-concept="${esc(item.doubtId)}">
        <span><small class="muted">${esc(label(k))}</small> <s lang="hu">${esc(item.doubts[k] || "–")}</s></span>
        <input name="form" lang="hu" placeholder="goede vorm" value="" autocomplete="off" spellcheck="false" aria-label="Goede vorm">
        <button type="submit" class="btn secondary" name="fix">Verbeteren</button>
        <button type="button" class="linklike" data-rv-resolve-action="correct">Klopt toch</button>
        <button type="button" class="linklike" data-rv-resolve-action="skip">Niet oefenen</button>
      </form>`).join("")}
  </div>`;
}

function reviewItemHTML(item) {
  const card = (e) => `${esc(e.prompt)}${e.context ? ` <span class="chip">${esc(e.context)}</span>` : ""} →
    <b lang="${esc(e.to)}">${esc(e.answers.join(", "))}</b>`;
  const [main, ...rest] = item.cards;
  const example = item.cards.find((e) => e.example)?.example;
  const groups = [...new Set(item.cards.flatMap((e) => e.groups || []))];
  return `
    <article class="card rv-item ${main.id === rv.current ? "current" : ""}" data-rv-id="${esc(main.id)}">
      <div class="rv-words">
        <div class="rv-main" lang="${esc(main.from)}">${card(main)}</div>
        ${rest.map((e) => `<div class="rv-rev muted">${card(e)}</div>`).join("")}
        ${example ? `<p class="example" lang="hu">“${esc(example)}”</p>` : ""}
        <div class="rv-tags">
          ${item.released ? `<span class="m-released">wordt al geleerd</span>`
            : item.queue ? `<span class="m-queued">in de wachtrij</span>` : `<span class="m-group">al in de herhalingen</span>`}
          ${groups.map((g) => `<span class="m-group">${esc(g)}</span>`).join("")}
        </div>
        ${item.note ? `<p class="m-warn rv-note">Opmerking: ${esc(item.note)}</p>` : ""}
        ${doubtsHTML(item)}
        ${formsHTML(item)}
      </div>
      <div class="rv-actions">
        ${item.verified ? "" : `<button type="button" class="btn" data-rv-approve>Goedkeuren</button>`}
        <button type="button" class="btn secondary" data-rv-note>Opmerking</button>
        ${item.queue ? `<button type="button" class="linklike" data-rv-release="${item.released ? "off" : "on"}">${item.released ? "Niet alvast leren" : "Alvast leren"}</button>` : ""}
        <button type="button" class="linklike" data-rv-fix>Verbeteren</button>
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
// Zet een woord alvast in de les (of haalt het eruit) zonder het goed te keuren
async function releaseWord(id, on) {
  const res = await api("POST", `/api/concepts/${encodeURIComponent(id)}/release`, { released: on });
  for (const e of rv.entries) if (res.ids.includes(e.id)) e.released = on;
}

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
  if (id) {
    rv.current = id;
    rv.currentIndex = [...reviewEl.querySelectorAll("[data-rv-id]")].findIndex((el) => el.dataset.rvId === id);
  }
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
    } else if (b.matches("[data-rv-release]")) {
      b.disabled = true;
      await releaseWord(id, b.dataset.rvRelease === "on");
      rerenderReview();
      refreshStats();
    } else if (b.matches("[data-rv-release-all]")) {
      const ids = b.dataset.rvReleaseAll.split(" ");
      b.disabled = true;
      for (const one of ids) await releaseWord(one, true);
      toast(`${ids.length} woorden worden alvast geleerd; ze blijven hier staan tot ze zijn goedgekeurd`);
      rerenderReview();
      refreshStats();
    } else if (b.dataset.rvDoubt) {
      // Deze vorm klopt niet: twijfel erop, dan wordt hij niet meer geoefend
      b.disabled = true;
      const res = await api("POST", "/api/grammar/doubt", { conceptId: id, key: b.dataset.rvDoubt, form: b.dataset.rvForm });
      rv.entries = res.entries;
      delete rv.forms[id];
      toast(`Gemeld: ${b.dataset.rvForm} wordt niet geoefend tot het is opgelost`);
      rerenderReview();
    } else if (b.dataset.rvResolveAction) {
      const form = b.closest("[data-rv-resolve]");
      await resolveDoubt(form, b.dataset.rvResolveAction);
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

// Een twijfel oplossen: de vorm klopt toch, de goede vorm komt bij het woord, of niet oefenen
async function resolveDoubt(form, action) {
  const value = form.querySelector("input").value.trim();
  if (action === "fix" && !value) return form.querySelector("input").focus();
  const id = form.dataset.rvConcept;
  const res = await api("POST", `/api/concepts/${encodeURIComponent(id)}/doubt`, { key: form.dataset.rvResolve, action, form: value });
  rv.entries = res.entries;
  const main = form.closest("[data-rv-id]").dataset.rvId;
  delete rv.forms[main];
  toast({ correct: "De vorm klopt: hij wordt weer geoefend", fix: `${value} staat nu bij het woord`, skip: "Deze vorm wordt niet geoefend" }[action]);
  rerenderReview();
}

reviewEl.addEventListener("submit", async (e) => {
  const doubtForm = e.target.closest("[data-rv-resolve]");
  if (doubtForm) {
    e.preventDefault();
    try {
      await resolveDoubt(doubtForm, "fix");
    } catch (err) {
      toast(err.message);
    }
    return;
  }
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

// Sneltoetsen, zodat honderden woorden nakijken niet al het werk met de muis is
document.addEventListener("keydown", (e) => {
  if (reviewEl.hidden || e.ctrlKey || e.metaKey || e.altKey) return;
  if (e.target instanceof Element && e.target.closest("input, textarea")) return;
  const items = [...reviewEl.querySelectorAll("[data-rv-id]")];
  if (!items.length) return;
  const i = Math.max(0, items.findIndex((el) => el.dataset.rvId === rv.current));
  const press = (sel) => items[i].querySelector(sel)?.click();
  const go = (j) => {
    rv.currentIndex = Math.min(Math.max(j, 0), items.length - 1);
    rv.current = items[rv.currentIndex].dataset.rvId;
    items.forEach((el) => el.classList.toggle("current", el.dataset.rvId === rv.current));
    reviewEl.querySelector(".rv-item.current")?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  };
  const actions = {
    j: () => go(i + 1), ArrowDown: () => go(i + 1), k: () => go(i - 1), ArrowUp: () => go(i - 1),
    a: () => press("[data-rv-approve]"), o: () => press("[data-rv-note]"), b: () => press("[data-rv-fix]"),
  };
  const action = actions[e.key.length === 1 ? e.key.toLowerCase() : e.key];
  if (!action) return;
  e.preventDefault();
  action();
});
