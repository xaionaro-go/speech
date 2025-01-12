package pkgconfig

import "C"

//go:generate go run github.com/mutablelogic/go-whisper/sys/pkg-config --version "0.0.0" --prefix "../../../../../thirdparty/whisper.cpp/" --cflags "-I$DOLLAR{prefix}/include -I$DOLLAR{prefix}/ggml/include" libwhisper.pc
//go:generate go run github.com/mutablelogic/go-whisper/sys/pkg-config --version "0.0.0" --prefix "../../../../../thirdparty/whisper.cpp/" --cflags "-fopenmp" --libs "-L$DOLLAR{prefix}/build/ggml/src -L$DOLLAR{prefix}/build/src -lwhisper -lggml -lggml-base -lggml-cpu -lgomp -lm -lstdc++" libwhisper-linux.pc
//go:generate go run github.com/mutablelogic/go-whisper/sys/pkg-config --version "0.0.0" --prefix "../../../../../thirdparty/whisper.cpp/" --libs "-L$DOLLAR{prefix}/build/ggml/src -L$DOLLAR{prefix}/build/ggml/src/ggml-blas -L$DOLLAR{prefix}/build/ggml/src/ggml-metal -L$DOLLAR{prefix}/build/src -lwhisper -lggml -lggml-base -lggml-cpu -lggml-blas -lggml-metal -lm -lstdc++ -framework Accelerate -framework Metal -framework Foundation -framework CoreGraphics" libwhisper-darwin.pc
//go:generate go run github.com/mutablelogic/go-whisper/sys/pkg-config --version "0.0.0" --prefix "../../../../../thirdparty/whisper.cpp/" --libs "-L$DOLLAR{prefix}/build/ggml/src/ggml-cuda -lggml-cuda" libwhisper-cuda.pc
