//go:build cgo && darwin && arm64

package mp3

// #cgo LDFLAGS: -L${SRCDIR}/deps/darwin_arm64
// #cgo LDFLAGS: -lmpg123
import "C"
