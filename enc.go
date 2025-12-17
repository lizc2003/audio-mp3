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

// EncoderConfig specifies MP3 encoding parameters.
type EncoderConfig struct {
	// SampleRate sets input sample rate in Hz.
	// Default is 44100.
	SampleRate int

	// NumChannels sets number of channels in input stream.
	// Default is 2 (stereo).
	NumChannels int

	// Bitrate in kbps for CBR encoding.
	// Supported values: 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320
	// Default is 128.
	Bitrate int

	// Quality is the encoding quality level (0-9).
	// 0 = best quality (very slow)
	// 2 = near-best quality, not too slow (recommended)
	// 5 = good quality, fast
	// 7 = ok quality, really fast
	// 9 = worst quality
	// Default is 2.
	Quality int

	// VbrMode sets the VBR (Variable Bit Rate) mode.
	// Default is VbrModeOff (CBR).
	VbrMode VBRMode

	// MpegMode sets the output audio mode.
	// Default: LAME picks based on compression ratio and input channels.
	MpegMode MpegMode
}

// Encoder is an MP3 encoder instance wrapping the LAME library.
// It encodes PCM audio data to MP3 format.
// Note: Encoder is NOT safe for concurrent use.
type Encoder struct {
	handle      *C.lame_global_flags
	remainData  []byte // Buffer for incomplete sample frames
	NumChannels int
	FrameLength int
}

// NewEncoder creates a new MP3 encoder with the given configuration.
// If config is nil or has zero values, defaults will be used.
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

	return enc, nil
}

func (enc *Encoder) Close() {
	if enc.handle != nil {
		C.lame_close(enc.handle)
		enc.handle = nil
	}
}

// Encode encodes PCM audio data to MP3 format.
// in: input PCM buffer (16-bit signed samples)
// out: output buffer for MP3 data (should be at least EstimateOutBufBytes(len(in)))
// Returns: number of MP3 bytes written to out buffer
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

// Flush flushes the internal encoder buffer to get remaining MP3 data.
// Should be called after all input data has been encoded.
// out: output buffer for remaining MP3 data
// Returns: number of MP3 bytes written to out buffer
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
		// MpegMode constants are offset by +1 to avoid conflict with C enum values
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
	enc.FrameLength = int(frameSize)
	enc.NumChannels = c.NumChannels

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
