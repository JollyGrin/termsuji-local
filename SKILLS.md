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
