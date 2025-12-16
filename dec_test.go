package mp3_test

import (
	mp3 "github.com/lizc2003/audio-mp3"
	"os"
	"path/filepath"
	"testing"
)

// TestCase defines a test case for MP3 decoding
type TestCase struct {
	filename      string
	expectedRate  int
	expectedChans int
	expectedBits  int
	minSamples    int // Minimum expected samples (accounting for gapless)
	maxSamples    int // Maximum expected samples
}

// getTestCases returns all test cases for different MP3 encodings
func getTestCases() []TestCase {
	return []TestCase{
		{
			filename:      "mpeg1_44100_stereo_cbr128.mp3",
			expectedRate:  44100,
			expectedChans: 2,
			expectedBits:  16,
			minSamples:    130000, // ~3 seconds
			maxSamples:    135000,
		},
		{
			filename:      "mpeg1_44100_mono_cbr64.mp3",
			expectedRate:  44100,
			expectedChans: 1,
			expectedBits:  16,
			minSamples:    130000,
			maxSamples:    135000,
		},
		{
			filename:      "mpeg1_48000_stereo_cbr192.mp3",
			expectedRate:  48000,
			expectedChans: 2,
			expectedBits:  16,
			minSamples:    142000, // ~3 seconds at 48kHz
			maxSamples:    146000,
		},
		{
			filename:      "mpeg1_32000_stereo_cbr96.mp3",
			expectedRate:  32000,
			expectedChans: 2,
			expectedBits:  16,
			minSamples:    94000, // ~3 seconds at 32kHz
			maxSamples:    98000,
		},
		{
			filename:      "mpeg2_22050_stereo_cbr64.mp3",
			expectedRate:  22050,
			expectedChans: 2,
			expectedBits:  16,
			minSamples:    64000, // ~3 seconds at 22.05kHz
			maxSamples:    68000,
		},
		{
			filename:      "mpeg2_24000_mono_cbr48.mp3",
			expectedRate:  24000,
			expectedChans: 1,
			expectedBits:  16,
			minSamples:    70000, // ~3 seconds at 24kHz
			maxSamples:    74000,
		},
		{
			filename:      "mpeg25_16000_mono_cbr32.mp3",
			expectedRate:  16000,
			expectedChans: 1,
			expectedBits:  16,
			minSamples:    46000, // ~3 seconds at 16kHz
			maxSamples:    50000,
		},
		{
			filename:      "mpeg25_8000_mono_cbr24.mp3",
			expectedRate:  8000,
			expectedChans: 1,
			expectedBits:  16,
			minSamples:    22000, // ~3 seconds at 8kHz
			maxSamples:    26000,
		},
		{
			filename:      "mpeg1_44100_stereo_vbr_q2.mp3",
			expectedRate:  44100,
			expectedChans: 2,
			expectedBits:  16,
			minSamples:    130000,
			maxSamples:    135000,
		},
		{
			filename:      "mpeg1_44100_stereo_vbr_q7.mp3",
			expectedRate:  44100,
			expectedChans: 2,
			expectedBits:  16,
			minSamples:    130000,
			maxSamples:    135000,
		},
	}
}

