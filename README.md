# audio-mp3

Go bindings for [mpg123](https://www.mpg123.de/download.shtml) to provide mp3 decoding.

Go bindings for [mp3lame](https://sourceforge.net/p/lame/svn/HEAD/tree/) to provide mp3 encoding.

## compile mpg123

```bash
./configure --disable-components --enable-libmpg123 --enable-static --prefix=$(pwd)/bin
mkae
make install
```

## compile mp3lame

```bash
svn checkout https://svn.code.sf.net/p/lame/svn/trunk/lame lame-svn
cd lame-svn
./configure --disable-frontend --disable-decoder --disable-gtktest --disable-analyzer-hooks --prefix=$(pwd)/bin
make
make install
```
