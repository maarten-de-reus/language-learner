// Kleine hulpjes voor tekst, datums en animaties, gedeeld door alle onderdelen.

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
