package plugins

import (
	"fmt"
	"github.com/alexandre-normand/figlet4go"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/actions"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugin"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	figletFontURLKey = "figletFontUrl" // Optional, string (url) to a figlet font. Default font is used if not set. Fonts can be found on http://www.figlet.org/fontdb.cgi and url should be for the raw .flf file like http://www.figlet.org/fonts/banner.flf
)

const (
	// EmojiBannerPluginName holds identifying name for the emoji banner plugin
	EmojiBannerPluginName = "emojiBanner"
	bannerMaxWordLength   = 4
)

// EmojiBannerMaker holds the plugin data for the emoji banner maker plugin
type EmojiBannerMaker struct {
	*slackscot.Plugin
	tempDirFontPath string
}

// NewEmojiBannerMaker creates a new instance of the plugin. Note that since it creates a temporary
// directory to store fonts, the caller should make sure to defer Close on shutdown
func NewEmojiBannerMaker(c *config.PluginConfig) (toClose io.Closer, emojiBannerPlugin *slackscot.Plugin, err error) {
	emojiBannerRegex := regexp.MustCompile("(?i)(emoji banner) (.*)")

	options := figlet4go.NewRenderOptions()
	renderer := figlet4go.NewAsciiRender()

	tempDirFontPath, err := ioutil.TempDir("", EmojiBannerPluginName)
	if err != nil {
		os.RemoveAll(tempDirFontPath)
		return nil, nil, err
	}

	// Download all fonts and write them in the fontPath
	fontURL := c.GetString(figletFontURLKey)
	if fontURL != "" {
		fontName, err := downloadFontToDir(fontURL, tempDirFontPath)
		if err != nil {
			os.RemoveAll(tempDirFontPath)
			return nil, nil, err
		}

		err = renderer.LoadFont(tempDirFontPath)
		if err != nil {
			os.RemoveAll(tempDirFontPath)
			return nil, nil, fmt.Errorf("[%s] Can't load fonts from [%s]: %v", EmojiBannerPluginName, tempDirFontPath, err)
		}

		options.FontName = fontName
	}

	ebm := new(EmojiBannerMaker)
	ebm.Plugin = plugin.New(EmojiBannerPluginName).
		WithCommand(actions.NewCommand().
			WithMatcher(func(m *slackscot.IncomingMessage) bool {
				return strings.HasPrefix(m.NormalizedText, "emoji banner")
			}).
			WithUsage("emoji banner <word of 5 characters or less> <emoji>").
			WithDescription("Renders a single-word banner with the provided emoji").
			WithAnswerer(func(m *slackscot.IncomingMessage) *slackscot.Answer {
				return validateAndRenderEmoji(m.NormalizedText, emojiBannerRegex, renderer, options)
			}).
			Build()).
		Build()
	ebm.tempDirFontPath = tempDirFontPath

	return ebm, ebm.Plugin, nil
}

// Close cleans up resources (temp font directory) used by the plugin
func (e *EmojiBannerMaker) Close() (err error) {
	return os.RemoveAll(e.tempDirFontPath)
}

func downloadFontToDir(fontURL string, fontPath string) (fontName string, err error) {
	url, b, err := downloadURL(fontURL)
	if err != nil {
		return "", errors.Wrapf(err, "Error reading data from font url [%s]: %v", fontURL, err)
	}

	filename := path.Base(url.EscapedPath())

	fullpath := filepath.Join(fontPath, filename)
	err = ioutil.WriteFile(fullpath, b, 0644)
	if err != nil {
		return "", errors.Wrapf(err, "Error saving file [%s] from font url [%s]", fullpath, fontURL)
	}

	return strings.TrimSuffix(filename, ".flf"), nil
}

func downloadURL(fontURL string) (parsedURL *url.URL, content []byte, err error) {
	url, err := url.Parse(fontURL)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Invalid font url [%s]", fontURL)
	}

	resp, err := http.Get(fontURL)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Error loading font url [%s]", fontURL)
	}

	b, err := ioutil.ReadAll(resp.Body)

	return url, b, err
}

func validateAndRenderEmoji(message string, regex *regexp.Regexp, renderer *figlet4go.AsciiRender, options *figlet4go.RenderOptions) *slackscot.Answer {
	commandParameters := regex.FindStringSubmatch(message)

	if len(commandParameters) > 0 {
		parameters := strings.Split(commandParameters[2], " ")

		if len(parameters) == 2 {
			word := parameters[0]
			emoji := parameters[1]

			if len(word) <= bannerMaxWordLength {
				return renderBanner(word, emoji, renderer, options)
			}

			return &slackscot.Answer{Text: fmt.Sprintf("`Wrong usage` (word *longer* than `%d` characters): emoji banner `<word of 5 characters or less>` `<emoji>`", bannerMaxWordLength)}
		}
	}

	return &slackscot.Answer{Text: fmt.Sprintf("`Wrong usage`: emoji banner `<word of %d characters or less>` `<emoji>`", bannerMaxWordLength)}
}

func renderBanner(word, emoji string, renderer *figlet4go.AsciiRender, options *figlet4go.RenderOptions) *slackscot.Answer {
	rendered, err := renderer.RenderOpts(word, options)
	if err != nil {
		return &slackscot.Answer{Text: fmt.Sprintf("Error generating: %v", err)}
	}

	var result strings.Builder
	result.WriteString("\r\n")
	for _, character := range rendered {
		if unicode.IsPrint(character) && character != ' ' {
			result.WriteString(emoji)
		} else if character == ' ' {
			result.WriteString("⬜️")
		} else {
			result.WriteString(string(character))
		}
	}

	return &slackscot.Answer{Text: result.String()}
}
