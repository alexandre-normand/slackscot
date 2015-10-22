package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/james-bowman/slack"
	"log"
	"regexp"
)

type Action struct {
	// indicates whether the action should be omitted from the bot's help message.
	// Defaults to false.
	Hidden bool

	// pattern to match in the message for the action's Answerer function to execute
	Regex *regexp.Regexp

	// usage example
	Usage string

	// textual help description for the action
	Description string

	// function to execute if the Regex matches
	Answerer ActionFunc
}

func (a Action) String() string {
	return fmt.Sprintf("`%s` - %s", a.Usage, a.Description)
}

type ActionFunc func(*slack.Message) string

type Slackscot struct {
	bundles []ExtentionBundle
}

type ExtentionBundle interface {
	fmt.Stringer
	Init(config.Configuration) (commands []Action, listeners []Action, err error)
	Close()
}

var tellCommand = regexp.MustCompile("(?i)tell (<.*?>)( .*)? <#(.*?)>? (.*)")

var defaultAction ActionFunc

var hearActions actionList
var commands actionList

type actionList []Action

func NewSlackscot(bundles []ExtentionBundle) (bot *Slackscot) {
	return &Slackscot{bundles: bundles}
}

func (a actionList) handle(message *slack.Message, command bool) bool {
	handled := false

	var action Action
	for _, action = range a {
		matches := action.Regex.FindStringSubmatch(message.Text)

		if len(matches) > 0 {
			response := action.Answerer(message)

			if response != "" {
				handled = true
				log.Printf("me-> %s", response)
				if command {
					if err := message.Respond(response); err != nil {
						// gulp!
						log.Printf("Error responding to message: %s\nwith Message: '%s'", err, response)
					}
				} else {
					if err := message.Send(response); err != nil {
						// gulp!
						log.Printf("Error responding to message: %s\nwith Message: '%s'", err, response)
					}
				}

			}
		}
	}

	return handled
}

func registerCommand(newAction Action) {
	log.Printf("\nRegistering command: %s", newAction)
	commands = append(commands, newAction)
}

func registerAction(newAction Action) {
	log.Printf("\nRegistering Action: %s", newAction)
	hearActions = append(hearActions, newAction)
}

func registerDefault(action ActionFunc) {
	log.Printf("\nRegistering Default Action: %#v", action)
	if defaultAction != nil {
		panic(fmt.Sprintf("Attempted to set default action failed because one has already been registered: %#v", defaultAction))
	}
	defaultAction = action
}

func init() {
	registerCommand(Action{
		Regex:       regexp.MustCompile("(?i)help"),
		Usage:       "help",
		Description: "Reply with usage instructions",
		Answerer: func(dummy *slack.Message) string {
			response := "I am a robot that listens to the team's chat and provides automated functions." +
				"  I currently support the following commands:\n"

			for _, value := range commands {
				if value.Usage != "" && !value.Hidden {
					response = fmt.Sprintf("%s\n\t%s", response, value)
				}
			}

			for _, value := range hearActions {
				if value.Usage != "" && !value.Hidden {
					response = fmt.Sprintf("%s\n\t%s", response, value)
				}
			}

			return response + "\n\nI can do some other things too - try asking me something!"
		},
	})
}

func onHeardMessage(message *slack.Message) {
	hearActions.handle(message, false)
}

func onAskedMessage(message *slack.Message) {
	log.Printf("%s-> %s", message.From, message.Text)

	matches := tellCommand.FindStringSubmatch(message.Text)
	if len(matches) > 0 {
		message.Tell(matches[3], matches[1]+": "+matches[4])
		return
	}

	handled := commands.handle(message, true)

	if !handled {
		var response string

		if defaultAction != nil {
			// if sentence not matched to a supported request then fallback to a default
			response = defaultAction(message)
		} else {
			response = "Je comprends pas.\n_Dis-moi_ `help` _pour avoir la liste des commandes_"
		}

		log.Printf("me-> %s", response)
		if err := message.Respond(response); err != nil {
			// gulp!
			log.Printf("Error responding to message: %s\nwith Message: '%s'", err, response)
		}
	}
}

func Run(slackscot Slackscot, config config.Configuration) (err error) {
	conn, err := slack.Connect(config.Token)
	if err != nil {
		log.Fatal(err)
	}

	// Register all commands and listeners

	var commands []Action
	var listeners []Action

	for _, b := range slackscot.bundles {
		c, l, err := b.Init(config)
		if err != nil {
			return err
		}

		commands = append(commands, c...)
		listeners = append(listeners, l...)
	}

	// Register the close of resources when the process terminates
	defer Close(slackscot)

	for _, c := range commands {
		registerCommand(c)
	}

	for _, l := range listeners {
		registerAction(l)
	}

	slack.EventProcessor(conn, onAskedMessage, onHeardMessage)

	return nil
}

func Close(slackscot Slackscot) {
	for _, b := range slackscot.bundles {
		b.Close()
	}
}
