// Het begin van de app: wisselen tussen de tabbladen. Elk onderdeel heeft zijn eigen module
// met een show-functie; hier wordt alleen bepaald welke er zichtbaar is.

import { showQuiz } from "./quiz.js";
import { showCards } from "./cards.js";
import { showStatistics } from "./stats.js";
import { showManage } from "./manage.js";
import { showReview } from "./review.js";

const TABS = {
  oefenen: { section: "app", show: showQuiz },
  kaarten: { section: "cards", show: showCards },
  statistiek: { section: "stats", show: showStatistics },
  beheer: { section: "manage", show: showManage },
  controleren: { section: "review", show: showReview },
};

function showTab() {
  const tab = location.hash.slice(1) in TABS ? location.hash.slice(1) : "oefenen";
  for (const [name, { section }] of Object.entries(TABS)) {
    document.getElementById(section).hidden = name !== tab;
    document.getElementById(`tab-${name}`).setAttribute("aria-current", name === tab ? "page" : "false");
  }
  document.getElementById("quiz-foot").hidden = tab !== "oefenen";
  document.getElementById("mode-switch").hidden = tab !== "oefenen";
  // De voortgangsbalk hoort bij oefenen en statistiek; elders kost hij alleen ruimte
  document.getElementById("steps").hidden = tab !== "oefenen" && tab !== "statistiek";
  document.body.classList.toggle("mode-manage", tab !== "oefenen");
  TABS[tab].show();
}

window.addEventListener("hashchange", showTab);
showTab();
