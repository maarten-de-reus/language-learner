// Regels: Hongaarse vormen leren maken (naamvallen, meervoud, vervoegen) met je eigen woorden.
// Je kiest zelf welke regels je oefent; de antwoorden tellen niet mee voor je planning.

import { ACCENTS, esc, insertAtCursor, pulse, shake } from "./util.js";
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
};

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
  // Vanuit de feedback in Oefenen of Kaarten: #regels/ww-onbep opent die regel
  const wanted = decodeURIComponent(location.hash.split("/")[1] ?? "");
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
  const active = d.rules.filter((r) => r.active && r.words > 0);
  const groups = [...new Set(d.rules.map((r) => r.group))];
  grammarEl.innerHTML = `
    <article class="card gr-head">
      <h2>Regels</h2>
      <p class="muted">Leer de vormen maken in plaats van ze te onthouden: elke opgave gebruikt een woord uit je eigen lijst,
        en na je antwoord zie je stap voor stap hoe de vorm ontstaat. <b>Telt niet mee voor je planning.</b></p>
      ${d.untyped ? `<p class="gr-note">${d.untyped} ${d.untyped === 1 ? "kaart heeft" : "kaarten hebben"} nog geen woordsoort en
        ${d.untyped === 1 ? "doet" : "doen"} niet mee. Vul die in bij <a href="#beheer">Beheer</a> (filter <i>woordsoort onbekend</i>).</p>` : ""}
      <div class="gr-start">
        <button class="btn" id="gr-start" type="button" ${active.length ? "" : "disabled"}>Oefenen</button>
        <span class="muted">${active.length
          ? `${active.length} ${active.length === 1 ? "regel" : "regels"} aan: ${esc(active.map((r) => r.name).join(", "))}`
          : "Zet hieronder een of meer regels aan."}</span>
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
  const none = r.words === 0;
  return `
    <li class="gr-rule ${open ? "open" : ""} ${none ? "none" : ""}">
      <div class="gr-rule-head">
        <label class="switch" title="${none ? "Geen woorden met deze woordsoort" : r.active ? "Uitzetten" : "Aanzetten"}">
          <input type="checkbox" data-rule-toggle="${esc(r.id)}" ${r.active ? "checked" : ""} ${none && !r.active ? "disabled" : ""}>
          <span class="switch-track" aria-hidden="true"></span>
          <span class="sr-only">${esc(r.name)} oefenen</span>
        </label>
        <button type="button" class="gr-rule-title" data-rule-open="${esc(r.id)}" aria-expanded="${open}">
          <span class="gr-name" lang="hu">${esc(r.name)}</span>
          <span class="gr-meaning">${esc(r.meaning)}</span>
          <span class="gr-meta">
            <span class="gr-score ${scoreClass(r.score)}">${scoreText(r.score)}</span>
            <span class="muted">${none ? "geen woorden" : `${r.words} ${r.words === 1 ? "woord" : "woorden"}`}</span>
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
          ${none ? `<p class="muted">Nog geen woorden waar deze regel op werkt: geef je woorden een woordsoort in Beheer.</p>`
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
  grammarEl.querySelectorAll("[data-rule-try]").forEach((b) =>
    b.addEventListener("click", () => startPractice(b.dataset.ruleTry)));
}

// ---------- Oefenen ----------

async function startPractice(onlyRule) {
  Object.assign(g, { onlyRule, session: { n: 0, good: 0 }, ex: null });
  await nextExercise();
}

async function nextExercise() {
  const params = new URLSearchParams();
  if (g.onlyRule) params.set("rule", g.onlyRule);
  if (g.ex) params.set("avoid", g.ex.word);
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
      <span>${g.onlyRule ? `Alleen ${esc(rule.name)}` : "Regels oefenen"}</span>
      <span class="run-count">${g.session.n ? `${g.session.good} van ${g.session.n} goed` : ""}</span>
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
  // De score in het overzicht meteen bijwerken
  const rule = ruleById(g.ex.rule);
  if (rule) rule.score = g.result.score;
  renderPractice();
}
