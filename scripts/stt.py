import sys
import os
import numpy as np
import sounddevice as sd
import soundfile as sf
import webrtcvad
from groq import Groq
import signal

def handle_exit(sig, frame):
    sd.stop()
    sys.exit(0)

signal.signal(signal.SIGINT, handle_exit)
signal.signal(signal.SIGTERM, handle_exit)
    
from env_utils import load_env
load_env()

SAMPLE_RATE = 16000
FRAME_DURATION_MS = 30
FRAME_SIZE = int(SAMPLE_RATE * FRAME_DURATION_MS / 1000)
SILENCE_TIMEOUT_MS = 1200
SILENCE_FRAMES = SILENCE_TIMEOUT_MS // FRAME_DURATION_MS
MAX_RECORD_SECONDS = 15
VAD_AGGRESSIVENESS = 0

vad = webrtcvad.Vad(VAD_AGGRESSIVENESS)
client = Groq(api_key=os.getenv("API_KEY", ""))

def record_until_silence():
    frames = []
    silence_count = 0
    triggered = False
    max_frames = int(MAX_RECORD_SECONDS * 1000 / FRAME_DURATION_MS)

    with sd.RawInputStream(samplerate=SAMPLE_RATE, blocksize=FRAME_SIZE,
                            dtype="int16", channels=1) as stream:
        for _ in range(max_frames):
            frame, _ = stream.read(FRAME_SIZE)
            frame_bytes = bytes(frame)
            frames.append(frame_bytes)

            is_speech = vad.is_speech(frame_bytes, SAMPLE_RATE)

            if is_speech:
                triggered = True
                silence_count = 0
            elif triggered:
                silence_count += 1

            if triggered and silence_count >= SILENCE_FRAMES:
                break

    return b"".join(frames), triggered

HALLUCINATION_PHRASES = {
    "thank you", "thanks for watching", "thank you for watching",
    "you", "bye", "subscribe", "subtitles by",
}

def process_once():
    raw_audio, got_speech = record_until_silence()
    audio = np.frombuffer(raw_audio, dtype=np.int16)

    mean_amp = np.abs(audio).mean()
    print(f"DEBUG mean amplitude: {mean_amp}", file=sys.stderr)

    if not got_speech or mean_amp < 1200:
        return "no input"

    buf_path = "/tmp/mortis_stt_tmp.wav"
    sf.write(buf_path, audio, SAMPLE_RATE)

    with open(buf_path, "rb") as f:
        transcription = client.audio.transcriptions.create(
            model="whisper-large-v3",
            file=f,
            language="en"
        )

    text = transcription.text.strip()

    if not text or len(text.split()) < 2:
        return ""

    normalized = text.lower().strip(".").strip()
    if normalized in HALLUCINATION_PHRASES:
        print(f"DEBUG filtered hallucination: {text!r}", file=sys.stderr)
        return "no input"

    return text

def main():
    print("READY", flush=True)
    for line in sys.stdin:
        cmd = line.strip()
        if not cmd:
            continue
        if cmd == "__EXIT__":
            break
        if cmd == "listen":
            try:
                result = process_once()
            except Exception as e:
                result = f"ERROR:{e}"
            print(result.replace("\n", " "), flush=True)

if __name__ == "__main__":
    main()