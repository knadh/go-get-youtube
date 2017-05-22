/**
	go-get-youtube v0.2

	A tiny Go library + client for download Youtube videos.
	The library is capable of fetching Youtube video metadata,
	in addition to downloading videos.

	Kailash Nadh, http://nadh.in
	27 February 2014

	License: GPL v2
**/

package main

import (
	"flag"
	"fmt"
	"os"

	youtube "github.com/knadh/go-get-youtube/youtube"
)

func intro() {
	txt :=
		`
Go Get Youtube
==============
A simple Youtube video downloader written in Go
- Kailash Nadh, github.com/knadh/go-get-youtube

ytdownload -id=VIDEO_ID
`

	fmt.Println(txt)
	flag.Usage()
}

func printVideoMeta(video youtube.Video) {
	txt :=
		`
	ID	:	%s
	Title	:	%s
	Author	:	%s
	Views	:	%d
	Rating	:	%f`

	fmt.Printf(txt, video.Id, video.Title, video.Author, video.View_count, video.Avg_rating)
	fmt.Println("\n\n\tFormats")

	for i := 0; i < len(video.Formats); i++ {
		fmt.Printf("\t%d\tItag %d: %s\t%s\n", i, video.Formats[i].Itag, video.Formats[i].Quality, video.Formats[i].Video_type)
	}
	fmt.Println()
	fmt.Println()
}

func getItag(max int) (i int) {
	for {
		fmt.Printf("Pick a format [0-%d]: ", max)
		if _, err := fmt.Scanf("%d", &i); err == nil {
			if i >= 0 && i <= max {
				return
			}
		}
		fmt.Println("Invalid entry:", i)
	}
}

func downloadVideo(video youtube.Video, index int, option *youtube.Option) error {

	ext := video.GetExtension(index)

	fmt.Printf("Downloading to '%s.%s'\n", video.Id, ext)

	filename := fmt.Sprintf("%s.%s", video.Id, ext)
	err := video.Download(index, filename, option)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Unable to download video content from Youtube.")
	} else {
		fmt.Println("Downloaded video:", video.Filename)
	}
	return err
}

func main() {
	// get the video id from the command line
	video_id := flag.String("id", "", "Youtube video ID to download")
	resume := flag.Bool("resume", false, "Resume cancelled download")
	itag := flag.Int("itag", 0, "Select video format by Itag number")
	rename := flag.Bool("rename", false, "Rename downloaded file using video title")
	mp3 := flag.Bool("mp3", false, "Extract MP3 audio using ffmpeg")
	flag.Parse()

	// no id supplied, show help text
	if *video_id == "" {
		intro()
		return
	}

	fmt.Println("Hold on ...")

	// feth the video metadata
	video, err := youtube.Get(*video_id)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	printVideoMeta(video)

	// get the format choice from the user
	var index int
	if *itag > 0 {
		idx, format := video.IndexByItag(*itag)
		if format == nil {
			fmt.Println("Unknown Itag number:", *itag)
			os.Exit(1)
		}
		index = idx
		fmt.Printf("Selected: %s %s\n", format.Video_type, format.Quality)
	} else {
		max := len(video.Formats) - 1
		index = getItag(max)
	}

	option := &youtube.Option{
		Resume: *resume,
		Rename: *rename,
		Mp3:    *mp3,
	}

	err = downloadVideo(video, index, option)
	if err != nil {
		os.Exit(1)
	}
}
