# Magyar – Dutch ↔ Hungarian vocabulary trainer

A small spaced-repetition trainer for learning Hungarian from Dutch (and back), with a web interface in Dutch. It is a single Go binary with no external dependencies; the frontend is embedded.

## Features

- **Spaced repetition** with fixed steps of 1, 3, 7, 16, 40 and 90 days (with a little random spread on the longer intervals). A new review day starts at 04:00, so a word practised in the evening comes back the next morning.
- **Bulk import** (*Beheer → Importeren*): paste one word per line as `hongaars;nederlands;context;voorbeeldzin` (context and example optional, spaces around `;` allowed, commas for several Hungarian answers). A preview marks each line as new, already present, an error, or needing a same/different-meaning choice; the import is all-or-nothing and new words enter the queue.
- **A queue for new words**: newly added words are introduced at a daily limit (default 20, adjustable in *Beheer*, 0 = no limit), so adding 100 words at once doesn't create a peak. "Nog 10 nieuwe woorden" lets you learn more on a good day.
- **Load balancing**: for intervals with some leeway (7 days and longer) the next review is put on the quietest day within that leeway.
- **Rounds of 10 words**, shuffled, never containing both directions of the same word (one would give away the other). Wrong answers are asked again at the end of the round; those re-asks don't change the schedule.
- **Helpful feedback**: a hint when only the accents are wrong (*kave* → *kávé*), and an explanation when you gave another meaning of the word (*idő* can mean *tijd*, but *weer* was meant here).
- **Hints and example sentences** to tell meanings apart (*idő* ☀️ vs *idő* ⏰).
- **Management tab** (*Beheer*): search (accent-insensitive), browse, filter, add, edit and delete concepts. When a Hungarian word already exists, you are asked whether the new word is the same meaning (an extra accepted answer, like *ő* → *hij / zij*) or a different one (a separate concept with its own hint).
- **Statistics tab** (*Statistiek*): streak, today's progress and first-attempt score, activity over the last 30 days, a 14-day forecast of upcoming reviews, the hardest words (click through to edit them), types of mistakes, and whether re-asking within a round actually helps the next day.
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
| `instellingen.json` | Settings changed in the interface, such as new words per day. |

Files are written atomically (temporary file + rename), and the app refuses to start on a corrupt file instead of overwriting it.

## Production

The same binary runs in production; things to take care of:

- **Authentication**: there are no user accounts yet. When the app is reachable from a network, set `AUTH_USER` and `AUTH_PASSWORD` (the app logs a warning otherwise), and serve it over **HTTPS** behind a reverse proxy (Caddy, nginx, Traefik) – basic auth sends the password with every request.
- **Time zone**: servers often run in UTC; set `TZ` to the learner's time zone so reviews start at 04:00 local time.
- **Data directory**: point `DATA_DIR` at persistent storage outside the code checkout, and back it up (it is four small files).
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
internal/storage/   JSON files, progress and the answer log
```

## Credits

Built by Maarten de Reus together with [Claude](https://claude.com/claude-code) (Anthropic).
