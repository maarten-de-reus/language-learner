# Magyar – Dutch ↔ Hungarian vocabulary trainer

A small spaced-repetition trainer for learning Hungarian from Dutch (and back), with a web interface in Dutch. It is a single Go binary with no external dependencies; the frontend is embedded.

## Features

- **Spaced repetition** with fixed steps of 1, 3, 7, 16, 40 and 90 days (with a little random spread on the longer intervals). A word that is right 4 times in a row at 90 days becomes **mastered** (shown in gold): from then on it only comes back once a year, and a mistake sends it back to 40 days. In Beheer you can put a mastered word back into the reviews. A new review day starts at 04:00, so a word practised in the evening comes back the next morning. A wrong answer brings the word back the next day, two steps lower instead of back to the start: forgetting a word at 40 days costs a review tomorrow and one at 16 days, not the whole ladder again. A mistake in the accents only costs one step, so after the next day the word is back at its old interval.
- **Groups**: put words in one or more groups (e.g. *Week 1*, *Eten en drinken*) when adding, importing or editing; a new name creates the group. Beheer shows the groups per word and can filter on them.
- **Bulk import** (*Beheer → Importeren*): paste one word per line as `hongaars;nederlands;context;voorbeeldzin;woordsoort;vormen` (everything after the Dutch word optional; *vormen* lists exceptions the grammar rules can't make themselves, e.g. `t=házat, k=házak`, spaces around `;` allowed, commas for several Hungarian answers). A preview marks each line as new, already present, an error, or needing a same/different-meaning choice. A line counts as already present when every Hungarian word is already an answer for that Dutch word, whatever the hint (*régi;oud* is the existing *oud (dingen) → régi*); such words still get the import's groups, so you can import a word list just to group existing words; the preview shows the lines that need attention first (the counters are filters), and any line can be skipped (*overslaan*), one by one or all open lines at once; the import is all-or-nothing and new words enter the queue. Every Hungarian synonym gets its own Hungarian → Dutch card (*pad,kanapé;bank* also asks *kanapé → bank*), unless that word already has another meaning: then the preview says so and you decide yourself.
- **A queue for new words**: newly added words are introduced at a daily limit (default 20, adjustable in *Beheer*, 0 = no limit), so adding 100 words at once doesn't create a peak. "Nog 10 nieuwe woorden" lets you learn more on a good day. Reviews and new words never mix: choose *Herhalen* or *Nieuwe woorden* above the card, so a big import never gets in the way of your reviews; a new word joins the reviews once you've answered it the first time.
- **Load balancing**: for intervals with some leeway (7 days and longer) the next review is put on the quietest day within that leeway.
- **Rounds of 10 words**, fully shuffled: short and long intervals mixed, not first all words from yesterday. Once one direction of a word has been asked, the other direction waits until the next day (one would give away the other), and new words arrive one direction per day. Wrong answers are asked again at the end of the round; those re-asks don't change the schedule. Don't know it? Press *Weet ik niet* (or Esc) to see the answer right away; it counts as wrong and shows up separately in the statistics.
- **Helpful feedback**: punctuation at the end and the kind of dash don't matter (*mi* = *mi?*, *tai-csi* = *tai–csi*); one typo in a Dutch answer of 4+ letters counts as correct (*vrinedelijk*), unless what you typed is another word from your list, while Hungarian spelling stays strict; *Toch goed* after a wrong answer counts it as correct anyway (a typo the check missed, or a synonym not in your list); a hint when only the accents are wrong (*kave* → *kávé*), and an explanation when you gave another meaning of the word (*idő* can mean *tijd*, but *weer* was meant here).
- **Hints and example sentences** to tell meanings apart (*idő* ☀️ vs *idő* ⏰).
- **Management tab** (*Beheer*): search (accent-insensitive), browse, filter, add, edit and delete concepts. When the Dutch word already exists, you are asked whether the Hungarian word is an extra accepted answer for an existing meaning (a Hungarian synonym, like *bank (meubel)* → *pad / kanapé*) or a new meaning that needs its own distinguishing hint (*bank (geld)*). When a Hungarian word already exists, you are asked whether the new word is the same meaning (an extra accepted answer, only for true synonyms like *szia* → *hoi / hallo*) or a different one (a separate concept with its own hint, like *idő* → *weer* / *tijd*, or *ő* → *hij* / *zij*, so that each option is learned on its own).
- **Checked before class** (*Controleren*): someone else can check new words before they reach the lessons: approve, fix in Beheer, or leave a note (keyboard: J/K next/previous, A approve, O note, B fix). A new word only enters *Nieuwe woorden* once approved, unless you choose *Alvast leren* (when importing or adding, or later in Controleren for one word or everything you searched for, e.g. a group): then it is learned already but stays in the review list until someone approves it. Words already in your reviews keep going. Changing a word's content sends it back for checking.
- **Flashcards** (*Kaarten*): study words before they reach your reviews, e.g. a fresh import (the words in the queue), the approved new words that come first in your lessons, a group, or the words you got wrong. Flip the card and say whether you knew it, or type the answer and have it checked like in the quiz (accent hints and other meanings included); cards you didn't know come back a few cards later. Separate from spaced repetition: your answers aren't saved, but you can mark a new word with ★ to give it priority in *Nieuwe woorden* (or, after a session, all new words you knew right away).
- **Grammar rules** (*Regels*): learn to *make* forms instead of memorising them. Each word has a word type (*zelfstandig naamwoord*, *werkwoord*, …; set it in Beheer, as the fifth import column, or filter on *woordsoort onbekend* to fill in the gaps), and the rules work on your own words: case endings (-ban/-ben, -ba/-be, -ból/-ből, -n/-on/-en/-ön, -ra/-re, -ról/-ről, -nál/-nél, -hoz/-hez/-höz, -tól/-től, -nak/-nek, -val/-vel, -vá/-vé, -ig, -ért, -ként), plural -k, object -t, the adjective ending -i (practised with a fixed list of places and time words, since after other nouns -i means something else), and verbs (present tense indefinite and definite, and the infinitive, made from the 3rd person singular you learn). You choose which rules are on; each exercise picks a rule (weaker rules more often) and a word you already know, and after your answer shows step by step how the form is made (vowel harmony, -a → -á, *busz* → *busszal*, *olvas* → *olvasol*). Irregular words come from an exceptions table (*ház* → *házat*, *ló* → *lovat*, *megy* → *mész*) or from the exceptions stored with the word itself (sixth import column, or *Afwijkende vormen* in Beheer, e.g. `t=szekeret, k=szekerek`; `?` = don't practise that form, `bep=?` = verb without object), compounds follow their last part (*kórház* like *ház*); where the rules aren't sure (a short noun that isn't in the lists yet), the word is simply not asked. A score per rule is kept in `regels.jsonl`; this practice doesn't touch your review schedule.
- **Statistics tab** (*Statistiek*): streak, today's progress and first-attempt score, activity over the last 30 days, progress over time (how your practised cards moved from short to long intervals, as numbers or as a share, reconstructed from the answer log), a 14-day forecast of upcoming reviews, the hardest words (click through to edit them), your score per direction (recognising Hungarian vs. producing it, with first-time answers kept apart), types of mistakes, whether re-asking within a round actually helps the next day, and whether a word you forgot after knowing it for a week comes back the next day and then sticks at its lower interval (to see whether two steps back is the right amount).
- **Answer log**: every answer is appended to `antwoorden.jsonl`; the statistics are computed from it.

## Running locally

Requires Go 1.25 or newer.

```bash
go run ./cmd/api
```

Then open <http://127.0.0.1:8080>. By default the app only listens on this machine and reads its data from the current directory.

```bash
go test ./...   # run all tests
```

## Configuration

All configuration is done with environment variables:

| Variable | Default | Meaning |
|---|---|---|
| `ADDR` | `127.0.0.1:8080` | Address to listen on. Use `:8080` to listen on all interfaces. |
| `DATA_DIR` | `.` | Directory containing the data files (see below). |
| `AUTH_USER`, `AUTH_PASSWORD` | – | Enables HTTP basic auth when both are set. `/healthz` stays open. |
| `TZ` | system | Time zone for the 04:00 day boundary, e.g. `Europe/Amsterdam`. |

## Data

All data lives in `DATA_DIR` as plain files:

| File | Contents |
|---|---|
| `woorden.json` | Words and concepts (a concept is one direction, e.g. *koffie* → *kávé*, with hint and example). |
| `voortgang.json` | Progress per concept: current step and next review. |
| `antwoorden.jsonl` | Append-only log of every answer (one JSON object per line). |
| `regels.jsonl` | Append-only log of the answers in *Regels* (the score per grammar rule). |
| `instellingen.json` | Settings changed in the interface, such as new words per day and the grammar rules you practise. |

Files are written atomically (temporary file + rename), and the app refuses to start on a corrupt file instead of overwriting it.

## Production

The same binary runs in production; things to take care of:

- **Authentication**: there are no user accounts yet. When the app is reachable from a network, set `AUTH_USER` and `AUTH_PASSWORD` (the app logs a warning otherwise), and serve it over **HTTPS** behind a reverse proxy (Caddy, nginx, Traefik) – basic auth sends the password with every request.
- **Time zone**: servers often run in UTC; set `TZ` to the learner's time zone so reviews start at 04:00 local time.
- **Data directory**: point `DATA_DIR` at persistent storage outside the code checkout, and back it up (it is a handful of small files).
- **Single instance**: the app keeps its state in memory and in local files, so run exactly one instance.
- **Health check**: `GET /healthz` returns `200 ok`.
- **Shutdown**: on `SIGTERM` / `SIGINT` the server finishes running requests (and their writes) before exiting.

Built-in protections: cross-site request protection (Go's `CrossOriginProtection`), JSON-only write requests with a size limit, a strict Content-Security-Policy and other security headers, and server timeouts.

Example systemd unit:

```ini
[Service]
ExecStart=/usr/local/bin/magyar
Environment=ADDR=127.0.0.1:8080 DATA_DIR=/var/lib/magyar TZ=Europe/Amsterdam
Environment=AUTH_USER=maarten AUTH_PASSWORD=change-me
Restart=on-failure
```

Build the binary with `go build -o magyar ./cmd/api`.

## Project layout

```
cmd/api/            HTTP server, configuration, security middleware, embedded web frontend (web/)
internal/quiz/      a practice session: rounds, re-asks, feedback, answer log
internal/engine/    scheduling (intervals, day boundary, round selection) and answer checking
internal/catalog/   adding, editing and deleting concepts, with duplicate and meaning checks
internal/stats/     statistics computed from the answer log and current progress
internal/grammar/   grammar rules (vowel harmony, case endings, conjugation) with exceptions, and the rule practice
internal/storage/   JSON files, progress and the answer log
```

The frontend in `cmd/api/web/` uses plain ES modules, no build step: `main.js` switches between the tabs, each tab has its own module (`quiz.js` Oefenen, `cards.js`, `grammar.js` Regels, `stats.js`, `manage.js` with `import.js` and `groups.js`, `review.js`), and `api.js` / `util.js` hold what they share. Every module imports what it uses, so nothing depends on global names.

## Generating import files

`.claude/skills/hongaarse-woordenlijst-csv/SKILL.md` is a Claude skill that turns a word list (PDF or notes) into an import file: base forms only, verbs as infinitive + 3rd person singular, the word type, and the exceptions to the grammar rules in the sixth column. It describes the rules in `internal/grammar`, so it is updated together with them; the same text is used as the skill on claude.ai.

## Credits

Built by Maarten de Reus together with [Claude](https://claude.com/claude-code) (Anthropic).
