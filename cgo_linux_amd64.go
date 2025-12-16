//go:build cgo && linux && amd64

package mp3

// #cgo LDFLAGS: -L${SRCDIR}/deps/linux_amd64
// #cgo LDFLAGS: -lmpg123 -lm
import "C"
