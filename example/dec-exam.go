package main

import (
	"fmt"
	"os"

	"github.com/lizc2003/audio-mp3"
)

func main() {
	decodeToWav()
}

func decodeToWav() {
	inFile, err := os.Open("samples/sample.mp3")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer inFile.Close()

	wavFile, err := os.Create("output.wav")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer wavFile.Close()

	totalBytes, totalSamples, sampleRate, err := mp3.DecodeToWav(inFile, wavFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("decoded %d bytes, total samples: %d, sample rate: %d\n", totalBytes, totalSamples, sampleRate)
}
