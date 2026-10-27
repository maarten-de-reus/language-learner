"use strict";

// Kaarten: woorden vooraf oefenen (bijv. een nieuwe import) met omdraaibare flashcards.
// Staat los van de spaced repetition: er wordt niets opgeslagen en niets uit de wachtrij gehaald.
// Gebruikt helpers uit app.js (esc, highlight, LANG) en manage.js (api).

const cardsEl = document.getElementById("cards");
const FC_DIRECTIONS = [
  { key: "hu-nl", label: "Hongaars → Nederlands", hint: "herkennen, de makkelijkste start" },
  { key: "nl-hu", label: "Nederlands → Hongaars", hint: "zelf het Hongaarse woord bedenken" },
  { key: "mixed", label: "Gemengd", hint: "per woord een willekeurige richting" },
];
const FC_REQUEUE_AFTER = 3; // "Nog niet" komt na zoveel andere kaarten terug

const fc = {
  entries: [],
  setKey: null,
  dir: "hu-nl",
  queue: [], // kaarten die nog moeten; de eerste is de huidige
  flipped: false,
  total: 0,
  misses: new Map(), // kaart-ID -> aantal keer "Nog niet"
  done: false,
};

// ---------- Sets en stapels ----------

function fcSets() {
  const groups = [...new Set(fc.entries.flatMap((e) => e.groups || []))]
    .sort((a, b) => a.localeCompare(b, "nl", { sensitivity: "base" }));
  return [
    { key: "queue", label: "In wachtrij", note: "nieuwe woorden die nog moeten komen", filter: (e) => e.new },
    { key: "wrong", label: "Fout gehad", note: "volgens je antwoorden", filter: (e) => e.wrong > 0 },
    ...groups.map((g) => ({ key: `g:${g}`, label: g, note: "groep", filter: (e) => (e.groups || []).includes(g) })),
    { key: "all", label: "Alles", note: "je hele woordenlijst", filter: () => true },
  ];
}

// fcDeck maakt van een set een stapel met één kaart per woord, in de gekozen richting
function fcDeck(set, dir) {
  const byId = new Map(fc.entries.map((e) => [e.id, e]));
  const used = new Set();
  const deck = [];
  for (const e of fc.entries.filter(set.filter)) {
    if (used.has(e.id)) continue;
    const reverse = (e.reverse || []).map((id) => byId.get(id)).filter(Boolean);
    const options = [e, ...reverse];
    let card;
    if (dir === "mixed") card = options[Math.floor(Math.random() * options.length)];
    else card = options.find((o) => `${o.from}-${o.to}` === dir) ?? e;
    for (const o of options) used.add(o.id); // elk woord maar één keer per sessie
    deck.push(card);
  }
  return deck;
}

function shuffle(list) {
  for (let i = list.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [list[i], list[j]] = [list[j], list[i]];
  }
  return list;
}

function fcRemember() {
  try { localStorage.setItem("fc-choice", JSON.stringify({ setKey: fc.setKey, dir: fc.dir })); } catch { /* geen opslag */ }
}

function fcRecall() {
  try { return JSON.parse(localStorage.getItem("fc-choice")) || {}; } catch { return {}; }
}

// ---------- Schermen ----------

async function showCards() {
  cardsEl.innerHTML = `<div class="skeleton"></div>`;
  refreshStats(); // kop en stappenbalk bijwerken, net als op de andere tabbladen
  try {
    fc.entries = (await api("GET", "/api/concepts")).entries;
  } catch (err) {
    cardsEl.innerHTML = `<p class="error">Kon de woorden niet laden: ${esc(err.message)}</p>`;
    return;
  }
  if (fc.queue.length && !fc.done) return fcRenderCard(); // lopende sessie hervatten

  const sets = fcSets();
  const saved = fcRecall();
  fc.dir = FC_DIRECTIONS.some((d) => d.key === saved.dir) ? saved.dir : fc.dir;
  const count = (s) => fcDeck(s, "hu-nl").length;
  if (!sets.some((s) => s.key === fc.setKey && count(s) > 0)) {
    // Standaard: wat nieuw binnenkomt, anders wat je fout had, anders alles
    fc.setKey = [saved.setKey, "queue", "wrong", "all"].find((k) => {
      const s = sets.find((x) => x.key === k);
      return s && count(s) > 0;
    }) ?? "all";
  }
  fcRenderSetup();
}

