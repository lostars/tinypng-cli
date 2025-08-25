# A TinyPNG CLI Developed by Go

This CLI using TinyPNG official API, you can get the API key [here](https://tinify.com/developers) before you start.

Only packaging to linux_amd64 windows_amd64 darwin_arm64 currently, create a issue if u need other platforms.

And create a issue if u encounter any bug.

[Official Docs](https://tinypng.com/developers/reference)

## Installation

Download from release or install my own usage brew tap:
```
brew tap lostars/homebrew-mybrew
brew install lostars/mybrew/tinypng-cli
```

Or you can download the source code and compile:
```
git clone https://github.com/lostars/tinypng-cli.git
cd tinypng-cli
make build
```

## Usage

You can get details by flag `-h` or `--help`

### compress

`compress` command supports single local file or single local directory or a image url.

`--save-to` only supports `local` by default currently.

Compressed file will be renamed with a `-compressed` suffix:
```
a.jpg -> a-compressed.jpg
```

`--output` is the compressed file local save path, make sure it exists.
Compressed file will be created near by original file if output path is not set
and web url compressed file will be saved at current location where the command executed.

`--max-upload` controls the max upload parallelism when compress a directory, u can set `--recursive` to list files recursively.
Take care of you upload bandwidth when you change `--max-upload` to a bigger number.

`--extensions` provides a image file filter, default is `png,jpg,jpeg,webp`

### web-compress

Same as `compress`, but without a API key. 

And it doesn't support advanced features like resize.
Only support basic compress: local file or local directory compress.