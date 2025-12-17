package mp3

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	WavHeaderSize = 44
)

// EncodeFromWav encodes a WAV audio stream into mp3 format.
// This function parses the WAV header to extract SampleRate and MaxChannels, overriding the values in config.
func EncodeFromWav(wavStream io.Reader, writer io.Writer, config *EncoderConfig) (totalBytes int, totalFrames int, sampleRate int, err error) {
	pcmSize, sampleRate, numChannels, bitsPerSample, err := ParseWavHeader(wavStream)
	if err != nil {
		return 0, 0, 0, err
	}
	if bitsPerSample != SampleBitDepth {
		return 0, 0, 0, fmt.Errorf("unsupported bits per sample: %d (only 16-bit supported)", bitsPerSample)
	}

	config.SampleRate = sampleRate
	config.NumChannels = numChannels
	// Limit the reader to the data size to avoid reading trailing metadata as audio.
	wavStream = io.LimitReader(wavStream, int64(pcmSize))

	encoder, err := NewEncoder(config)
	if err != nil {
		return 0, 0, 0, err
	}
	defer encoder.Close()

	// Buffer for reading input PCM data
	chunkSize := 2048
	inBuf := make([]byte, chunkSize)
	outBuf := make([]byte, encoder.EstimateOutBufBytes(chunkSize))

	for {
		n, err := wavStream.Read(inBuf)
		if n > 0 {
			encodedBytes, encErr := encoder.Encode(inBuf[:n], outBuf)
			if encErr != nil {
				return 0, 0, 0, encErr
			}
			if encodedBytes > 0 {
				totalBytes += encodedBytes
				if _, wErr := writer.Write(outBuf[:encodedBytes]); wErr != nil {
					return 0, 0, 0, wErr
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, 0, 0, err
		}
	}

	encodedBytes, flushErr := encoder.Flush(outBuf)
	if flushErr != nil {
		return 0, 0, 0, flushErr
	}
	if encodedBytes > 0 {
		totalBytes += encodedBytes
		if _, wErr := writer.Write(outBuf[:encodedBytes]); wErr != nil {
			return 0, 0, 0, wErr
		}
	}

	totalFrames, err = encoder.GetFrameNum()
	if err != nil {
		return 0, 0, 0, err
	}

	return totalBytes, totalFrames, sampleRate, nil
}

// DecodeToWav decodes a mp3 stream to WAV format and writes it to the output writer.
func DecodeToWav(inStream io.Reader, writer io.WriteSeeker) (totalBytes int, totalSamples int, sampleRate int, err error) {
	decoder, err := NewDecoder()
	if err != nil {
		return 0, 0, 0, err
	}
	defer decoder.Close()

	pcmBuf := make([]byte, decoder.EstimateOutBufBytes())
	chunk := make([]byte, 2048)

	for {
		n, readErr := inStream.Read(chunk)
		if n > 0 {
			decodedN, decErr := decoder.Decode(chunk[:n], pcmBuf)
			if decErr != nil {
				return 0, 0, 0, decErr
			}

			if decodedN > 0 {
				if totalBytes == 0 {
					// Write placeholder WAV header
					headerBuf := make([]byte, WavHeaderSize)
					if _, err := writer.Write(headerBuf); err != nil {
						return 0, 0, 0, fmt.Errorf("write placeholder header failed: %w", err)
					}
				}

				if _, wErr := writer.Write(pcmBuf[:decodedN]); wErr != nil {
					return 0, 0, 0, wErr
				}
				totalBytes += decodedN
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return 0, 0, 0, readErr
		}
	}

	if totalBytes == 0 {
		return 0, 0, 0, errors.New("no audio frames decoded")
	}

	// Update WAV header
	if _, err := writer.Seek(0, io.SeekStart); err != nil {
		// If we can't seek, the file will have invalid header.
		return 0, 0, 0, fmt.Errorf("seek to start failed: %w", err)
	}

	header := GenerateWavHeader(totalBytes, decoder.SampleRate, decoder.NumChannels, decoder.SampleBitDepth)
	if _, err := writer.Write(header); err != nil {
		return 0, 0, 0, fmt.Errorf("write real header failed: %w", err)
	}

	// Not strictly necessary but good practice.
	writer.Seek(0, io.SeekEnd)

	totalSamples = totalBytes / (decoder.NumChannels * decoder.SampleBitDepth / 8)
	return totalBytes + WavHeaderSize, totalSamples, decoder.SampleRate, nil
}

func GenerateWavHeader(pcmSize int, sampleRate int, numChannels int, bitsPerSample int) []byte {
	header := make([]byte, WavHeaderSize)
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8

	// RIFF
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+pcmSize))
	copy(header[8:12], []byte("WAVE"))

	// fmt
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16) // Subchunk1Size for PCM
	binary.LittleEndian.PutUint16(header[20:22], 1)  // AudioFormat 1 = PCM
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))

	// data
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], uint32(pcmSize))

	return header
}

func ParseWavHeader(wavStream io.Reader) (pcmSize int, sampleRate int, numChannels int, bitsPerSample int, err error) {
	var (
		riffHeader    [12]byte
		chunkHeader   [8]byte
		fmtChunkFound bool
	)

	// Read RIFF header
	if _, err := io.ReadFull(wavStream, riffHeader[:]); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("read RIFF header failed: %w", err)
	}
	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return 0, 0, 0, 0, errors.New("invalid WAV header: missing RIFF/WAVE")
	}

	// Loop chunks
	for {
		if _, err := io.ReadFull(wavStream, chunkHeader[:]); err != nil {
			return 0, 0, 0, 0, fmt.Errorf("read chunk header failed: %w", err)
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			if chunkSize < 16 {
				return 0, 0, 0, 0, fmt.Errorf("invalid fmt chunk size: %d", chunkSize)
			}
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(wavStream, fmtData); err != nil {
				return 0, 0, 0, 0, fmt.Errorf("read fmt chunk failed: %w", err)
			}

			audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
			numChannels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))

			if audioFormat != 1 {
				return 0, 0, 0, 0, fmt.Errorf("unsupported audio format: %d (only PCM supported)", audioFormat)
			}
			fmtChunkFound = true
		} else if chunkID == "data" {
			if !fmtChunkFound {
				return 0, 0, 0, 0, errors.New("data chunk found before fmt chunk")
			}
			// We found data chunk, stop parsing.
			pcmSize = int(chunkSize)
			break
		} else {
			// Skip other chunks
			if _, err := io.CopyN(io.Discard, wavStream, int64(chunkSize)); err != nil {
				return 0, 0, 0, 0, fmt.Errorf("skip chunk %s failed: %w", chunkID, err)
			}
		}
	}
	return pcmSize, sampleRate, numChannels, bitsPerSample, nil
}
