# Mortis

Mortis is a local, voice-first assistant written in Go with Python speech bridges.
It listens for a wake word, transcribes speech, asks an LLM to plan actions as JSON commands, executes those commands through modules, and speaks the results.

## Features

- Wake-word activation (`hey_jarvis`) using OpenWakeWord
- Speech-to-text with Groq Whisper (`whisper-large-v3`)
- Text-to-speech with Piper voice models
- LLM command planning via Groq chat completions
- Built-in action modules:
  - `file` (create/read/write/delete files)
  - `browser` (open URLs/search, explain active page)
  - `calendar` (Google Calendar list/add/update/delete)
  - `question` (general Q&A)
  - `system` (OS info, run commands, open apps)
  - `unknown` fallback
- Lightweight memory:
  - user profile facts (`PROFILE_FILE`)
  - rapport notes (`RAPPORT_FILE`)

## Architecture (high level)

1. `scripts/wake.py` detects wake word.
2. `scripts/stt.py` records audio and transcribes speech.
3. `internal/ai` builds prompts and asks Groq for a JSON command plan.
4. `internal/core/CommandRouter` routes commands to modules.
5. `scripts/tts.py` speaks each module result.
6. Post-exchange classification updates user profile and rapport stores.

## Requirements

- Go 1.22+
- Python 3.10+
- Audio input/output devices configured on host machine
- A Piper voice model file and its `.json` config
- Groq API key

## Setup

### 1) Clone and enter project

```bash
git clone https://github.com/GabGaitanidis/mortisv.git
cd mortisv
```

### 2) Create Python environment and install dependencies

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### 3) Configure environment

```bash
cp .env.example .env
```

Edit `.env`:

| Variable | Required | Description |
| --- | --- | --- |
| `PYTHON_BIN` | Recommended | Python executable used by Go speech bridges (example: `venv/bin/python`) |
| `API_KEY` | Yes | Groq API key used for chat completions and transcription |
| `PROFILE_FILE` | Yes | Path to JSON user profile store |
| `RAPPORT_FILE` | Yes | Path to rapport log file |
| `MORTIS_PIPER_VOICE_PATH` | Yes | Path to Piper `.onnx` voice model |
| `GOOGLE_CALENDAR_ACCESS_TOKEN` | Optional | OAuth bearer token for Calendar module |
| `GOOGLE_CALENDAR_ID` | Optional | Calendar ID (`primary` by default) |

Create storage directories/files if missing:

```bash
mkdir -p files
printf '{"fields":{}}' > files/profile.json
touch files/rapport.json
```

### 4) Add Piper voice assets

Place your Piper voice model and config where `MORTIS_PIPER_VOICE_PATH` points (for example `piper-voices/en_US-lessac-medium.onnx` and `piper-voices/en_US-lessac-medium.onnx.json`).

## Run

```bash
go run .
```

You should see logs indicating Mortis is ready and listening for the wake word.

## Notes and limitations

- `browser current` / `browser explain` active-tab detection is implemented for macOS; Linux currently requires additional browser integration.
- `calendar` actions require a valid Google Calendar OAuth access token.
- File module paths are rooted under your `~/Desktop`.

## Project structure

```text
.
├── main.go
├── internal/
│   ├── ai/            # prompting, Groq client, post-exchange updates
│   ├── core/          # command model + router
│   ├── modules/       # executable modules (browser/file/calendar/system/...)
│   ├── session/       # STT/TTS warm session lifecycle
│   ├── speech/        # Python subprocess bridges
│   ├── userprofile/   # profile model + persistence
│   └── rapport/       # rapport note persistence
├── scripts/
│   ├── wake.py
│   ├── stt.py
│   └── tts.py
└── .env.example
```

## Development

Run tests:

```bash
go test ./...
```

