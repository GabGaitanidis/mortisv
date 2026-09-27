# Mortis

Mortis is a voice-controlled assistant. It listens for a wake word, captures speech, processes commands, and responds using text-to-speech.

## Architecture

```text
┌──────────────────────┐
│   Wake-word engine   │
│    scripts/wake.py   │
└──────────┬───────────┘
           │ wake word detected
           ▼
┌──────────────────────┐
│   Exchange session   │
│       Go runtime     │
└──────┬───────────────┘
       │
       ├── starts ──► scripts/stt.py
       │              Speech-to-text
       │
       ├── processes ──► Commands and modules
       │
       └── starts ──► scripts/tts.py
                      Piper text-to-speech
                              │
                              ▼
                        OS play .wav file
```

## Main components

### Application

This Go application coordinates the assistant lifecycle and manages exchange sessions.

Responsibilities:

- Start and stop the Python subprocesses from a registry hub
- Coordinate wake-word, STT, and TTS states
- Receive transcripts
- Route commands to application modules
- Handle shutdown and errors

### Wake-word detection

File: `scripts/wake.py`

The wake-word process continuously listens from microphone ready to capture the selected (currently hey_jarvis) wake word


### Speech-to-text

File: `scripts/stt.py`

This STT process captures microphone input and converts speech into text.


### Text-to-speech

File: `scripts/tts.py`

The TTS process uses Piper to synthesize speech into a temporary WAV file.

On macOS, audio is played with based on the OS: 

| Operating system | Audio command |
|---|---|
| macOS | `/usr/bin/afplay` |
| Linux | `paplay`, `aplay`, or `ffplay` |
| Windows | Built-in `winsound` (needs more config) |

The Piper model is configured with:

```dotenv
MORTIS_PIPER_VOICE_PATH=/Users/gabrizio/Desktop/projects/mortisv/piper-voices/en_US-lessac-medium.onnx
```

The model and configuration files MUST both exist:

```text
piper-voices/
├── en_US-lessac-medium.onnx
└── en_US-lessac-medium.onnx.json
```

The TTS subprocess communicates through standard input and output (known as IPC (inter process communication)):
This means that in the main go runtime, in its state where the wake word has been detected
all 3 subprocesses are running in the background waiting to receive in their stdin 
the selected string that indicates that they must run some code (captures input for stt selected word listen and output with voice with selected word empty string )
```text
READY
DONE
ERROR: <message>
```

## Browser module

File: `internal/modules/browser/handler.go`

The browser module can (ONLY HTTP/HTTPS):

- Open URLs using the operating system
- Read the active browser tab on macOS
- Fetch HTTP and HTTPS pages
- Strip HTML tags
- Normalize page text
- Limit fetched content to prevent excessive input size



On macOS, active URLs are read using AppleScript from Google Chrome or Safari.

## Session lifecycle

```text
Wake word detected
        │
        ▼
Start STT and TTS subprocesses
        │
        ▼
Capture user speech
        │
        ▼
Transcribe speech
        │
        ▼
Interpret and execute command
        │
        ▼
Generate response
        │
        ▼
Synthesize and play response
        │
        ▼
Return to wake-word listening
```

## Configuration

Example `.env`:

```dotenv
PYTHON_BIN=/Users/gabrizio/Desktop/projects/mortisv/venv/bin/python3
PROFILE_FILE=files/profile.json
RAPPORT_FILE=files/rapport.json
MORTIS_PIPER_VOICE_PATH=/Users/gabrizio/Desktop/projects/mortisv/piper-voices/en_US-lessac-medium.onnx

GOOGLE_CALENDAR_ACCESS_TOKEN=
GOOGLE_CALENDAR_ID=primary

API_KEY=
```

Do not commit `.env` or expose API keys.

## Running Mortis

Create and activate the virtual environment:

```bash
python3 -m venv venv
source venv/bin/activate
```

Install Python dependencies:

```bash
pip install --upgrade pip
pip install -r requirements.txt
```

Start the application executing from the root go run .


## FEATURES
If you want mortis to handle your calendar you need to generate an api key and add it as
```dotenv
GOOGLE_CALENDAR_ACCESS_TOKEN=
GOOGLE_CALENDAR_ID=primary
```


## Piper voice model

The Piper voice model is not committed to the repository. Download both files:

```bash
mkdir -p piper-voices

curl -L -o piper-voices/en_US-lessac-medium.onnx \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx

curl -L -o piper-voices/en_US-lessac-medium.onnx.json \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json
```

Expected files:

```text
piper-voices/
├── en_US-lessac-medium.onnx
└── en_US-lessac-medium.onnx.json
```