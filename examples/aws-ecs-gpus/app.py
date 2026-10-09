from pathlib import Path
import glob
import os

from fastapi import FastAPI


app = FastAPI()


def read_file(path: str):
    try:
        return Path(path).read_text(encoding="utf-8").strip()
    except FileNotFoundError:
        return None


@app.get("/health")
def health():
    return {"ok": True}


@app.get("/")
def index():
    return {
        "message": "hello from ecs managed instances",
        "gpu": {
            "visibleDevicesEnv": os.getenv("NVIDIA_VISIBLE_DEVICES"),
            "deviceFiles": sorted(glob.glob("/dev/nvidia*")),
            "procGpus": sorted(glob.glob("/proc/driver/nvidia/gpus/*")),
            "driverVersion": read_file("/proc/driver/nvidia/version"),
            "cudaVersion": os.getenv("CUDA_VERSION"),
        },
    }
