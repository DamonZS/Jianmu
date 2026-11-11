# Integrating your app or agent with jianmu

For developers of an AI app, agent or client who want it to work well with
jianmu. There are three levels, from no work on jianmu's side to a row of
your own on the Agents page. The web version is
<https://usejianmu.ai/docs/integrate>.

| You want                                              | Route                          | Change to jianmu |
| ----------------------------------------------------- | ------------------------------ | ---------------- |
| Your app's requests served by jianmu, counted under its own name in Usage | [Talk to the gateway](#1-talk-to-the-gateway) | none |
| A row on the Agents page: detected, its model picked there, put back on disconnect | [An agent adapter](#2-a-row-on-the-agents-page) | a pull request |
| A new model source (a subscription, a vendor), or rewriting requests | [A plugin](#3-plugins) | none |

## 1. Talk to the gateway

jianmu runs a local LLM gateway. Every model the user has in jianmu
(subscriptions, API keys, local models, routing groups) is served there
under one id each, so your app needs one provider entry, not one per
vendor.

**Find it.** The gateway listens on `127.0.0.1:3425`, or on the port in
`port` of jianmu's `settings.json` (`~/.config/jianmu/settings.json`, the
same path under the user's folder on Windows; `$XDG_CONFIG_HOME/jianmu`
when that is set). `GET /api/hello` answers
`{"name":"jianmu","version":"…"}` when it is jianmu. Read that file, never
write it. jianmu must be running (the app or `jianmu serve`). An app that
finds nothing there should offer its own providers as usual.

**Endpoints.** Requests go straight through when the vendor speaks your
API, and are translated otherwise, streaming, tool calls, images and
reasoning included:

| Path | API |
| ---- | --- |
| `/v1/chat/completions` | OpenAI Chat Completions |
| `/v1/responses` | OpenAI Responses |
| `/v1/messages`, `/v1/messages/count_tokens` | Anthropic Messages |
| `/v1beta/models/{model}:generateContent` (and `:streamGenerateContent`) | Google Gemini |
| `/v1/models` | the catalog |

So the base URL is `http://127.0.0.1:<port>/v1` for an OpenAI client and
`http://127.0.0.1:<port>` for an Anthropic or Gemini one.

**The key names your app.** On this machine the gateway takes any key.
Send `jianmu-<your-app-id>` (`Authorization: Bearer jianmu-acme`, or
`x-api-key`) and every call is counted as `acme` in Usage, the Requests
log and the routing rules that match on agents. Without it, jianmu counts
the call by the first word of your `User-Agent` (`Acme/1.4 (darwin)` is
`Acme`); an SDK's own User-Agent (`OpenAI/JS …`, `ai-sdk/…`) would count it
as that SDK. From another computer (Settings → Network and sharing → Share on local network) the
key must be one of the user's gateway keys, and the User-Agent names your
app.

**Models.** `GET /v1/models` lists what the user has turned on: `id`
(`provider/model`, or a routing group's name, which you send as it is),
`display_name`, `owned_by` (the provider), `reasoning` and
`supported_reasoning_levels` (`[{"effort":"low"},…]`, and the same as
plain names in `reasoning_levels`), `context_window`
and `max_output_tokens` when known, `modalities` when jianmu knows whether
it takes images. List them from there rather than shipping a list: the
user adds and removes providers in jianmu. Reasoning is asked in your
API's own field (`reasoning_effort`, `reasoning.effort`, Anthropic's
`thinking` or `output_config.effort`). `?format=text` gives the ids one a
line. The list holds chat models only. Ask for image and video models
with `?kind=image,video` (or `?kind=chat,image,video`, or `?kind=all`).
Each of them is marked `"kind": "image"` or `"kind": "video"` and goes to
`/v1/images/generations` or `/v1/videos`, never to a chat route.

**A vendor's own fields.** `/v1/images/generations` and `/v1/videos`
take OpenAI's fields and pass the others on to a vendor that takes them:
Volcengine Ark's `content` (reference video and audio, `draft_task`),
`ratio`, `resolution`, `duration: -1`, `tools`, `watermark`,
`sequential_image_generation` and the rest. A video's answer carries
Ark's own fields too (`content.last_frame_url`, `usage`, `resolution`).
For the rest of a vendor's API use
`/v1/jianmu/native/<provider>/<path>`: GET, POST, PUT, PATCH or DELETE,
with `<path>` joined to the provider's host (not its base URL), sent and
answered in the vendor's own shape on the user's key for that provider,
a stream as it comes. So Ark's task list is `GET
/v1/jianmu/native/ark/api/v3/contents/generations/tasks`. It serves a
provider added with an API key, not a subscription or sign-in. Ark's
short paths (`images/generations`, `contents/generations/tasks[/<id>]`)
still work under the key's base. It takes the same gateway key as every
route, and a key held to models, accounts or a budget is refused it.

**Optional.** Send `X-Jianmu-Session: <id>` with a conversation's requests
and `GET /v1/jianmu/route?session=<id>` says which model a routing group
picked for the turn and the fallbacks it tried, before the first token
(long-poll with `after=<seq>&wait=<s>`). `GET /v1/jianmu/quotas` lists the
user's subscription allowances, and `GET /v1/jianmu/usage` and
`/v1/jianmu/usage/requests` the Usage page's numbers and its requests
(with `session`, and `subagent`/`parent_agent` for a Claude Code
subagent's call). They answer only this machine, or another with a gateway
key; a key held to a budget, models or accounts sees its own calls and
routes alone, and is refused the quotas unless its owner lets it see
those of what it may use (`jianmu gateway-key quotas <id> on`, #2216;
its own limit, and what holds it, is `GET /v1/jianmu/limit`).
Calls your app makes to its vendor itself can be reported with `POST
/v1/jianmu/usage/requests` (one record or up to 5000 a post, in
`usage.jsonl`'s shape, each with a `rid`; past times are kept, so a
history can be backfilled). They show on the Usage page as reported, and
never count against a key's limit. [Connecting anything else](reference.md#connecting-anything-else)
has the details.

**Keep the user's providers.** Add jianmu as one more provider (named
Jianmu) beside the ones the user set up in your app, rather than replacing
them, so turning jianmu off is picking another provider.

## 2. A row on the Agents page

The agents jianmu lists (Claude Code, Codex, OpenCode, dsh, Alma, Cindy…)
are adapters in [`internal/agent`](../internal/agent), one file each,
listed in [`agents.go`](../internal/agent/agents.go). A row gets your
app's name and icon, shows when your app is installed, lets the user pick
its model (and reasoning) from jianmu's catalog, follows the catalog as
providers change, and on disconnect puts back what your app had before.
There is no runtime registration: a row is added by a pull request to
this repository, and ships in the next release.

An adapter is an [`Agent`](../internal/agent/agent.go) value. What your
app offers decides its shape:

- **jianmu edits your config file** (most adapters; [`fx.go`](../internal/agent/fx.go),
  [`empryo.go`](../internal/agent/empryo.go)). `Dir`, `Path` and `Bin` find
  the app; each `Field` (`model`, `effort`, …) has `Get`, `Set` and
  `Options`. Connecting adds a `jianmu` provider at the gateway, keyed
  `jianmu-<id>`, and sets your model to one of jianmu's; `Set("")` (the
  agent's default) or `Unwire` takes jianmu's entries out and restores
  what was there. Only the keys jianmu sets are touched; the user's other
  settings, providers and comments stay. `Notice` says when your app must
  be restarted to see a change; `Sync` rewrites a model list your app keeps
  in a file rather than asking `/v1/models`.
- **Your app takes a provider only through its own import link, which the
  user confirms in your app** ([`cindy.go`](../internal/agent/cindy.go)).
  `Import` returns the link (`yourapp://provider/import?…` carrying the
  gateway's URL, the key `jianmu-<id>` and the endpoints), the row's button
  opens it, and the user says yes once, in your app. `Added` reads,
  read-only, whether your app has jianmu already. Leave `Added` nil if your
  app has no public status reader (as [Lody](../internal/agent/lody.go) does);
  the row then asks the user to confirm in your app without claiming an
  import status. jianmu writes nothing of yours. This suits an app that keeps
  its providers in a database or encrypted.
- **Your app has a local API while it runs** (Alma, at `localhost:23001`;
  Shelley, on its Unix socket): jianmu adds itself as a provider and sets the
  default model through it.

A pull request needs: the adapter and its registration, `UA` (what your
User-Agent begins with, lower-case) so Usage knows your calls, an icon in
`internal/gui/assets/icons`, tests that run under a temporary `HOME`
(connect, switch, disconnect restoring the original file), and a row in
the [Agents table](reference.md#agents). What helps us most is a stable,
documented config format or import link, and where your app keeps it on
each system.

## 3. Plugins

A plugin is an npm package jianmu runs; the user installs it from
Settings → Plugins or with `jianmu plugin add <package>`. There are two
kinds ([plugin guide](https://usejianmu.ai/docs/plugins)):

- **Provider plugins**, OpenCode's `auth` hook or a pi package: a new
  source of models, such as a subscription jianmu doesn't sign in to
  itself. They sign in, list models and make requests.
- **Gateway middleware**: rewrites requests and replies as they pass
  (`onRequest`, `onAccount`, `onProvider`, `onEvent`, `onResponse`).

A plugin can't add a row to the Agents page or write another app's
config, so it is not how an app registers itself.

## What isn't there

- No API or link lets another app register itself as an agent, or install
  a plugin, without the user: plugins are installed in jianmu by the user,
  and agent rows come from adapters in jianmu's source.
- `jianmu://import` links add a *provider* (a source of models, with its
  key), not an agent or an app; see [Import links](reference.md#import-links).

Questions, or an adapter you'd like to discuss first: open an issue at
<https://github.com/DamonZS/Jianmu/issues> or ask on
[Discord](https://discord.gg/vGSnD3ZKQF).
