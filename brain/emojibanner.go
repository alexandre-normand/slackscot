package brain

import (
	"fmt"
	"github.com/alexandre-normand/slack"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/getwe/figlet4go"
	"regexp"
	"strings"
	"unicode"
)

type EmojiBannerMaker struct {
}

func NewEmojiBannerMaker() *EmojiBannerMaker {
	return &EmojiBannerMaker{}
}

func (emojiBannerMaker EmojiBannerMaker) String() string {
	return "emojiBanner"
}

func (emojiBannerMaker EmojiBannerMaker) Init(config config.Configuration) (commands []slackscot.Action, listeners []slackscot.Action, err error) {
	emojiBannerRegex := regexp.MustCompile("(?i)(emoji banner) (.*)")

	commands = append(commands, slackscot.Action{
		Regex:       emojiBannerRegex,
		Usage:       "emoji banner <word> <emoji>",
		Description: "Renders a single-word banner with the provided emoji",
		Answerer: func(message *slack.Message) string {
			return validateAndRenderEmoji(message.Text, emojiBannerRegex)
		},
	})

	return commands, listeners, nil
}

func (emojiBannerMaker EmojiBannerMaker) Close() {
}

func validateAndRenderEmoji(message string, regex *regexp.Regexp) string {
	commandParameters := regex.FindStringSubmatch(message)

	if len(commandParameters) > 0 {
		parameters := strings.Split(commandParameters[2], " ")

		if len(parameters) != 2 {
			return "Wrong usage: emoji banner <word> <emoji>"
		}

		return RenderBanner(parameters[0], parameters[1])
	}

	return "Wrong usage: emoji banner <word> <emoji>"
}

func RenderBanner(word, emoji string) string {
	render := figlet4go.NewAsciiRender()

	rendered, err := render.Render(word)
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
