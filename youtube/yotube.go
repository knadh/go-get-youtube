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
	"encoding/json"
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
	"runtime"
	"strconv"
	"strings"
	"time"
)

// _________________________________________________________________

const (
	// Youtube video meta source url
	URL_META = "https://www.youtube.com/get_video_info?&video_id="
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

// Response Data
type playerResponse struct {
	ResponseContext struct {
		ServiceTrackingParams []struct {
			Service string `json:"service"`
			Params  []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"params"`
		} `json:"serviceTrackingParams"`
	} `json:"responseContext"`
	PlayabilityStatus struct {
		Status          string `json:"status"`
		PlayableInEmbed bool   `json:"playableInEmbed"`
	} `json:"playabilityStatus"`
	StreamingData struct {
		ExpiresInSeconds string `json:"expiresInSeconds"`
		Formats          []struct {
			Itag             int    `json:"itag"`
			URL              string `json:"url"`
			MimeType         string `json:"mimeType"`
			Bitrate          int    `json:"bitrate"`
			Width            int    `json:"width"`
			Height           int    `json:"height"`
			LastModified     string `json:"lastModified"`
			ContentLength    string `json:"contentLength"`
			Quality          string `json:"quality"`
			QualityLabel     string `json:"qualityLabel"`
			ProjectionType   string `json:"projectionType"`
			AverageBitrate   int    `json:"averageBitrate,omitempty"`
			AudioQuality     string `json:"audioQuality"`
			ApproxDurationMs string `json:"approxDurationMs,omitempty"`
			AudioSampleRate  string `json:"audioSampleRate,omitempty"`
			AudioChannels    int    `json:"audioChannels,omitempty"`
		} `json:"formats"`
		AdaptiveFormats []struct {
			Itag      int    `json:"itag"`
			URL       string `json:"url"`
			MimeType  string `json:"mimeType"`
			Bitrate   int    `json:"bitrate"`
			Width     int    `json:"width,omitempty"`
			Height    int    `json:"height,omitempty"`
			InitRange struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"initRange"`
			IndexRange struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"indexRange"`
			LastModified     string `json:"lastModified"`
			ContentLength    string `json:"contentLength"`
			Quality          string `json:"quality"`
			Fps              int    `json:"fps,omitempty"`
			QualityLabel     string `json:"qualityLabel,omitempty"`
			ProjectionType   string `json:"projectionType"`
			AverageBitrate   int    `json:"averageBitrate"`
			ApproxDurationMs string `json:"approxDurationMs"`
			HighReplication  bool   `json:"highReplication,omitempty"`
			AudioQuality     string `json:"audioQuality,omitempty"`
			AudioSampleRate  string `json:"audioSampleRate,omitempty"`
			AudioChannels    int    `json:"audioChannels,omitempty"`
		} `json:"adaptiveFormats"`
	} `json:"streamingData"`
	PlaybackTracking struct {
		VideostatsPlaybackURL struct {
			BaseURL string `json:"baseUrl"`
		} `json:"videostatsPlaybackUrl"`
		VideostatsDelayplayURL struct {
			BaseURL string `json:"baseUrl"`
		} `json:"videostatsDelayplayUrl"`
		VideostatsWatchtimeURL struct {
			BaseURL string `json:"baseUrl"`
		} `json:"videostatsWatchtimeUrl"`
		PtrackingURL struct {
			BaseURL string `json:"baseUrl"`
		} `json:"ptrackingUrl"`
		QoeURL struct {
			BaseURL string `json:"baseUrl"`
		} `json:"qoeUrl"`
		SetAwesomeURL struct {
			BaseURL                 string `json:"baseUrl"`
			ElapsedMediaTimeSeconds int    `json:"elapsedMediaTimeSeconds"`
		} `json:"setAwesomeUrl"`
		AtrURL struct {
			BaseURL                 string `json:"baseUrl"`
			ElapsedMediaTimeSeconds int    `json:"elapsedMediaTimeSeconds"`
		} `json:"atrUrl"`
	} `json:"playbackTracking"`
	Captions struct {
		PlayerCaptionsRenderer struct {
			BaseURL    string `json:"baseUrl"`
			Visibility string `json:"visibility"`
		} `json:"playerCaptionsRenderer"`
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []struct {
				BaseURL string `json:"baseUrl"`
				Name    struct {
					SimpleText string `json:"simpleText"`
				} `json:"name"`
				VssID          string `json:"vssId"`
				LanguageCode   string `json:"languageCode"`
				Kind           string `json:"kind"`
				IsTranslatable bool   `json:"isTranslatable"`
			} `json:"captionTracks"`
			AudioTracks []struct {
				CaptionTrackIndices []int  `json:"captionTrackIndices"`
				Visibility          string `json:"visibility"`
			} `json:"audioTracks"`
			TranslationLanguages []struct {
				LanguageCode string `json:"languageCode"`
				LanguageName struct {
					SimpleText string `json:"simpleText"`
				} `json:"languageName"`
			} `json:"translationLanguages"`
			DefaultAudioTrackIndex int `json:"defaultAudioTrackIndex"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
	VideoDetails struct {
		VideoID          string   `json:"videoId"`
		Title            string   `json:"title"`
		LengthSeconds    string   `json:"lengthSeconds"`
		Keywords         []string `json:"keywords"`
		ChannelID        string   `json:"channelId"`
		IsOwnerViewing   bool     `json:"isOwnerViewing"`
		ShortDescription string   `json:"shortDescription"`
		IsCrawlable      bool     `json:"isCrawlable"`
		Thumbnail        struct {
			Thumbnails []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"thumbnails"`
		} `json:"thumbnail"`
		UseCipher         bool    `json:"useCipher"`
		AverageRating     float64 `json:"averageRating"`
		AllowRatings      bool    `json:"allowRatings"`
		ViewCount         string  `json:"viewCount"`
		Author            string  `json:"author"`
		IsPrivate         bool    `json:"isPrivate"`
		IsUnpluggedCorpus bool    `json:"isUnpluggedCorpus"`
		IsLiveContent     bool    `json:"isLiveContent"`
	} `json:"videoDetails"`
	Annotations []struct {
		PlayerAnnotationsUrlsRenderer struct {
			InvideoURL         string `json:"invideoUrl"`
			LoadPolicy         string `json:"loadPolicy"`
			AllowInPlaceSwitch bool   `json:"allowInPlaceSwitch"`
		} `json:"playerAnnotationsUrlsRenderer"`
	} `json:"annotations"`
	PlayerConfig struct {
		AudioConfig struct {
			LoudnessDb           float64 `json:"loudnessDb"`
			PerceptualLoudnessDb float64 `json:"perceptualLoudnessDb"`
		} `json:"audioConfig"`
		StreamSelectionConfig struct {
			MaxBitrate string `json:"maxBitrate"`
		} `json:"streamSelectionConfig"`
		MediaCommonConfig struct {
			DynamicReadaheadConfig struct {
				MaxReadAheadMediaTimeMs int `json:"maxReadAheadMediaTimeMs"`
				MinReadAheadMediaTimeMs int `json:"minReadAheadMediaTimeMs"`
				ReadAheadGrowthRateMs   int `json:"readAheadGrowthRateMs"`
			} `json:"dynamicReadaheadConfig"`
		} `json:"mediaCommonConfig"`
	} `json:"playerConfig"`
	Storyboards struct {
		PlayerStoryboardSpecRenderer struct {
			Spec string `json:"spec"`
		} `json:"playerStoryboardSpecRenderer"`
	} `json:"storyboards"`
	Microformat struct {
		PlayerMicroformatRenderer struct {
			Thumbnail struct {
				Thumbnails []struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"thumbnails"`
			} `json:"thumbnail"`
			Embed struct {
				IframeURL      string `json:"iframeUrl"`
				FlashURL       string `json:"flashUrl"`
				Width          int    `json:"width"`
				Height         int    `json:"height"`
				FlashSecureURL string `json:"flashSecureUrl"`
			} `json:"embed"`
			Title struct {
				SimpleText string `json:"simpleText"`
			} `json:"title"`
			Description struct {
				SimpleText string `json:"simpleText"`
			} `json:"description"`
			LengthSeconds        string   `json:"lengthSeconds"`
			OwnerProfileURL      string   `json:"ownerProfileUrl"`
			OwnerGplusProfileURL string   `json:"ownerGplusProfileUrl"`
			ExternalChannelID    string   `json:"externalChannelId"`
			AvailableCountries   []string `json:"availableCountries"`
			IsUnlisted           bool     `json:"isUnlisted"`
			HasYpcMetadata       bool     `json:"hasYpcMetadata"`
			ViewCount            string   `json:"viewCount"`
			Category             string   `json:"category"`
			PublishDate          string   `json:"publishDate"`
			OwnerChannelName     string   `json:"ownerChannelName"`
			UploadDate           string   `json:"uploadDate"`
		} `json:"playerMicroformatRenderer"`
	} `json:"microformat"`
	TrackingParams string `json:"trackingParams"`
	Attestation    struct {
		PlayerAttestationRenderer struct {
			Challenge    string `json:"challenge"`
			BotguardData struct {
				Program        string `json:"program"`
				InterpreterURL string `json:"interpreterUrl"`
			} `json:"botguardData"`
		} `json:"playerAttestationRenderer"`
	} `json:"attestation"`
	Messages []struct {
		MealbarPromoRenderer struct {
			MessageTexts []struct {
				Runs []struct {
					Text string `json:"text"`
				} `json:"runs"`
			} `json:"messageTexts"`
			ActionButton struct {
				ButtonRenderer struct {
					Style string `json:"style"`
					Size  string `json:"size"`
					Text  struct {
						Runs []struct {
							Text string `json:"text"`
						} `json:"runs"`
					} `json:"text"`
					NavigationEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						URLEndpoint         struct {
							URL    string `json:"url"`
							Target string `json:"target"`
						} `json:"urlEndpoint"`
					} `json:"navigationEndpoint"`
					TrackingParams string `json:"trackingParams"`
				} `json:"buttonRenderer"`
			} `json:"actionButton"`
			DismissButton struct {
				ButtonRenderer struct {
					Style string `json:"style"`
					Size  string `json:"size"`
					Text  struct {
						Runs []struct {
							Text string `json:"text"`
						} `json:"runs"`
					} `json:"text"`
					ServiceEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						FeedbackEndpoint    struct {
							FeedbackToken string `json:"feedbackToken"`
							UIActions     struct {
								HideEnclosingContainer bool `json:"hideEnclosingContainer"`
							} `json:"uiActions"`
						} `json:"feedbackEndpoint"`
					} `json:"serviceEndpoint"`
					TrackingParams string `json:"trackingParams"`
				} `json:"buttonRenderer"`
			} `json:"dismissButton"`
			TriggerCondition    string `json:"triggerCondition"`
			Style               string `json:"style"`
			TrackingParams      string `json:"trackingParams"`
			ImpressionEndpoints []struct {
				ClickTrackingParams string `json:"clickTrackingParams"`
				FeedbackEndpoint    struct {
					FeedbackToken string `json:"feedbackToken"`
					UIActions     struct {
						HideEnclosingContainer bool `json:"hideEnclosingContainer"`
					} `json:"uiActions"`
				} `json:"feedbackEndpoint"`
			} `json:"impressionEndpoints"`
			IsVisible    bool `json:"isVisible"`
			MessageTitle struct {
				Runs []struct {
					Text string `json:"text"`
				} `json:"runs"`
			} `json:"messageTitle"`
		} `json:"mealbarPromoRenderer"`
	} `json:"messages"`
	AdSafetyReason struct {
		ApmUserPreference struct {
		} `json:"apmUserPreference"`
		IsEmbed bool `json:"isEmbed"`
	} `json:"adSafetyReason"`
}

