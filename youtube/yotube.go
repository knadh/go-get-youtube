/**
	go-get-youtube v0.2

	A tiny Go library that can fetch Youtube video metadata
	and download videos.

	Kailash Nadh, http://nadh.in
	27 February 2014

	License: GPL v2
**/

package youtube

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// _________________________________________________________________

// Youtube video meta source url
const URL_META = "http://www.youtube.com/get_video_info?&video_id="

var FORMATS []string = []string{"3gp", "mp4", "flv", "webm", "avi"}

// holds a video's information
type Video struct {
	Id, Title, Author, Keywords, Thumbnail_url string
	Avg_rating                                 float32
	View_count, Length_seconds                 int
	Formats                                    []Format
}

type Format struct {
	Itag                     int
	Video_type, Quality, Url string
}

// _________________________________________________________________
// given a video id, get it's information from youtube
func Get(video_id string) (Video, error) {
	// fetch video meta from youtube
	query_string, err := fetchMeta(video_id)
	if err != nil {
		return Video{}, err
	}

	meta, err := parseMeta(video_id, query_string)

	if err != nil {
		return Video{}, err
	}

	return meta, nil
}

func (video *Video) Download(index int, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Unable to write to file %q: %s", filename, err)
	}
	defer out.Close()

	url := video.Formats[index].Url

	// Check if server accepts range request
	if resp, err := http.Head(url); err != nil {
		return fmt.Errorf("Head request failed: %s", err)
	} else if resp.Header.Get("Accept-Ranges") == "bytes" {
		var length int64
		if size := resp.Header.Get("Content-Length"); len(size) == 0 {
			return errors.New("Content-Length header is missing")
		} else if length, err = strconv.ParseInt(size, 10, 64); err != nil {
			return fmt.Errorf("Invalid Content-Length: %s", err)
		}
		return video.DownloadChunks(out, length, url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Request failed: %s", err)
	}
	defer resp.Body.Close()

	if err != nil {
		return errors.New("Unable to download video content from Yotutube")
	}

	io.Copy(out, resp.Body)

	return nil
}

// Downloads video content in chunks and prints progress
func (video *Video) DownloadChunks(out *os.File, length int64, url string) error {

	var (
		offset int64
		chunk  int64 = 10 * 1 << 20 // 10MB
		ticker       = time.NewTicker(time.Second)
	)

	defer ticker.Stop()

	const (
		KB float64 = 1 << (10 * (iota + 1))
		MB
		GB
	)

	abbr := func(byteSize int64) string {
		size := float64(byteSize)
		switch {
		case size > GB:
			return fmt.Sprintf("%0.1fGB", size/GB)
		case size > MB:
			return fmt.Sprintf("%0.1fMB", size/MB)
		case size > KB:
			return fmt.Sprintf("%0.1fKB", size/KB)
		}
		return fmt.Sprintf("%.0f", size)
	}

	printProgress := func() {
		start := time.Now()
		tail := offset
		for now := range ticker.C {
			duration := now.Sub(start)
			duration -= duration % time.Second
			speed := offset - tail
			percent := int(100 * offset / length)
			progress := fmt.Sprintf(
				"%s\t %s/%s\t %d%%\t %s/s",
				duration, abbr(offset), abbr(length), percent, abbr(speed))
			fmt.Println(progress)
			tail = offset
			if tail == length {
				break
			}
		}
	}

	if length > 0 {
		go printProgress()
	}

	for {
		rangeHeader := fmt.Sprintf("bytes=%d-%d", offset, offset+chunk)
		if length < chunk {
			rangeHeader = fmt.Sprintf("bytes=%d-", offset)
		}
		//fmt.Println(length, rangeHeader)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("Invalid request: %s", err)
		}
		req.Header.Set("Range", rangeHeader)

		for n := 0; ; n++ {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("Request failed: %s", err)
			}

			//fmt.Println("status", resp.StatusCode)
			if resp.StatusCode != http.StatusPartialContent {
				if n == 10 {
					fmt.Println(
						"\nDownload failed after 10 attempts at offset %d: HTTP %d",
						offset, resp.StatusCode)
					defer resp.Body.Close()
					return errors.New("Unable to download video content from Yotutube")
				}
				fmt.Println(http.StatusText(resp.StatusCode), "..")
				time.Sleep(time.Second)
				resp.Body.Close()
				continue
			}

			offset += resp.ContentLength
			if _, err := io.Copy(out, resp.Body); err != nil {
				return err
			}
			resp.Body.Close()
			break
		}

		if offset == length {
			time.Sleep(time.Second)
			fmt.Println()
			break
		}
		if length-offset < chunk {
			chunk = length - offset
		}
	}

	return nil
}

// figure out the file extension from a codec string
func (video *Video) GetExtension(index int) string {
	for i := 0; i < len(FORMATS); i++ {
		if strings.Contains(video.Formats[index].Video_type, FORMATS[i]) {
			return FORMATS[i]
		}
	}

	return "avi"
}

// _________________________________________________________________
// fetch video meta from http
func fetchMeta(video_id string) (string, error) {
	resp, err := http.Get(URL_META + video_id)

	// fetch the meta information from http
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	query_string, _ := ioutil.ReadAll(resp.Body)

	return string(query_string), nil
}

// parse youtube video metadata and return a Video object
func parseMeta(video_id, query_string string) (Video, error) {
	// parse the query string
	u, _ := url.Parse("?" + query_string)

	// parse url params
	query := u.Query()

	// no such video
	if query.Get("errorcode") != "" || query.Get("status") == "fail" {
		return Video{}, errors.New(query.Get("reason"))
	}

	// collate the necessary params
	video := Video{
		Id:            video_id,
		Title:         query.Get("title"),
		Author:        query.Get("author"),
		Keywords:      query.Get("keywords"),
		Thumbnail_url: query.Get("thumbnail_url"),
	}

	v, _ := strconv.Atoi(query.Get("view_count"))
	video.View_count = v

	r, _ := strconv.ParseFloat(query.Get("avg_rating"), 32)
	video.Avg_rating = float32(r)

	l, _ := strconv.Atoi(query.Get("length_seconds"))
	video.Length_seconds = l

	// further decode the format data
	format_params := strings.Split(query.Get("url_encoded_fmt_stream_map"), ",")

	// every video has multiple format choices. collate the list.
	for _, f := range format_params {
		furl, _ := url.Parse("?" + f)
		fquery := furl.Query()

		itag, _ := strconv.Atoi(fquery.Get("itag"))

		video.Formats = append(video.Formats, Format{
			Itag:       itag,
			Video_type: fquery.Get("type"),
			Quality:    fquery.Get("quality"),
			Url:        fquery.Get("url") + "&signature=" + fquery.Get("sig"),
		})
	}

	return video, nil
}
