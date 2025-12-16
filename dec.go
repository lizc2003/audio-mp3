package mp3

/*
#include "deps/include/mpg123.h"

int mpg123_DecodeWrapped(mpg123_handle *mh,
			unsigned char *pBuffer, int bufferSize, unsigned char *pOut, int outSize, int *bytesDecode) {
	int errNo;
	size_t szDone;
	int done;

	errNo = mpg123_feed(mh, pBuffer, (size_t)bufferSize);
	if(errNo != MPG123_OK) {
		return errNo;
	}

	*bytesDecode = 0;
	for(;;) {
		errNo = mpg123_read(mh, pOut, (size_t)outSize, &szDone);
		done = (int)szDone;
		if(errNo != MPG123_OK) {
			if (errNo == MPG123_NEED_MORE || errNo == MPG123_DONE) {
				*bytesDecode += done;
				break;
			}
			if (errNo == MPG123_NEW_FORMAT) {
				continue;
			}
			return errNo;
		}

		if (done == 0) {
			break;
		}

		*bytesDecode += done;
		outSize -= done;
		pOut += done;
	}
	return MPG123_OK;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// Decoder represents an MP3 decoder instance wrapping mpg123.
// It is NOT safe for concurrent use.
type Decoder struct {
	handle         *C.mpg123_handle
	SampleRate     int
	NumChannels    int
	SampleBitDepth int
}

var mpg123Initialized bool
var mpg123once sync.Once

func initializeMpg123() {
	mpg123once.Do(func() {
		err := C.mpg123_init()
		if err != C.MPG123_OK {
			fmt.Println("failed to initialize mpg123")
			return
		}
		mpg123Initialized = true
	})
}

// NewDecoder creates a new mpg123 decoder instance
func NewDecoder() (*Decoder, error) {
	initializeMpg123()
	if !mpg123Initialized {
		return nil, errors.New("mpg123 not initialized")
	}

	var errNo C.int
	var mh *C.mpg123_handle
	mh = C.mpg123_new(nil, &errNo)
	if mh == nil {
		return nil, fmt.Errorf("error initializing mpg123 decoder: %s", plainStrError(errNo))
	}

	errNo = C.mpg123_open_feed(mh)
	if errNo != C.MPG123_OK {
		C.mpg123_delete(mh)
		return nil, fmt.Errorf("error open feed: %s", plainStrError(errNo))
	}

	// Set QUIET flag to suppress mpg123 printouts
	errNo = C.mpg123_param(mh, C.MPG123_ADD_FLAGS, C.MPG123_QUIET, 0.0)
	if errNo != C.MPG123_OK {
		C.mpg123_delete(mh)
		return nil, fmt.Errorf("error setting quiet flag: %s", plainStrError(errNo))
	}

	return &Decoder{
		handle: mh,
	}, nil
}

func (d *Decoder) Close() {
	if d.handle != nil {
		C.mpg123_delete(d.handle)
		d.handle = nil
	}
}

func (d *Decoder) EstimateOutBufBytes() int {
	// 1 frame: 1152 samples * 8 channels * 4 bytes = 36864 bytes
	return (1152 * 8 * 4) * 5 // 5 frames
}

// Decode
func (d *Decoder) Decode(in, out []byte) (n int, err error) {
	szIn := len(in)
	szOut := len(out)
	if szIn == 0 {
		return 0, errors.New("input buffer is empty")
	}
	if szOut < d.EstimateOutBufBytes() {
		return 0, errors.New("output buffer size is not enough")
	}

	inPtr := (*C.uchar)(unsafe.Pointer(&in[0]))
	inLen := C.int(szIn)
	outPtr := (*C.uchar)(unsafe.Pointer(&out[0]))
	outLen := C.int(szOut)
	bytesDecoded := C.int(0)

	if errNo := C.mpg123_DecodeWrapped(d.handle, inPtr, inLen, outPtr, outLen, &bytesDecoded); errNo != C.MPG123_OK {
		return 0, errors.New(plainStrError(errNo))
	}

	if d.SampleRate == 0 && bytesDecoded > 0 {
		if err = d.getFormat(); err != nil {
			return 0, err
		}
	}

	return int(bytesDecoded), nil
}

func (d *Decoder) getFormat() error {
	var cRate C.long
	var cChans, cEnc C.int
	errNo := C.mpg123_getformat(d.handle, &cRate, &cChans, &cEnc)
	if errNo != C.MPG123_OK {
		return errors.New(plainStrError(errNo))
	}

	d.SampleRate = int(cRate)
	d.NumChannels = int(cChans)

	//if d.SampleRate > 24000 { // MPEG-1 (32, 44.1, 48 kHz)
	//	d.FrameLength = 1152
	//} else { // MPEG-2/2.5 (<=24 kHz)
	//	d.FrameLength = 576
	//}

	switch cEnc {
	case C.MPG123_ENC_UNSIGNED_8:
		d.SampleBitDepth = 8
	case C.MPG123_ENC_SIGNED_16:
		d.SampleBitDepth = 16
	case C.MPG123_ENC_SIGNED_24:
		d.SampleBitDepth = 24
	case C.MPG123_ENC_SIGNED_32:
		d.SampleBitDepth = 32
	default:
		return fmt.Errorf("unsupported encoding: %d", int(cEnc))
	}

	return nil
}

func plainStrError(errNo C.int) string {
	return C.GoString(C.mpg123_plain_strerror(errNo))
}
