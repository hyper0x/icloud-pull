# icloud-pull

> A command-line tool to download evicted (dataless) iCloud files on macOS — without Finder, without brctl, without any GUI.

## Why?

This tool was born from a real problem: **AI agents cannot access evicted iCloud files.**

### The Problem

macOS has a storage optimization feature called **iCloud eviction**. When local disk space is tight, the system automatically offloads file *contents* to iCloud, keeping only metadata (filename, size, modification time) locally. The files still appear to exist — `ls` shows them, `stat` reports their original size — but reading them returns **0 bytes**.

This is called being **dataless** in APFS terms. The file's BSD flags include `UF_DATALESS` (0x40000000), and the actual data lives only in iCloud.

For a human user, this is invisible — you double-click the file in Finder, macOS transparently fetches it from iCloud, and it opens. But for **automated workflows** — backup scripts, AI agents, CI pipelines, server-side processing — evicted files are a silent landmine:

- `cp` copies a **0-byte shell** instead of the real content
- `ditto` reports "No such file or directory" (it can't read the data fork)
- `rsync` triggers a background download mid-transfer, causing hash mismatches
- `tar`/`zip` archives empty files without warning
- AI agents read file contents and get nothing, with no error to signal why

### How We Discovered This

While building an automated backup system for [Obsidian](https://obsidian.md) vaults stored in iCloud Drive, we hit a wall: backup scripts would silently produce **corrupt archives** — files that existed in `ls` but contained zero bytes. The root cause was iCloud eviction. Three out of 389 files in one vault had been evicted; `stat` reported their sizes (4793, 4507, 4516 bytes), but reading them returned nothing.

The diagnosis took hours. The fix took longer. And the realization was worse: **there was no good command-line tool to download evicted files.**

### The Tool Gap

| Approach | Problem |
|----------|---------|
| `brctl download <path>` | Apple **removed** this subcommand in recent macOS versions. It no longer works. |
| `fileproviderctl materialize <path>` | Undocumented, unstable, and not available on all macOS versions. |
| Finder (double-click) | Requires GUI interaction. Impossible for scripts, agents, or headless servers. |
| `open <file>` | Opens the file in its default app (e.g., Obsidian, Preview), which is disruptive and unreliable. |
| `cat <file> > /dev/null` | Triggers a download as a side effect, but is slow, loads the entire file into memory, and is a hack. |

The only existing open-source tool we found — [`icanhasjonas/icloud-tools`](https://github.com/icanhasjonas/icloud-tools) — is written in Swift (3 stars, requires Xcode toolchain) and uses Foundation APIs that are heavier than necessary.

### The Insight

The key discovery: **reading a single byte from an evicted file triggers APFS transparent fetch.** 

```bash
head -c1 <evicted-file> > /dev/null
```

This one-liner tells macOS "I need this file's data," and iCloud fetches it in the background. The `dataless` flag disappears within seconds. No Finder, no Foundation framework, no GUI.

This is the core mechanism `icloud-pull` is built on — simple, fast, and dependency-free.

## What It Does

```bash
# Scan a directory for evicted files
icloud-pull status ~/Library/Mobile\ Documents/iCloud~md~obsidian/Documents/MyVault

# Download all evicted files in a directory (recursively)
icloud-pull download ~/Library/Mobile\ Documents/iCloud~md~obsidian/Documents/MyVault

# Download with concurrency control
icloud-pull download --concurrency 10 ~/path/to/vault

# JSON output for scripting / AI agent integration
icloud-pull status --json ~/path/to/vault
```

### Features

- **Detect evicted files** — scans for `UF_DATALESS` BSD flags via Go's native `syscall.Stat_t`, no external commands needed
- **Batch download** — triggers APFS transparent fetch via single-byte reads, with configurable concurrency (goroutines)
- **Progress feedback** — real-time progress bar showing files downloaded / remaining / failed
- **JSON output** — machine-readable format for AI agents, scripts, and CI pipelines
- **Single binary** — no dependencies, no Xcode, no Homebrew formula required (but we'll provide one)
- **macOS only** — this is an APFS/iCloud feature; the tool will clearly error on other platforms

## Installation

```bash
# Homebrew (coming soon)
brew install hyper0x/tap/icloud-pull

# Build from source
go install github.com/hyper0x/icloud-pull@latest

# Or just download the binary from Releases
```

## How It Works

1. **Scan**: Walks the directory tree, calls `os.Stat()` on each file, reads `Stat_t.Flags` to check for `UF_DATALESS` (0x40000000). This is a native Go syscall on Darwin — no shelling out to `stat` or `brctl`.

2. **Download**: For each evicted file, opens the file and reads 1 byte (`head -c1` equivalent). This triggers APFS to request the file content from iCloud. The download happens asynchronously — the tool polls the `dataless` flag until it clears (or times out).

3. **Verify**: After download, re-checks the BSD flags to confirm `UF_DATALESS` is gone. Reports any files that failed to download.

## Who Is This For?

- **AI agent developers** — your agent reads files from iCloud Drive and needs them to actually contain data
- **Backup script authors** — ensure all files are local before archiving
- **DevOps / sysadmins** — pre-download iCloud files on headless Macs (CI runners, build servers)
- **Power users** — bulk-download iCloud Drive folders without clicking through Finder

## Limitations

- **macOS only**. iCloud eviction is an APFS feature.
- **`.icloud` placeholder files** (fully offloaded files that don't even have local metadata) require a different approach — they need to be resolved through the File Provider API. This tool focuses on *evicted* files that still have local metadata.
- **Download speed** depends on iCloud's backend, not this tool. We can trigger the fetch, but iCloud controls the actual transfer.
- **Pin / Keep Downloaded** (preventing future eviction) may require Foundation API integration, which is a future feature.

## License

MIT

## Contributing

Contributions welcome. This tool solves a real pain point — if you've hit iCloud eviction in your workflows, you know how frustrating it is. Help make it better.
