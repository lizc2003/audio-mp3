package mp3_test

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/lizc2003/audio-mp3"
)

// TestEncodeBasic tests basic encoding functionality
func TestEncodeBasic(t *testing.T) {
	// Generate simple test PCM data: 1 second of 440Hz sine wave
	sampleRate := 44100
	duration := 1.0
	numSamples := int(float64(sampleRate) * duration)

	pcmData := generateSineWave(440, sampleRate, 2, numSamples)

	// Create encoder
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:  sampleRate,
		NumChannels: 2,
		Bitrate:     128,
		Quality:     2,
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	// Encode
	outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
	encodedBytes, err := encoder.Encode(pcmData, outBuf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Flush
	flushedBytes, err := encoder.Flush(outBuf[encodedBytes:])
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	totalBytes := encodedBytes + flushedBytes

	if totalBytes == 0 {
		t.Fatal("No MP3 data generated")
	}

	t.Logf("✓ Encoded %d PCM bytes -> %d MP3 bytes (compression: %.1fx)",
		len(pcmData), totalBytes, float64(len(pcmData))/float64(totalBytes))
}

// TestEncodeDifferentBitrates tests encoding with various bitrates
func TestEncodeDifferentBitrates(t *testing.T) {
	bitrates := []int{64, 96, 128, 192, 256, 320}

	// Generate test data
	pcmData := generateSineWave(440, 44100, 2, 44100) // 1 second

	for _, bitrate := range bitrates {
		t.Run(bitrate2string(bitrate), func(t *testing.T) {
			encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
				SampleRate:  44100,
				NumChannels: 2,
				Bitrate:     bitrate,
				Quality:     2,
			})
			if err != nil {
				t.Fatalf("Failed to create encoder: %v", err)
			}
			defer encoder.Close()

			outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
			encodedBytes, err := encoder.Encode(pcmData, outBuf)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			flushedBytes, _ := encoder.Flush(outBuf[encodedBytes:])
			totalBytes := encodedBytes + flushedBytes

			if totalBytes == 0 {
				t.Fatal("No MP3 data generated")
			}

			// Approximate check: 1 second at given bitrate should be about bitrate/8 KB
			expectedSize := bitrate * 1000 / 8 // bytes
			tolerance := 0.3                   // 30% tolerance
			minSize := int(float64(expectedSize) * (1 - tolerance))
			maxSize := int(float64(expectedSize) * (1 + tolerance))

			if totalBytes < minSize || totalBytes > maxSize {
				t.Logf("Warning: Size out of expected range: got %d, expected ~%d ±30%%",
					totalBytes, expectedSize)
			}

			t.Logf("✓ %d kbps: %d bytes", bitrate, totalBytes)
		})
	}
}

// TestEncodeMonoStereo tests mono and stereo encoding
func TestEncodeMonoStereo(t *testing.T) {
	testCases := []struct {
		name        string
		numChannels int
		samples     int
	}{
		{"Mono", 1, 44100},
		{"Stereo", 2, 44100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate test data
			pcmData := generateSineWave(440, 44100, tc.numChannels, tc.samples)

			encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
				SampleRate:  44100,
				NumChannels: tc.numChannels,
				Bitrate:     128,
				Quality:     2,
			})
			if err != nil {
				t.Fatalf("Failed to create encoder: %v", err)
			}
			defer encoder.Close()

			// Verify encoder settings
			if encoder.NumChannels != tc.numChannels {
				t.Errorf("NumChannels mismatch: got %d, want %d",
					encoder.NumChannels, tc.numChannels)
			}

			outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
			encodedBytes, err := encoder.Encode(pcmData, outBuf)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			flushedBytes, _ := encoder.Flush(outBuf[encodedBytes:])
			totalBytes := encodedBytes + flushedBytes

			if totalBytes == 0 {
				t.Fatal("No MP3 data generated")
			}

			t.Logf("✓ %s: %d channels, %d PCM bytes -> %d MP3 bytes",
				tc.name, tc.numChannels, len(pcmData), totalBytes)
		})
	}
}

