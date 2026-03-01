# hugo-buffer

Posts the latest entry from a Hugo blog's RSS feed to your social channels via
the [Buffer](https://buffer.com) API. Run it from the root of any Hugo blog
after a production build.

## What it does

1. Discovers the RSS feed in `public/` (`index.xml`, `feed.xml`, or `rss.xml`)
2. Takes the first (latest) item
3. Guards against dev builds by rejecting feeds that link to `localhost`
4. Renders configurable post text per channel using a Go template
5. Submits to each configured Buffer channel for immediate publication

## Building

Requires Go 1.22+.

```sh
git clone https://github.com/aaronbieber/hugo-buffer
cd hugo-buffer
```

With [just](https://github.com/casey/just):

```sh
just build    # build ./hugo-buffer
just install  # build and move to ~/.local/bin
```

Or directly with Go:

```sh
go build -o hugo-buffer .
mv hugo-buffer ~/.local/bin/
```

## Configuration

### Global config

Create `~/.config/hugo-buffer/config.yaml` (start from the example):

```sh
cp config.yaml.example ~/.config/hugo-buffer/config.yaml
```

```yaml
buffer:
  token: "abc_XYZ"          # Buffer API token; or set BUFFER_TOKEN env var

channels:
  - id: "channel_abc123"
    name: "Bluesky"
    char_limit: 300
  - id: "channel_def456"
    name: "Mastodon"
    char_limit: 500

post:
  char_limit: 500           # fallback if a channel omits char_limit
  template: |
    {{ .Description }}

    {{ .Link }}
```

**Template variables:** `.Title` `.Description` `.Link` `.PubDate`

The description is HTML-stripped, entity-decoded, and truncated to fit within
the channel's character limit while always preserving the link.

**Finding your channel IDs** — they aren't shown in the Buffer UI, so use:

```sh
hugo-buffer --list-channels
```

This queries the API and prints a table of channel names, services, and IDs.

**Config file location** is resolved in this order:

1. `$HUGO_BUFFER_CONFIG` (explicit path)
2. `$XDG_CONFIG_HOME/hugo-buffer/config.yaml`
3. `~/.config/hugo-buffer/config.yaml`

The `BUFFER_TOKEN` environment variable overrides `buffer.token` if set.

### Per-blog template override

Place a `hugo-buffer.yaml` in the blog's root directory to override the post
template for that blog:

```yaml
post:
  template: |
    {{ .Title }}

    {{ .Link }}
```

Only `post.template` is read from this file; all other settings come from the
global config.

## Running

Build your Hugo site with your production `baseURL`, then run `hugo-buffer`
from the blog root:

```sh
cd ~/src/my-blog
hugo
hugo-buffer
```

### Flags

| Flag | Description |
|---|---|
| `--dry-run` | Render and print post text for each channel without calling the API |
| `--list-channels` | Query the Buffer API and print available channel IDs |

### Dry run

Preview exactly what will be posted before sending:

```sh
hugo-buffer --dry-run
```

```
─── Bluesky (187 chars) ───
My post description, trimmed to fit…

https://example.com/posts/my-post

─── Mastodon (312 chars) ───
My post description with more room to breathe on Mastodon.

https://example.com/posts/my-post
```

## License

```
DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
Version 2, December 2004

Copyright (C) 2004 Sam Hocevar <sam@hocevar.net>

Everyone is permitted to copy and distribute verbatim or modified
copies of this license document, and changing it is allowed as long
as the name is changed.

DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
TERMS AND CONDITIONS FOR COPYING, DISTRIBUTION AND MODIFICATION

0. You just DO WHAT THE FUCK YOU WANT TO.
```
