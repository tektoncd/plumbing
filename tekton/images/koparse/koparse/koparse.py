#!/usr/bin/env python3

"""
koparse.py parses release.yaml files from `ko`

The `ko` tool (https://github.com/google/go-containerregistry/tree/master/cmd/ko)
builds images and embeds the full names of the built images in the resulting
yaml files.

This script does two things:

* Parses those image names out of the release.yaml, including their digests, and
  outputs those to stdout
* Verifies the list of built images against an expected list, to be sure that all
  expected images were built (and no extra images were built)
"""

import argparse
import hashlib
import re
import string
import sys
from typing import List


DIGEST_MARKER = "@sha256"


class ImagesMismatchError(Exception):
    def __init__(self, missing: List[str], extra: List[str]):
        self.missing = missing
        self.extra = extra

    def __str__(self):
        errs = []
        if self.missing:
            errs.append("Images %s were expected but missing." % self.missing)
        if self.extra:
            errs.append("Images %s were present but not expected." %
                        self.extra)
        return " ".join(errs)


class BadActualImageFormatError(Exception):
    def __init__(self, image: str):
        self.image = image

    def __str__(self):
        return "Format of image %s was unexpected, did not contain %s" % (self.image, DIGEST_MARKER)


def parse_release(base: str, path: str) -> List[str]:
    """Extracts built images from the release.yaml at path

    Args:
        base: The built images will be expected to start with this string,
            other images will be ignored
        path: The path to the file (release.yaml) that will contain the built images
    Returns:
        list of the images parsed from the file
    """
    images = []
    pattern = re.compile(base + r"[0-9a-z\-/\.]+(?::[0-9a-zA-Z\-\._]+)?" + DIGEST_MARKER + r":[0-9a-f]+")
    with open(path) as f:
        for line in f:
            found = re.findall(pattern, line)
            for image in found:
                images.append(image)
    return images


def md5(input: str) -> str:
    return hashlib.md5(input.encode()).hexdigest()


def convert_image_path(image: str) -> str:
    # Separate the image name from path in <container-reg>/<path>/<image-name>:<tag>
    parts = image.split(":")
    image_without_tag = parts[0]
    tag = ""

    if len(parts) == 2:
        tag = f":{parts[1]}"
    
    image_name = image_without_tag.split("/")[-1]

    return f"{image_name}-{md5(image_without_tag)}{tag}"

def compare_expected_images(expected: List[str], actual: List[str], container_registry: str, preserve_path: bool) -> None:
    """Ensures that the list of actual images includes only the expected images

    Args:
        expected: A list of all of the names of images that are expected to have
            been built, including the path to the image without the digest
        actual: A list of the names of the built images, including the path to the
            image and the digest
    """
    for image in actual:
        if DIGEST_MARKER not in image:
            raise BadActualImageFormatError(image)

    if not preserve_path:
        expected = [convert_image_path(image) for image in expected]

    expected = ["/".join([container_registry, image]) for image in expected]
    actual_no_digest = [image.split(DIGEST_MARKER)[0] for image in actual]

    missing = set(expected) - set(actual_no_digest)
    extra = set(actual_no_digest) - set(expected)

    if missing or extra:
        raise ImagesMismatchError(list(missing), list(extra))


def backwards_compatible_params(container_registry: str, base: str, images: List[str]) -> tuple[str, str, List[str]]:
    # For backwards compatibility, if container registry is not provided, we assume it's
    # the first section of the base, which is the legacy behavior. We then need to strip
    # container registry out of the expected images and base
    if not container_registry:
        container_registry = "/".join(base.split("/")[:2])
        base = "/".join(base.split("/")[2:])
        images = ["/".join(image.split("/")[2:]) for image in images]
    return container_registry, base, images


if __name__ == "__main__":
    arg_parser = argparse.ArgumentParser(
        description="Parse expected built images from a release.yaml created by `ko`")
    arg_parser.add_argument("--path", type=str, required=True,
                            help="Path to the release.yaml")
    arg_parser.add_argument("--container-registry", dest="container_registry", type=str, required=False,
                            help="Container registry URI and path e.g. gcr.io/tekton-releases")
    arg_parser.add_argument("--base", type=str, required=True,
                            help="String prefix which is used to find images within the release.yaml")
    arg_parser.add_argument("--images", type=str, required=True, nargs="+",
                            help="List of all images expected to be built, without digests")
    arg_parser.add_argument("--preserve-path", dest="preserve_path", type=bool, required=False, default=True,
                            help="Whether ko is configured to preserve the images base path")
    args = arg_parser.parse_args()

    try:
        container_registry, expected_images, base = backwards_compatible_params(
            args.container_registry, args.images, args.base)

        # For backwards compatibility, if container registry is not provided, we assume it's
        # the first section of the base, which is the legacy behavior. We then need to strip
        # container registry out of the expected images and base
        if not args.container_registry:
            container_registry = "/".join(args.base.split("/")[:1])
            base = "/".join(args.base.split("/")[2:])
            expected_images = ["/".join(image.split("/")[2:]) for image in args.images]

        if args.preserve_path:
            search_path = "/".join([container_registry, args.base])
        else:
            search_path = container_registry

        images = parse_release(search_path, args.path)
        compare_expected_images(images, images, container_registry, args.preserve_path)
    except (IOError, BadActualImageFormatError) as e:
        sys.stderr.write("Error determining built images: %s\n" % e)
        sys.exit(1)
    except (ImagesMismatchError) as e:
        sys.stderr.write("Expected images did not match: %s\n" % e)
        with open(args.path) as f:
            sys.stderr.write(f.read())
        sys.exit(1)

    print("\n".join(images))
