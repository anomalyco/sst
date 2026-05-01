import json
import os
import importlib
import pytest

import sst


class TestSSTResourcesJSON:
    def test_loads_resources_from_json(self, monkeypatch):
        monkeypatch.setenv(
            "SST_RESOURCES_JSON",
            json.dumps({"MyBucket": {"name": "my-bucket"}, "App": {"name": "app", "stage": "dev"}}),
        )
        importlib.reload(sst)

        assert sst.Resource.MyBucket.name == "my-bucket"

    def test_merges_with_individual_vars(self, monkeypatch):
        monkeypatch.setenv("SST_RESOURCE_MyTable", json.dumps({"name": "my-table"}))
        monkeypatch.setenv(
            "SST_RESOURCES_JSON",
            json.dumps({"MyBucket": {"name": "my-bucket"}, "App": {"name": "app", "stage": "dev"}}),
        )
        importlib.reload(sst)

        assert sst.Resource.MyTable.name == "my-table"
        assert sst.Resource.MyBucket.name == "my-bucket"

    def test_json_overrides_individual_vars(self, monkeypatch):
        monkeypatch.setenv("SST_RESOURCE_MyBucket", json.dumps({"name": "from-env-var"}))
        monkeypatch.setenv(
            "SST_RESOURCES_JSON",
            json.dumps({"MyBucket": {"name": "from-json"}, "App": {"name": "app", "stage": "dev"}}),
        )
        importlib.reload(sst)

        assert sst.Resource.MyBucket.name == "from-json"

    def test_invalid_json_is_ignored(self, monkeypatch):
        monkeypatch.setenv("SST_RESOURCES_JSON", "not-json")
        importlib.reload(sst)

        with pytest.raises(Exception, match='"MyBucket" is not linked'):
            _ = sst.Resource.MyBucket

    def test_links_not_active_without_app_or_json(self, monkeypatch):
        monkeypatch.delenv("SST_RESOURCE_App", raising=False)
        monkeypatch.delenv("SST_RESOURCES_JSON", raising=False)
        importlib.reload(sst)

        with pytest.raises(Exception, match="It does not look like SST links are active"):
            _ = sst.Resource.MyBucket

    def test_no_links_active_error_with_json(self, monkeypatch):
        monkeypatch.delenv("SST_RESOURCE_App", raising=False)
        monkeypatch.setenv(
            "SST_RESOURCES_JSON",
            json.dumps({"MyBucket": {"name": "my-bucket"}, "App": {"name": "app", "stage": "dev"}}),
        )
        importlib.reload(sst)

        # Should not raise "links not active"
        assert sst.Resource.MyBucket.name == "my-bucket"
