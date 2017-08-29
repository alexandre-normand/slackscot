package plugins

import (
	"errors"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/getwe/figlet4go"
	"github.com/mitchellh/go-homedir"
	"log"
	"regexp"
	"strings"
	"unicode"
)

const (
	FONT_PATH = "fontPath"
	FONT_NAME = "fontName"
)

type EmojiBannerMaker struct {
}

func NewEmojiBannerMaker() *EmojiBannerMaker {
	return &EmojiBannerMaker{}
}

func (emojiBannerMaker EmojiBannerMaker) String() string {
	return "emojiBanner"
}

func (emojiBannerMaker EmojiBannerMaker) Init(config config.Configuration) (commands []slackscot.ActionDefinition, listeners []slackscot.ActionDefinition, err error) {
	options := figlet4go.NewRenderOptions()
	renderer := figlet4go.NewAsciiRender()

	if pluginConfig, ok := config.Plugins[emojiBannerMaker.String()]; !ok {
		return nil, nil, errors.New(fmt.Sprintf("Missing extention config for %s", emojiBannerMaker.String()))
	} else {
		if fontPath, ok := pluginConfig[FONT_PATH]; !ok {
			return nil, nil, errors.New(fmt.Sprintf("Missing %s config key: %s", emojiBannerMaker.String(), FONT_PATH))
		} else {
			fontPath, err = homedir.Expand(fontPath)
			if err != nil {
				return nil, nil, errors.New(fmt.Sprintf("[%s] Can't load fonts from [%s]: %v", emojiBannerMaker.String(), fontPath, err))
			}

			err := renderer.LoadFont(fontPath)
			if err != nil {
				return nil, nil, errors.New(fmt.Sprintf("[%s] Can't load fonts from [%s]: %v", emojiBannerMaker.String(), fontPath, err))
			}
			log.Printf("Loaded fonts from [%s]", fontPath)
		}

		if fontName, ok := pluginConfig[FONT_NAME]; !ok {
			return nil, nil, errors.New(fmt.Sprintf("Missing %s config key: %s", emojiBannerMaker.String(), FONT_NAME))
		} else {
			options.FontName = fontName
			log.Printf("Using font name [%s] if it exists", fontName)
		}
	}

	emojiBannerRegex := regexp.MustCompile("(?i)(emoji banner) (.*)")

	commands = append(commands, slackscot.ActionDefinition{
		Regex:       emojiBannerRegex,
		Usage:       "emoji banner <word> <emoji>",
		Description: "Renders a single-word banner with the provided emoji",
		Answerer: func(message *slackscot.IncomingMessageEvent) string {
			return validateAndRenderEmoji(message.Text, emojiBannerRegex, renderer, options)
		},
	})

	return commands, listeners, nil
}

func (emojiBannerMaker EmojiBannerMaker) Close() {
}

func validateAndRenderEmoji(message string, regex *regexp.Regexp, renderer *figlet4go.AsciiRender, options *figlet4go.RenderOptions) string {
	commandParameters := regex.FindStringSubmatch(message)

	if len(commandParameters) > 0 {
		parameters := strings.Split(commandParameters[2], " ")

		if len(parameters) != 2 {
			return "Wrong usage: emoji banner <word> <emoji>"
		}

		return RenderBanner(parameters[0], parameters[1], renderer, options)
	}

	return "Wrong usage: emoji banner <word> <emoji>"
}

func RenderBanner(word, emoji string, renderer *figlet4go.AsciiRender, options *figlet4go.RenderOptions) string {
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
