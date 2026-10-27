"use strict";

// Statistiek: reeks, score, activiteit, vooruitblik, moeilijkste woorden en het effect van herhalen.
// Gebruikt helpers uit app.js (esc, days) en manage.js (api, refreshStats, view).

const statsEl = document.getElementById("stats");

const dayLabel = (iso, opts = { day: "numeric", month: "short" }) => {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d).toLocaleDateString("nl-NL", opts);
};
const pct = (x) => `${Math.round(x * 100)}%`;
const times = (n) => (n === 1 ? "1 keer" : `${n} keer`);

async function showStatistics() {
  statsEl.innerHTML = `<div class="skeleton"></div>`;
  try {
    const [report, quiz] = await Promise.all([api("GET", "/api/stats"), api("GET", "/api/state")]);
    renderStatistics(report, quiz.stats);
    renderStats(quiz.stats);
  } catch (err) {
    statsEl.innerHTML = `<p class="error">Kon de statistiek niet laden: ${esc(err.message)}</p>`;
  }
}

function renderStatistics(r, quiz) {
  if (r.answers === 0 && r.queue.waiting === 0) {
    statsEl.innerHTML = `
      <article class="card center">
        <div class="big-emoji">📊</div>
        <h2>Nog geen statistiek</h2>
        <p class="muted">Zodra je geoefend hebt, zie je hier je reeks, je score en welke woorden lastig zijn.</p>
        <a class="btn" href="#oefenen">Beginnen met oefenen</a>
      </article>`;
    return;
  }

  const todayTotal = r.today.correct + r.today.wrong;
  const streakSub = r.streak === 0
    ? (r.longestStreak > 1 ? `langste: ${r.longestStreak} dagen` : "oefen vandaag om te beginnen")
    : !r.practicedToday ? "oefen vandaag om hem te houden"
    : r.longestStreak > r.streak ? `langste: ${r.longestStreak} dagen` : "je langste tot nu toe";

  statsEl.innerHTML = `
    <div class="sx-kpis">
      ${kpi("Reeks", r.streak, r.streak === 1 ? "dag op rij" : "dagen op rij", streakSub)}
      ${kpi("Vandaag", todayTotal, todayTotal === 1 ? "woord" : "woorden",
        todayTotal ? `${r.today.correct} goed · ${r.today.wrong} fout` : "nog niet geoefend")}
      ${kpi("Score", pct(r.accuracy), "goed bij de 1e poging", `${r.firstCorrect} van ${r.firstAttempts}`)}
      ${kpi("Geoefend", r.wordsPracticed, "verschillende woorden", `op ${r.daysPracticed} ${r.daysPracticed === 1 ? "dag" : "dagen"}`)}
    </div>

    <section class="sx-card">
      <div class="sx-head">
        <div><h3>Activiteit <span class="sx-tag">terugblik</span></h3>
          <p class="muted">Eerste pogingen per dag, van ${esc(dayLabel(r.activity[0].date, { day: "numeric", month: "long" }))} t/m vandaag</p></div>
        <ul class="sx-legend" aria-label="Legenda">
          <li><span class="sx-swatch good"></span>✓ goed</li>
          <li><span class="sx-swatch bad"></span>✕ fout</li>
        </ul>
      </div>
      ${activityChart(r.activity)}
    </section>

    <section class="sx-card">
      <div class="sx-head">
        <div><h3>Planning: komende 2 weken <span class="sx-tag ahead">vooruitblik</span></h3>
          <p class="muted">Hoeveel woorden er per dag aan de beurt zijn, van vandaag t/m
          ${esc(dayLabel(r.forecast[r.forecast.length - 1].date, { weekday: "long", day: "numeric", month: "long" }))}.
          Wat je nog moet inhalen telt bij vandaag.</p></div>
      </div>
      ${forecastChart(r.forecast)}
    </section>

    <div class="sx-grid">
      <section class="sx-card">
        <h3>Moeilijkste woorden</h3>
        <p class="muted">Vaakst fout bij de eerste poging. Klik om te bewerken, bijv. een hint of voorbeeldzin toevoegen.</p>
        ${hardestList(r.hardest)}
      </section>

      <div class="sx-stack">
        ${queueCard(r.queue, quiz)}
        <section class="sx-card">
          <h3>Per richting</h3>
          ${directions(r.directions)}
        </section>
        <section class="sx-card">
          <h3>Soort fouten</h3>
          ${mistakes(r.mistakes)}
        </section>
        <section class="sx-card">
          <h3>Helpt herhalen?</h3>
          ${retryEffect(r.retry)}
        </section>
      </div>
    </div>
    ${r.skipped ? `<p class="muted sx-note">${r.skipped} onleesbare regel(s) in het antwoordenlog overgeslagen.</p>` : ""}`;

  bindCharts();
  statsEl.querySelectorAll("[data-word]").forEach((row) => {
    const open = () => {
      view.query = row.dataset.word; // zoekterm voor het beheer
      view.openId = row.dataset.id;
      location.hash = "#beheer";
    };
    row.addEventListener("click", open);
    row.addEventListener("keydown", (e) => { if (e.key === "Enter") open(); });
  });
}