function fcRenderSetup() {
  fc.done = false;
  const sets = fcSets();
  const set = sets.find((s) => s.key === fc.setKey) ?? sets[sets.length - 1];
  const size = fcDeck(set, fc.dir).length;

  cardsEl.innerHTML = `
    <article class="card fc-setup">
      <h2>Kaarten</h2>
      <p class="muted">Oefen woorden vooraf, bijvoorbeeld een nieuwe import, voordat ze in je herhalingen komen.
        Draai de kaart om en zeg zelf of je het wist. <b>Dit telt niet mee voor je planning.</b></p>

      <h3>Welke woorden</h3>
      <div class="fc-options" role="radiogroup" aria-label="Welke woorden">
        ${sets.map((s) => {
          const n = fcDeck(s, fc.dir).length;
          return `<button type="button" class="fchip" role="radio" data-set="${esc(s.key)}"
            aria-checked="${s.key === set.key}" ${n ? "" : "disabled"} title="${esc(s.note)}">${esc(s.label)} <span>${n}</span></button>`;
        }).join("")}
      </div>

      <h3>Richting</h3>
      <div class="fc-dirs" role="radiogroup" aria-label="Richting">
        ${FC_DIRECTIONS.map((d) => `
          <label class="m-radio"><input type="radio" name="fc-dir" value="${d.key}" ${d.key === fc.dir ? "checked" : ""}>
            <span>${d.label} <small class="muted">${d.hint}</small></span></label>`).join("")}
      </div>

      <button type="button" class="btn block" id="fc-start" ${size ? "" : "disabled"}>
        ${size ? `Start met ${size} ${size === 1 ? "kaart" : "kaarten"}` : "Geen woorden in deze selectie"}</button>
    </article>`;

  cardsEl.querySelectorAll("[data-set]").forEach((b) =>
    b.addEventListener("click", () => { fc.setKey = b.dataset.set; fcRemember(); fcRenderSetup(); }));
  cardsEl.querySelectorAll('input[name="fc-dir"]').forEach((r) =>
    r.addEventListener("change", () => { fc.dir = r.value; fcRemember(); fcRenderSetup(); }));
  document.getElementById("fc-start").addEventListener("click", () => fcStartSession(fcDeck(set, fc.dir)));
}

function fcStartSession(deck) {
  Object.assign(fc, { queue: shuffle([...deck]), total: deck.length, misses: new Map(), flipped: false, done: false });
  fcRenderCard();
}

function fcRenderCard() {
  const card = fc.queue[0];
  if (!card) return fcRenderSummary();
  const doneCount = fc.total - new Set(fc.queue.map((c) => c.id)).size;
  const pct = (doneCount / fc.total) * 100;
  // Een Hongaarse voorbeeldzin verraadt het antwoord niet als de vraag Hongaars is
  const exampleFront = card.example && card.from === "hu";

  cardsEl.innerHTML = `
    <article class="card fc-card ${fc.flipped ? "flipped" : ""}">
      <div class="run-progress">
        <div class="run-track"><div class="run-fill" style="width:${pct}%"></div></div>
        <span class="run-count">${doneCount} / ${fc.total}</span>
      </div>
      <div class="chips">
        <span class="chip">${esc(LANG[card.from] || card.from)} → ${esc(LANG[card.to] || card.to)}</span>
        ${card.context ? `<span class="chip">${esc(card.context)}</span>` : ""}
        ${fc.misses.get(card.id) ? `<span class="chip retry">↻ nog eens</span>` : ""}
      </div>
      <h2 class="prompt" lang="${esc(card.from)}">${esc(card.prompt)}</h2>
      ${exampleFront ? `<p class="example" lang="hu">“${highlight(card.example, [card.prompt])}”</p>` : ""}

      ${fc.flipped ? `
        <div class="fc-back" lang="${esc(card.to)}">
          <p class="fc-answer">${esc(card.answers.join(" · "))}</p>
          ${card.example ? `<p class="example" lang="hu">“${highlight(card.example, [card.prompt, ...card.answers])}”</p>` : ""}
        </div>
        <div class="fc-rate">
          <button type="button" class="btn secondary" id="fc-again">← Nog niet</button>
          <button type="button" class="btn" id="fc-known">Ken ik →</button>
        </div>`
      : `<button type="button" class="btn block fc-flip" id="fc-flip">Omdraaien <kbd>spatie</kbd></button>`}
    </article>
    <p class="muted fc-foot">Telt niet mee voor je planning ·
      <button type="button" class="linklike" id="fc-stop">stoppen</button></p>`;

  document.getElementById("fc-flip")?.addEventListener("click", fcFlip);
  document.getElementById("fc-again")?.addEventListener("click", () => fcRate(false));
  document.getElementById("fc-known")?.addEventListener("click", () => fcRate(true));
  document.getElementById("fc-stop").addEventListener("click", () => { fc.queue = []; fcRenderSummary(); });
  (document.getElementById("fc-known") ?? document.getElementById("fc-flip"))?.focus();
}

