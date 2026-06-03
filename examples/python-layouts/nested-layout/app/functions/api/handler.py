import json
from pathlib import Path

from shared.utils import json_response


CONFIG_PATH = Path(__file__).parents[2] / "data" / "config.json"


def main(event, context):
    config = json.loads(CONFIG_PATH.read_text())
    return json_response(
        {
            "layout": "nested",
            "app": config["app_name"],
            "feature_flag": config["feature_flag"],
        }
    )
