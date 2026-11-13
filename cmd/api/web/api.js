// Contact met de server en korte meldingen onderin het scherm, gedeeld door alle onderdelen.

const toastEl = document.getElementById("toast");

export async function api(method, path, body) {
  // Alle wijzigingen gaan als JSON: de server weigert andere verzoeken (bescherming tegen CSRF)
  const res = await fetch(path, method === "GET" ? { method } : {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body ?? {}),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = data.error || `Serverfout (${res.status})`;
    throw new Error(msg.charAt(0).toUpperCase() + msg.slice(1) + (/[.!?]$/.test(msg) ? "" : "."));
  }
  return data;
}

export function toast(msg) {
  toastEl.textContent = msg;
  toastEl.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => (toastEl.hidden = true), Math.max(2600, msg.length * 60)); // langere tekst blijft langer staan
}
