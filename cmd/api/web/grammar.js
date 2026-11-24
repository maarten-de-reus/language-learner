// Regels: Hongaarse vormen leren maken (naamvallen, meervoud, vervoegen) met je eigen woorden.
// Je kiest zelf welke regels je oefent; de antwoorden tellen niet mee voor je planning.

import { ACCENTS, diffHTML, esc, insertAtCursor, pulse, shake } from "./util.js";
import { api, toast } from "./api.js";
import { refreshStats } from "./quiz.js";

const grammarEl = document.getElementById("grammar");

const g = {
  data: null, // GET /api/grammar
  view: "overview", // overview | practice
  open: new Set(), // regels waarvan de uitleg openstaat
  onlyRule: "", // oefenen met één regel (vanuit de uitleg)
  ex: null, // huidige opgave
  result: null, // nagekeken antwoord
  input: "",
  session: { n: 0, good: 0 },
  review: null, // regels herhalen: { queue, index, count, correct, size, pass, results }
  seen: new Set(), // varianten die in deze sessie al geweest zijn: eerst komen de andere
};

// De niveaus van de herhalingen (engine.AnkiIntervals), voor "woorden die je kent sinds minstens"
const LEVELS = ["1 dag", "3 dagen", "7 dagen", "16 dagen", "40 dagen", "90 dagen", "beheerst (★)"];

const ruleById = (id) => g.data?.rules.find((r) => r.id === id);

// scoreText: hoe gaat het met deze regel, over de laatste antwoorden
function scoreText(s) {
  if (!s?.recentN) return "nog niet geoefend";
  return s.attempts > s.recentN ? `${s.recentCorrect} van de laatste ${s.recentN} goed` : `${s.recentCorrect} van ${s.recentN} goed`;
}

function scoreClass(s) {
  if (!s?.recentN) return "";
  const p = s.recentCorrect / s.recentN;
  return p >= 0.9 ? "good" : p >= 0.6 ? "mid" : "low";
}

export async function showGrammar() {
  refreshStats();
  // Vanuit de feedback in Oefenen of Kaarten: #regels/ww-onbep opent die regel.
  // Vanuit Oefenen (knop Regels): #regels/herhalen herhaalt de regels die aan de beurt zijn.
  const wanted = decodeURIComponent(location.hash.split("/")[1] ?? "");
  if (wanted === "herhalen") {
    if (g.review && g.view === "practice" && g.ex) return renderPractice();
    return startReview();
  }
  g.review = null;
  if (wanted) {
    g.open.add(wanted);
    Object.assign(g, { view: "overview", ex: null });
  }
  if (g.view === "practice" && g.ex) return renderPractice();
  grammarEl.innerHTML = `<div class="skeleton"></div>`;
  try {
    g.data = await api("GET", "/api/grammar");
  } catch (err) {
    grammarEl.innerHTML = `<p class="error">Kon de regels niet laden: ${esc(err.message)}</p>`;
    return;
  }
  renderOverview();
  if (wanted) grammarEl.querySelector(`[data-rule-open="${CSS.escape(wanted)}"]`)?.scrollIntoView({ block: "center" });
}

// ---------- Overzicht ----------

function renderOverview() {
  g.view = "overview";
  const d = g.data;
  const active = d.rules.filter((r) => r.active && r.ready);
  const waiting = d.rules.filter((r) => r.active && !r.ready);
  const groups = [...new Set(d.rules.map((r) => r.group))];
  grammarEl.innerHTML = `
    <article class="card gr-head">
      <h2>Regels</h2>
      <p class="muted">Leer de vormen maken in plaats van ze te onthouden: elke opgave gebruikt een woord uit je eigen lijst,
        en na je antwoord zie je stap voor stap hoe de vorm ontstaat. <b>Telt niet mee voor je planning.</b></p>
      ${d.untyped ? `<p class="gr-note">${d.untyped} ${d.untyped === 1 ? "kaart heeft" : "kaarten hebben"} nog geen woordsoort en
        ${d.untyped === 1 ? "doet" : "doen"} niet mee. Vul die in bij <a href="#beheer">Beheer</a> (filter <i>woordsoort onbekend</i>).</p>` : ""}
      <label class="gr-level">Oefen met woorden die je kent sinds minstens
        <select id="gr-level">${LEVELS.map((l, i) => `<option value="${i}" ${i === d.minStep ? "selected" : ""}>${l}</option>`).join("")}</select>
        <small class="muted">Dan zie je wat de regel met het woord doet, zonder de vertaling nodig te hebben. Een regel heeft
          minstens ${d.rules[0]?.need ?? 5} van zulke woorden nodig; tot die tijd is hij nog niet klaar en komt hij vanzelf vrij.</small>
      </label>
      <div class="gr-start">
        <button class="btn" id="gr-start" type="button" ${active.length ? "" : "disabled"}>Oefenen</button>
        <span class="muted">${active.length
          ? `${active.length} ${active.length === 1 ? "regel" : "regels"} aan: ${esc(active.map((r) => r.name).join(", "))}`
          : waiting.length ? "Je regels zijn nog niet klaar: je kent nog te weinig woorden waar ze op werken."
          : "Zet hieronder een of meer regels aan."}${waiting.length && active.length
          ? ` · ${waiting.length} nog niet klaar` : ""}</span>
      </div>
    </article>
    ${groups.map((group) => `
      <section class="gr-group" aria-label="${esc(group)}">
        <h3>${esc(group)}</h3>
        <ul class="gr-list">${d.rules.filter((r) => r.group === group).map(ruleRowHTML).join("")}</ul>
      </section>`).join("")}`;
  bindOverview();
}

