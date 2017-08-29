package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/nlopes/slack"
	"log"
	"os"
	"regexp"
)

// Slackscot represents what defines a Slack Mascot (mostly, a name and its plugins)
type Slackscot struct {
	name    string
	plugins []Plugin
	botEngine
}

// Plugin defines the lifecycle functions of a plugin: Init and Close
type Plugin interface {
	fmt.Stringer
	Init(config.Configuration) (commands []ActionDefinition, listeners []ActionDefinition, err error)
	Close()
}

// ActionDefinition represents how an action is triggered, published, used and described
// along with defining the function defining its behavior
type ActionDefinition struct {
	// Indicates whether the action should be omitted from the help message
	Hidden bool

	// Pattern to match in the message for the action's Answerer function to execute
	Regex *regexp.Regexp

	// Usage example
	Usage string

	// Help description for the action
	Description string

	// Function to execute if the Regex matches
	Answerer ActionFunc
}

// String returns a friendly description of an ActionDefinition
func (a ActionDefinition) String() string {
	return fmt.Sprintf("`%s` - %s", a.Usage, a.Description)
}

// ActionFunc is what gets executed when an ActionDefinition is triggered
type ActionFunc func(ime *IncomingMessageEvent) string

// responseStrategy defines how a slack.OutgoingMessage is generated from a response
type responseStrategy func(rtm *slack.RTM, ime *IncomingMessageEvent, response string) *slack.OutgoingMessage

// botEngine is the internal representation of a running bot
type botEngine struct {
	defaultAction ActionFunc
	hearActions   []ActionDefinition
	commands      []ActionDefinition
}

// IncomingMessageEvent embeds a slack MessageEvent and adds a few internal slackscot bits of context
// Namely: the existingReplyMsgId which is set by slackscot if a modified or deleted message is seen to which
// he/she had replied to already. This can be used to modify/delete that response message instead of ignoring it or
// responding with a new message.
type IncomingMessageEvent struct {
	slack.MessageEvent
	// TODO: fill this in when receiving a new message event so that handling can then use it to update the existing message
	// or delete it
	existingReplyMsgId SlackMsgId
}

// SlackMsgId holds the elements that form a unique message identifier for slack. Technically, slack also uses
// the workspace id as the first part of that unique identifier but since an instance of slackscot only lives within
// a single workspace, that part is left out
type SlackMsgId struct {
	channelId string
	timestamp int
}

// NewSlackscot creates a new slackscot from an array of plugins and a name
func NewSlackscot(name string, plugins []Plugin) (bot *Slackscot) {
	return &Slackscot{name: name, plugins: plugins}
}

// init initializes all plugins and creates the internal botEngine to start processing messages
func (s *Slackscot) init(config config.Configuration) error {
	commands := make([]ActionDefinition, 0)
	hearActions := make([]ActionDefinition, 0)

	// Keep track of initialized plugins in case one fails to initialize and we need to close the ones
	// already initialized before returning
	initializedPlugins := make([]Plugin, 0)
	for _, p := range s.plugins {
		c, h, err := p.Init(config)
		if err != nil {
			// Close all plugins already initialized already before returning error and shutting down everything
			for _, ip := range initializedPlugins {
				ip.Close()
			}

			return err
		}

		initializedPlugins = append(initializedPlugins, p)
		commands = append(commands, c...)
		hearActions = append(hearActions, h...)
	}

	helpCommand := generateHelpCommand(s.name, commands, hearActions)
	allCommands := append(commands, helpCommand)

	// Set internal state representing the bot's engine
	s.botEngine = botEngine{defaultAction: func(ime *IncomingMessageEvent) string {
		return fmt.Sprintf("I don't understand, ask me for \"%s\" to get a list of things I do", helpCommand.Usage)
	}, hearActions: hearActions, commands: allCommands}

	return nil
}

// generateHelpCommand generates a command providing a list of all of the slackscot commands and hear actions.
// Note that ActionDefinitions with the flag Hidden set to true won't be included in the list
func generateHelpCommand(name string, commands []ActionDefinition, hearActions []ActionDefinition) ActionDefinition {
	return ActionDefinition{
		Regex:       regexp.MustCompile("(?i)help"),
		Usage:       "help",
		Description: "Reply with usage instructions",
		Answerer: func(ime *IncomingMessageEvent) string {
			response := fmt.Sprintf("I'm `%s` (engine version `%s`) that listens to the team's chat and provides automated functions."+
				"  I currently support the following commands:\n", name, VERSION)

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

			return response
		},
	}
}

// Close closes slackscot (its plugins) gracefully
func (s *Slackscot) Close() {
	for _, p := range s.plugins {
		p.Close()
	}
}

