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
