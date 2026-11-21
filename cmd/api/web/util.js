// Kleine hulpjes voor tekst, datums en animaties, gedeeld door alle onderdelen.

// De laatste stap: 4 keer achter elkaar goed op 90 dagen (engine.MasteredStep), daarna een
// controle per jaar. Overal in goud.
export const MASTERED_STEP = 6;

// Woordsoorten, zoals de server ze kent (catalog.WordTypes)
export const WORD_TYPES = [
  ["zn", "zelfstandig naamwoord"], ["ww", "werkwoord"], ["bn", "bijvoeglijk naamwoord"], ["bw", "bijwoord"],
  ["vnw", "voornaamwoord"], ["telw", "telwoord"], ["lw", "lidwoord"], ["vw", "voegwoord"],
  ["vz", "achterzetsel"], ["tw", "tussenwerpsel"], ["uitdr", "uitdrukking"],
];
export const wordTypeName = (code) => WORD_TYPES.find(([c]) => c === code)?.[1] ?? "";

// wordTypeOptions: <option>s voor een keuzelijst, met "onbekend" bovenaan
export const wordTypeOptions = (selected = "") =>
  `<option value="">onbekend</option>` +
  WORD_TYPES.map(([c, name]) => `<option value="${c}" ${c === selected ? "selected" : ""}>${name}</option>`).join("");

// ---------- Per letter laten zien waar je antwoord afwijkt ----------

const letters = (s) => Array.from(String(s ?? "").normalize("NFC").trim().toLowerCase());
const base = (c) => c.normalize("NFD").replace(/\p{M}/gu, "");

// alignLetters zet je antwoord en het goede antwoord letter voor letter naast elkaar (Levenshtein
// met terugweg). Stappen: same, acc (alleen het accent anders), sub (andere letter), miss (ontbrak),
// extra (te veel getypt).
function alignLetters(input, expected) {
  const a = letters(input), b = letters(expected);
  const d = Array.from({ length: a.length + 1 }, (_, i) => Array.from({ length: b.length + 1 }, (_, j) => (i === 0 ? j : j === 0 ? i : 0)));
  for (let i = 1; i <= a.length; i++) {
    for (let j = 1; j <= b.length; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : base(a[i - 1]) === base(b[j - 1]) ? 0.5 : 1;
      d[i][j] = Math.min(d[i - 1][j] + 1, d[i][j - 1] + 1, d[i - 1][j - 1] + cost);
    }
  }
  const steps = [];
  let i = a.length, j = b.length;
  while (i > 0 || j > 0) {
    const cost = i > 0 && j > 0 ? (a[i - 1] === b[j - 1] ? 0 : base(a[i - 1]) === base(b[j - 1]) ? 0.5 : 1) : Infinity;
    if (i > 0 && j > 0 && d[i][j] === d[i - 1][j - 1] + cost) {
      steps.unshift({ op: cost === 0 ? "same" : cost === 0.5 ? "acc" : "sub", got: a[i - 1], want: b[j - 1] });
      i--; j--;
    } else if (j > 0 && d[i][j] === d[i][j - 1] + 1) {
      steps.unshift({ op: "miss", want: b[j - 1] });
      j--;
    } else {
      steps.unshift({ op: "extra", got: a[i - 1] });
      i--;
    }
  }
  return { steps, distance: d[a.length][b.length] };
}

