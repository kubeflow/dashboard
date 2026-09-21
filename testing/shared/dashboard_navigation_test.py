#!/usr/bin/env python3
"""Validate navigation in the rendered classic and Angular Dashboard profiles."""

import json
import subprocess
import unittest
from pathlib import Path

import yaml

REPOSITORY_ROOT = Path(__file__).resolve().parents[2]
DASHBOARDS = {
    "centraldashboard": "dashboard-config",
    "centraldashboard-angular": "dashboard-angular-config",
}
PROFILES = ("base", "overlays/istio", "overlays/kserve")
MLFLOW_LINK = {
    "icon": "assessment",
    "link": "/mlflow",
    "text": "MLflow Experiments",
    "type": "item",
}
EXISTING_LINKS = {
    "/jupyter/",
    "/tensorboards/",
    "/volumes/",
    "/katib/",
    "/pipeline/#/pipelines",
    "/pipeline/#/experiments",
    "/pipeline/#/runs",
    "/pipeline/#/recurringruns",
    "/pipeline/#/artifacts",
    "/pipeline/#/executions",
}


def menu_items(items):
    for item in items:
        yield item
        yield from menu_items(item.get("items", []))


class DashboardNavigationTest(unittest.TestCase):
    def test_navigation_in_every_profile(self):
        for dashboard, configuration_name in DASHBOARDS.items():
            for profile in PROFILES:
                with self.subTest(dashboard=dashboard, profile=profile):
                    directory = (
                        REPOSITORY_ROOT
                        / "components"
                        / dashboard
                        / "manifests/kustomize"
                        / profile
                    )
                    rendered = subprocess.run(
                        ["kustomize", "build", str(directory)],
                        check=True,
                        capture_output=True,
                        text=True,
                        timeout=60,
                    ).stdout
                    configurations = [
                        resource
                        for resource in yaml.safe_load_all(rendered)
                        if resource
                        and resource.get("kind") == "ConfigMap"
                        and resource["metadata"]["name"] == configuration_name
                    ]
                    self.assertEqual(1, len(configurations))
                    navigation = json.loads(configurations[0]["data"]["links"])
                    items = list(menu_items(navigation["menuLinks"]))
                    mlflow_entries = [
                        item
                        for item in items
                        if item.get("text") == MLFLOW_LINK["text"]
                        or item.get("link", "").rstrip("/") == "/mlflow"
                    ]
                    self.assertEqual([MLFLOW_LINK], mlflow_entries)
                    links = {item["link"] for item in items if "link" in item}
                    self.assertTrue(EXISTING_LINKS.issubset(links))
                    self.assertEqual(
                        profile == "overlays/kserve", "/kserve-endpoints/" in links
                    )


if __name__ == "__main__":
    unittest.main()
