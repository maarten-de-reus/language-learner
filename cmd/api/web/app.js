"use strict";

const LANG = { nl: "Nederlands", hu: "Hongaars", en: "Engels" };
const ACCENTS = ["á", "é", "í", "ó", "ö", "ő", "ú", "ü", "ű"];
const rtf = new Intl.RelativeTimeFormat("nl", { numeric: "auto" });

const app = document.getElementById("app");
const errorBox = document.getElementById("error");
const tooltip = document.getElementById("tooltip");

let state = null;
let busy = false;
let renderedCardId = null;

// ---------- Helpers ----------

const esc = (s) =>
  String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

function relative(iso) {
  const min = (new Date(iso) - Date.now()) / 60000;
  if (Math.abs(min) < 60) return rtf.format(Math.round(min), "minute");
  const h = min / 60;
  if (Math.abs(h) < 20) return rtf.format(Math.round(h), "hour");
  return rtf.format(Math.round(h / 24), "day");
}

// highlight markeert de woorden in een (ge-escapete) voorbeeldzin, hoofdletterongevoelig
function highlight(sentence, words) {
  let html = esc(sentence);
  for (const w of words) {
    if (!w || !w.trim()) continue;
    const re = new RegExp(esc(w.trim()).replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "giu");
    html = html.replace(re, (m) => `<mark>${m}</mark>`);
  }
  return html;
}

const days = (n) => (n === 1 ? "1 dag" : `${n} dagen`);