// Run starts the Slackscot and loops until the process is interrupted
func (s *Slackscot) Run(config config.Configuration) (err error) {
	api := slack.New(
		config.Token,
		slack.OptionDebug(config.Debug),
		slack.OptionLog(log.New(os.Stdout, "slackscot: ", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// Initialize slackscot (and all its plugins)
	err = s.init(config)
	if err != nil {
		return err
	}

	// Register the close of slackscot when the process terminates
	defer s.Close()

	for msg := range rtm.IncomingEvents {
		switch e := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			log.Println("Infos:", e.Info)
			log.Println("Connection counter:", e.ConnectionCount)

		case *slack.MessageEvent:
			s.processMessageEvent(rtm, e)

		case *slack.PresenceChangeEvent:
			log.Printf("Presence Change: %v\n", e)

		case *slack.LatencyReport:
			log.Printf("Current latency: %v\n", e.Value)

		case *slack.RTMError:
			log.Printf("Error: %s\n", e.Error())

		case *slack.InvalidAuthEvent:
			log.Printf("Invalid credentials")
			return

		default:
			// Ignore other events
		}
	}

	return nil
}

// processMessageEvent handles high-level processing of all slack message events.
//
// TODO (coming soon):
// In the case of updated messages or deleted messages,
// slackscot will look for an existing reply to that message. In the case of a deleted message that triggered a response from the bot,
// that message will be deleted from slack. For a changed message, slackscot will either delete the message if not handled anymore or
// will update the original response if it is still handled
func (b *botEngine) processMessageEvent(c *slack.RTM, msgEvent *slack.MessageEvent) {
	// reply_to is an field set to 1 sent by slack when a sent message has been acknowledged and should be considered
	// officially sent to others. Therefore, we ignore all of those since it's mostly for clients/UI to show status
	isReply := msgEvent.ReplyTo > 0

	// TODO retrieve stored reply (if any) and change reaction (if appropriate)
	subtype := msgEvent.SubType
	isMessageChangedEvent := subtype == "message_changed" || subtype == "message_deleted"

	ime := &IncomingMessageEvent{MessageEvent: *msgEvent}
	if !isReply && !isMessageChangedEvent {
		if msgEvent.Type == "message" {
			b.routeMessage(c, ime)
		} else {
			log.Printf("message type is %v", msgEvent.Type)
		}
	}
}

// routeMessage handles routing the message to commands or hear actions according to the context
// The rules are the following:
// 	1. If the message is on a channel with a direct mention to us (@name), we route to commands
// 	2. If the message is a direct message to us, we route to commands
// 	3. If the message is on a channel without mention (regular conversation), we route to hear actions
func (b *botEngine) routeMessage(rtm *slack.RTM, ime *IncomingMessageEvent) {
	selfId := rtm.GetInfo().User.ID
	selfName := rtm.GetInfo().User.Name

	// Built regex to detect if message was directed at "us"
	r, _ := regexp.Compile("^(<@" + selfId + ">|@?" + selfName + "):? (.+)")

	matches := r.FindStringSubmatch(ime.Text)

	if len(matches) == 3 {
		if b.commands != nil {
			handleCommand(b.defaultAction, b.commands, rtm, matches[2], ime, reply)
		}
	} else if ime.Channel[0] == 'D' {
		if b.commands != nil {
			handleCommand(b.defaultAction, b.commands, rtm, ime.Text, ime, directReply)
		}
	} else {
		if b.hearActions != nil {
			handleMessage(b.hearActions, rtm, ime.Text, ime, send)
		}
	}
}

// handleCommand handles a command by trying a match with all known actions. If no match is found, the default action is invoked
// Note that in the case of the default action being executed, the return value is still false to indicate no bot actions were triggered
func handleCommand(defaultAction ActionFunc, actions []ActionDefinition, rtm *slack.RTM, content string, ime *IncomingMessageEvent, rs responseStrategy) bool {
	handled := handleMessage(actions, rtm, content, ime, rs)
	if !handled {
		response := defaultAction(ime)

		outMsg := rs(rtm, ime, response)
		rtm.SendMessage(outMsg)
	}

	return handled
}

// processMessage loops over all action definitions and invokes its action if the incoming message matches it's regular expression
// Note that more than one action can be triggered during the processing of a single message
func handleMessage(actions []ActionDefinition, rtm *slack.RTM, content string, ime *IncomingMessageEvent, rs responseStrategy) bool {
	handled := false

	var action ActionDefinition
	for _, action = range actions {
		matches := action.Regex.FindStringSubmatch(ime.Text)

		if len(matches) > 0 {
			response := action.Answerer(ime)

			if response != "" {
				handled = true

				outMsg := rs(rtm, ime, response)
				rtm.SendMessage(outMsg)
			}
		}
	}

	return handled
}

// reply sends a reply to the user (using @user) who sent the message on the channel it was sent on
func reply(rtm *slack.RTM, ime *IncomingMessageEvent, response string) *slack.OutgoingMessage {
	om := rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", ime.User, response), ime.Channel)
	return om
}

// directReply sends a reply to a direct message (which is internally a channel id for slack). It is essentially
// the same as send but it's kept separate for clarity
func directReply(rtm *slack.RTM, ime *IncomingMessageEvent, response string) *slack.OutgoingMessage {
	return send(rtm, ime, response)
}

// send creates a message to be sent on the same channel as received (which can be a direct message since
// slack internally uses a channel id for private conversations)
func send(rtm *slack.RTM, ime *IncomingMessageEvent, response string) *slack.OutgoingMessage {
	om := rtm.NewOutgoingMessage(response, ime.Channel)
	return om
}
