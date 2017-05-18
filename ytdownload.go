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
		fmt.Printf("\t%d\t%d-%s\t%s\n", i, video.Formats[i].Itag, video.Formats[i].Quality, video.Formats[i].Video_type)
	}
	fmt.Println("\n")
}

func getItag(max int) int {
	var i int
	for {
		fmt.Printf("Pick a format [0-%d]: ", max)
		if _, err := fmt.Scanf("%d", &i); err == nil {
			return i
		}
		fmt.Println("Invalid entry")
	}
	return i
}

func downloadVideo(video youtube.Video, index int, resume, rename bool) error {

	ext := video.GetExtension(index)

	fmt.Printf("Downloading to %s.%s ... This could take a while\n", video.Id, ext)

	filename := fmt.Sprintf("%s.%s", video.Id, ext)
	err := video.Download(index, filename, resume, rename)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Unable to download video content from Youtube.")
	} else {
		fmt.Println("Downloaded", video.Filename)
	}
	return err
}

func main() {
	// get the video id from the command line
	video_id := flag.String("id", "", "Youtube video ID to download")
	resume := flag.Bool("resume", false, "Resume failed download")
	itag := flag.Int("itag", 0, "Select video format by Itag number")
	rename := flag.Bool("rename", false, "Rename downloaded file using video title")
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
		index = video.IndexByItag(*itag)
		if index == -1 {
			fmt.Println("Unknown Itag number:", *itag)
			os.Exit(1)
		}
	} else {
		max := len(video.Formats) - 1
		index = getItag(max)
	}

	err = downloadVideo(video, index, *resume, *rename)
	if err != nil {
		os.Exit(1)
	}
}