function ruleRowHTML(r) {
  const open = g.open.has(r.id);
  const none = !r.ready;
  return `
    <li class="gr-rule ${open ? "open" : ""} ${none ? "none" : ""}">
      <div class="gr-rule-head">
        <label class="switch" title="${r.active ? "Uitzetten" : none ? "Aanzetten: komt vanzelf vrij als je genoeg woorden kent" : "Aanzetten"}">
          <input type="checkbox" data-rule-toggle="${esc(r.id)}" ${r.active ? "checked" : ""}>
          <span class="switch-track" aria-hidden="true"></span>
          <span class="sr-only">${esc(r.name)} oefenen</span>
        </label>
        <button type="button" class="gr-rule-title" data-rule-open="${esc(r.id)}" aria-expanded="${open}">
          <span class="gr-name" lang="hu">${esc(r.name)}</span>
          <span class="gr-meaning">${esc(r.meaning)}</span>
          <span class="gr-meta">
            <span class="gr-score ${scoreClass(r.score)}">${scoreText(r.score)}</span>
            ${none ? `<span class="gr-not-ready" title="Deze regel heeft minstens ${r.need} woorden nodig die je goed kent">nog niet klaar · ${r.words} van ${r.need}</span>`
              : `<span class="muted">${r.words} ${r.words === 1 ? "woord" : "woorden"}</span>`}
            <span aria-hidden="true">${open ? "▴" : "▾"}</span>
          </span>
        </button>
      </div>
      ${open ? `
        <div class="gr-explain">
          ${r.explain.split("\n").map((p) => `<p>${esc(p)}</p>`).join("")}
          <p class="gr-fixed" lang="hu">Bijvoorbeeld: ${esc(r.example)}</p>
          ${r.examples?.length ? `<p class="gr-ex-title">Met jouw woorden:</p>
            <ul class="gr-examples">${r.examples.map((e) => `<li><span lang="hu">${esc(e.word)}</span>
              <span class="muted">${esc(e.dutch)}</span> → ${e.person ? `<span class="muted">${esc(e.person)}</span> ` : ""}<b lang="hu">${esc(e.form)}</b></li>`).join("")}</ul>` : ""}
          ${none ? `<p class="muted">Nog niet klaar: je kent ${r.words === 0 ? "nog geen" : `pas ${r.words}`} van de minstens ${r.need} woorden
            die deze regel nodig heeft. Oefen je woorden in de herhalingen; de regel komt vanzelf vrij.</p>`
            : `<button type="button" class="btn secondary" data-rule-try="${esc(r.id)}">Alleen deze regel oefenen</button>`}
        </div>` : ""}
    </li>`;
}

function bindOverview() {
  document.getElementById("gr-start").addEventListener("click", () => startPractice(""));
  grammarEl.querySelectorAll("[data-rule-open]").forEach((b) =>
    b.addEventListener("click", () => {
      const id = b.dataset.ruleOpen;
      g.open.has(id) ? g.open.delete(id) : g.open.add(id);
      renderOverview();
    }));
  grammarEl.querySelectorAll("[data-rule-toggle]").forEach((box) =>
    box.addEventListener("change", async () => {
      const active = g.data.rules.filter((r) => (r.id === box.dataset.ruleToggle ? box.checked : r.active)).map((r) => r.id);
      try {
        g.data = await api("PUT", "/api/grammar/active", { active });
        renderOverview();
      } catch (err) {
        toast(err.message);
        box.checked = !box.checked;
      }
    }));
  document.getElementById("gr-level").addEventListener("change", async (e) => {
    try {
      g.data = await api("PUT", "/api/grammar/level", { minStep: Number(e.target.value) });
      renderOverview();
    } catch (err) {
      toast(err.message);
    }
  });
  grammarEl.querySelectorAll("[data-rule-try]").forEach((b) =>
    b.addEventListener("click", () => startPractice(b.dataset.ruleTry)));
}