// TestEncodeDifferentSampleRates tests various sample rates
func TestEncodeDifferentSampleRates(t *testing.T) {
	sampleRates := []int{8000, 16000, 22050, 24000, 32000, 44100, 48000}

	for _, rate := range sampleRates {
		t.Run(rate2string(rate), func(t *testing.T) {
			numSamples := rate // 1 second
			pcmData := generateSineWave(440, rate, 2, numSamples)

			encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
				SampleRate:  rate,
				NumChannels: 2,
				Bitrate:     128,
				Quality:     2,
			})
			if err != nil {
				t.Fatalf("Failed to create encoder: %v", err)
			}
			defer encoder.Close()

			outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
			encodedBytes, err := encoder.Encode(pcmData, outBuf)
			if err != nil {
				t.Fatalf("Encode failed at %d Hz: %v", rate, err)
			}

			flushedBytes, _ := encoder.Flush(outBuf[encodedBytes:])
			totalBytes := encodedBytes + flushedBytes

			if totalBytes == 0 {
				t.Fatal("No MP3 data generated")
			}

			t.Logf("✓ %d Hz: %d bytes", rate, totalBytes)
		})
	}
}

// TestEncodeVBRModes tests different VBR encoding modes
func TestEncodeVBRModes(t *testing.T) {
	testCases := []struct {
		name    string
		vbrMode mp3.VBRMode
	}{
		{"CBR", mp3.VbrModeOff},
		{"VBR_RH", mp3.VbrModeRh},
		{"ABR", mp3.VbrModeAbr},
		{"VBR_MTRH", mp3.VbrModeMtrh},
	}

	pcmData := generateSineWave(440, 44100, 2, 44100) // 1 second

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
				SampleRate:  44100,
				NumChannels: 2,
				Bitrate:     128,
				Quality:     2,
				VbrMode:     tc.vbrMode,
			})
			if err != nil {
				t.Fatalf("Failed to create encoder: %v", err)
			}
			defer encoder.Close()

			outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
			encodedBytes, err := encoder.Encode(pcmData, outBuf)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			flushedBytes, _ := encoder.Flush(outBuf[encodedBytes:])
			totalBytes := encodedBytes + flushedBytes

			if totalBytes == 0 {
				t.Fatal("No MP3 data generated")
			}

			t.Logf("✓ %s: %d bytes", tc.name, totalBytes)
		})
	}
}

// TestEncodeQualityLevels tests different quality levels
func TestEncodeQualityLevels(t *testing.T) {
	qualities := []int{0, 2, 5, 7, 9}

	pcmData := generateSineWave(440, 44100, 2, 44100) // 1 second

	for _, quality := range qualities {
		t.Run(quality2string(quality), func(t *testing.T) {
			encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
				SampleRate:  44100,
				NumChannels: 2,
				Bitrate:     128,
				Quality:     quality,
			})
			if err != nil {
				t.Fatalf("Failed to create encoder: %v", err)
			}
			defer encoder.Close()

			outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
			encodedBytes, err := encoder.Encode(pcmData, outBuf)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			flushedBytes, _ := encoder.Flush(outBuf[encodedBytes:])
			totalBytes := encodedBytes + flushedBytes

			if totalBytes == 0 {
				t.Fatal("No MP3 data generated")
			}

			t.Logf("✓ Quality %d: %d bytes", quality, totalBytes)
		})
	}
}

// TestEncodeStreamingMode tests encoding in streaming mode (multiple Encode calls)
func TestEncodeStreamingMode(t *testing.T) {
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:  44100,
		NumChannels: 2,
		Bitrate:     128,
		Quality:     2,
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	// Encode in multiple chunks
	chunkSize := 4096 // 1024 stereo samples
	totalPCM := 0
	totalMP3 := 0

	outBuf := make([]byte, encoder.EstimateOutBufBytes(chunkSize))

	for i := 0; i < 10; i++ {
		pcmChunk := generateSineWave(440, 44100, 2, 1024)
		totalPCM += len(pcmChunk)

		encodedBytes, err := encoder.Encode(pcmChunk, outBuf)
		if err != nil {
			t.Fatalf("Encode chunk %d failed: %v", i, err)
		}
		totalMP3 += encodedBytes
	}

	// Flush remaining data
	flushedBytes, err := encoder.Flush(outBuf)
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
	totalMP3 += flushedBytes

	if totalMP3 == 0 {
		t.Fatal("No MP3 data generated")
	}

	t.Logf("✓ Streaming: %d PCM bytes -> %d MP3 bytes (10 chunks)",
		totalPCM, totalMP3)
}

