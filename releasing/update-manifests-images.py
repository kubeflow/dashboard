#!/usr/bin/env python3

import logging
import os
import sys
import ruamel.yaml

log = logging.getLogger(__name__)


class YAMLEmitterNoVersionDirective(ruamel.yaml.emitter.Emitter):
    def write_version_directive(self, version_text):
        pass

    def expect_document_start(self, first=False):
        if not isinstance(self.event, ruamel.yaml.events.DocumentStartEvent):
            return super().expect_document_start(first=first)
        version = self.event.version
        self.event.version = None
        ret = super().expect_document_start(first=first)
        self.event.version = version
        return ret


class YAML(ruamel.yaml.YAML):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.version = (1, 1)
        self.Emitter = YAMLEmitterNoVersionDirective
        self.preserve_quotes = True


yaml = YAML()

components = [
    {
        "name": "Access Management",
        "kustomization": "components/profile-controller/manifests/kustomize/components/kfam/kustomization.yaml",
        "images": [
            {
                "name": "access-management",
                "newName": "ghcr.io/kubeflow/dashboard/access-management",
            },
        ],
    },
    {
        "name": "Central Dashboard",
        "kustomization": "components/centraldashboard/manifests/kustomize/base/kustomization.yaml",
        "images": [
            {
                "name": "dashboard",
                "newName": "ghcr.io/kubeflow/dashboard/dashboard",
            },
        ],
    },
    {
        "name": "Central Dashboard Angular",
        "kustomization": "components/centraldashboard-angular/manifests/kustomize/base/kustomization.yaml",
        "images": [
            {
                "name": "dashboard-angular",
                "newName": "ghcr.io/kubeflow/dashboard/dashboard-angular",
            },
        ],
    },
    {
        "name": "PodDefaults Webhooks",
        "kustomization": "components/poddefaults-webhooks/manifests/kustomize/base/kustomization.yaml",
        "images": [
            {
                "name": "poddefaults-webhook",
                "newName": "ghcr.io/kubeflow/dashboard/poddefaults-webhook",
            },
        ],
    },
    {
        "name": "Profile Controller",
        "kustomization": "components/profile-controller/manifests/kustomize/base/manager/kustomization.yaml",
        "images": [
            {
                "name": "profile-controller",
                "newName": "ghcr.io/kubeflow/dashboard/profile-controller",
            },
        ],
    },
]


def update_manifests_images(components, tag):
    for component in components:
        log.info("Updating manifests for Dashboard component `%s`", component["name"])
        with open(component["kustomization"], "r") as file:
            kustomize = yaml.load(file)

        images = kustomize.get("images", [])
        for target_image in component["images"]:
            found = False
            for image in images:
                if image["name"] == target_image["name"]:
                    image["newName"] = target_image["newName"]
                    image["newTag"] = tag
                    found = True
                    break
            if not found:
                images.append(
                    {
                        "name": target_image["name"],
                        "newName": target_image["newName"],
                        "newTag": tag,
                    }
                )
        kustomize["images"] = images

        with open(component["kustomization"], "w") as file:
            yaml.dump(kustomize, file)


def main():
    logging.basicConfig(level=logging.INFO)

    # read the tag from the VERSION file
    base_dir = os.path.dirname(os.path.abspath(__file__))
    version_file_path = os.path.join(base_dir, "./version/VERSION")
    with open(version_file_path, "r") as file:
        version = file.read().strip()

    update_manifests_images(components, version)


if __name__ == "__main__":
    sys.exit(main())