// ---------- Oefenen ----------

// ---------- Regels herhalen (de derde modus van Oefenen) ----------

async function startReview() {
  grammarEl.innerHTML = `<div class="skeleton"></div>`;
  let rv;
  try {
    [g.data, rv] = await Promise.all([api("GET", "/api/grammar"), api("GET", "/api/grammar/review")]);
  } catch (err) {
    grammarEl.innerHTML = `<p class="error">Kon de regels niet laden: ${esc(err.message)}</p>`;
    return;
  }
  if (!rv.due.length) return renderReviewDone(rv, []);
  g.review = { queue: rv.due, index: 0, size: rv.size, pass: rv.pass, perDay: rv.perDay, doneToday: rv.doneToday, results: [] };
  startReviewRule();
}

function startReviewRule() {
  const r = g.review;
  Object.assign(r, { count: 0, correct: 0 });
  Object.assign(g, { onlyRule: r.queue[r.index].id, session: { n: 0, good: 0 }, ex: null, seen: new Set() });
  nextExercise();
}

// finishReviewRule: de opgaven van deze regel zijn klaar; planning bijwerken en uitslag tonen
async function finishReviewRule() {
  const r = g.review;
  const rule = r.queue[r.index];
  let res;
  try {
    res = await api("POST", `/api/grammar/review/${encodeURIComponent(rule.id)}`, { correct: r.correct });
  } catch (err) {
    toast(err.message);
    return;
  }
  r.results.push({ rule, correct: r.correct, size: r.size, ...res });
  refreshStats();
  const last = r.index + 1 >= r.queue.length;
  const when = res.passed ? (res.interval === 1 ? "morgen weer" : `over ${res.interval} dagen weer`) : "morgen nog een keer";
  g.view = "between";
  grammarEl.innerHTML = `
    <article class="card gr-review-step">
      <div class="feedback ${res.passed ? "good" : "bad"}" role="status">
        <span class="icon" aria-hidden="true">${res.passed ? "✓" : "✕"}</span>
        <div>
          <strong lang="hu">${esc(rule.name)}</strong>
          <p>${r.correct} van ${r.size} goed · ${when}</p>
          ${res.passed ? "" : `<p class="muted">Bij ${r.pass} of meer goed gaat een regel een stap verder; anders komt hij morgen terug.</p>`}
        </div>
      </div>
      <div class="next-row">
        <button class="btn block" id="gr-review-next" type="button">${last ? "Klaar" : `Volgende regel (${r.index + 2} van ${r.queue.length})`}</button>
      </div>
    </article>`;
  const next = document.getElementById("gr-review-next");
  next.focus();
  next.addEventListener("click", () => {
    if (last) return renderReviewDone(null, r.results);
    r.index++;
    startReviewRule();
  });
}

function renderReviewDone(rv, results) {
  g.review = null;
  g.view = "between";
  const passed = results.filter((x) => x.passed).length;
  grammarEl.innerHTML = `
    <article class="card gr-head">
      <h2>Regels herhalen</h2>
      ${results.length
        ? `<p>${results.length} ${results.length === 1 ? "regel" : "regels"} herhaald, ${passed} een stap verder.</p>
           <ul class="gr-review-list">${results.map((x) => `<li>${x.passed ? "✓" : "✕"} <span lang="hu">${esc(x.rule.name)}</span>
             <span class="muted">${x.correct} van ${x.size} goed · ${x.passed ? `over ${x.interval} ${x.interval === 1 ? "dag" : "dagen"}` : "morgen"}</span></li>`).join("")}</ul>`
        : `<p class="muted">${rv && rv.doneToday >= rv.perDay
            ? `Je hebt vandaag al ${rv.doneToday} regels herhaald (het maximum per dag). Morgen meer.`
            : "Er zijn nu geen regels aan de beurt. Regels die aan staan en klaar zijn komen hier vanzelf, net als je woorden."}</p>`}
      <div class="m-actions">
        <a class="btn secondary" href="#regels">Naar Regels</a>
        <a class="btn" href="#oefenen">Terug naar Oefenen</a>
      </div>
    </article>`;
}