function kpi(label, value, unit, sub) {
  return `<div class="sx-kpi">
    <span class="sx-kpi-label">${label}</span>
    <span class="sx-kpi-value">${esc(value)}</span>
    <span class="sx-kpi-unit">${esc(unit)}</span>
    <span class="sx-kpi-sub muted">${esc(sub)}</span>
  </div>`;
}

// ---------- Grafieken ----------

// niceMax rondt het maximum op tot een rond getal voor de hulplijn
function niceMax(n) {
  if (n <= 5) return Math.max(n, 1);
  const step = 10 ** Math.floor(Math.log10(n));
  return Math.ceil(n / (step / 2)) * (step / 2);
}

function chartFrame({ cols, max, label, tableHead, tableRows, ticks }) {
  return `
    <div class="sx-chart" role="img" aria-label="${esc(label)}">
      <div class="sx-gridline top"><span>${max}</span></div>
      <div class="sx-gridline mid"><span>${max / 2}</span></div>
      <div class="sx-plot">${cols}</div>
      <div class="sx-axis">${ticks}</div>
      <div class="sx-tip" hidden></div>
    </div>
    <details class="sx-table">
      <summary>Als tabel</summary>
      <div class="sx-table-wrap"><table>
        <thead><tr>${tableHead.map((h) => `<th>${h}</th>`).join("")}</tr></thead>
        <tbody>${tableRows.map((r) => `<tr>${r.map((c) => `<td>${c}</td>`).join("")}</tr>`).join("")}</tbody>
      </table></div>
    </details>`;
}

// ticksFor zet om de `every` dagen een datum onder de as, gerekend vanaf het anker ("vandaag").
// Het andere uiteinde krijgt ook een datum, zodat het bereik van de grafiek altijd zichtbaar is.
function ticksFor(dates, every, anchor) {
  const n = dates.length;
  return dates.map((d, i) => {
    const k = anchor === "end" ? n - 1 - i : i;
    const otherEnd = k === n - 1;
    const show = k % every === 0 || (otherEnd && k % every >= every / 2);
    const text = k === 0 ? "vandaag" : dayLabel(d);
    return `<span>${show ? esc(text) : ""}</span>`;
  }).join("");
}

function activityChart(activity) {
  const max = niceMax(Math.max(...activity.map((d) => d.correct + d.wrong)));
  const h = (n) => `${(n / max) * 100}%`;
  const cols = activity.map((d) => {
    const tip = `${dayLabel(d.date, { weekday: "short", day: "numeric", month: "short" })}: ✓ ${d.correct} goed · ✕ ${d.wrong} fout${d.retries ? ` · ${d.retries} herhaald` : ""}`;
    return `<div class="sx-col" data-tip="${esc(tip)}">
      <div class="sx-bar">
        ${d.wrong ? `<span class="sx-seg bad" style="height:${h(d.wrong)}"></span>` : ""}
        ${d.correct ? `<span class="sx-seg good" style="height:${h(d.correct)}"></span>` : ""}
      </div>
    </div>`;
  }).join("");

  const active = activity.filter((d) => d.correct + d.wrong > 0);
  return chartFrame({
    cols, max,
    label: `Activiteit: op ${active.length} van de ${activity.length} dagen geoefend, in totaal ${active.reduce((a, d) => a + d.correct, 0)} goed en ${active.reduce((a, d) => a + d.wrong, 0)} fout.`,
    ticks: ticksFor(activity.map((d) => d.date), 7, "end"),
    tableHead: ["Dag", "Goed", "Fout", "Herhaald"],
    tableRows: active.map((d) => [dayLabel(d.date), d.correct, d.wrong, d.retries]),
  });
}

