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
	"os/exec"
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
	Formats = []string{"3gp", "mp4", "flv", "webm", "avi"}
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

// Download options
type Option struct {
	Resume bool // resume failed or cancelled download
	Rename bool // rename output file using video title
	Mp3    bool // extract audio using ffmpeg
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

	return *meta, nil
}

func (video *Video) Download(index int, filename string, option *Option) error {
	var (
		out    *os.File
		err    error
		offset int64
		length int64
	)

	if option.Resume {
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

	// Get video content length
	if resp, err := http.Head(url); err != nil {
		return fmt.Errorf("Head request failed: %s", err)
	} else {
		if size := resp.Header.Get("Content-Length"); len(size) == 0 {
			return errors.New("Content-Length header is missing")
		} else if length, err = strconv.ParseInt(size, 10, 64); err != nil {
			return fmt.Errorf("Invalid Content-Length: %s", err)
		}
		if length <= offset {
			fmt.Println("Video file is already downloaded.")
			return nil
		}
	}

	if length > 0 {
		go printProgress(out, offset, length)
	}

	// Not using range requests by default, because Youtube is throttling
	// download speed. Using a single GET request for max speed.
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Request failed: %s", err)
	}
	defer resp.Body.Close()

	if length, err = io.Copy(out, resp.Body); err != nil {
		return err
	}

	// Download stats
	duration := time.Now().Sub(start)
	speed := float64(length) / float64(duration/time.Second)
	if duration > time.Second {
		duration -= duration % time.Second
	} else {
		speed = float64(length)
	}

	if option.Rename {
		// Rename output file using video title
		wspace := regexp.MustCompile(`\W+`)
		fname := strings.Split(filename, ".")[0]
		ext := filepath.Ext(filename)
		title := wspace.ReplaceAllString(video.Title, "-")
		if len(title) > 64 {
			title = title[:64]
		}
		title = strings.TrimRight(strings.ToLower(title), "-")
		video.Filename = fmt.Sprintf("%s-%s%s", fname, title, ext)
		if err := os.Rename(filename, video.Filename); err != nil {
			fmt.Println("Failed to rename output file:", err)
		}
	}

	// Extract audio from downloaded video using ffmpeg
	if option.Mp3 {
		if err := out.Close(); err != nil {
			fmt.Println("Error:", err)
		}
		ffmpeg, err := exec.LookPath("ffmpeg")
		if err != nil {
			fmt.Println("ffmpeg not found")
		} else {
			fmt.Println("Extracting autio ..")
			fname := video.Filename
			mp3 := strings.TrimRight(fname, filepath.Ext(fname)) + ".mp3"
			cmd := exec.Command(ffmpeg, "-y", "-loglevel", "quiet", "-i", fname, "-vn", mp3)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("Failed to extract audio:", err)
			} else {
				fmt.Println()
				fmt.Println("Extracted audio:", mp3)
			}
		}
	}

	fmt.Printf("Download duration: %s\n", duration)
	fmt.Printf("Average speed: %s/s\n", abbr(int64(speed)))

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

// Measure download speed using output file offset
func printProgress(out *os.File, offset, length int64) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	start := time.Now()
	tail := offset

	var err error
	for now := range ticker.C {
		duration := now.Sub(start)
		duration -= duration % time.Second
		offset, err = out.Seek(0, os.SEEK_CUR)
		if err != nil {
			return
		}
		speed := offset - tail
		percent := int(100 * offset / length)
		progress := fmt.Sprintf(
			"%s\t %s/%s\t %d%%\t %s/s",
			duration, abbr(offset), abbr(length), percent, abbr(speed))
		fmt.Println(progress)
		tail = offset
		if tail >= length {
			break
		}
	}
}

// figure out the file extension from a codec string
func (v *Video) GetExtension(index int) string {
	for _, format := range Formats {
		if strings.Contains(v.Formats[index].Video_type, format) {
			return format
		}
	}

	return "avi"
}

// Returns video format index by Itag number, or nil if unknown
func (v *Video) IndexByItag(itag int) (int, *Format) {
	for i := range v.Formats {
		format := &v.Formats[i]
		if format.Itag == itag {
			return i, format
		}
	}
	return 0, nil
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
func parseMeta(video_id, query_string string) (*Video, error) {
	// parse the query string
	u, _ := url.Parse("?" + query_string)

	// parse url params
	query := u.Query()

	// no such video
	if query.Get("errorcode") != "" || query.Get("status") == "fail" {
		return nil, errors.New(query.Get("reason"))
	}

	// collate the necessary params
	video := &Video{
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
