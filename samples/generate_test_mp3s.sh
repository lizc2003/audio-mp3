#!/bin/bash
# Generate various MP3 test files with different encodings

# Generate a 3-second test tone (440Hz sine wave)
ffmpeg -f lavfi -i "sine=frequency=440:duration=3" -y source.wav 2>/dev/null

echo "Generating MP3 test files..."

# 1. MPEG-1 Layer III, 44.1kHz, Stereo, CBR 128kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 128k -ar 44100 -ac 2 -y mpeg1_44100_stereo_cbr128.mp3 2>/dev/null
echo "✓ mpeg1_44100_stereo_cbr128.mp3"

# 2. MPEG-1 Layer III, 44.1kHz, Mono, CBR 64kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 64k -ar 44100 -ac 1 -y mpeg1_44100_mono_cbr64.mp3 2>/dev/null
echo "✓ mpeg1_44100_mono_cbr64.mp3"

# 3. MPEG-1 Layer III, 48kHz, Stereo, CBR 192kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 192k -ar 48000 -ac 2 -y mpeg1_48000_stereo_cbr192.mp3 2>/dev/null
echo "✓ mpeg1_48000_stereo_cbr192.mp3"

# 4. MPEG-1 Layer III, 32kHz, Stereo, CBR 96kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 96k -ar 32000 -ac 2 -y mpeg1_32000_stereo_cbr96.mp3 2>/dev/null
echo "✓ mpeg1_32000_stereo_cbr96.mp3"

# 5. MPEG-2 Layer III, 22.05kHz, Stereo, CBR 64kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 64k -ar 22050 -ac 2 -y mpeg2_22050_stereo_cbr64.mp3 2>/dev/null
echo "✓ mpeg2_22050_stereo_cbr64.mp3"

# 6. MPEG-2 Layer III, 24kHz, Mono, CBR 48kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 48k -ar 24000 -ac 1 -y mpeg2_24000_mono_cbr48.mp3 2>/dev/null
echo "✓ mpeg2_24000_mono_cbr48.mp3"

# 7. MPEG-2.5 Layer III, 16kHz, Mono, CBR 32kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 32k -ar 16000 -ac 1 -y mpeg25_16000_mono_cbr32.mp3 2>/dev/null
echo "✓ mpeg25_16000_mono_cbr32.mp3"

# 8. MPEG-2.5 Layer III, 8kHz, Mono, CBR 24kbps
ffmpeg -i source.wav -codec:a libmp3lame -b:a 24k -ar 8000 -ac 1 -y mpeg25_8000_mono_cbr24.mp3 2>/dev/null
echo "✓ mpeg25_8000_mono_cbr24.mp3"

# 9. MPEG-1 Layer III, 44.1kHz, Stereo, VBR quality 2 (high quality)
ffmpeg -i source.wav -codec:a libmp3lame -q:a 2 -ar 44100 -ac 2 -y mpeg1_44100_stereo_vbr_q2.mp3 2>/dev/null
echo "✓ mpeg1_44100_stereo_vbr_q2.mp3"

# 10. MPEG-1 Layer III, 44.1kHz, Stereo, VBR quality 7 (lower quality)
ffmpeg -i source.wav -codec:a libmp3lame -q:a 7 -ar 44100 -ac 2 -y mpeg1_44100_stereo_vbr_q7.mp3 2>/dev/null
echo "✓ mpeg1_44100_stereo_vbr_q7.mp3"

# Clean up
rm -f source.wav

echo ""
echo "Generated $(ls -1 *.mp3 | wc -l) test MP3 files"
ls -lh *.mp3