function forecastChart(forecast) {
  const peak = forecast.reduce((a, d) => (d.due > a.due ? d : a), forecast[0]);
  const max = niceMax(peak.due);
  const cols = forecast.map((d, i) => {
    const when = i === 0 ? "vandaag" : i === 1 ? "morgen" : dayLabel(d.date, { weekday: "short", day: "numeric", month: "short" });
    return `<div class="sx-col" data-tip="${esc(`${when}: ${d.due} ${d.due === 1 ? "woord" : "woorden"}`)}">
      <div class="sx-bar">${d.due ? `<span class="sx-seg due ${d === peak ? "peak" : ""}" style="height:${(d.due / max) * 100}%"></span>` : ""}</div>
    </div>`;
  }).join("");

  return `
    ${peak.due ? `<p class="sx-callout">Drukste dag: <b>${esc(dayLabel(peak.date, { weekday: "long", day: "numeric", month: "long" }))}</b> met ${peak.due} woorden</p>` : ""}
    ${chartFrame({
      cols, max,
      label: `Komende ${forecast.length} dagen: ${forecast.reduce((a, d) => a + d.due, 0)} herhalingen, het meest op ${dayLabel(peak.date)} (${peak.due}).`,
      ticks: ticksFor(forecast.map((d) => d.date), 7, "start"),
      tableHead: ["Dag", "Woorden"],
      tableRows: forecast.map((d) => [dayLabel(d.date), d.due]),
    })}`;
}

function bindCharts() {
  statsEl.querySelectorAll(".sx-chart").forEach((chart) => {
    const tip = chart.querySelector(".sx-tip");
    chart.querySelectorAll(".sx-col").forEach((col) => {
      col.addEventListener("mouseenter", () => {
        tip.textContent = col.dataset.tip;
        tip.hidden = false;
        const c = chart.getBoundingClientRect();
        const b = col.getBoundingClientRect();
        // Binnen de grafiek houden
        const x = Math.min(Math.max(b.left - c.left + b.width / 2, tip.offsetWidth / 2), c.width - tip.offsetWidth / 2);
        tip.style.left = `${x}px`;
        col.classList.add("hover");
      });
      col.addEventListener("mouseleave", () => {
        tip.hidden = true;
        col.classList.remove("hover");
      });
    });
  });
}

// ---------- Wachtrij ----------

function queueCard(q, quiz) {
  const unlimited = q.perDay === 0;
  const today = quiz ? `<p class="muted">Vandaag ${quiz.newToday} ${quiz.newToday === 1 ? "nieuw woord" : "nieuwe woorden"} geleerd${unlimited ? "" : ` (limiet ${q.perDay} per dag)`}.</p>` : "";
  let body;
  if (q.waiting === 0) {
    body = `<p class="sx-empty">De wachtrij is leeg: alle woorden zijn al eens gevraagd.</p>`;
  } else if (unlimited) {
    body = `<p class="sx-big">${q.waiting} <small>in de wachtrij</small></p>
      <p class="muted">Geen daglimiet: ze komen allemaal meteen aan de beurt.</p>`;
  } else {
    body = `<p class="sx-big">${q.waiting} <small>in de wachtrij</small></p>
      <p class="muted">Bij ${q.perDay} per dag ben je er over ongeveer <b>${q.days} ${q.days === 1 ? "dag" : "dagen"}</b> doorheen.
      Wil je sneller? Verhoog de limiet in <a href="#beheer">Beheer</a>, of kies na een ronde voor "Nog 10 nieuwe woorden".</p>`;
  }
  return `<section class="sx-card"><h3>Nieuwe woorden</h3>${today}${body}</section>`;
}

// ---------- Lijsten ----------