// diffHTML: het goede antwoord met daarin gemarkeerd wat er anders was dan wat je typte. Leeg als
// je niets typte of als je antwoord er te ver van af zit (dan is het een ander woord, geen tikfout).
export function diffHTML(input, expected) {
  if (!String(input ?? "").trim() || !expected?.length) return "";
  const best = expected.map((e) => ({ e, ...alignLetters(input, e) })).sort((x, y) => x.distance - y.distance)[0];
  const len = letters(best.e).length;
  if (best.distance === 0 || best.distance > Math.max(2, Math.ceil(len / 2))) return "";
  const html = best.steps.map((s) => {
    switch (s.op) {
      case "same": return esc(s.want);
      case "acc": return `<mark class="d-acc" title="accent: jij typte ${esc(s.got)}">${esc(s.want)}</mark>`;
      case "sub": return `<del class="d-extra" title="jij typte ${esc(s.got)}">${esc(s.got)}</del><mark class="d-sub" title="hier hoort ${esc(s.want)}">${esc(s.want)}</mark>`;
      case "miss": return `<mark class="d-miss" title="deze letter ontbrak">${esc(s.want)}</mark>`;
      default: return `<del class="d-extra" title="te veel getypt">${esc(s.got)}</del>`;
    }
  }).join("");
  const n = (op) => best.steps.filter((s) => s.op === op).length;
  const parts = [];
  if (n("acc")) parts.push(n("acc") === 1 ? "1 accent" : `${n("acc")} accenten`);
  if (n("sub")) parts.push(n("sub") === 1 ? "1 verkeerde letter" : `${n("sub")} verkeerde letters`);
  if (n("miss")) parts.push(n("miss") === 1 ? "1 letter vergeten" : `${n("miss")} letters vergeten`);
  if (n("extra")) parts.push(n("extra") === 1 ? "1 letter te veel" : `${n("extra")} letters te veel`);
  return `<p class="diff">Waar het verschilt: <span class="diff-word">${html}</span> <span class="muted">(${parts.join(", ")})</span></p>`;
}

// formHintHTML: je typte een andere vorm van het gevraagde woord (olvasok bij olvas). Zeg welke
// regel die vorm maakt, met een link naar die regel in Regels.
export function formHintHTML(form, input, expected) {
  if (!form) return "";
  const what = form.person ? `${form.ruleName}, ${form.person}` : form.ruleName;
  return `<p class="hint form-hint">Bijna: <b lang="hu">${esc(input.trim())}</b> is <b lang="hu">${esc(form.word)}</b>
    in de vorm <i>${esc(what)}</i>. Gevraagd werd <b lang="hu">${esc(expected.join(", "))}</b>.
    <a href="#regels/${encodeURIComponent(form.rule)}">Regel bekijken</a></p>`;
}

export const LANG = { nl: "Nederlands", hu: "Hongaars", en: "Engels" };
export const ACCENTS = ["á", "é", "í", "ó", "ö", "ő", "ú", "ü", "ű"];
const rtf = new Intl.RelativeTimeFormat("nl", { numeric: "auto" });

// Een herhaaldag begint om 04:00, net als op de server (engine.DayStartHour)
const DAY_START_HOUR = 4;

export const esc = (s) =>
  String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

// relative toont wanneer een herhaling is, in herhaaldagen: "morgen", "over 3 dagen".
// Binnen dezelfde herhaaldag in uren of minuten.
export function relative(iso) {
  const t = new Date(iso);
  const ms = t - Date.now();
  if (ms <= 0) return "nu";
  const reviewDay = (d) => {
    const shifted = new Date(d.getTime() - DAY_START_HOUR * 3600000);
    return Date.UTC(shifted.getFullYear(), shifted.getMonth(), shifted.getDate());
  };
  const days = Math.round((reviewDay(t) - reviewDay(new Date())) / 86400000);
  if (days > 0) return rtf.format(days, "day");
  if (ms < 3600000) return rtf.format(Math.max(1, Math.round(ms / 60000)), "minute");
  return rtf.format(Math.round(ms / 3600000), "hour");
}

// highlight markeert de woorden in een (ge-escapete) voorbeeldzin, hoofdletterongevoelig
export function highlight(sentence, words) {
  let html = esc(sentence);
  for (const w of words) {
    if (!w || !w.trim()) continue;
    const re = new RegExp(esc(w.trim()).replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "giu");
    html = html.replace(re, (m) => `<mark>${m}</mark>`);
  }
  return html;
}

export const days = (n) => (n === 1 ? "1 dag" : `${n} dagen`);

// fold: kleine letters zonder accenten, zodat "ido" ook "idő" vindt
export const fold = (s) => String(s ?? "").normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().trim();

export function insertAtCursor(input, text) {
  if (input.disabled) return;
  const { selectionStart: s, selectionEnd: e, value } = input;
  input.value = value.slice(0, s) + text + value.slice(e);
  input.setSelectionRange(s + text.length, s + text.length);
  input.focus();
}

export function shake(el) {
  el.classList.remove("shake");
  void el.offsetWidth;
  el.classList.add("shake");
}

export function pulse(el) {
  el.classList.remove("pop");
  void el.offsetWidth;
  el.classList.add("pop");
}
