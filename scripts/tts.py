import os
import shutil
import subprocess
import sys
import tempfile
import wave
from pathlib import Path

from piper.voice import PiperVoice


PROJECT_ROOT = Path(__file__).resolve().parent.parent
VOICE_PATH = Path(os.environ.get(
    "MORTIS_PIPER_VOICE_PATH",
    str(PROJECT_ROOT / "piper-voices/en_US-lessac-medium.onnx"),
))

if not VOICE_PATH.is_file():
    raise FileNotFoundError(f"Piper voice model not found: {VOICE_PATH}")

CONFIG_PATH = Path(f"{VOICE_PATH}.json")
if not CONFIG_PATH.is_file():
    raise FileNotFoundError(f"Piper voice config not found: {CONFIG_PATH}")

voice = PiperVoice.load(str(VOICE_PATH))


def play_audio(audio_path: Path) -> None:
    if sys.platform == "darwin":
        subprocess.run(
            ["/usr/bin/afplay", "-v", "1.0", str(audio_path)],
            check=True,
        )
        return

    if sys.platform == "win32":
        import winsound

        winsound.PlaySound(
            str(audio_path),
            winsound.SND_FILENAME | winsound.SND_SYNC,
        )
        return

    if sys.platform.startswith("linux"):
        players = (
            ("paplay", [str(audio_path)]),
            ("aplay", ["-q", str(audio_path)]),
            ("ffplay", ["-nodisp", "-autoexit", "-loglevel", "quiet", str(audio_path)]),
        )

        for executable, arguments in players:
            if shutil.which(executable):
                subprocess.run([executable, *arguments], check=True)
                return

        raise RuntimeError(
            "No Linux audio player found. Install pulseaudio-utils, alsa-utils, "
            "or ffmpeg."
        )

    raise RuntimeError(f"Unsupported operating system: {sys.platform}")


def speak(text: str) -> None:
    text = text.strip()
    if not text:
        return

    with tempfile.NamedTemporaryFile(
        prefix="mortis-",
        suffix=".wav",
        delete=False,
    ) as temporary_file:
        audio_path = Path(temporary_file.name)

    try:
        with wave.open(str(audio_path), "wb") as wav_file:
            wav_file.setnchannels(1)
            wav_file.setsampwidth(2)
            wav_file.setframerate(voice.config.sample_rate)

            for chunk in voice.synthesize(text):
                wav_file.writeframes(chunk.audio_int16_bytes)

        play_audio(audio_path)
    finally:
        audio_path.unlink(missing_ok=True)


def main() -> None:
    print("READY", flush=True)

    try:
        for line in sys.stdin:
            text = line.strip()

            if not text:
                continue
            if text == "__EXIT__":
                break

            try:
                speak(text)
                print("DONE", flush=True)
            except Exception as error:
                print(f"ERROR: {error}", flush=True)
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()