function hardestList(hardest) {
  if (!hardest.length) return `<p class="sx-empty">Nog geen fouten gemaakt. Knap!</p>`;
  return `<ol class="sx-hardest">
    ${hardest.map((h) => `
      <li data-id="${esc(h.id)}" data-word="${esc(h.prompt)}" tabindex="0" role="link">
        <span class="sx-word"><b lang="${esc(h.from)}">${esc(h.prompt)}</b>${h.context ? ` <span class="chip">${esc(h.context)}</span>` : ""}
          <span class="muted">→ ${esc(h.answers.join(", "))}</span></span>
        <span class="sx-count"><span class="sx-badge">✕ ${times(h.wrong)}</span> <span class="muted">van ${h.attempts}</span></span>
      </li>`).join("")}
  </ol>`;
}

function mistakes(m) {
  const total = m.accentOnly + m.otherMeaning + m.other;
  if (!total) return `<p class="sx-empty">Nog geen fouten.</p>`;
  const row = (label, n, note) => `
    <li><span>${label}<small class="muted">${note}</small></span>
      <span class="sx-meter"><span style="width:${(n / total) * 100}%"></span></span>
      <b>${n}</b></li>`;
  return `<ul class="sx-mistakes">
    ${row("Alleen accenten", m.accentOnly, "bijv. kave i.p.v. kávé")}
    ${row("Andere betekenis", m.otherMeaning, "bijv. tijd i.p.v. weer")}
    ${row("Anders fout", m.other, "")}
  </ul>`;
}

// Minimaal aantal herhalingen per richting voordat een vergelijking iets zegt
const MIN_DIRECTION_REVIEWS = 10;
const DIRECTION_NOTE = { hu: "herkennen", nl: "zelf bedenken" };

function directions(dirs) {
  if (!dirs.length) return `<p class="sx-empty">Nog geen antwoorden.</p>`;
  const share = (s) => (s.attempts ? s.correct / s.attempts : 0);
  const rows = dirs.map((d) => `
    <li>
      <span class="sx-dir-head"><span>${esc(LANG[d.from] || d.from)} → ${esc(LANG[d.to] || d.to)}
        <small class="muted">${esc(DIRECTION_NOTE[d.from] || "")}</small></span>
        <b>${d.review.attempts ? pct(share(d.review)) : "–"}</b></span>
      <span class="sx-meter good"><span style="width:${share(d.review) * 100}%"></span></span>
      <small class="muted">${d.review.correct} van ${d.review.attempts} herhalingen goed${
        d.new.attempts ? ` · nieuw: ${d.new.correct} van ${d.new.attempts} meteen goed` : ""}</small>
    </li>`).join("");

  // Een conclusie pas als beide richtingen genoeg herhalingen hebben en duidelijk verschillen
  let note = "";
  const enough = dirs.filter((d) => d.review.attempts >= MIN_DIRECTION_REVIEWS);
  if (enough.length >= 2) {
    const [weak, strong] = [...enough].sort((a, b) => share(a.review) - share(b.review));
    note = share(strong.review) - share(weak.review) >= 0.1
      ? `${esc(LANG[weak.from])} → ${esc(LANG[weak.to])} blijft achter. Oefen die richting extra onder <a href="#kaarten">Kaarten</a>.`
      : "Beide richtingen gaan ongeveer even goed.";
  } else if (dirs.some((d) => d.review.attempts)) {
    note = `Een vergelijking volgt bij minimaal ${MIN_DIRECTION_REVIEWS} herhalingen per richting.`;
  }
  return `<ul class="sx-dirs">${rows}</ul><p class="muted sx-dir-note">${note}</p>`;
}

function retryEffect(e) {
  const intro = `<p class="muted">Woorden die je fout had en pas bij de herhaling aan het eind van de ronde wist: wist je ze bij de volgende echte herhaling ook?</p>`;
  if (!e.cases) return `${intro}<p class="sx-empty">Nog geen gegevens. Die komen vanzelf: na een paar dagen oefenen komen zulke woorden terug.</p>`;
  const share = e.nextCorrect / e.cases;
  return `${intro}
    <p class="sx-big">${e.nextCorrect} <small>van de ${e.cases} keer goed</small></p>
    <span class="sx-meter wide" aria-hidden="true"><span style="width:${share * 100}%"></span></span>
    <p class="muted">${e.enough
      ? (share >= 0.6 ? "De herhaling lijkt te helpen: de meeste van deze woorden blijven hangen."
        : "Veel van deze woorden zijn een dag later weer weg: de herhaling helpt vooral op korte termijn.")
      : "Nog te weinig om iets te zeggen (minimaal 5 keer)."}</p>`;
}