function fcFlip() {
  fc.flipped = true;
  fcRenderCard();
}

function fcRate(known) {
  const card = fc.queue.shift();
  if (!known) {
    fc.misses.set(card.id, (fc.misses.get(card.id) || 0) + 1);
    fc.queue.splice(Math.min(FC_REQUEUE_AFTER, fc.queue.length), 0, card);
  }
  fc.flipped = false;
  fcRenderCard();
}

function fcRenderSummary() {
  fc.done = true;
  const hard = [...fc.misses].sort((a, b) => b[1] - a[1]);
  const byId = new Map(fc.entries.map((e) => [e.id, e]));
  const seen = fc.total;
  const firstTime = seen - hard.length;

  cardsEl.innerHTML = `
    <article class="card center">
      <div class="big-emoji">🃏</div>
      <h2>${hard.length ? "Sessie klaar" : "Alles in één keer!"}</h2>
      <p class="score">${firstTime}<small> / ${seen}</small></p>
      <p class="muted">meteen gekend${hard.length ? `; ${hard.length} ${hard.length === 1 ? "kaart kostte" : "kaarten kostten"} meer moeite` : ""}</p>
      ${hard.length ? `<ul class="results">${hard.map(([id, n]) => {
        const e = byId.get(id);
        return e ? `<li class="ko"><span class="mark">↻</span><span class="src">${esc(e.prompt)}</span>
          <span class="exp">${esc(e.answers.join(", "))} <span class="muted">· ${n}× nog niet</span></span></li>` : "";
      }).join("")}</ul>` : ""}
      <div class="done-actions">
        ${hard.length ? `<button type="button" class="btn" id="fc-hard">Alleen de lastige nog een keer</button>` : ""}
        <button type="button" class="btn secondary" id="fc-new">Andere kaarten kiezen</button>
      </div>
    </article>`;

  document.getElementById("fc-hard")?.addEventListener("click", () =>
    fcStartSession(hard.map(([id]) => byId.get(id)).filter(Boolean)));
  document.getElementById("fc-new").addEventListener("click", () => { fc.queue = []; fcRenderSetup(); });
  (document.getElementById("fc-hard") ?? document.getElementById("fc-new")).focus();
}

// Toetsen: spatie draait om, pijl links = nog niet, pijl rechts = ken ik
document.addEventListener("keydown", (e) => {
  const typing = e.target instanceof Element && e.target.closest("input, textarea");
  if (cardsEl.hidden || fc.done || !fc.queue.length || typing) return;
  if (!fc.flipped && (e.key === " " || e.key === "Enter")) {
    e.preventDefault();
    fcFlip();
  } else if (fc.flipped && (e.key === "ArrowLeft" || e.key === "1")) {
    e.preventDefault();
    fcRate(false);
  } else if (fc.flipped && (e.key === "ArrowRight" || e.key === "2")) {
    e.preventDefault();
    fcRate(true);
  }
});
