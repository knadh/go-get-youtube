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

	youtube "github.com/knadh/go-get-youtube/youtube"
)

func intro() {
	txt :=
	`

	Go Get Youtube
	==============
	A simple Youtube video downloader written in Go
	- Kailash Nadh, github.com/knadh/go-get-youtube

	Usage:
	ytdownload -id=VIDEO_ID
	`

	fmt.Println(txt)
}

func printVideoMeta(video youtube.Video) {
	txt :=
	`
	ID	:	%s
	Title	:	%s
	Author	:	%s
	Views	:	%d
	Rating	:	%f`

	fmt.Printf(txt, video.Id, video.Title, video.Author, video.View_count, video.Avg_rating);
	fmt.Println("\n\n\tFormats");

	for i:=0; i<len(video.Formats); i++ {
		fmt.Printf("\t%d\t%d-%s\t%s\n", i, video.Formats[i].Itag, video.Formats[i].Quality, video.Formats[i].Video_type)
	}
	fmt.Println("\n")
}

func getItag(max int) int {
	i := -1

	fmt.Printf("Pick a format [0-%d]: ", max)
	_, err := fmt.Scanf("%d", &i)

	if err != nil {
		return -1
	}

	return i
}

func downloadVideo(video youtube.Video, index int) {
	ext := video.GetExtension(index)

	fmt.Printf("Downloading to %s.%s ... This could take a while", video.Id, ext)
	
	video.Download(index, video.Id + "." + ext)
	
	fmt.Println("Done")
}

func main() {
	// get the video id from the command line
	video_id := flag.String("id", "", "Youtube video id")
	flag.Parse()

	// no id supplied, show help text
	if *video_id == "" {
		intro()
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
	i := -1; max := len(video.Formats) - 1
	for {
		if (i == -1) || (i < 0) || (i > max) {
			i = getItag(max)
		} else {
			downloadVideo(video, i)
			break
		}
	}	
}
