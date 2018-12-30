package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/hashicorp/golang-lru"
	"github.com/nlopes/slack"
	"log"
	"os"
	"regexp"
)

// Slackscot represents what defines a Slack Mascot (mostly, a name and its plugins)
type Slackscot struct {
	name                    string
	config                  config.Configuration
	defaultAction           ActionFunc
	plugins                 []*Plugin
	triggeringMsgToResponse *lru.ARCCache

	// Internal state as an optimization when looping through all commands/hearActions
	commandsWithId    []ActionDefinitionWithId
	hearActionsWithId []ActionDefinitionWithId

	selfId   string
	selfName string
}

// Plugin represents a plugin (its name and action definitions)
type Plugin struct {
	Name        string
	Commands    []ActionDefinition
	HearActions []ActionDefinition
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

// ActionDefinitionWithId holds an action definition along with its identifier string
type ActionDefinitionWithId struct {
	ActionDefinition
	id string
}

// String returns a friendly description of an ActionDefinition
func (a ActionDefinition) String() string {
	return fmt.Sprintf("`%s` - %s", a.Usage, a.Description)
}

// ActionFunc is what gets executed when an ActionDefinition is triggered
type ActionFunc func(m *slack.Msg) string

// responseStrategy defines how a slack.OutgoingMessage is generated from a response
type responseStrategy func(rtm *slack.RTM, m *slack.Msg, response string) *slack.OutgoingMessage

// SlackMessageId holds the elements that form a unique message identifier for slack. Technically, slack also uses
// the workspace id as the first part of that unique identifier but since an instance of slackscot only lives within
// a single workspace, that part is left out
type SlackMessageId struct {
	channelId string
	timestamp string
}

// OutgoingMessage holds a plugin generated slack outgoing message along with the plugin identifier
type OutgoingMessage struct {
	*slack.OutgoingMessage

	// The identifier of the source of the outgoing message. The format being: pluginName.c[commandIndex] (for a command) or pluginName.h[actionIndex] (for an hear action)
	pluginIdentifier string
}

// NewSlackscot creates a new slackscot from an array of plugins and a name
func NewSlackscot(name string, config config.Configuration) (bot *Slackscot, err error) {
	triggeringMsgToResponseCache, err := lru.NewARC(config.ResponseCacheSize)
	if err != nil {
		return nil, err
	}

	return &Slackscot{name: name, config: config, defaultAction: func(m *slack.Msg) string {
		return fmt.Sprintf("I don't understand, ask me for \"%s\" to get a list of things I do", helpPluginName)
	}, plugins: []*Plugin{}, triggeringMsgToResponse: triggeringMsgToResponseCache}, nil
}

// RegisterPlugin registers a plugin with the Slackscot engine. This should be invoked
// prior to calling Run
func (s *Slackscot) RegisterPlugin(p *Plugin) {
	s.plugins = append(s.plugins, p)
}

// Run starts the Slackscot and loops until the process is interrupted
func (s *Slackscot) Run() (err error) {
	// Start by adding the help command now that we know all plugins have been registered
	helpPlugin := newHelpPlugin(s.name, VERSION, s.plugins)
	s.RegisterPlugin(&helpPlugin.Plugin)
	s.attachIdentifiersToPluginActions()

	api := slack.New(
		s.config.Token,
		slack.OptionDebug(s.config.Debug),
		slack.OptionLog(log.New(os.Stdout, "slackscot: ", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()

	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch e := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			log.Println("Infos:", e.Info)
			log.Println("Connection counter:", e.ConnectionCount)
			s.cacheSelfIdentity(rtm)

		case *slack.MessageEvent:
			s.processMessageEvent(api, rtm, e)

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
			// Ignoring other messages
		}
	}

	return nil
}

// attachIdentifiersToPluginActions attaches an action identifier to every plugin action and sets them accordingly
// in the internal state of Slackscot
// The identifiers are generated the following way:
//  - pluginName.c[pluginIndexOfTheCommand] for commands
//  - pluginName.h[pluginIndexOfTheHearAction] for hear actions
//
//  The identifier remains the same for the duration of an execution but might change if the slackscot instance
//  reorders/replaces actions. Since the identifier isn't used for any durable functionality at the moment, this seems
//  adequate. If this ever changes, we might formalize an action identifier that could be generated by users and validated
//  to be unique.
func (s *Slackscot) attachIdentifiersToPluginActions() {
	s.commandsWithId = make([]ActionDefinitionWithId, 0)
	s.hearActionsWithId = make([]ActionDefinitionWithId, 0)

	for _, p := range s.plugins {
		if p.Commands != nil {
			for i, c := range p.Commands {
				s.commandsWithId = append(s.commandsWithId, ActionDefinitionWithId{ActionDefinition: c, id: fmt.Sprintf("%s.c[%d]", p.Name, i)})
			}
		}

		if p.HearActions != nil {
			for i, c := range p.HearActions {
				s.hearActionsWithId = append(s.hearActionsWithId, ActionDefinitionWithId{ActionDefinition: c, id: fmt.Sprintf("%s.c[%d]", p.Name, i)})
			}
		}
	}
}

// cacheSelfIdentity gets "our" identity and keeps the selfId and selfName to avoid having to look it up every time
func (s *Slackscot) cacheSelfIdentity(rtm *slack.RTM) {
	s.selfId = rtm.GetInfo().User.ID
	s.selfName = rtm.GetInfo().User.Name
}

// processMessageEvent handles high-level processing of all slack message events.
//
// TODO (coming soon):
// In the case of updated messages or deleted messages,
// slackscot will look for an existing reply to that message. In the case of a deleted message that triggered a response from the bot,
// that message will be deleted from slack. For a changed message, slackscot will either delete the message if not handled anymore or
// will update the original response if it is still handled
func (s *Slackscot) processMessageEvent(api *slack.Client, c *slack.RTM, msgEvent *slack.MessageEvent) {
	// reply_to is an field set to 1 sent by slack when a sent message has been acknowledged and should be considered
	// officially sent to others. Therefore, we ignore all of those since it's mostly for clients/UI to show status
	isReply := msgEvent.ReplyTo > 0

	Debugf(s.config, "Processing event : %v\n", msgEvent)

	if !isReply && msgEvent.Type == "message" {
		slackMessageId := SlackMessageId{channelId: msgEvent.Channel, timestamp: msgEvent.Timestamp}

		if msgEvent.SubType == "message_deleted" {
			deletedMessageId := SlackMessageId{channelId: msgEvent.Channel, timestamp: msgEvent.DeletedTimestamp}

			Debugf(s.config, "Message deleted: [%s] and cache contains: [%s]", deletedMessageId, s.triggeringMsgToResponse.Keys())

			if existingResponses, exists := s.triggeringMsgToResponse.Get(deletedMessageId); exists {
				byAction := existingResponses.(map[string]SlackMessageId)

				for _, v := range byAction {
					// Delete existing response since the triggering message was deleted
					_, _, err := c.DeleteMessage(v.channelId, v.timestamp)
					if err != nil {
						log.Printf("Error deleting existing response to triggering message [%s]: %s: %v", deletedMessageId, v, err)
					}
				}

				s.triggeringMsgToResponse.Remove(deletedMessageId)
			}
		} else {
			// We need to keep the new responses by plugin action identifier as we update/send them
			newResponseByActionId := make(map[string]SlackMessageId)

			if msgEvent.SubType == "message_changed" {
				editedSlackMessageId := SlackMessageId{channelId: msgEvent.Channel, timestamp: msgEvent.SubMessage.Timestamp}

				outMsgs := s.routeMessage(c, combineIncomingMessageToHandle(msgEvent))

				Debugf(s.config, "Updated message: [%s] and cache contains: [%s]", editedSlackMessageId, s.triggeringMsgToResponse.Keys())

				if cachedResponses, exists := s.triggeringMsgToResponse.Get(editedSlackMessageId); exists {
					responsesByAction := cachedResponses.(map[string]SlackMessageId)

					Debugf(s.config, "Detected %d existing responses to message [%s]\n", len(responsesByAction), editedSlackMessageId)

					for _, om := range outMsgs {
						// We had a previous response for that same plugin action so edit it instead of posting a new message
						if r, ok := responsesByAction[om.pluginIdentifier]; ok {
							Debugf(s.config, "slackscot: Trying to update response at [%s].[%s] with message [%s]\n", r.channelId, r.timestamp, om.OutgoingMessage.Text)

							channelId, newOutgoingMsgTimestamp, _, err := api.UpdateMessage(r.channelId, r.timestamp, slack.MsgOptionText(om.OutgoingMessage.Text, false))
							rId := SlackMessageId{channelId: channelId, timestamp: newOutgoingMsgTimestamp}
							if err != nil {
								log.Printf("slackscot: Unable to update message [%s] to triggering message [%s]: %v\n", r, editedSlackMessageId, err)
							} else {
								// Add the new updated message to the new responses
								newResponseByActionId[om.pluginIdentifier] = rId

								// Remove entries for plugin actions as we process them so that we can detect afterwards if a plugin isn't triggering
								// anymore (to delete those responses).
								delete(responsesByAction, om.pluginIdentifier)
							}
						} else {
							Debugf(s.config, "slackscot: New response triggered to updated message [%s] [%s].[%s]: [%s]\n", om.OutgoingMessage.Text, r.channelId, r.timestamp, om.OutgoingMessage.Text)

							// It's a new message for that action so post it as a new message
							channelId, newOutgoingMsgTimestamp, _, err := api.SendMessage(om.OutgoingMessage.Channel, slack.MsgOptionText(om.OutgoingMessage.Text, false))
							rId := SlackMessageId{channelId: channelId, timestamp: newOutgoingMsgTimestamp}
							if err != nil {
								log.Printf("slackscot: Unable to send new message to updated message [%s]: %v\n", r, err)
							} else {
								// Add the new updated message to the new responses
								newResponseByActionId[om.pluginIdentifier] = rId
							}
						}
					}

					// Delete any previous triggered responses that aren't triggering anymore
					for _, r := range responsesByAction {
						c.DeleteMessage(r.channelId, r.timestamp)
					}

					// Since the updated message now has new responses, update the entry with those or remove if no actions are triggered
					if len(newResponseByActionId) > 0 {
						Debugf(s.config, "Updating responses to edited message [%s]\n", editedSlackMessageId)

						s.triggeringMsgToResponse.Add(editedSlackMessageId, newResponseByActionId)
					} else {
						Debugf(s.config, "Deleting entry for edited message [%s] since no more triggered response\n", editedSlackMessageId)

						s.triggeringMsgToResponse.Remove(editedSlackMessageId)
					}
				} else {
					for _, o := range outMsgs {
						Debugf(s.config, "slackscot: New response triggered to updated message [%s].[%s]: [%s]\n", slackMessageId.channelId, slackMessageId.timestamp, o.OutgoingMessage.Text)

						// It's a new message for that action so post it as a new message
						channelId, newOutgoingMsgTimestamp, _, err := api.SendMessage(o.OutgoingMessage.Channel, slack.MsgOptionText(o.OutgoingMessage.Text, false))
						rId := SlackMessageId{channelId: channelId, timestamp: newOutgoingMsgTimestamp}
						if err != nil {
							log.Printf("slackscot: Unable to send new message trigged from updated message [%s]: %v\n", slackMessageId, err)
						} else {
							// Add the new updated message to the new responses
							newResponseByActionId[o.pluginIdentifier] = rId
						}
					}
				}
			} else {
				outMsgs := s.routeMessage(c, &msgEvent.Msg)

				for _, o := range outMsgs {
					// Send the message and keep track of our response in cache to be able to update it as needed later
					channelId, newOutgoingMsgTimestamp, _, err := api.SendMessage(o.OutgoingMessage.Channel, slack.MsgOptionText(o.OutgoingMessage.Text, false))
					rId := SlackMessageId{channelId: channelId, timestamp: newOutgoingMsgTimestamp}
					if err != nil {
						log.Printf("slackscot: Unable to send new message to updated message [%s]: %v\n", slackMessageId, err)
					} else {
						// Add the new updated message to the new responses
						newResponseByActionId[o.pluginIdentifier] = rId
					}
				}

				if len(newResponseByActionId) > 0 {
					Debugf(s.config, "Adding responses to triggering message [%s]: %s", slackMessageId, newResponseByActionId)

					// Add current responses for that triggering message
					s.triggeringMsgToResponse.Add(slackMessageId, newResponseByActionId)
				}
			}
		}
	}
}

// combineIncomingMessageToHandle combined a main message and its sub message to form what would be an intuitive message to process for
// a bot. That is, a message with the new updated text (in the case of a changed message) along with the channel being the one where the message
// is visible and with the user correctly set to the person who updated/sent the message
// Given the current behavior, this means
//   1. Returning the mainMessage when the subType is not "message_changed"
//   2. If the subType is "message_changed", take everything from the main message except for the text and user that is set on
//      the SubMessage
func combineIncomingMessageToHandle(messageEvent *slack.MessageEvent) (combinedMessage *slack.Msg) {
	if messageEvent.SubType == "message_changed" {
		combined := messageEvent.Msg
		combined.Text = messageEvent.SubMessage.Text
		combined.User = messageEvent.SubMessage.User
		return &combined
	}

	return &messageEvent.Msg
}

// routeMessage handles routing the message to commands or hear actions according to the context
// The rules are the following:
// 	1. If the message is on a channel with a direct mention to us (@name), we route to commands
// 	2. If the message is a direct message to us, we route to commands
// 	3. If the message is on a channel without mention (regular conversation), we route to hear actions
func (s *Slackscot) routeMessage(rtm *slack.RTM, m *slack.Msg) (responses []*OutgoingMessage) {
	// Built regex to detect if message was directed at "us"
	r, _ := regexp.Compile("^(<@" + s.selfId + ">|@?" + s.selfName + "):? (.+)")
	matches := r.FindStringSubmatch(m.Text)

	responses = make([]*OutgoingMessage, 0)

	// Ignore messages send by "us"
	if m.User == s.selfId {
		Debugf(s.config, "Ignoring message from user [%s] because that's \"us\" [%s]", m.User, s.selfId)

		return responses
	}

	if len(matches) == 3 {
		if s.commandsWithId != nil {
			omsgs := handleCommand(s.defaultAction, s.commandsWithId, rtm, matches[2], m, reply)
			if len(omsgs) > 0 {
				responses = append(responses, omsgs...)
			}
		}
	} else if m.Channel[0] == 'D' {
		if s.commandsWithId != nil {
			omsgs := handleCommand(s.defaultAction, s.commandsWithId, rtm, m.Text, m, directReply)
			if len(omsgs) > 0 {
				responses = append(responses, omsgs...)
			}
		}
	} else if s.hearActionsWithId != nil {
		omsgs := handleMessage(s.hearActionsWithId, rtm, m.Text, m, send)
		if len(omsgs) > 0 {
			responses = append(responses, omsgs...)
		}
	}

	return responses
}

// handleCommand handles a command by trying a match with all known actions. If no match is found, the default action is invoked
// Note that in the case of the default action being executed, the return value is still false to indicate no bot actions were triggered
func handleCommand(defaultAction ActionFunc, actions []ActionDefinitionWithId, rtm *slack.RTM, content string, m *slack.Msg, rs responseStrategy) (outMsgs []*OutgoingMessage) {
	outMsgs = handleMessage(actions, rtm, content, m, rs)
	if len(outMsgs) == 0 {
		response := defaultAction(m)

		slackOutMsg := rs(rtm, m, response)
		outMsg := OutgoingMessage{OutgoingMessage: slackOutMsg, pluginIdentifier: "default"}
		return []*OutgoingMessage{&outMsg}
	}

	return outMsgs
}

// processMessage loops over all action definitions and invokes its action if the incoming message matches it's regular expression
// Note that more than one action can be triggered during the processing of a single message
func handleMessage(actions []ActionDefinitionWithId, rtm *slack.RTM, t string, m *slack.Msg, rs responseStrategy) (outMsgs []*OutgoingMessage) {
	outMsgs = make([]*OutgoingMessage, 0)

	for _, action := range actions {
		matches := action.Regex.FindStringSubmatch(t)

		if len(matches) > 0 {
			response := action.Answerer(m)

			if response != "" {
				slackOutMsg := rs(rtm, m, response)
				outMsg := OutgoingMessage{OutgoingMessage: slackOutMsg, pluginIdentifier: action.id}

				outMsgs = append(outMsgs, &outMsg)
			}
		}
	}

	return outMsgs
}

// reply sends a reply to the user (using @user) who sent the message on the channel it was sent on
func reply(rtm *slack.RTM, rm *slack.Msg, response string) *slack.OutgoingMessage {
	om := rtm.NewOutgoingMessage(fmt.Sprintf("<@%s>: %s", rm.User, response), rm.Channel)
	return om
}

// directReply sends a reply to a direct message (which is internally a channel id for slack). It is essentially
// the same as send but it's kept separate for clarity
func directReply(rtm *slack.RTM, rm *slack.Msg, response string) *slack.OutgoingMessage {
	return send(rtm, rm, response)
}

// send creates a message to be sent on the same channel as received (which can be a direct message since
// slack internally uses a channel id for private conversations)
func send(rtm *slack.RTM, rm *slack.Msg, response string) *slack.OutgoingMessage {
	om := rtm.NewOutgoingMessage(response, rm.Channel)
	return om
}

// Debugf logs a debug line after checking if the configuration is in debug mode
func Debugf(config config.Configuration, format string, v ...interface{}) {
	if config.Debug {
		log.Printf(format, v...)
	}
}
