# Images

This directory contains sources and Dockerfiles used to build images used for
Tekton's CI system.

Images can be built with the Makefile at the top-level of this repository:

```shell
make images
```

The images are pushed to the repository: `gcr.io/tekton-releases/tests`.
They can be pushed with:

```shell
make push
```