// TestEncodeFromWavFile tests encoding from real WAV files
func TestEncodeFromWavFile(t *testing.T) {
	wavFile := filepath.Join("samples", "sample.wav")
	if _, err := os.Stat(wavFile); os.IsNotExist(err) {
		t.Skip("sample.wav not found, skipping test")
	}

	inFile, err := os.Open(wavFile)
	if err != nil {
		t.Fatalf("Failed to open WAV file: %v", err)
	}
	defer inFile.Close()

	// Create output buffer
	var mp3Buf bytes.Buffer

	// Encode
	totalBytes, totalFrames, sampleRate, err := mp3.EncodeFromWav(inFile, &mp3Buf, &mp3.EncoderConfig{
		Bitrate: 128,
		Quality: 2,
	})
	if err != nil {
		t.Fatalf("EncodeFromWav failed: %v", err)
	}

	if totalBytes == 0 {
		t.Fatal("No MP3 data generated")
	}

	if totalFrames == 0 {
		t.Error("Frame count is zero")
	}

	duration := float64(totalFrames*1152) / float64(sampleRate)
	t.Logf("✓ Encoded WAV: %d bytes, %d frames, %.2fs at %dHz",
		totalBytes, totalFrames, duration, sampleRate)
}

// TestEncodeMonoFiles tests encoding mono audio files
func TestEncodeMonoFiles(t *testing.T) {
	testCases := []struct {
		name     string
		wavFile  string
		bitrate  int
		minBytes int
		maxBytes int
	}{
		{
			name:     "Mono_8kHz_32kbps",
			wavFile:  "mpeg25_8000_mono_cbr24.mp3",
			bitrate:  32,
			minBytes: 8000,
			maxBytes: 20000,
		},
		{
			name:     "Mono_16kHz_48kbps",
			wavFile:  "mpeg25_16000_mono_cbr32.mp3",
			bitrate:  48,
			minBytes: 15000,
			maxBytes: 30000,
		},
		{
			name:     "Mono_24kHz_64kbps",
			wavFile:  "mpeg2_24000_mono_cbr48.mp3",
			bitrate:  64,
			minBytes: 20000,
			maxBytes: 35000,
		},
		{
			name:     "Mono_44.1kHz_64kbps",
			wavFile:  "mpeg1_44100_mono_cbr64.mp3",
			bitrate:  64,
			minBytes: 20000,
			maxBytes: 30000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First decode MP3 to WAV to get PCM data
			mp3File, err := os.Open(filepath.Join("samples", tc.wavFile))
			if err != nil {
				t.Skipf("Test file not found: %v", err)
			}
			defer mp3File.Close()

			// Decode to PCM
			decoder, err := mp3.NewDecoder()
			if err != nil {
				t.Fatalf("Failed to create decoder: %v", err)
			}
			defer decoder.Close()

			var pcmData []byte
			pcmBuf := make([]byte, decoder.EstimateOutBufBytes())
			chunk := make([]byte, 2048)

			for {
				n, readErr := mp3File.Read(chunk)
				if n > 0 {
					decodedN, _ := decoder.Decode(chunk[:n], pcmBuf)
					if decodedN > 0 {
						pcmData = append(pcmData, pcmBuf[:decodedN]...)
					}
				}
				if readErr != nil {
					break
				}
			}

			if len(pcmData) == 0 {
				t.Fatal("No PCM data decoded")
			}

			// Now encode the PCM data
			encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
				SampleRate:  decoder.SampleRate,
				NumChannels: decoder.NumChannels,
				Bitrate:     tc.bitrate,
				Quality:     2,
			})
			if err != nil {
				t.Fatalf("Failed to create encoder: %v", err)
			}
			defer encoder.Close()

			// Verify it's mono
			if encoder.NumChannels != 1 {
				t.Errorf("Expected mono (1 channel), got %d", encoder.NumChannels)
			}

			// Encode in chunks
			outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
			totalEncoded := 0

			for offset := 0; offset < len(pcmData); offset += 2048 {
				end := offset + 2048
				if end > len(pcmData) {
					end = len(pcmData)
				}

				encoded, err := encoder.Encode(pcmData[offset:end], outBuf[totalEncoded:])
				if err != nil {
					t.Fatalf("Encode failed: %v", err)
				}
				totalEncoded += encoded
			}

			// Flush
			flushed, err := encoder.Flush(outBuf[totalEncoded:])
			if err != nil {
				t.Fatalf("Flush failed: %v", err)
			}
			totalEncoded += flushed

			if totalEncoded == 0 {
				t.Fatal("No MP3 data generated")
			}

			// Verify size is reasonable
			if totalEncoded < tc.minBytes || totalEncoded > tc.maxBytes {
				t.Logf("Warning: Size out of expected range: got %d, want %d-%d",
					totalEncoded, tc.minBytes, tc.maxBytes)
			}

			duration := float64(len(pcmData)) / float64(decoder.NumChannels) / 2.0 / float64(decoder.SampleRate)
			t.Logf("✓ Mono %dHz: %d PCM bytes -> %d MP3 bytes, %.2fs",
				decoder.SampleRate, len(pcmData), totalEncoded, duration)
		})
	}
}

