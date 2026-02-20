import json
import os
import time
from pathlib import Path


def generate_text(prompt: str):
    text = f"百炼回复: {prompt}"
    for ch in text:
        yield ch
        time.sleep(0.01)


def main():
    data_dir = Path(os.getenv("DATA_DIR", "./data"))
    req_dir = data_dir / "requests"
    done_dir = data_dir / "processed"
    stream_dir = data_dir / "streams"
    req_dir.mkdir(parents=True, exist_ok=True)
    done_dir.mkdir(parents=True, exist_ok=True)
    stream_dir.mkdir(parents=True, exist_ok=True)

    while True:
        for req_file in req_dir.glob("*.json"):
            mark = done_dir / req_file.name
            if mark.exists():
                continue
            payload = json.loads(req_file.read_text(encoding="utf-8"))
            req_id = req_file.stem
            out = stream_dir / f"{req_id}.stream"
            with out.open("w", encoding="utf-8") as f:
                for tok in generate_text(payload.get("prompt", "")):
                    f.write(tok + "\n")
                    f.flush()
                f.write("[DONE]\n")
                f.flush()
            mark.write_text("ok", encoding="utf-8")
        time.sleep(0.1)


if __name__ == "__main__":
    main()
