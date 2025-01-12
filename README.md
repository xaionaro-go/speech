# `speech`

This is a library for Speech-To-Text operations in Go

Currently, we provide API for using Whisper directly and/or for using whisper via HTTP API.

An example how to use Whisper directly is provided in [`./cmd/stt`](./cmd/stt/main.go).

# Quick start

If you use Linux:
```sh
WHISPER_MODEL=medium ENABLE_CUDA=true make example-stt
```
(keep in mind: the larger model is the more time it takes to warm up)

It likely will fail to build, because you don't have CUDA libraries install. You need to install them. But if it will run; it will start listening the microphone, and you can start speaking. It should print the translation of your speech to English.

For example in my case:
```
WHISPER_MODEL=medium ENABLE_CUDA=true make example-stt
[...a lot of log...]
   23.3s -    26.3s:  Hello.
     28s -    30.6s:  This is just a demonstration that the thing works properly.
   30.7s -    37.9s:  And somehow it does work properly, which is weird.
```