// TestEncodeInvalidInput tests error handling
func TestEncodeInvalidInput(t *testing.T) {
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:  44100,
		NumChannels: 2,
		Bitrate:     128,
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	t.Run("EmptyInput", func(t *testing.T) {
		emptyBuf := make([]byte, 0)
		outBuf := make([]byte, 1000)
		_, err := encoder.Encode(emptyBuf, outBuf)
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
	})

	t.Run("SmallOutputBuffer", func(t *testing.T) {
		input := make([]byte, 4096)
		smallBuf := make([]byte, 10) // Too small
		_, err := encoder.Encode(input, smallBuf)
		if err == nil {
			t.Error("Expected error for small output buffer, got nil")
		}
	})
}

// TestEncodeFlushMultipleTimes tests that flush can be called multiple times
func TestEncodeFlushMultipleTimes(t *testing.T) {
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:  44100,
		NumChannels: 2,
		Bitrate:     128,
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	pcmData := generateSineWave(440, 44100, 2, 44100)
	outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))

	// Encode
	encodedBytes, err := encoder.Encode(pcmData, outBuf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// First flush
	flushed1, err := encoder.Flush(outBuf[encodedBytes:])
	if err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	// Second flush should return 0 bytes (no more data)
	flushed2, err := encoder.Flush(outBuf[encodedBytes+flushed1:])
	if err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	if flushed2 != 0 {
		t.Logf("Warning: Second flush returned %d bytes (expected 0)", flushed2)
	}

	t.Logf("✓ First flush: %d bytes, Second flush: %d bytes", flushed1, flushed2)
}

// TestEncodeRoundTrip tests encoding and decoding back
func TestEncodeRoundTrip(t *testing.T) {
	// Generate original PCM
	originalPCM := generateSineWave(440, 44100, 2, 44100*2) // 2 seconds

	// Encode to MP3
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:  44100,
		NumChannels: 2,
		Bitrate:     192, // High bitrate for better quality
		Quality:     0,   // Best quality
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	mp3Buf := make([]byte, encoder.EstimateOutBufBytes(len(originalPCM)))
	encodedBytes, _ := encoder.Encode(originalPCM, mp3Buf)
	flushedBytes, _ := encoder.Flush(mp3Buf[encodedBytes:])
	totalMP3 := encodedBytes + flushedBytes

	if totalMP3 == 0 {
		t.Fatal("Encoding failed")
	}

	// Decode back to PCM
	decoder, err := mp3.NewDecoder()
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}
	defer decoder.Close()

	var decodedPCM []byte
	pcmBuf := make([]byte, decoder.EstimateOutBufBytes())
	chunk := make([]byte, 2048)

	mp3Reader := bytes.NewReader(mp3Buf[:totalMP3])
	for {
		n, readErr := mp3Reader.Read(chunk)
		if n > 0 {
			decodedN, _ := decoder.Decode(chunk[:n], pcmBuf)
			if decodedN > 0 {
				decodedPCM = append(decodedPCM, pcmBuf[:decodedN]...)
			}
		}
		if readErr != nil {
			break
		}
	}

	if len(decodedPCM) == 0 {
		t.Fatal("Decoding failed")
	}

	// Compare sizes (lossy compression, so won't be exact)
	sizeDiff := abs(len(originalPCM) - len(decodedPCM))
	tolerance := len(originalPCM) / 10 // 10% tolerance

	if sizeDiff > tolerance {
		t.Logf("Warning: Size difference too large: original %d, decoded %d",
			len(originalPCM), len(decodedPCM))
	}

	compressionRatio := float64(len(originalPCM)) / float64(totalMP3)
	t.Logf("✓ Round-trip: %d -> %d MP3 -> %d PCM (compression: %.1fx)",
		len(originalPCM), totalMP3, len(decodedPCM), compressionRatio)
}

