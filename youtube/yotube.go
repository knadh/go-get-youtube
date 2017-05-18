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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// _________________________________________________________________

const (
	// Youtube video meta source url
	URL_META = "http://www.youtube.com/get_video_info?&video_id="
)

const (
	KB float64 = 1 << (10 * (iota + 1))
	MB
	GB
)

var (
	// Video formats
	FORMATS = []string{"3gp", "mp4", "flv", "webm", "avi"}
)

// holds a video's information
type Video struct {
	Id, Title, Author, Keywords, Thumbnail_url string
	Avg_rating                                 float32
	View_count, Length_seconds                 int
	Formats                                    []Format
	Filename                                   string
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

func (video *Video) Download(index int, filename string, resume, rename bool) error {
	var (
		out    *os.File
		err    error
		offset int64
	)

	if resume {
		// Resume download from last known offset
		flags := os.O_WRONLY | os.O_CREATE
		out, err = os.OpenFile(filename, flags, 0644)
		if err != nil {
			return fmt.Errorf("Unable to open file %q: %s", filename, err)
		}
		offset, err = out.Seek(0, os.SEEK_END)
		if err != nil {
			return fmt.Errorf("Unable to seek file %q: %s", filename, err)
		}
		fmt.Printf("Resuming from offset %d (%s)\n", offset, abbr(offset))

	} else {
		// Start new download
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		out, err = os.OpenFile(filename, flags, 0644)
		if err != nil {
			return fmt.Errorf("Unable to write to file %q: %s", filename, err)
		}
	}
	defer out.Close()

	url := video.Formats[index].Url
	video.Filename = filename

	// Check if server accepts range request
	if resp, err := http.Head(url); err != nil {
		return fmt.Errorf("Head request failed: %s", err)

	} else if resp.Header.Get("Accept-Ranges") == "bytes" {
		// Download in chunks
		var length int64
		if size := resp.Header.Get("Content-Length"); len(size) == 0 {
			return errors.New("Content-Length header is missing")
		} else if length, err = strconv.ParseInt(size, 10, 64); err != nil {
			return fmt.Errorf("Invalid Content-Length: %s", err)
		}
		if length == offset {
			fmt.Println("Video file is already dowloaded.")
			return nil
		}
		if err := video.DownloadChunks(out, length, url, offset); err != nil {
			return err
		}

	} else {
		// No range support, download without progress print
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("Request failed: %s", err)
		}
		defer resp.Body.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return err
		}
	}

	if rename {
		// Rename output file using video title
		wspace := regexp.MustCompile(`\W+`)
		fname := strings.Split(filename, ".")[0]
		ext := filepath.Ext(filename)
		title := wspace.ReplaceAllString(video.Title, "-")
		if len(title) > 64 {
			title = title[:64]
		}
		title = strings.ToLower(title)
		video.Filename = fmt.Sprintf("%s-%s%s", fname, title, ext)
		if err := os.Rename(filename, video.Filename); err != nil {
			fmt.Println("Failed to rename output file:", err)
		}
	}

	return nil
}

func abbr(byteSize int64) string {
	size := float64(byteSize)
	switch {
	case size > GB:
		return fmt.Sprintf("%.1fGB", size/GB)
	case size > MB:
		return fmt.Sprintf("%.1fMB", size/MB)
	case size > KB:
		return fmt.Sprintf("%.1fKB", size/KB)
	}
	return fmt.Sprintf("%d", byteSize)
}

// Downloads video content in chunks and prints progress
func (video *Video) DownloadChunks(out *os.File, length int64,
	url string, offset int64) error {

	var (
		chunk  int64 = 1 << 20 // 1MB
		ticker       = time.NewTicker(time.Second)
	)

	defer ticker.Stop()

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
					fmt.Printf(
						"\nDownload failed after 10 attempts at offset %v: HTTP %d %s\n",
						offset, resp.StatusCode, http.StatusText(resp.StatusCode))
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

		if offset >= length {
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
func (v *Video) GetExtension(index int) string {
	for _, format := range FORMATS {
		if strings.Contains(v.Formats[index].Video_type, format) {
			return format
		}
	}

	return "avi"
}

// Returns video format index by Itag number, or -1 if unknown
func (v *Video) IndexByItag(itag int) int {
	for i, format := range v.Formats {
		if format.Itag == itag {
			return i
		}
	}
	return -1
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
