# Skills

## Clean Asciicast

Strip personal info and speed up terminal recordings.

```sh
./scripts/clean-cast.sh <file.cast> --inplace --max-delay 0.5
```

| Flag | Description |
|------|-------------|
| `--inplace` | Overwrite the input file |
| `-o FILE` | Write to a different file |
| `--max-delay N` | Cap delays at N seconds (default: 0.5) |

**What it does:**
- Removes `grins@Deans-MacBook-Pro termsuji-local` from shell prompts, leaving just `% `
- Caps long pauses so the recording plays back without awkward hesitation
- Preserves the asciicast v3 header and all TUI rendering intact

## Cast to MP4

Convert a `.cast` recording to MP4 via `agg` (GIF) + `ffmpeg`.

```sh
./scripts/cast-to-mp4.sh docs/termsuji-demo.cast -o docs/termsuji-demo.mp4
```

| Flag | Description |
|------|-------------|
| `-o FILE` | Output path (default: input with `.mp4` extension) |
| `--font-size N` | agg font size (default: 16) |
| `--cols N` | Override terminal columns for agg |
| `--rows N` | Override terminal rows for agg |

**Requires:** `agg` (`cargo install --git https://github.com/asciinema/agg`) and `ffmpeg` (`brew install ffmpeg`)
