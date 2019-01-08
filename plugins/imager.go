package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/alexandre-normand/slackscot/v2"
	"github.com/nlopes/slack"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var reUrl = regexp.MustCompile(`(?i).(png|jpe?g|gif)$`)

type giphyImageData struct {
	URL      string `json:"url,omitempty"`
	Width    string `json:"width,omitempty"`
	Height   string `json:"height,omitempty"`
	Size     string `json:"size,omitempty"`
	Frames   string `json:"frames,omitempty"`
	Mp4      string `json:"mp4,omitempty"`
	Mp4Size  string `json:"mp4_size,omitempty"`
	Webp     string `json:"webp,omitempty"`
	WebpSize string `json:"webp_size,omitempty"`
}

type giphyResponse struct {
	Data       []giphyGif      `json:"data"`
	Status     giphyStatus     `json:"meta"`
	Pagination giphyPagination `json:"pagination"`
}

type giphyPagination struct {
	Total  int `json:"total_count"`
	Count  int `json:"count"`
	Offset int `json:"offset"`
}

type giphyStatus struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

type giphyGif struct {
	Type               string
	Id                 string
	URL                string
	Tags               []string
	BitlyGifURL        string `json:"bitly_gif_url"`
	BitlyFullscreenURL string `json:"bitly_fullscreen_url"`
	BitlyTiledURL      string `json:"bitly_tiled_url"`
	Images             struct {
		Original               giphyImageData
		FixedHeight            giphyImageData `json:"fixed_height"`
		FixedHeightStill       giphyImageData `json:"fixed_height_still"`
		FixedHeightDownsampled giphyImageData `json:"fixed_height_downsampled"`
		FixedWidth             giphyImageData `json:"fixed_width"`
		FixedwidthStill        giphyImageData `json:"fixed_width_still"`
		FixedwidthDownsampled  giphyImageData `json:"fixed_width_downsampled"`
	}
}

// Imager holds the plugin data for the imager plugin
type Imager struct {
	slackscot.Plugin
}

const (
	// ImagerPluginName holds identifying name for the imager plugin
	ImagerPluginName = "imager"
)

// NewImager creates a new instance of the plugin
func NewImager() (imager *Imager) {
	imageRegex := regexp.MustCompile("(?i)(image|img) (.*)")
	animateRegex := regexp.MustCompile("(?i)(animate) (.*)")
	moosificateRegex := regexp.MustCompile("(?i)(moosificate) (.*)")
	urlRegex := regexp.MustCompile("(?i).*https?://.*")

	commands := []slackscot.ActionDefinition{
		{
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "image")
			},
			Usage:       "image <search expression>",
			Description: "Queries Google Images for _search expression_ and returns random result",
			Answer: func(message *slack.Msg) string {
				return processQueryAndSearch(message.Text, imageRegex, false)
			},
		}, {
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "animate")
			},
			Usage:       "animate <search expression>",
			Description: "The sames as `image` except requests an animated gif matching _search expression_",
			Answer: func(message *slack.Msg) string {
				searchExpression := animateRegex.FindAllStringSubmatch(message.Text, -1)[0]

				return searchGiphy(searchExpression[2], "dc6zaTOxFJmzC")
			},
		}, {
			Match: func(t string, m *slack.Msg) bool {
				return strings.HasPrefix(t, "moosificate")
			},
			Usage:       "moosificate <search expression or image url>",
			Description: "Moosificates an image from either an image search for the _search expression_ or a direct image URL",
			Answer: func(message *slack.Msg) string {
				match := moosificateRegex.FindAllStringSubmatch(message.Text, -1)[0]
				toMoosificate := match[2]

				if !urlRegex.MatchString(toMoosificate) {
					toMoosificate = imageSearch(toMoosificate, false, false, 1)
				} else {
					toMoosificate = toMoosificate[1 : len(toMoosificate)-1]
				}

				return fmt.Sprintf("https://moosificator.herokuapp.com/api/moose?image=%s", url.QueryEscape(toMoosificate))
			},
		}}

	return &Imager{Plugin: slackscot.Plugin{Name: ImagerPluginName, Commands: commands, HearActions: nil}}
}

func processQueryAndSearch(message string, regex *regexp.Regexp, animated bool) string {
	searchExpression := regex.FindStringSubmatch(message)

	if len(searchExpression) > 0 {
		return imageSearch(searchExpression[2], animated, false, 1)
	}
	return ""
}

func imageSearch(expr string, animated bool, faces bool, count int) string {
	googleURL, err := url.Parse("http://ajax.googleapis.com/ajax/services/search/images")
	if err != nil {
		log.Printf("Error parsing Google Images URL: %s", err)
		return ""
	}

	q := googleURL.Query()
	q.Set("v", "1.0")
	q.Set("rsz", "8")
	q.Set("safe", "active")
	q.Set("q", expr)

	if animated {
		q.Set("as_filetype", "gif")
	}

	if faces {
		q.Set("imgtype", "face")
	}

	googleURL.RawQuery = q.Encode()
	resp, err := http.Get(googleURL.String())

	if err != nil {
		log.Printf("Error calling url '%s' : %s ", googleURL, err)
		return "Sorry I had a problem finding that image from Google"
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading results from HTTP Request '%s': %s", googleURL, err)
		return "Sorry I had a problem finding that image from Google"
	}

	var results map[string]interface{}
	if err = json.Unmarshal(body, &results); err != nil {
		log.Printf("%T\n%s\n%#v\n", err, err, err)
		switch v := err.(type) {
		case *json.SyntaxError:
			log.Println(string(body[v.Offset-40 : v.Offset]))
		}
		log.Printf("%s", body)
		return "Sorry I had a problem finding that image from Google"
	}

	if responseData, someResult := results["responseData"]; someResult && responseData != nil {
		imageList, ok := responseData.(map[string]interface{})["results"]

		var selectedImages []string
		var images []interface{}
		if ok {
			images = imageList.([]interface{})

			for i, image := range images {
				if i >= count {
					break
				}

				element := image.(map[string]interface{})
				imageUrl := element["unescapedUrl"].(string)
				log.Printf("Result image : %v", imageUrl)

				if !reUrl.MatchString(imageUrl) {
					imageUrl = imageUrl + ".png"
				}

				selectedImages = append(selectedImages, imageUrl)
			}
		}

		return strings.Join(selectedImages, "\n")
	}

	return fmt.Sprintf("https://media.giphy.com/media/9J7tdYltWyXIY/giphy.gif?index=%d", time.Now().Unix())

}

func searchGiphy(q string, key string) string {
	url := fmt.Sprintf("http://api.giphy.com/v1/gifs/search?q=%s&api_key=%s", url.QueryEscape(q), key)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return "Arggggg..."
	}

	var giphyResp giphyResponse
	parseErr := json.Unmarshal(content, &giphyResp)
	if parseErr != nil {
		log.Print(parseErr)
		return "Arrggggggggg...."
	}

	msg := "No results. Look away, I'm hideous."
	if len(giphyResp.Data) > 0 {
		msg = fmt.Sprintf("%s", giphyResp.Data[rand.Intn(len(giphyResp.Data))].Images.Original.URL)
	}

	return msg
}
