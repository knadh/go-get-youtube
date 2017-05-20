# go-get-youtube v 0.2
A tiny Go library + client (command line Youtube video downloader) for downloading Youtube videos. The library is capable of fetching Youtube video metadata, in addition to downloading videos. If ffmpeg is available, client can extract MP3 audio from downloaded video files.

Kailash Nadh, http://nadh.in --
27 February 2014

License: GPL v2

# Client
Once you have compiled or [downloaded](https://github.com/knadh/go-get-youtube/releases) the binary, simply run the following on your terminal:

`ytdownload -id=youtube_video_id`

## Building
```
$ export GOPATH=$PWD/go-get-youtube
$ go get github.com/knadh/go-get-youtube
$ cd go-get-youtube/bin
$ ./go-get-youtube -id=cN_DpYBzKso -itag 18 -rename -mp3

Extracted audio: cN_DpYBzKso-rob-pike-concurrency-is-not-parallelis.mp3
Download duration: 5s
Average speed: 16.1MB/s
Downloaded video: cN_DpYBzKso-rob-pike-concurrency-is-not-parallelism.mp4
```
# Library

## Methods

### youtube.Get(youtube_video_id)
`video, err = youtube.Get(youtube_video_id)`

Initializes a `Video` object by fetching its metdata from Youtube. `Video` is a struct with the following structure

```go
{
	Id, Title, Author, Keywords, Thumbnail_url string
	Avg_rating float32
	View_count,	Length_seconds int
	Formats []Format
}
```

`Video.Formats` is an array of the `Format` struct, which looks like this:

```
type Format struct {
	Itag int
	Video_type, Quality, Url string
}

type Option struct {
	Resume bool // resume failed or cancelled download
	Rename bool // rename output file using video title
	Mp3    bool // extract audio using ffmpeg
}
```

### youtube.Download(format_index, output_file, option)
`format_index` is the index of the format listed in the `Video.Formats` array. Youtube offers a number of video formats (mp4, webm, 3gp etc.)

### youtube.GetExtension(format_index)
Guesses the file extension (avi, 3gp, mp4, webm) based on the format chosen

## Example
```go
import (
	youtube "github.com/knadh/go-get-youtube/youtube"
)

func main() {
	// get the video object (with metdata)
	video, err := youtube.Get("FTl0tl9BGdc")

	// download the video and write to file
	option := &youtube.Option{
		Rename: true,  // rename file using video title
		Resume: true,  // resume cancelled download
		Mp3:    true,  // extract audio to MP3
	}
	video.Download(0, "video.mp4", option)
}
```


## Contributors

Batbold Dashzeveg
