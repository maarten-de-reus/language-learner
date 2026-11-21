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