async function startPractice(onlyRule) {
  g.review = null;
  Object.assign(g, { onlyRule, session: { n: 0, good: 0 }, ex: null, seen: new Set() });
  await nextExercise();
}

async function nextExercise() {
  if (g.review && g.review.count >= g.review.size) return finishReviewRule();
  const params = new URLSearchParams();
  if (g.onlyRule) params.set("rule", g.onlyRule);
  if (g.ex) params.set("avoid", g.ex.word);
  for (const v of g.seen) params.append("seen", v);
  try {
    g.ex = await api("GET", `/api/grammar/exercise?${params}`);
  } catch (err) {
    toast(err.message);
    return showOverview();
  }
  Object.assign(g, { view: "practice", result: null, input: "" });
  renderPractice();
}

function showOverview() {
  g.view = "overview";
  g.ex = null;
  showGrammar();
}

function taskText(ex, rule) {
  if (rule.id === "pers") return `${esc(ex.dutch)} ${["mij", "jou", "hem/haar", "ons", "jullie", "hen"][ex.person]} <span class="muted">· ${esc(ex.personLabel)}</span>`;
  if (rule.id === "bb") return "vergrotende trap (-bb)";
  if (rule.id === "leg") return "overtreffende trap (leg-…-bb)";
  const tense = { "ww-mult": "verleden tijd", "ww-mult-bep": "verleden tijd, bepaald (met lijdend voorwerp)",
    "ww-fog": "toekomende tijd (fog + infinitief)", "ww-hat": "met -hat/-het (kunnen, mogen)",
    "ww-felsz": "gebiedende wijs", "ww-felsz-bep": "gebiedende wijs, bepaald (met lijdend voorwerp)" }[rule.id];
  if (rule.id === "ww-lak") return "ik … jou <span class=\"muted\">(-lak/-lek)</span>";
  if (tense) return `${esc(ex.personLabel)}, ${tense}`;
  if (rule.persons) return `${esc(ex.personLabel)}, ${rule.id === "ww-bep" ? "bepaald (met lijdend voorwerp)" : "onbepaald"}`;
  if (rule.id === "ww-inf") return "de infinitief";
  return `+ <span lang="hu">${esc(rule.name)}</span> <span class="muted">· ${esc(rule.meaning)}</span>`;
}

function renderPractice() {
  const ex = g.ex;
  const rule = ruleById(ex.rule);
  const r = g.result;
  grammarEl.innerHTML = `
    <div class="run-progress gr-progress">
      <span>${g.review ? `Regels herhalen · regel ${g.review.index + 1} van ${g.review.queue.length}`
        : g.onlyRule ? `Alleen ${esc(rule.name)}` : "Regels oefenen"}</span>
      <span class="run-count">${g.review ? `opgave ${Math.min(g.review.count + (g.result ? 0 : 1), g.review.size)} van ${g.review.size}`
        : g.session.n ? `${g.session.good} van ${g.session.n} goed` : ""}</span>
      <button type="button" class="linklike" id="gr-stop">Stoppen</button>
    </div>
    <article class="card gr-card">
      <div class="chips">
        <span class="chip">${esc(rule.group)}</span>
        <span class="chip" lang="hu">${esc(rule.name)}</span>
      </div>
      <p class="prompt-label">${esc(ex.dutch)}</p>
      <p class="prompt" lang="hu">${esc(ex.word)}</p>
      <p class="gr-task">${taskText(ex, rule)}</p>
      <form id="gr-form" autocomplete="off">
        <div class="answer-row">
          <input class="answer-input ${r ? (r.correct ? "is-good" : "is-bad") : ""}" id="gr-answer" lang="hu"
                 value="${esc(g.input)}" ${r ? "disabled" : ""} placeholder="De vorm…"
                 autocapitalize="off" autocorrect="off" spellcheck="false" aria-label="Jouw antwoord">
          ${r ? "" : `<button class="btn" type="submit">Controleer</button>`}
        </div>
        <div class="accents" aria-label="Hongaarse tekens">
          ${ACCENTS.map((a) => `<button type="button" data-char="${a}" tabindex="-1" ${r ? "disabled" : ""}>${a}</button>`).join("")}
        </div>
        ${r ? "" : `<button class="linklike dont-know" id="gr-dont-know" type="button">Weet ik niet <kbd>Esc</kbd></button>`}
      </form>
      ${r ? feedbackHTML(r, rule) : ""}
    </article>`;
  bindPractice();
}

