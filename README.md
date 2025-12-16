# audio-mp3

Go bindings for [mpg123](https://www.mpg123.de/download.shtml) to provide mp3 decoding.

## compile mpg123

```bash
./configure --disable-components --enable-libmpg123 --enable-static --prefix=$(pwd)/bin
mkae
make install
```