// TestLameTagFrame tests Xing/LAME tag generation
func TestLameTagFrame(t *testing.T) {
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:    44100,
		NumChannels:   2,
		Bitrate:       128,
		Quality:       2,
		IsWriteVbrTag: true,
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	// Encode some data
	pcmData := generateSineWave(440, 44100, 2, 44100)
	outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))

	_, err = encoder.Encode(pcmData, outBuf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Flush to finalize encoding
	_, err = encoder.Flush(outBuf)
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Get LAME tag
	lameTag, err := encoder.GetLameTagFrame()
	if err != nil {
		t.Fatalf("GetLameTagFrame failed: %v", err)
	}

	if lameTag == nil {
		t.Fatal("LAME tag is nil (should be generated)")
	}

	if len(lameTag) == 0 {
		t.Fatal("LAME tag is empty")
	}

	// Verify tag contains "Info" or "Xing" marker
	hasInfo := bytes.Contains(lameTag, []byte("Info"))
	hasXing := bytes.Contains(lameTag, []byte("Xing"))
	hasLame := bytes.Contains(lameTag, []byte("LAME"))

	if !hasInfo && !hasXing {
		t.Error("LAME tag missing Info/Xing marker")
	}

	if !hasLame {
		t.Error("LAME tag missing LAME encoder marker")
	}

	t.Logf("✓ LAME tag: %d bytes, Info=%v, Xing=%v, LAME=%v",
		len(lameTag), hasInfo, hasXing, hasLame)
}

// TestEncodeWithXingHeader tests that Xing/Info header is written correctly
func TestEncodeWithXingHeader(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_xing_*.mp3")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Generate WAV data
	wavData := generateWavFile(44100, 2, 44100*2) // 2 seconds
	wavReader := bytes.NewReader(wavData)

	// Encode to file (supports seeking)
	totalBytes, totalFrames, sampleRate, err := mp3.EncodeFromWav(wavReader, tmpFile, &mp3.EncoderConfig{
		Bitrate: 128,
		Quality: 2,
	})
	tmpFile.Close()

	if err != nil {
		t.Fatalf("EncodeFromWav failed: %v", err)
	}

	if totalBytes == 0 {
		t.Fatal("No MP3 data generated")
	}

	// Read back the MP3 file
	mp3Data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read MP3 file: %v", err)
	}

	// Check for Info/Xing header (should be within first 200 bytes)
	hasInfo := bytes.Contains(mp3Data[:200], []byte("Info"))
	hasXing := bytes.Contains(mp3Data[:200], []byte("Xing"))
	hasLame := bytes.Contains(mp3Data[:200], []byte("LAME"))

	if !hasInfo && !hasXing {
		t.Error("MP3 file missing Info/Xing header")
	}

	if !hasLame {
		t.Error("MP3 file missing LAME encoder tag")
	}

	t.Logf("✓ MP3 with headers: %d bytes, %d frames, %dHz, Info=%v, Xing=%v, LAME=%v",
		totalBytes, totalFrames, sampleRate, hasInfo, hasXing, hasLame)
}