// extract the video Id from an URL
func extractId(input string) (string, error) {
	u, err := url.Parse(input)

	if err != nil {
		return "", err
	}

	queries := u.Query()
	for key, value := range queries {
		if key == "v" {
			return value[0], nil
		}
	}
	return "", fmt.Errorf("No video ID detectable")
}

// given a video id, get it's information from youtube
func Get(video_id string) (Video, error) {
	if strings.Contains(video_id, "youtube.com/watch?") {
		video_id, _ = extractId(video_id)
	}
	
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
		if resp.StatusCode == 403 {
			return errors.New("Head request failed: Video is 403 forbidden")
		}

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
			fmt.Println("Extracting audio ..")
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
	var clear string
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
			"%s%s\t %s/%s\t %d%%\t %s/s",
			clear, duration, abbr(offset), abbr(length), percent, abbr(speed))
		fmt.Println(progress)
		tail = offset
		if tail >= length {
			break
		}
		if clear == "" {
			switch runtime.GOOS {
				case "darwin":
					clear = "\033[A\033[2K\r"
				case "linux":
					clear = "\033[A\033[2K\r"
				case "windows":
			}
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

	var player_response playerResponse
	json.Unmarshal([]byte(query.Get("player_response")), &player_response)

	// collate the necessary params
	video := &Video{
		Id:            video_id,
		Title:         player_response.VideoDetails.Title,
		Author:        player_response.VideoDetails.Author,
		Keywords:      fmt.Sprint(player_response.VideoDetails.Keywords),
		Thumbnail_url: player_response.VideoDetails.Thumbnail.Thumbnails[0].URL,
	}

	v, _ := strconv.Atoi(player_response.VideoDetails.ViewCount)
	video.View_count = v
	
	video.Avg_rating = float32(player_response.VideoDetails.AverageRating)

	l, _ := strconv.Atoi(player_response.VideoDetails.LengthSeconds)
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
			Url:        fquery.Get("url"),
		})
	}

	return video, nil
}
