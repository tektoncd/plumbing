#!/usr/bin/env python3.6

import os
import unittest

import koparse


IMAGE_PATH = "github.com/tektoncd/pipeline/cmd/"
CONTAINER_REGISTRY = "gcr.io/tekton-releases"
IMAGE_BASE = CONTAINER_REGISTRY + "/" + IMAGE_PATH
PATH_TO_TEST_RELEASE_YAML = os.path.join(os.path.dirname(
    os.path.abspath(__file__)), "test_release.yaml")
PATH_TO_TEST_RELEASE_YAML_NO_PATH = os.path.join(os.path.dirname(
    os.path.abspath(__file__)), "test_release_no_preserve_path.yaml")
PATH_TO_WRONG_FILE = os.path.join(os.path.dirname(
    os.path.abspath(__file__)), "koparse.py")
BUILT_IMAGES = [
    "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/kubeconfigwriter:v20201022-ceeec6463e.1_1A@sha256:68453f5bb4b76c0eab98964754114d4f79d3a50413872520d8919a6786ea2b35",
    "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:7d5520efa2d55e1346c424797988c541327ee52ef810a840b5c6f278a9de934a",
    "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller@sha256:bdc6f22a44944c829983c30213091b60f490b41f89577e8492f6a2936be0df41",
    "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/webhook@sha256:cca7069a11aaf0d9d214306d456bc40b2e33e5839429bf07c123ad964d495d8a",
]
BUILT_IMAGES_NO_PATH = [
    "gcr.io/tekton-releases/kubeconfigwriter-3d37fea0b053ea82d66b7c0bae03dcb0:v20201022-ceeec6463e.1_1A@sha256:68453f5bb4b76c0eab98964754114d4f79d3a50413872520d8919a6786ea2b35",
    "gcr.io/tekton-releases/git-init-4874978a9786b6625dd8b6ef2a21aa70@sha256:7d5520efa2d55e1346c424797988c541327ee52ef810a840b5c6f278a9de934a",
    "gcr.io/tekton-releases/controller-10a3e32792f33651396d02b6855a6e36@sha256:bdc6f22a44944c829983c30213091b60f490b41f89577e8492f6a2936be0df41",
    "gcr.io/tekton-releases/webhook-d4749e605405422fd87700164e31b2d1@sha256:cca7069a11aaf0d9d214306d456bc40b2e33e5839429bf07c123ad964d495d8a",
]
EXPECTED_IMAGES = [
    "github.com/tektoncd/pipeline/cmd/kubeconfigwriter:v20201022-ceeec6463e.1_1A",
    "github.com/tektoncd/pipeline/cmd/git-init",
    "github.com/tektoncd/pipeline/cmd/controller",
    "github.com/tektoncd/pipeline/cmd/webhook",
]


class TestKoparse(unittest.TestCase):

    def test_parse_release(self):
        images = koparse.parse_release(IMAGE_BASE, PATH_TO_TEST_RELEASE_YAML)
        self.assertListEqual(images, BUILT_IMAGES)

    def test_parse_release_no_path(self):
        images = koparse.parse_release(CONTAINER_REGISTRY, PATH_TO_TEST_RELEASE_YAML_NO_PATH)
        self.assertListEqual(images, BUILT_IMAGES_NO_PATH)

    def test_parse_release_no_file(self):
        with self.assertRaises(IOError):
            koparse.parse_release(IMAGE_BASE, "whoops")

    def test_parse_release_wrong_contents(self):
        images = koparse.parse_release(IMAGE_BASE, PATH_TO_WRONG_FILE)
        self.assertEqual(images, [])

    def test_compare_expected_images(self):
        koparse.compare_expected_images(EXPECTED_IMAGES, BUILT_IMAGES, CONTAINER_REGISTRY, True)

    def test_compare_expected_images_no_path(self):
        koparse.compare_expected_images(EXPECTED_IMAGES, BUILT_IMAGES_NO_PATH, CONTAINER_REGISTRY, False)

    def test_compare_expected_images_bad_format(self):
        with self.assertRaises(koparse.BadActualImageFormatError):
            koparse.compare_expected_images(EXPECTED_IMAGES, EXPECTED_IMAGES, CONTAINER_REGISTRY, True)

    def test_compare_expected_images_missing(self):
        extra_expected = (EXPECTED_IMAGES[:] +
                          ["gcr.io/knative-releases/something-else"])
        with self.assertRaises(koparse.ImagesMismatchError):
            koparse.compare_expected_images(extra_expected, BUILT_IMAGES, CONTAINER_REGISTRY, True)

    def test_compare_expected_images_too_many(self):
        extra_actual = (BUILT_IMAGES[:] +
                        ["gcr.io/knative-releases/something-else@sha256:somedigest"])
        with self.assertRaises(koparse.ImagesMismatchError):
            koparse.compare_expected_images(EXPECTED_IMAGES, extra_actual, CONTAINER_REGISTRY, True)

    def test_md5(self):
        self.assertEqual(koparse.md5("some input test string"), "e426c968ded84c5ea8b2e3b5e3e7a865")

    def test_convert_image_path(self):
        self.assertEqual(koparse.convert_image_path(EXPECTED_IMAGES[1]), "git-init-4874978a9786b6625dd8b6ef2a21aa70")

    def test_convert_image_path_with_tag(self):
        self.assertEqual(koparse.convert_image_path(EXPECTED_IMAGES[0]), "kubeconfigwriter-3d37fea0b053ea82d66b7c0bae03dcb0:v20201022-ceeec6463e.1_1A")

    def test_backwards_compatible_params_no_cr(self):
        self.assertEqual(
            koparse.backwards_compatible_params(
                "", "cr.io/test/a/b/c", ["cr.io/test/a/b/c/image1", "cr.io/test/a/b/c/image2"]),
            ("cr.io/test", "a/b/c", ["a/b/c/image1", "a/b/c/image2"]))

    def test_backwards_compatible_params_no_cr(self):
        params = ("cr.io/test", "a/b/c", ["a/b/c/image1", "a/b/c/image2"])
        self.assertEqual(koparse.backwards_compatible_params(*params), params)


if __name__ == "__main__":
    unittest.main()
