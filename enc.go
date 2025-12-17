package mp3

/*
#include "deps/include/lame.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

const (
	SampleBitDepth = 16
)

type MpegMode int

const (
	MpegStereo      MpegMode = C.STEREO + 1
	MpegJointStereo MpegMode = C.JOINT_STEREO + 1
	MpegDualChannel MpegMode = C.DUAL_CHANNEL + 1 /* LAME doesn't supports this! */
	MpegMono        MpegMode = C.MONO + 1
	MpegNotSet      MpegMode = C.NOT_SET + 1
)

type VBRMode int

const (
	VbrModeOff  VBRMode = C.vbr_off
	VbrModeRh   VBRMode = C.vbr_rh
	VbrModeAbr  VBRMode = C.vbr_abr
	VbrModeMtrh VBRMode = C.vbr_mtrh
)

var (
	ErrorBufferTooSmall         = errors.New("buffer too small")
	ErrorMalloc                 = errors.New("could not allocate malloc")
	ErrorParamsNotInitialized   = errors.New("lame_init_params not called")
	ErrorPsychoAcousticProblems = errors.New("psycho acoustic problems")
	ErrorUnknown                = errors.New("unknown error")
)

type EncoderConfig struct {
	//  sets input sample rate in Hz
	//  default is 44100
	SampleRate int
	//  sets number of channels in input stream
	//  default is 2
	NumChannels int
	//  bitrate in kbps.
	//  32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320
	Bitrate int
	//  quality=0..9.  0=best (very slow).  9=worst.
	//  recommended:  2     near-best quality, not too slow
	//                5     good quality, fast
	//                7     ok quality, really fast
	Quality int
	//  sets VBR mode
	VbrMode VBRMode
	//  sets output audio mode
	//  default: lame picks based on compression ration and input channels
	MpegMode MpegMode
}

type Encoder struct {
	handle      *C.lame_global_flags
	remainData  []byte
	NumChannels int
	FrameSize   int
}

func NewEncoder(c *EncoderConfig) (*Encoder, error) {
	h := C.lame_init()
	if h == nil {
		return nil, errors.New("failed to initialize lame")
	}

	enc := &Encoder{
		handle: h,
	}
	err := enc.initParams(populateEncConfig(c))
	if err != nil {
		C.lame_close(h)
		return nil, err
	}
	enc.NumChannels = c.NumChannels

	return enc, nil
}

func (enc *Encoder) Close() {
	if enc.handle != nil {
		C.lame_close(enc.handle)
		enc.handle = nil
	}
}

func (enc *Encoder) Encode(in, out []byte) (n int, err error) {
	szIn := len(in)
	szOut := len(out)

	if szIn == 0 {
		return 0, errors.New("input buffer is empty")
	}
	if szOut < enc.EstimateOutBufBytes(szIn) {
		return 0, errors.New("output buffer is too small")
	}

	if len(enc.remainData) > 0 {
		in = append(enc.remainData, in...)
		szIn = len(in)
		enc.remainData = nil
	}

	bytesPerSample := enc.NumChannels * SampleBitDepth / 8
	remain := szIn % bytesPerSample
	if remain > 0 {
		szIn -= remain
		enc.remainData = append(enc.remainData, in[szIn:]...)
		in = in[:szIn]
	}

	if szIn == 0 {
		return 0, nil
	}

	inPtr := (*C.short)(unsafe.Pointer(&in[0]))
	outPtr := (*C.uchar)(unsafe.Pointer(&out[0]))
	numSamples := C.int(szIn / bytesPerSample)
	nWr := C.int(0)

	if enc.NumChannels == 2 {
		nWr = C.lame_encode_buffer_interleaved(enc.handle,
			inPtr, numSamples, outPtr, C.int(szOut))
	} else {
		nWr = C.lame_encode_buffer(enc.handle,
			inPtr, nil, numSamples, outPtr, C.int(szOut))
	}
	if nWr < 0 {
		return 0, toError(nWr)
	}

	return int(nWr), nil
}

