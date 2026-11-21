// Het begin van de app: wisselen tussen de tabbladen. Elk onderdeel heeft zijn eigen module
// met een show-functie; hier wordt alleen bepaald welke er zichtbaar is.

import { showQuiz } from "./quiz.js";
import { showCards } from "./cards.js";
import { showGrammar } from "./grammar.js";
import { showStatistics } from "./stats.js";
import { showManage } from "./manage.js";
import { showReview } from "./review.js";

const TABS = {
  oefenen: { section: "app", show: showQuiz },
  kaarten: { section: "cards", show: showCards },
  regels: { section: "grammar", show: showGrammar },
  statistiek: { section: "stats", show: showStatistics },
  beheer: { section: "manage", show: showManage },
  controleren: { section: "review", show: showReview },
};

function showTab() {
  // #regels/ban opent het tabblad Regels met de uitleg van -ban/-ben
  const name = location.hash.slice(1).split("/")[0];
  const tab = name in TABS ? name : "oefenen";
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
