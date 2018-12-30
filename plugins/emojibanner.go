package plugins

import (
	"errors"
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
	FONT_PATH         = "fontPath"
	FONT_NAME         = "fontName"
	EMOJI_BANNER_NAME = "emojiBanner"
)

type EmojiBannerMaker struct {
	slackscot.Plugin
}

func NewEmojiBannerMaker(config config.Configuration) (emojiBannerPlugin *EmojiBannerMaker, err error) {
	emojiBannerRegex := regexp.MustCompile("(?i)(emoji banner) (.*)")

	options := figlet4go.NewRenderOptions()
	renderer := figlet4go.NewAsciiRender()

	if pluginConfig, ok := config.Plugins[EMOJI_BANNER_NAME]; !ok {
		return nil, errors.New(fmt.Sprintf("Missing plugin config for %s", EMOJI_BANNER_NAME))
	} else {
		if fontPath, ok := pluginConfig[FONT_PATH]; !ok {
			return nil, errors.New(fmt.Sprintf("Missing %s config key: %s", EMOJI_BANNER_NAME, FONT_PATH))
		} else {
			fontPath, err = homedir.Expand(fontPath)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("[%s] Can't load fonts from [%s]: %v", EMOJI_BANNER_NAME, fontPath, err))
			}

			err := renderer.LoadFont(fontPath)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("[%s] Can't load fonts from [%s]: %v", EMOJI_BANNER_NAME, fontPath, err))
			}
			log.Printf("Loaded fonts from [%s]", fontPath)
		}

		if fontName, ok := pluginConfig[FONT_NAME]; !ok {
			return nil, errors.New(fmt.Sprintf("Missing %s config key: %s", EMOJI_BANNER_NAME, FONT_NAME))
		} else {
			options.FontName = fontName
			log.Printf("Using font name [%s] if it exists", fontName)
		}
	}

	return &EmojiBannerMaker{slackscot.Plugin{Name: EMOJI_BANNER_NAME, Commands: []slackscot.ActionDefinition{slackscot.ActionDefinition{
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
