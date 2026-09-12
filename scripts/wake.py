import os
import sys
from openwakeword.model import Model
import sounddevice as sd

os.environ["OMP_NUM_THREADS"] = "1"

MODEL_PATH = os.getenv("MORTIS_WAKEWORD_MODEL_PATH", "/home/gabz/Downloads/hey_mortis.onnx "
                       )
WAKEWORD_NAME = "hey_mortis"  

def wait_for_wake_word(model):
    samplerate = 16000
    blocksize = 4000
    detected = False

    def callback(indata, frames, time, status):
        nonlocal detected
        audio = indata[:, 0]
        prediction = model.predict(audio)
        for wakeword, score in prediction.items():
            if score > 0.5 and wakeword == WAKEWORD_NAME:
                detected = True
                raise sd.CallbackStop

    try:
        with sd.InputStream(samplerate=samplerate, channels=1, dtype="int16",
                             blocksize=blocksize, callback=callback):
            while not detected:
                sd.sleep(100)
    except sd.CallbackStop:
        pass
    sd.stop()

def main():
    model = Model(wakeword_model_paths=["/home/gabz/Downloads/hey_mortis.onnx"])
    print("READY", flush=True)

    for line in sys.stdin:
        cmd = line.strip()
        if cmd == "__EXIT__":
            break
        if cmd == "listen":
            model.reset()  
            wait_for_wake_word(model)
            print("detected", flush=True)

if __name__ == "__main__":
    main()