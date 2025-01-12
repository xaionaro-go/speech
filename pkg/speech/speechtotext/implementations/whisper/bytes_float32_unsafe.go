package whisper

import (
	"unsafe"
)

func convertBytesToFloat32Slice(b []byte) []float32 {
	ptr := unsafe.SliceData(b)
	return unsafe.Slice((*float32)(unsafe.Pointer(ptr)), len(b)/4)
}