function feedbackHTML(r, rule) {
  const forms = [r.form.text, ...(r.form.also || [])].map((f) => `<span class="answer" lang="hu">${esc(f)}</span>`).join(" of ");
  return `
    <div class="feedback ${r.correct ? "good" : "bad"}" role="status">
      <span class="icon" aria-hidden="true">${r.correct ? "✓" : "✕"}</span>
      <div>
        <strong>${r.correct ? "Goed zo!" : r.unknown ? "Zo maak je hem" : r.accentOnly ? "Bijna! Alleen de accenten kloppen niet" : "Nog niet"}</strong>
        ${r.correct && !r.form.also?.length ? "" : `<p>${r.correct ? "Allebei goed" : "De vorm"}: ${forms}</p>`}
        ${r.correct || r.unknown ? "" : diffHTML(g.input, [r.form.text, ...(r.form.also || [])])}
        <ol class="gr-steps">${r.form.steps.filter(Boolean).map((s) => `<li>${esc(s)}</li>`).join("")}</ol>
        <p class="gr-rule-score">${esc(rule.name)}: ${scoreText(r.score)}
          ${["i", "kor", "pers"].includes(rule.id) ? "" : r.doubted
            ? ` · <span class="gr-doubted">gemeld: wordt niet meer gevraagd tot het in Controleren is opgelost</span>`
            : ` · <button type="button" class="linklike" id="gr-doubt" title="Deze vorm lijkt niet te kloppen: niet meer vragen en laten nakijken in Controleren">Klopt de vorm niet?</button>`}</p>
        ${r.saveError ? `<p class="hint">${esc(r.saveError)}</p>` : ""}
      </div>
    </div>
    <div class="next-row">
      <button class="btn block" id="gr-next" type="button">Volgende →</button>
    </div>`;
}

function bindPractice() {
  document.getElementById("gr-stop").addEventListener("click", () => {
    if (g.review) {
      // Een half afgemaakte herhaling telt niet: de regel blijft aan de beurt
      g.review = null;
      location.hash = "#oefenen";
      return;
    }
    if (g.session.n) toast(`${g.session.good} van ${g.session.n} goed`);
    showOverview();
  });
  const card = grammarEl.querySelector(".gr-card");
  if (g.result) {
    g.result.correct ? pulse(card) : shake(card);
    const next = document.getElementById("gr-next");
    next.addEventListener("click", nextExercise);
    document.getElementById("gr-doubt")?.addEventListener("click", reportDoubt);
    next.focus();
    return;
  }
  const input = document.getElementById("gr-answer");
  input.focus();
  grammarEl.querySelectorAll(".accents [data-char]").forEach((b) =>
    b.addEventListener("click", () => insertAtCursor(input, b.dataset.char)));
  document.getElementById("gr-form").addEventListener("submit", (e) => {
    e.preventDefault();
    if (!input.value.trim()) return shake(input);
    check(input.value);
  });
  document.getElementById("gr-dont-know").addEventListener("click", () => check("", true));
  input.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      check("", true);
    }
  });
}

// reportDoubt: de vorm van deze opgave klopt volgens jou niet. Het woord wordt voor deze vorm niet
// meer gevraagd en komt in Controleren, waar iemand de goede vorm kan invullen.
async function reportDoubt() {
  try {
    await api("POST", "/api/grammar/doubt", { rule: g.ex.rule, word: g.ex.word, person: g.ex.person });
    g.result.doubted = true;
    toast(`Gemeld: ${g.ex.word} staat nu in Controleren`);
    renderPractice();
  } catch (err) {
    toast(err.message);
  }
}

async function check(value, unknown = false) {
  g.input = value;
  try {
    const res = await api("POST", "/api/grammar/check", { rule: g.ex.rule, word: g.ex.word, person: g.ex.person, input: value });
    g.result = { ...res, unknown };
  } catch (err) {
    toast(err.message);
    return;
  }
  g.session.n++;
  if (g.result.correct) g.session.good++;
  if (g.ex.variant) g.seen.add(g.ex.variant);
  if (g.review) {
    g.review.count++;
    if (g.result.correct) g.review.correct++;
  }
  // De score in het overzicht meteen bijwerken
  const rule = ruleById(g.ex.rule);
  if (rule) rule.score = g.result.score;
  renderPractice();
}
