from piper.voice import PiperVoice
import wave
import subprocess
import sys
import os
from pathlib import Path
from env_utils import load_env

load_env()

voice = PiperVoice.load(
    os.getenv("MORTIS_PIPER_VOICE_PATH", str(Path.home() / "piper-voices" / "en_US-lessac-medium.onnx"))
)

AUDIO_PATH = os.getenv("MORTIS_ANSWER_AUDIO_PATH", str(Path.home() / "mortisAnswer.wav"))

def speak(text):
    if not text:
        return

    with wave.open(AUDIO_PATH, "wb") as wav_file:
        wav_file.setnchannels(1)
        wav_file.setsampwidth(2)
        wav_file.setframerate(voice.config.sample_rate)

        for chunk in voice.synthesize(text):
            wav_file.writeframes(chunk.audio_int16_bytes)

    subprocess.run(["paplay", AUDIO_PATH])

def main():
    print("READY", flush=True)  

    for line in sys.stdin:
        text = line.strip()
        if not text:
            continue
        if text == "__EXIT__":
            break

        try:
            speak(text)
            print("DONE", flush=True)
        except Exception as e:
            print(f"ERROR: {e}".replace("\n", " "), flush=True) 

    if os.path.exists(AUDIO_PATH):
        os.remove(AUDIO_PATH)  

if __name__ == "__main__":
    main()