# Website design concepts

- Created: 2026-07-09
- Source: Pi ideation session (`tmp/pi-marketing-session.md`) + follow-up curation in Claude Code
- Status: [DECIDED] 2026-07-09 — winner is the round-3 rebuild of `09-datasheet` ("Transformation, typeset"); it now lives in the repo as `website/index.html`. All prototypes and the review app were deleted; local preview served via `site.localhost`.

## Settled direction (from the Pi session)

- Hero copy family: **"Your local dev servers deserve names."** + "Friendly URLs for unfriendly local ports."
- Examples must carry *intent*, never generic: `checkout-redesign`, `docs-preview`, `webhook-tester`, `client-demo` — not `api`/`demo`/`app`.
- Core insight: **Ports name machines. Route names name work.** The port still exists; humans stop carrying it.
- Style direction: calm, developer-native, small-tool energy, "labels over plumbing". No network-infrastructure/cyber visuals, no SaaS-platform vibe.
- Honest scope framing: not a public tunnel, not a dev-server launcher, not a reverse-proxy framework — the small missing piece between random ports and readable local URLs.
- Tailnet support is a differentiator but secondary: "same local work, reachable from the machines you trust."

## Round 1 verdict (2026-07-09)

User review: nothing above 6/10 overall. Best mixture = **hero idea from #1 (ports→names transformation) + look of #2 (browser-chrome, warm gray chrome on tinted paper, burnt-orange accent)**. Round-1 prototypes deleted; round 2 replaces them with the hybrid plus four fully new concepts.

## Round 2 verdict (2026-07-09)

- `06-hybrid-chrome` — **best so far, keep as-is**. Current front-runner.
- `09-datasheet` — the appliance-drawing viz idea rejected, but the *look* (typeset paper/ink/stamp-red print documentation) liked → rebuilt as **transformation hero in the datasheet look**.
- `07-switchboard`, `08-transit-map`, `10-playground` — rejected, deleted.

## Final round & decision

Final head-to-head was `06-hybrid-chrome` (transformation hero in browser-chrome look) vs the rebuilt `09-datasheet` (**Transformation, typeset**: same transformation hero expressed as a "Fig. 1 — Route assignment ledger" with rubber-stamped names, in the datasheet print language — paper/ink/stamp-red, numbered sections, footnotes, SCOPE box, no appliance drawing).

**Winner: 09 "Transformation, typeset"** → promoted to `website/index.html` (single self-contained file, zero external requests, light/dark via prefers-color-scheme). `06-hybrid-chrome` and the review BWA were deleted with the rest of the exploration artifacts. Not deployed anywhere yet — hosting is still an open item in `tmp/public-launch-path.md`.

## Round 2 rejected concepts

- `07-switchboard` — vintage patch panel identity.
- `08-transit-map` — metro-diagram identity.
- `10-playground` — interactive type-a-name hero.

## Round 1 concepts (superseded)

1. **Ports → names transformation** — hero is the before/after itself: a pile of `localhost:5173 / :3000 / :49152` resolving into named routes; ports dim into gray plumbing, names snap in as crisp labels. Clearest statement of the pitch.
2. **Tab strip** — hero is a browser tab row where every tab reads `localhost:xxxx`, indistinguishable; after, tabs are named. Most viscerally relatable pain.
3. **Real terminal → browser demo** — no metaphor: `local-router register checkout-redesign --port 5173`, then the browser opens the named URL. Product-honest; the asset doubles as the README GIF.
4. **Label-maker identity** — "labels over plumbing" taken literally as the whole visual system: route names as label-tape stickers over faint port/pipe linework. A design identity, not just a hero layout.
5. **"Send a name, not a port"** — copy-led, editorial: the stability/handoff angle. Paste a name into Slack / a script / an agent and it survives port changes. Where tailnet + agent messaging naturally live.

Demoted to mid-page sections (kept, not killed): route-card grid (dashboard section), tailnet dual-name illustration (features row).

## Prototypes

Each concept gets a full standalone one-page site (not just a hero) in `artifacts/site-concepts/<nn>-<slug>/index.html`, each with a deliberately different overall design. Shared content skeleton: hero → live demo/transformation → how it works (register/routes/pin) → dashboard → tailnet → honest comparison (Portless/Caddy/ngrok) → install (`go install forge.dikka.dev/lab/local-router@latest`) → footer (Forgejo source).

## Open questions

- [OPEN] Which concept (or hybrid) wins? Lean from discussion: transformation as hero, tab strip as the beat below.
- [OPEN] Hosting + repo polish items live in `tmp/public-launch-path.md` (license, v0.1.0 tag, GitHub mirror).