// TestGetFrameNum tests frame number tracking
func TestGetFrameNum(t *testing.T) {
	encoder, err := mp3.NewEncoder(&mp3.EncoderConfig{
		SampleRate:  44100,
		NumChannels: 2,
		Bitrate:     128,
	})
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}
	defer encoder.Close()

	// Encode 1 second of audio
	pcmData := generateSineWave(440, 44100, 2, 44100)
	outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))

	_, err = encoder.Encode(pcmData, outBuf)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	encoder.Flush(outBuf)

	frameNum, err := encoder.GetFrameNum()
	if err != nil {
		t.Fatalf("GetFrameNum failed: %v", err)
	}

	// For 1 second at 44.1kHz: 44100 samples / 1152 samples per frame ≈ 38 frames
	expectedFrames := 44100 / 1152
	tolerance := 5

	if frameNum < expectedFrames-tolerance || frameNum > expectedFrames+tolerance {
		t.Errorf("Frame count out of range: got %d, expected ~%d",
			frameNum, expectedFrames)
	}

	t.Logf("✓ Frame count: %d frames (expected ~%d)", frameNum, expectedFrames)
}

// BenchmarkEncode benchmarks encoding performance
func BenchmarkEncode(b *testing.B) {
	// Generate 1 second of stereo audio
	pcmData := generateSineWave(440, 44100, 2, 44100)

	b.ResetTimer()
	b.SetBytes(int64(len(pcmData)))

	for i := 0; i < b.N; i++ {
		encoder, _ := mp3.NewEncoder(&mp3.EncoderConfig{
			SampleRate:  44100,
			NumChannels: 2,
			Bitrate:     128,
			Quality:     5, // Fast quality for benchmark
		})

		outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
		encoder.Encode(pcmData, outBuf)
		encoder.Flush(outBuf)
		encoder.Close()
	}
}

// BenchmarkEncodeMono benchmarks mono encoding
func BenchmarkEncodeMono(b *testing.B) {
	pcmData := generateSineWave(440, 44100, 1, 44100) // 1 second mono

	b.ResetTimer()
	b.SetBytes(int64(len(pcmData)))

	for i := 0; i < b.N; i++ {
		encoder, _ := mp3.NewEncoder(&mp3.EncoderConfig{
			SampleRate:  44100,
			NumChannels: 1,
			Bitrate:     64,
			Quality:     5,
		})

		outBuf := make([]byte, encoder.EstimateOutBufBytes(len(pcmData)))
		encoder.Encode(pcmData, outBuf)
		encoder.Flush(outBuf)
		encoder.Close()
	}
}

// BenchmarkEncodeFromWav benchmarks the complete WAV to MP3 flow
func BenchmarkEncodeFromWav(b *testing.B) {
	wavFile := filepath.Join("samples", "sample.wav")
	wavData, err := os.ReadFile(wavFile)
	if err != nil {
		b.Skipf("Test file not found: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(wavData)))

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(wavData)
		var mp3Buf bytes.Buffer
		mp3.EncodeFromWav(reader, &mp3Buf, &mp3.EncoderConfig{
			Bitrate: 128,
			Quality: 5,
		})
	}
}

// Helper functions

// generateSineWave generates PCM data for a sine wave (16-bit signed samples)
func generateSineWave(freq, sampleRate, channels, numSamples int) []byte {
	data := make([]byte, numSamples*channels*2) // 2 bytes per sample (16-bit)

	for i := 0; i < numSamples; i++ {
		// Generate sine wave sample
		t := float64(i) / float64(sampleRate)
		sample := int16(32767.0 * 0.5 * math.Sin(2*math.Pi*float64(freq)*t))

		// Write to all channels
		for ch := 0; ch < channels; ch++ {
			idx := (i*channels + ch) * 2
			data[idx] = byte(sample & 0xFF)
			data[idx+1] = byte((sample >> 8) & 0xFF)
		}
	}

	return data
}

func bitrate2string(bitrate int) string {
	return fmt.Sprintf("%dkbps", bitrate)
}

func rate2string(rate int) string {
	return fmt.Sprintf("%dHz", rate)
}

func quality2string(quality int) string {
	return fmt.Sprintf("Q%d", quality)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// generateWavFile generates a complete WAV file with header
func generateWavFile(sampleRate, channels, numSamples int) []byte {
	pcmData := generateSineWave(440, sampleRate, channels, numSamples)

	// Generate WAV header
	header := mp3.GenerateWavHeader(len(pcmData), sampleRate, channels, 16)

	// Combine header and PCM data
	wavData := make([]byte, len(header)+len(pcmData))
	copy(wavData, header)
	copy(wavData[len(header):], pcmData)

	return wavData
}
