package plugins

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/getwe/figlet4go"
	"github.com/mitchellh/go-homedir"
	"github.com/nlopes/slack"
	"log"
	"regexp"
	"strings"
	"unicode"
)

const (
	fontPathKey           = "fontPath"
	fontNameKey           = "fontName"
	emojiBannerPluginName = "emojiBanner"
)

// EmojiBannerMaker holds the plugin data for the emoji banner maker plugin
type EmojiBannerMaker struct {
	slackscot.Plugin
}

// NewEmojiBannerMaker creates a new instance of the plugin
func NewEmojiBannerMaker(config config.Configuration) (emojiBannerPlugin *EmojiBannerMaker, err error) {
	emojiBannerRegex := regexp.MustCompile("(?i)(emoji banner) (.*)")

	options := figlet4go.NewRenderOptions()
	renderer := figlet4go.NewAsciiRender()

	if pluginConfig, ok := config.Plugins[emojiBannerPluginName]; !ok {
		return nil, fmt.Errorf("Missing plugin config for %s", emojiBannerPluginName)
	} else {

		if fontPath, ok := pluginConfig[fontPathKey]; !ok {
			return nil, fmt.Errorf("Missing %s config key: %s", emojiBannerPluginName, fontPathKey)
		} else {
			fontPath, err = homedir.Expand(fontPath)
			if err != nil {
				return nil, fmt.Errorf("[%s] Can't load fonts from [%s]: %v", emojiBannerPluginName, fontPath, err)
			}

			err := renderer.LoadFont(fontPath)
			if err != nil {
				return nil, fmt.Errorf("[%s] Can't load fonts from [%s]: %v", emojiBannerPluginName, fontPath, err)
			}
			log.Printf("Loaded fonts from [%s]", fontPath)

			if fontName, ok := pluginConfig[fontNameKey]; !ok {
				return nil, fmt.Errorf("Missing %s config key: %s", emojiBannerPluginName, fontNameKey)
			} else {
				options.FontName = fontName
				log.Printf("Using font name [%s] if it exists", fontName)
			}
		}
	}

	return &EmojiBannerMaker{slackscot.Plugin{Name: emojiBannerPluginName, Commands: []slackscot.ActionDefinition{{
		Regex:       emojiBannerRegex,
		Usage:       "emoji banner <word> <emoji>",
		Description: "Renders a single-word banner with the provided emoji",
		Answerer: func(message *slack.Msg) string {
			return validateAndRenderEmoji(message.Text, emojiBannerRegex, renderer, options)
		},
	}}, HearActions: nil}}, nil
}

func validateAndRenderEmoji(message string, regex *regexp.Regexp, renderer *figlet4go.AsciiRender, options *figlet4go.RenderOptions) string {
	commandParameters := regex.FindStringSubmatch(message)

	if len(commandParameters) > 0 {
		parameters := strings.Split(commandParameters[2], " ")

		if len(parameters) != 2 {
			return "Wrong usage: emoji banner <word> <emoji>"
		}

		return renderBanner(parameters[0], parameters[1], renderer, options)
	}

	return "Wrong usage: emoji banner <word> <emoji>"
}

func renderBanner(word, emoji string, renderer *figlet4go.AsciiRender, options *figlet4go.RenderOptions) string {
	rendered, err := renderer.RenderOpts(word, options)
	if err != nil {
		return fmt.Sprintf("Error generating: %v", err)
	}

	var result string
	for _, character := range rendered {
		if unicode.IsPrint(character) && character != ' ' {
			result = result + emoji
		} else if character == ' ' {
			result = result + "⬜️"
		} else {
			result = result + string(character)
		}
	}

	return "\r\n" + result
}
