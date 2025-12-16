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
