package main

import (
	"fmt"
	mp3 "github.com/lizc2003/audio-mp3"
	"os"
)

func main() {
	encodeFromWav()
}

func encodeFromWav() {
	in, err := os.Open("samples/sample.wav")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer in.Close()

	out, err := os.Create("output.mp3")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	totalBytes, totalFrames, sampleRate, err := mp3.EncodeFromWav(in, out, &mp3.EncoderConfig{
		Bitrate: 128,
		Quality: 2,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("totalBytes: %d, totalFrames: %d, sampleRate: %d\n", totalBytes, totalFrames, sampleRate)
}