func (enc *Encoder) Flush(out []byte) (n int, err error) {
	szOut := len(out)
	if szOut < enc.EstimateOutBufBytes(0) {
		return 0, errors.New("output buffer is too small")
	}

	outPtr := (*C.uchar)(unsafe.Pointer(&out[0]))
	bytesOut := C.lame_encode_flush(enc.handle, outPtr, C.int(szOut))
	if bytesOut < 0 {
		return 0, toError(bytesOut)
	}

	return int(bytesOut), nil
}

func (enc *Encoder) GetFrameNum() (int, error) {
	frameNum := C.lame_get_frameNum(enc.handle)
	if frameNum < 0 {
		return 0, toError(frameNum)
	}
	return int(frameNum), nil
}

func (enc *Encoder) EstimateOutBufBytes(inBytes int) int {
	//
	// From lame.h:
	// The required mp3buf_size can be computed from num_samples,
	// samplerate and encoding rate, but here is a worst case estimate:
	//
	// mp3buf_size in bytes = 1.25*num_samples + 7200
	//
	numSamples := inBytes/(enc.NumChannels*SampleBitDepth/8) + 1
	return int(1.25*float64(numSamples)) + 7200
}

func (enc *Encoder) initParams(c *EncoderConfig) error {
	handle := enc.handle
	errNo := C.lame_set_in_samplerate(handle, C.int(c.SampleRate))
	if errNo < 0 {
		return toError(errNo)
	}
	errNo = C.lame_set_num_channels(handle, C.int(c.NumChannels))
	if errNo < 0 {
		return toError(errNo)
	}
	errNo = C.lame_set_brate(handle, C.int(c.Bitrate))
	if errNo < 0 {
		return toError(errNo)
	}
	errNo = C.lame_set_brate(handle, C.int(c.Bitrate))
	if errNo < 0 {
		return toError(errNo)
	}
	if c.VbrMode != VbrModeOff {
		errNo = C.lame_set_VBR(handle, C.vbr_mode(c.VbrMode))
		if errNo < 0 {
			return toError(errNo)
		}
		errNo = C.lame_set_VBR_quality(handle, C.float(c.Quality))
		if errNo < 0 {
			return toError(errNo)
		}
	} else {
		errNo = C.lame_set_VBR(handle, C.vbr_mode(VbrModeOff))
		if errNo < 0 {
			return toError(errNo)
		}
		errNo = C.lame_set_quality(handle, C.int(c.Quality))
		if errNo < 0 {
			return toError(errNo)
		}
	}
	if c.MpegMode > 0 {
		errNo = C.lame_set_mode(handle, C.MPEG_mode(c.MpegMode-1))
		if errNo < 0 {
			return toError(errNo)
		}
	}

	errNo = C.lame_init_params(handle)
	if errNo < 0 {
		return toError(errNo)
	}

	frameSize := C.lame_get_framesize(handle)
	if frameSize < 0 {
		return toError(frameSize)
	}
	enc.FrameSize = int(frameSize)

	return nil
}

func toError(errNo C.int) error {
	switch errNo {
	case -1:
		return ErrorBufferTooSmall
	case -2:
		return ErrorMalloc
	case -3:
		return ErrorParamsNotInitialized
	case -4:
		return ErrorPsychoAcousticProblems
	default:
		return ErrorUnknown
	}
}

func populateEncConfig(c *EncoderConfig) *EncoderConfig {
	if c == nil {
		c = &EncoderConfig{}
	}
	if c.NumChannels == 0 {
		c.NumChannels = 2
	}
	if c.SampleRate == 0 {
		c.SampleRate = 44100
	}
	if c.Bitrate == 0 {
		c.Bitrate = 128
	}
	if c.Quality < 0 || c.Quality > 9 {
		c.Quality = 2
	}

	return c
}
