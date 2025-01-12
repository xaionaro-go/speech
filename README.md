# `speech`

This is a library for Speech-To-Text operations in Go

Currently, we provide API for using Whisper directly and/or for using whisper via HTTP API.

An example how to use Whisper directly is provided in [`./cmd/stt`](./cmd/stt/main.go).

# Quick start

```sh
WHISPER_MODEL=large-v3 ENABLE_CUDA=true make example-stt
```