from http.server import BaseHTTPRequestHandler, HTTPServer
import glob
import json
import os


def read_file(path):
    try:
        with open(path, "r", encoding="utf-8") as f:
            return f.read().strip()
    except FileNotFoundError:
        return None


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.respond({"ok": True})
            return

        self.respond(
            {
                "message": "hello from ecs managed instances",
                "gpu": {
                    "visibleDevicesEnv": os.getenv("NVIDIA_VISIBLE_DEVICES"),
                    "deviceFiles": sorted(glob.glob("/dev/nvidia*")),
                    "procGpus": sorted(glob.glob("/proc/driver/nvidia/gpus/*")),
                    "driverVersion": read_file("/proc/driver/nvidia/version"),
                    "cudaVersion": os.getenv("CUDA_VERSION"),
                },
            }
        )

    def log_message(self, format, *args):
        return

    def respond(self, payload):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8000"))
    server = HTTPServer(("0.0.0.0", port), Handler)
    server.serve_forever()
