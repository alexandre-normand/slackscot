// Package slackscot provides the building blocks to create a slack bot. It is
// easily extendable via plugins that can combine commands, hear actions (listeners) as well
// as scheduled actions. It also supports updating of triggered responses on message updates as well
// as deleting triggered responses when the triggering messages are deleted by users.
package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/hashicorp/golang-lru"
	"github.com/marcsantiago/gocron"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

// Slackscot represents what defines a Slack Mascot (mostly, a name and its plugins)
type Slackscot struct {
	name                    string
	config                  *viper.Viper
	defaultAction           Answerer
	plugins                 []*Plugin
	triggeringMsgToResponse *lru.ARCCache

	// Internal state as an optimization when looping through all commands/hearActions
	commandsWithId    []ActionDefinitionWithId
	hearActionsWithId []ActionDefinitionWithId

	selfId   string
	selfName string
	*log.Logger
}

// Plugin represents a plugin (its name and action definitions)
type Plugin struct {
	Name             string
	Commands         []ActionDefinition
	HearActions      []ActionDefinition
	ScheduledActions []ScheduledActionDefinition
}

// ActionDefinition represents how an action is triggered, published, used and described
// along with defining the function defining its behavior
type ActionDefinition struct {
	// Indicates whether the action should be omitted from the help message
	Hidden bool

	// Matcher that will determine whether or not the action should be triggered
	Match Matcher

	// Usage example
	Usage string

	// Help description for the action
	Description string

	// Function to execute if the Matcher matches
	Answer Answerer
}

// Matcher is the function that determines whether or not an action should be triggered. Note that a match doesn't guarantee that the action should
// actually respond with anything once invoked
type Matcher func(t string, m *slack.Msg) bool

// ScheduledActionDefinition represents when a scheduled action is triggered as well
// as what it does and how
type ScheduledActionDefinition struct {
	// Indicates whether the action should be omitted from the help message
	Hidden bool

	schedule.ScheduleDefinition

	// Help description for the scheduled action
	Description string

	// ScheduledAction is the function that is invoked when the schedule activates
	Action ScheduledAction
}

// ActionDefinitionWithId holds an action definition along with its identifier string
type ActionDefinitionWithId struct {
	ActionDefinition
	id string
}

// String returns a friendly description of a ScheduledActionDefinition
func (a ScheduledActionDefinition) String() string {
	return fmt.Sprintf("`%s` - %s", a.ScheduleDefinition, a.Description)
}

// String returns a friendly description of an ActionDefinition
func (a ActionDefinition) String() string {
	return fmt.Sprintf("`%s` - %s", a.Usage, a.Description)
}

// Answerer is what gets executed when an ActionDefinition is triggered
type Answerer func(m *slack.Msg) string

// ScheduledAction is what gets executed when a ScheduledActionDefinition is triggered (by its ScheduleDefinition)
type ScheduledAction func(rtm *slack.RTM)

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
func NewSlackscot(name string, v *viper.Viper) (bot *Slackscot, err error) {
	triggeringMsgToResponseCache, err := lru.NewARC(v.GetInt(config.ResponseCacheSizeKey))
	if err != nil {
		return nil, err
	}

	return &Slackscot{name: name, config: v, defaultAction: func(m *slack.Msg) string {
		return fmt.Sprintf("I don't understand, ask me for \"%s\" to get a list of things I do", helpPluginName)
	}, plugins: []*Plugin{}, triggeringMsgToResponse: triggeringMsgToResponseCache, Logger: log.New(os.Stdout, "slackscot: ", log.Lshortfile|log.LstdFlags)}, nil
}

// RegisterPlugin registers a plugin with the Slackscot engine. This should be invoked
// prior to calling Run
func (s *Slackscot) RegisterPlugin(p *Plugin) {
	s.plugins = append(s.plugins, p)
}