// TestDecodeVariousEncodings tests decoding of various MP3 encoding formats
func TestDecodeVariousEncodings(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			// Open test MP3 file
			mp3Path := filepath.Join("samples", tc.filename)
			mp3File, err := os.Open(mp3Path)
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", tc.filename, err)
			}
			defer mp3File.Close()

			// Create decoder
			decoder, err := mp3.NewDecoder()
			if err != nil {
				t.Fatalf("Failed to create decoder: %v", err)
			}
			defer decoder.Close()

			// Decode the entire file
			pcmBuf := make([]byte, decoder.EstimateOutBufBytes())
			chunk := make([]byte, 2048)
			totalBytes := 0

			for {
				n, readErr := mp3File.Read(chunk)
				if n > 0 {
					decodedN, decErr := decoder.Decode(chunk[:n], pcmBuf)
					if decErr != nil {
						t.Fatalf("Decode error: %v", decErr)
					}

					if decodedN > 0 {
						// Check format on first decoded frame
						if totalBytes == 0 {
							if decoder.SampleRate != tc.expectedRate {
								t.Errorf("Sample rate mismatch: got %d, want %d",
									decoder.SampleRate, tc.expectedRate)
							}
							if decoder.NumChannels != tc.expectedChans {
								t.Errorf("Channels mismatch: got %d, want %d",
									decoder.NumChannels, tc.expectedChans)
							}
							if decoder.SampleBitDepth != tc.expectedBits {
								t.Errorf("Bit depth mismatch: got %d, want %d",
									decoder.SampleBitDepth, tc.expectedBits)
							}
						}
						totalBytes += decodedN
					}
				}

				if readErr != nil {
					break
				}
			}

			// Verify we decoded some data
			if totalBytes == 0 {
				t.Fatal("No data decoded")
			}

			// Calculate samples
			bytesPerSample := decoder.NumChannels * (decoder.SampleBitDepth / 8)
			totalSamples := totalBytes / bytesPerSample

			// Verify sample count is within expected range
			if totalSamples < tc.minSamples || totalSamples > tc.maxSamples {
				t.Errorf("Sample count out of range: got %d, want %d-%d",
					totalSamples, tc.minSamples, tc.maxSamples)
			}

			// Log success info
			duration := float64(totalSamples) / float64(decoder.SampleRate)
			t.Logf("✓ Decoded: %d samples, %.2fs, %dHz, %dch, %dbit",
				totalSamples, duration, decoder.SampleRate,
				decoder.NumChannels, decoder.SampleBitDepth)
		})
	}
}

// TestInvalidInput tests decoder behavior with invalid input
func TestInvalidInput(t *testing.T) {
	decoder, err := mp3.NewDecoder()
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}
	defer decoder.Close()

	pcmBuf := make([]byte, decoder.EstimateOutBufBytes())

	t.Run("EmptyInput", func(t *testing.T) {
		emptyBuf := make([]byte, 0)
		_, err := decoder.Decode(emptyBuf, pcmBuf)
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
	})

	t.Run("SmallOutputBuffer", func(t *testing.T) {
		input := make([]byte, 1024)
		smallBuf := make([]byte, 100) // Too small
		_, err := decoder.Decode(input, smallBuf)
		if err == nil {
			t.Error("Expected error for small output buffer, got nil")
		}
	})

	t.Run("GarbageData", func(t *testing.T) {
		garbage := make([]byte, 1024)
		for i := range garbage {
			garbage[i] = byte(i % 256)
		}
		// This should not crash, but may not decode anything
		_, err := decoder.Decode(garbage, pcmBuf)
		// We don't require an error here as mpg123 may just skip invalid data
		t.Logf("Garbage data result: %v", err)
	})
}

// BenchmarkDecode benchmarks the decoding performance
func BenchmarkDecode(b *testing.B) {
	mp3Path := filepath.Join("samples", "mpeg1_44100_stereo_cbr128.mp3")

	// Read entire file into memory
	mp3Data, err := os.ReadFile(mp3Path)
	if err != nil {
		b.Skipf("Test file not found: %v", err)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(mp3Data)))

	for i := 0; i < b.N; i++ {
		decoder, err := mp3.NewDecoder()
		if err != nil {
			b.Fatal(err)
		}

		pcmBuf := make([]byte, decoder.EstimateOutBufBytes())
		chunk := make([]byte, 2048)

		for offset := 0; offset < len(mp3Data); offset += len(chunk) {
			end := offset + len(chunk)
			if end > len(mp3Data) {
				end = len(mp3Data)
			}

			decoder.Decode(mp3Data[offset:end], pcmBuf)
		}

		decoder.Close()
	}
}
