# go-get-youtube v 0.2
A tiny Go library + client for download Youtube videos. The library is capable of fetching Youtube video metadata, in addition to downloading videos.

Kailash Nadh, http://nadh.in --
27 February 2014

License: GPL v2

# Client
Once you have compiled or [downloaded](https://github.com/knadh/go-get-youtube/releases) the binary, simply run the following on your terminal:

`ytdownload -id=youtube_video_id`


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
```

### youtube.Download(format_index, output_file)
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
	video.download(0, "video.mp4")
}

```