async function call(path, body) {
  const opts = body === undefined ? { method: path === "/api/state" ? "GET" : "POST" } : {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
  const res = await fetch(path, opts);
  if (!res.ok) throw new Error(`${res.status} ${await res.text()}`);
  return res.json();
}

async function act(path, body) {
  if (busy) return;
  busy = true;
  try {
    state = await call(path, body);
    errorBox.hidden = true;
    render();
  } catch (err) {
    errorBox.textContent = `Er ging iets mis met de server: ${err.message}`;
    errorBox.hidden = false;
  } finally {
    busy = false;
  }
}

// ---------- Stats ----------

function renderStats(stats) {
  document.getElementById("kpi-due").textContent = stats.due;
  document.getElementById("kpi-total").textContent = stats.total;

  const bar = document.getElementById("steps-bar");
  const legend = document.getElementById("steps-legend");
  const total = stats.steps.reduce((a, b) => a + b, 0) || 1;

  bar.innerHTML = stats.steps
    .map((n, i) => n === 0 ? "" :
      `<div class="seg" style="flex-grow:${n};background:var(--s${i})" data-step="${i}"></div>`)
    .join("");
  bar.setAttribute("aria-label", stats.steps
    .map((n, i) => `${n} concepten met interval ${days(stats.intervals[i])}`).join(", "));

  legend.innerHTML = stats.steps
    .map((n, i) => `<li>
      <span class="lbl"><span class="dot" style="background:var(--s${i})"></span>${stats.intervals[i]}d</span>
      <span class="num">${n}</span></li>`)
    .join("");

  bar.querySelectorAll(".seg").forEach((seg) => {
    seg.addEventListener("mousemove", (e) => {
      const i = Number(seg.dataset.step);
      const n = stats.steps[i];
      tooltip.textContent = `${n} concepten · interval ${days(stats.intervals[i])} · ${Math.round((n / total) * 100)}%`;
      const box = seg.closest(".steps").getBoundingClientRect();
      tooltip.style.left = `${e.clientX - box.left}px`;
      tooltip.style.top = `${e.clientY - box.top - 10}px`;
      tooltip.hidden = false;
    });
    seg.addEventListener("mouseleave", () => (tooltip.hidden = true));
  });
}

// ---------- Quiz ----------

function renderCard(card) {
  const pct = ((card.position - 1) / card.runSize) * 100;
  const interval = state.stats.intervals[card.step];

  app.innerHTML = `
    <article class="card" id="card">
      <div class="run-progress">
        <div class="run-track"><div class="run-fill" style="width:${pct}%"></div></div>
        <span class="run-count">${card.position} / ${card.runSize}</span>
      </div>

      <div class="chips">
        <span class="chip">${esc(LANG[card.from] || card.from)} → ${esc(LANG[card.to] || card.to)}</span>
        ${card.retry
          ? `<span class="chip retry">↻ Herhaling · telt niet mee voor de planning</span>`
          : `<span class="chip"><span class="dot" style="background:var(--s${card.step})"></span>interval ${days(interval)}</span>`}
        ${card.context ? `<span class="chip">${esc(card.context)}</span>` : ""}
      </div>

      <p class="prompt-label">Vertaal</p>
      <h2 class="prompt">${esc(card.prompt)}</h2>
      ${card.example ? `<p class="example" lang="${esc(card.from)}">“${highlight(card.example, [card.prompt])}”</p>` : ""}

      <form id="answer-form" autocomplete="off">
        <div class="answer-row">
          <input class="answer-input" id="answer" name="answer" lang="${esc(card.to)}"
                 placeholder="Jouw antwoord in het ${esc((LANG[card.to] || card.to).toLowerCase())}…"
                 autocapitalize="off" autocorrect="off" spellcheck="false" aria-label="Jouw antwoord">
          <button class="btn" id="check" type="submit">Controleer</button>
        </div>
        ${card.to === "hu" ? `<div class="accents" aria-label="Hongaarse tekens">
          ${ACCENTS.map((a) => `<button type="button" data-char="${a}" tabindex="-1">${a}</button>`).join("")}
        </div>` : ""}
      </form>

      <div id="feedback-slot"></div>
    </article>`;

  const input = document.getElementById("answer");
  document.getElementById("answer-form").addEventListener("submit", (e) => {
    e.preventDefault();
    if (!input.value.trim()) {
      shake(input);
      return;
    }
    act("/api/answer", { input: input.value });
  });

  app.querySelectorAll(".accents button").forEach((b) => {
    b.addEventListener("mousedown", (e) => e.preventDefault()); // focus in het invoerveld houden
    b.addEventListener("click", () => insertAtCursor(input, b.dataset.char));
  });

  input.focus();
}

function insertAtCursor(input, text) {
  if (input.disabled) return;
  const { selectionStart: s, selectionEnd: e, value } = input;
  input.value = value.slice(0, s) + text + value.slice(e);
  input.setSelectionRange(s + text.length, s + text.length);
  input.focus();
}

function renderFeedback(fb) {
  const slot = document.getElementById("feedback-slot");
  const input = document.getElementById("answer");
  const check = document.getElementById("check");
  const lockAccents = (on) => app.querySelectorAll(".accents button").forEach((b) => (b.disabled = on));

  if (!fb) {
    slot.innerHTML = "";
    input.disabled = false;
    check.hidden = false;
    lockAccents(false);
    return;
  }

  input.value = fb.input;
  input.disabled = true;
  input.classList.toggle("is-good", fb.correct);
  input.classList.toggle("is-bad", !fb.correct);
  check.hidden = true;
  lockAccents(true);

  const answers = fb.expected.map((x) => `<span class="answer">${esc(x)}</span>`).join(" of ");
  const isLast = state.card.position === state.card.runSize;
  const title = fb.correct
    ? (fb.retry ? "Goed! Morgen volgt de echte test." : "Goed zo!")
    : (fb.retry ? "Nog niet goed" : "Helaas, niet goed");

  slot.innerHTML = `
    <div class="feedback ${fb.correct ? "good" : "bad"}" role="status">
      <span class="icon" aria-hidden="true">${fb.correct ? "✓" : "✕"}</span>
      <div>
        <strong>${title}</strong>
        ${fb.correct
          ? (fb.expected.length > 1 ? `<p>Ook goed: ${answers}</p>` : "")
          : `<p>Het juiste antwoord: ${answers}</p>`}
        <p>Volgende herhaling ${esc(relative(fb.nextReview))} · interval ${days(state.stats.intervals[fb.step])}</p>
        ${fb.otherMeaning ? `<p class="hint">Klopt, <b>${esc(state.card.prompt)}</b> kan ook
          ‘${esc(fb.otherMeaning.answer)}’ betekenen${fb.otherMeaning.context ? ` (${esc(fb.otherMeaning.context)})` : ""},
          maar hier wordt ‘${esc(fb.expected.join(", "))}’ bedoeld.</p>` : ""}
        ${fb.example ? `<p class="example">“${highlight(fb.example, [state.card.prompt, ...fb.expected])}”</p>` : ""}
        ${fb.willRetry ? `<p>Je krijgt dit woord aan het eind van de ronde nog een keer.</p>` : ""}
        ${fb.accentOnly ? `<span class="hint">Bijna! Alleen de accenten kloppen niet.</span>` : ""}
      </div>
    </div>
    <div class="next-row">
      <button class="btn block" id="next" type="button">${isLast ? "Bekijk resultaat" : "Volgende"} →</button>
    </div>`;

  const card = document.getElementById("card");
  fb.correct ? pulse(card) : shake(card);

  const next = document.getElementById("next");
  next.addEventListener("click", () => act("/api/next"));
  next.focus();
}

function shake(el) {
  el.classList.remove("shake");
  void el.offsetWidth;
  el.classList.add("shake");
}

function pulse(el) {
  el.classList.remove("pop");
  void el.offsetWidth;
  el.classList.add("pop");
}

// ---------- Run finished ----------

function retryNote(r) {
  if (!r.retryCorrect) return `↻ ook bij ${r.retries === 1 ? "de herhaling" : `${r.retries} herhalingen`} fout`;
  return `↻ goed bij de ${r.retries === 1 ? "1e" : `${r.retries}e`} herhaling`;
}

function renderDone() {
  const { results, stats } = state;
  const nextDue = stats.nextDue ? `De volgende herhaling is ${esc(relative(stats.nextDue))}.` : "";

  if (!results || results.length === 0) {
    app.innerHTML = `
      <article class="card center">
        <div class="big-emoji">🌿</div>
        <h2>Alles bijgewerkt</h2>
        <p class="muted">Er staan nu geen woorden klaar om te herhalen. ${nextDue}</p>
        <button class="btn secondary" id="again" type="button">Opnieuw controleren</button>
      </article>`;
  } else {
    const good = results.filter((r) => r.correct).length;
    const pct = good / results.length;
    const title = pct === 1 ? "Foutloos! Nagyszerű!" : pct >= 0.7 ? "Mooi gedaan!" : "Ronde voltooid";
    const more = stats.due > 0
      ? `Er staan nog ${stats.due} concepten klaar.`
      : `Je bent klaar voor nu. ${nextDue}`;

    app.innerHTML = `
      <article class="card center">
        <h2>${title}</h2>
        <p class="score">${good}<small> / ${results.length}</small></p>
        <p class="muted">${more}</p>
        <ul class="results">
          ${results.map((r) => `
            <li class="${r.correct ? "ok" : "ko"}">
              <span class="mark">${r.correct ? "✓" : "✕"}</span>
              <span class="src">${esc(r.prompt)}</span>
              <span class="exp">${r.correct ? "" : `<span class="yours">${esc(r.input)}</span>`}${esc(r.expected.join(", "))}</span>
              ${r.retries > 0 ? `<span class="retry-note ${r.retryCorrect ? "ok" : "ko"}">${retryNote(r)}</span>` : ""}
            </li>`).join("")}
        </ul>
        <button class="btn block" id="again" type="button">${stats.due > 0 ? "Nieuwe ronde" : "Opnieuw controleren"}</button>
      </article>`;
  }

  const again = document.getElementById("again");
  again.addEventListener("click", () => act("/api/run"));
  again.focus();
}

// ---------- Main render ----------

function render() {
  renderStats(state.stats);

  if (state.saveError) {
    errorBox.textContent = state.saveError;
    errorBox.hidden = false;
  }

  if (state.done) {
    renderedCardId = null;
    renderDone();
    return;
  }

  // Een herhaling heeft hetzelfde ID als de eerste poging; positie maakt de kaart uniek
  const cardKey = `${state.card.id}#${state.card.position}`;
  if (cardKey !== renderedCardId || !document.getElementById("card")) {
    renderCard(state.card);
    renderedCardId = cardKey;
  }
  renderFeedback(state.feedback);
}

app.innerHTML = `<div class="skeleton"></div>`;
// De start (en het wisselen tussen Oefenen en Beheer) regelt manage.js