// Run starts the Slackscot and loops until the process is interrupted
func (s *Slackscot) Run() (err error) {
	// Start by adding the help command now that we know all plugins have been registered
	helpPlugin := newHelpPlugin(s.name, VERSION, s.config, s.plugins)
	s.RegisterPlugin(&helpPlugin.Plugin)
	s.attachIdentifiersToPluginActions()

	// Push the Debug configuration to the global Viper instance so it's available to plugins too.
	// TODO: get a better debug logging solution in place that can be used for plugins as well
	viper.Set(config.DebugKey, s.config.GetBool(config.DebugKey))

	api := slack.New(
		s.config.GetString(config.TokenKey),
		slack.OptionDebug(s.config.GetBool(config.DebugKey)),
		slack.OptionLog(log.New(os.Stdout, "slack: ", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()

	go rtm.ManageConnection()

	// Load time zone location for the scheduler
	timeLoc, err := config.GetTimeLocation(s.config)
	if err != nil {
		return err
	}

	// Start scheduling of scheduled actions
	go s.startActionScheduler(timeLoc, rtm)
	go s.watchForTerminationSignalToAbort(rtm)

	for msg := range rtm.IncomingEvents {
		switch e := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			s.Logger.Println("Infos:", e.Info)
			s.Logger.Println("Connection counter:", e.ConnectionCount)
			s.cacheSelfIdentity(rtm)

		case *slack.MessageEvent:
			s.processMessageEvent(api, rtm, e)

		case *slack.PresenceChangeEvent:
			s.Logger.Printf("Presence Change: %v\n", e)

		case *slack.LatencyReport:
			s.Logger.Printf("Current latency: %v\n", e.Value)

		case *slack.RTMError:
			s.Logger.Printf("Error: %s\n", e.Error())

		case *slack.InvalidAuthEvent:
			s.Logger.Printf("Invalid credentials")
			return

		default:
			// Ignoring other messages
		}
	}

	return nil
}

// watchForTerminationSignalToAbort waits for a SIGTERM or SIGINT and closes the rtm's IncomingEvents channel to finish
// the main Run() loop and terminate cleanly. Note that this is meant to run in a go routine given that this is blocking
func (s *Slackscot) watchForTerminationSignalToAbort(rtm *slack.RTM) {
	tSignals := make(chan os.Signal, 1)
	// Register to be notified of termination signals so we can abort
	signal.Notify(tSignals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-tSignals

	s.Debugf("Received termination signal [%s], closing RTM's incoming events channel to terminate processing\n", sig)
	close(rtm.IncomingEvents)
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

	s.Debugf("Caching self id [%s] and self name [%s]\n", s.selfId, s.selfName)
}

// startActionScheduler creates all ScheduledActionDefinition from all plugins and registers them with the scheduler
// Very importantly, it also starts the scheduler
func (s *Slackscot) startActionScheduler(timeLoc *time.Location, rtm *slack.RTM) {
	gocron.ChangeLoc(timeLoc)
	sc := gocron.NewScheduler()

	for _, p := range s.plugins {
		if p.ScheduledActions != nil {
			for _, sa := range p.ScheduledActions {
				j := schedule.NewScheduledJob(sc, sa.ScheduleDefinition)
				s.Debugf("Adding job [%v] to scheduler\n", j)
				j.Do(sa.Action, rtm)
			}
		}
	}

	_, t := sc.NextRun()
	s.Debugf("Starting scheduler with first job scheduled at [%s]\n", t)

	// TODO: consider keeping track of the scheduler to stop it if it starts to appear necessary
	<-sc.Start()
}

// processMessageEvent handles high-level processing of all slack message events.
func (s *Slackscot) processMessageEvent(api *slack.Client, rtm *slack.RTM, msgEvent *slack.MessageEvent) {
	// reply_to is an field set to 1 sent by slack when a sent message has been acknowledged and should be considered
	// officially sent to others. Therefore, we ignore all of those since it's mostly for clients/UI to show status
	isReply := msgEvent.ReplyTo > 0

	s.Debugf("Processing event : %v\n", msgEvent)

	if !isReply && msgEvent.Type == "message" {
		slackMessageId := SlackMessageId{channelId: msgEvent.Channel, timestamp: msgEvent.Timestamp}

		if msgEvent.SubType == "message_deleted" {
			s.processDeletedMessage(rtm, msgEvent)
		} else {
			if msgEvent.SubType == "message_changed" {
				s.processUpdatedMessage(api, rtm, msgEvent, slackMessageId)
			} else {
				s.processNewMessage(api, rtm, msgEvent, slackMessageId)
			}
		}
	}
}

// processUpdatedMessage processes changed messages. This is a more complicated scenario but slackscot handles it by doing the following:
// 1. If the message isn't present in the triggering message cache, we process it as we would any other regular new message (check if it triggers an action and sends responses accordingly)
// 2. If the message is present in cache, we had pre-existing responses so we handle this by updating responses on a plugin action basis. A plugin action that isn't triggering anymore gets its previous
//    response deleted while a still triggering response will result in a message update. Newly triggered actions will be sent out as new messages.
// 3. The new state of responses replaces the previous one for the triggering message in the cache
func (s *Slackscot) processUpdatedMessage(api *slack.Client, rtm *slack.RTM, msgEvent *slack.MessageEvent, incomingMessageId SlackMessageId) {
	editedSlackMessageId := SlackMessageId{channelId: msgEvent.Channel, timestamp: msgEvent.SubMessage.Timestamp}

	s.Debugf("Updated message: [%s], does cache contain it => [%t]", editedSlackMessageId, s.triggeringMsgToResponse.Contains(editedSlackMessageId))

	if cachedResponses, exists := s.triggeringMsgToResponse.Get(editedSlackMessageId); exists {
		responsesByAction := cachedResponses.(map[string]SlackMessageId)
		newResponseByActionId := make(map[string]SlackMessageId)

		outMsgs := s.routeMessage(rtm, combineIncomingMessageToHandle(msgEvent))
		s.Debugf("Detected %d existing responses to message [%s]\n", len(responsesByAction), editedSlackMessageId)

		for _, o := range outMsgs {
			// We had a previous response for that same plugin action so edit it instead of posting a new message
			if r, ok := responsesByAction[o.pluginIdentifier]; ok {
				s.Debugf("Trying to update response at [%s] with message [%s]\n", r, o.OutgoingMessage.Text)

				rId, err := s.updateExistingMessage(api, r, o)
				if err != nil {
					s.Logger.Printf("Unable to update message [%s] to triggering message [%s]: %v\n", r, editedSlackMessageId, err)
				} else {
					// Add the new updated message to the new responses
					newResponseByActionId[o.pluginIdentifier] = rId

					// Remove entries for plugin actions as we process them so that we can detect afterwards if a plugin isn't triggering
					// anymore (to delete those responses).
					delete(responsesByAction, o.pluginIdentifier)
				}
			} else {
				s.Debugf("New response triggered to updated message [%s] [%s]: [%s]\n", o.OutgoingMessage.Text, r, o.OutgoingMessage.Text)

				// It's a new message for that action so post it as a new message
				rId, err := s.sendNewMessage(api, o, incomingMessageId.timestamp)
				if err != nil {
					s.Logger.Printf("Unable to send new message to updated message [%s]: %v\n", r, err)
				} else {
					// Add the new updated message to the new responses
					newResponseByActionId[o.pluginIdentifier] = rId
				}
			}
		}

		// Delete any previous triggered responses that aren't triggering anymore
		for _, r := range responsesByAction {
			rtm.DeleteMessage(r.channelId, r.timestamp)
		}

		// Since the updated message now has new responses, update the entry with those or remove if no actions are triggered
		if len(newResponseByActionId) > 0 {
			s.Debugf("Updating responses to edited message [%s]\n", editedSlackMessageId)
			s.triggeringMsgToResponse.Add(editedSlackMessageId, newResponseByActionId)
		} else {
			s.Debugf("Deleting entry for edited message [%s] since no more triggered response\n", editedSlackMessageId)
			s.triggeringMsgToResponse.Remove(editedSlackMessageId)
		}
	} else {
		outMsgs := s.routeMessage(rtm, combineIncomingMessageToHandle(msgEvent))
		s.sendOutgoingMessages(api, rtm, incomingMessageId, outMsgs)
	}
}

// processDeletedMessage handles a deleted message. Slackscot cares about those in order to
// delete any previous responses triggered by that now inexistant message
func (s *Slackscot) processDeletedMessage(rtm *slack.RTM, msgEvent *slack.MessageEvent) {
	deletedMessageId := SlackMessageId{channelId: msgEvent.Channel, timestamp: msgEvent.DeletedTimestamp}

	s.Debugf("Message deleted: [%s] and cache contains: [%s]", deletedMessageId, s.triggeringMsgToResponse.Keys())

	if existingResponses, exists := s.triggeringMsgToResponse.Get(deletedMessageId); exists {
		byAction := existingResponses.(map[string]SlackMessageId)

		for _, v := range byAction {
			// Delete existing response since the triggering message was deleted
			_, _, err := rtm.DeleteMessage(v.channelId, v.timestamp)
			if err != nil {
				s.Logger.Printf("Error deleting existing response to triggering message [%s]: %s: %v", deletedMessageId, v, err)
			}
		}

		s.triggeringMsgToResponse.Remove(deletedMessageId)
	}
}

// processNewMessage handles a regular new message and sends any triggered response
func (s *Slackscot) processNewMessage(api *slack.Client, rtm *slack.RTM, msgEvent *slack.MessageEvent, incomingMessageId SlackMessageId) {
	outMsgs := s.routeMessage(rtm, &msgEvent.Msg)

	s.sendOutgoingMessages(api, rtm, incomingMessageId, outMsgs)
}

// sendOutgoingMessages sends out any triggered plugin responses and keeps track of those in the internal cache
func (s *Slackscot) sendOutgoingMessages(api *slack.Client, rtm *slack.RTM, incomingMessageId SlackMessageId, outMsgs []*OutgoingMessage) {
	newResponseByActionId := make(map[string]SlackMessageId)

	for _, o := range outMsgs {
		// Send the message and keep track of our response in cache to be able to update it as needed later
		rId, err := s.sendNewMessage(api, o, incomingMessageId.timestamp)
		if err != nil {
			s.Logger.Printf("Unable to send new message triggered by [%s]: %v\n", incomingMessageId, err)
		} else {
			// Add the new updated message to the new responses
			newResponseByActionId[o.pluginIdentifier] = rId
		}
	}

	if len(newResponseByActionId) > 0 {
		s.Debugf("Adding responses to triggering message [%s]: %s", incomingMessageId, newResponseByActionId)

		// Add current responses for that triggering message
		s.triggeringMsgToResponse.Add(incomingMessageId, newResponseByActionId)
	}
}

// sendNewMessage sends a new outgoingMsg and waits for the response to return that message's identifier
func (s *Slackscot) sendNewMessage(api *slack.Client, o *OutgoingMessage, threadTS string) (rId SlackMessageId, err error) {
	options := []slack.MsgOption{slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionUser(s.selfId), slack.MsgOptionAsUser(true)}
	if s.config.GetBool(config.ThreadedRepliesKey) {
		options = append(options, slack.MsgOptionTS(threadTS))

		if s.config.GetBool(config.BroadcastThreadedRepliesKey) {
			options = append(options, slack.MsgOptionBroadcast())
		}
	}

	channelId, newOutgoingMsgTimestamp, _, err := api.SendMessage(o.OutgoingMessage.Channel, options...)
	rId = SlackMessageId{channelId: channelId, timestamp: newOutgoingMsgTimestamp}

	return rId, err
}

// updateExistingMessage updates an existing message with the content of a newly triggered OutgoingMessage
func (s *Slackscot) updateExistingMessage(api *slack.Client, r SlackMessageId, o *OutgoingMessage) (rId SlackMessageId, err error) {
	channelId, newOutgoingMsgTimestamp, _, err := api.UpdateMessage(r.channelId, r.timestamp, slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionUser(s.selfId), slack.MsgOptionAsUser(true))
	rId = SlackMessageId{channelId: channelId, timestamp: newOutgoingMsgTimestamp}

	return rId, err
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
	if m.User == s.selfId || m.BotID == s.selfId {
		s.Debugf("Ignoring message from user [%s] because that's \"us\" [%s]", m.User, s.selfId)

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
func handleCommand(defaultAnswer Answerer, actions []ActionDefinitionWithId, rtm *slack.RTM, content string, m *slack.Msg, rs responseStrategy) (outMsgs []*OutgoingMessage) {
	outMsgs = handleMessage(actions, rtm, content, m, rs)
	if len(outMsgs) == 0 {
		response := defaultAnswer(m)

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
		matches := action.Match(t, m)

		if matches {
			response := action.Answer(m)

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
func (s *Slackscot) Debugf(format string, v ...interface{}) {
	if s.config.GetBool(config.DebugKey) {
		s.Logger.Printf(format, v...)
	}
}

// Debugf logs a debug line after checking if the configuration is in debug mode
func Debugf(format string, v ...interface{}) {
	// TODO: formalize a better debug logging for slackscot and get rid of this global usage of viper
	if viper.GetBool(config.DebugKey) {
		log.Printf(format, v...)
	}
}
