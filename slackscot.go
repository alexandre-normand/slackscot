// Package slackscot provides the building blocks to create a slack bot. It is
// easily extendable via plugins that can combine commands, hear actions (listeners) as well
// as scheduled actions. It also supports updating of triggered responses on message updates as well
// as deleting triggered responses when the triggering messages are deleted by users.
package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/alexandre-normand/slackscot/v2/schedule"
	"github.com/hashicorp/golang-lru"
	"github.com/marcsantiago/gocron"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	defaultLogPrefix = "slackscot: "
	defaultLogFlag   = log.Lshortfile | log.LstdFlags
)

// Slackscot represents what defines a Slack Mascot (mostly, a name and its plugins)
type Slackscot struct {
	name                    string
	config                  *viper.Viper
	defaultAction           Answerer
	plugins                 []*Plugin
	triggeringMsgToResponse *lru.ARCCache

	// Internal state as an optimization when looping through all commands/hearActions
	commandsWithID    []ActionDefinitionWithID
	hearActionsWithID []ActionDefinitionWithID

	// Caching self identity used during message processing/filtering
	selfID   string
	selfName string

	// Logger
	log *sLogger
}

// Plugin represents a plugin (its name, action definitions and slackscot injected services)
type Plugin struct {
	Name             string
	Commands         []ActionDefinition
	HearActions      []ActionDefinition
	ScheduledActions []ScheduledActionDefinition

	// Those slackscot services are injected post-creation when slackscot is called.
	// A plugin shouldn't rely on those being available during creation
	UserInfoFinder UserInfoFinder
	Logger         SLogger
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

	// Schedule definition determining when the action runs
	Schedule schedule.Definition

	// Help description for the scheduled action
	Description string

	// ScheduledAction is the function that is invoked when the schedule activates
	Action ScheduledAction
}

// ActionDefinitionWithID holds an action definition along with its identifier string
type ActionDefinitionWithID struct {
	ActionDefinition
	id string
}

// String returns a friendly description of a ScheduledActionDefinition
func (a ScheduledActionDefinition) String() string {
	return fmt.Sprintf("`%s` - %s", a.Schedule, a.Description)
}

// String returns a friendly description of an ActionDefinition
func (a ActionDefinition) String() string {
	return fmt.Sprintf("`%s` - %s", a.Usage, a.Description)
}

// Answerer is what gets executed when an ActionDefinition is triggered
type Answerer func(m *slack.Msg) string

// ScheduledAction is what gets executed when a ScheduledActionDefinition is triggered (by its ScheduleDefinition)
type ScheduledAction func(sender RealTimeMessageSender)

// responseStrategy defines how a slack.OutgoingMessage is generated from a response
type responseStrategy func(m *slack.Msg, response string) *slack.OutgoingMessage

// SlackMessageID holds the elements that form a unique message identifier for slack. Technically, slack also uses
// the workspace id as the first part of that unique identifier but since an instance of slackscot only lives within
// a single workspace, that part is left out
type SlackMessageID struct {
	channelID string
	timestamp string
}

// OutgoingMessage holds a plugin generated slack outgoing message along with the plugin identifier
type OutgoingMessage struct {
	*slack.OutgoingMessage

	// The identifier of the source of the outgoing message. The format being: pluginName.c[commandIndex] (for a command) or pluginName.h[actionIndex] (for an hear action)
	pluginIdentifier string
}

// terminationEvent is an empty struct that is only used for whitebox testing in order to signal slackscot to terminate
// Any such events when executed as part of the normal API would be ignored
type terminationEvent struct {
}

// Option defines an option for a Slackscot
type Option func(*Slackscot)

// OptionLog sets a logger for Slackscot
func OptionLog(logger *log.Logger) func(*Slackscot) {
	return func(s *Slackscot) {
		s.log.logger = logger
	}
}

// OptionLogfile sets a logfile for Slackscot while using the other default logging prefix and options
func OptionLogfile(logfile *os.File) func(*Slackscot) {
	return func(s *Slackscot) {
		s.log.logger = log.New(logfile, defaultLogPrefix, defaultLogFlag)
	}
}

// NewSlackscot creates a new slackscot from an array of plugins and a name
func NewSlackscot(name string, v *viper.Viper, options ...Option) (s *Slackscot, err error) {
	s = new(Slackscot)

	s.triggeringMsgToResponse, err = lru.NewARC(v.GetInt(config.ResponseCacheSizeKey))
	if err != nil {
		return nil, err
	}

	s.name = name
	s.config = v
	s.defaultAction = func(m *slack.Msg) string {
		return fmt.Sprintf("I don't understand, ask me for \"%s\" to get a list of things I do", helpPluginName)
	}
	s.log = NewSLogger(log.New(os.Stdout, defaultLogPrefix, defaultLogFlag), v.GetBool(config.DebugKey))

	for _, opt := range options {
		opt(s)
	}

	s.commandsWithID = make([]ActionDefinitionWithID, 0)
	s.hearActionsWithID = make([]ActionDefinitionWithID, 0)

	return s, nil
}

// RegisterPlugin registers a plugin with the Slackscot engine. This should be invoked
// prior to calling Run
func (s *Slackscot) RegisterPlugin(p *Plugin) {
	s.plugins = append(s.plugins, p)
	s.attachIdentifiersToPluginActions(p)
}

// Run starts the Slackscot and loops until the process is interrupted
func (s *Slackscot) Run() (err error) {
	// Start by adding the help command now that we know all plugins have been registered
	helpPlugin := newHelpPlugin(s.name, VERSION, s.config, s.plugins)
	s.RegisterPlugin(&helpPlugin.Plugin)

	sc := slack.New(
		s.config.GetString(config.TokenKey),
		slack.OptionDebug(s.config.GetBool(config.DebugKey)),
	)
	//slack.OptionLog(log.New(s.log.logger.Writer(), "slack: ", defaultLogFlag)),

	s.injectServicesToPlugins(sc, s.log)

	// Load time zone location for the scheduler
	timeLoc, err := config.GetTimeLocation(s.config)
	if err != nil {
		return err
	}

	// This will initiate the connection to the slack RTM and start the reception of messages
	rtm := sc.NewRTM()
	go rtm.ManageConnection()

	// Wrap the slack API to expose access to it for advanced uses
	slackRealTimeMsgSender := slackRealTimeMsgSender{rtm: rtm}

	// Start scheduling of scheduled actions
	go s.startActionScheduler(timeLoc, &slackRealTimeMsgSender)

	termination := make(chan bool)

	// Register to receive a notification for a termination signal which will, in turn, send a termination message to the
	// termination channel
	go s.watchForTerminationSignalToAbort(rtm, termination)

	// This is a blocking call so it's running in a goroutine. The way slackscot would usually terminate
	// in a production scenario is by receiving a termination signal which
	go s.handleIncomingEvents(rtm.IncomingEvents, termination, sc, rtm, true)

	// Wait for termination
	<-termination

	return nil
}

// handleIncomingEvents handles all incoming events and acts as the main loop. It will essentially
// always process events as long as the process isn't interrupted. Normally, this happens
// by a kill signal being sent and slackscot gets notified and closes the events channel which
// terminates this loop and shuts down gracefully
func (s *Slackscot) handleIncomingEvents(events <-chan slack.RTMEvent, termination chan<- bool, driver chatDriver, selfInfoFinder selfInfoFinder, productionMode bool) {
	defer func() {
		termination <- true
	}()

	for msg := range events {
		switch e := msg.Data.(type) {
		case *slack.ConnectedEvent:
			s.log.Printf("Infos: %v\n", e.Info)
			s.log.Printf("Connection counter: %d\n", e.ConnectionCount)
			s.cacheSelfIdentity(selfInfoFinder)

		case *slack.MessageEvent:
			s.processMessageEvent(driver, e)

		case *slack.LatencyReport:
			s.log.Printf("Current latency: %v\n", e.Value)

		case *slack.RTMError:
			s.log.Printf("Error: %s\n", e.Error())

		case *slack.InvalidAuthEvent:
			s.log.Printf("Invalid credentials\n")
			return

		case *terminationEvent:
			if !productionMode {
				s.log.Printf("Received termination event in test mode, terminating\n")
				return
			}
		default:
			// Ignoring other messages
		}
	}
}

// injectServicesToPlugins assembles/creates the services and injects them in all plugins
func (s *Slackscot) injectServicesToPlugins(loadingUserInfoFinder UserInfoFinder, l SLogger) (err error) {
	uf, err := NewCachingUserInfoFinder(s.config, loadingUserInfoFinder, l)
	if err != nil {
		return err
	}

	for _, p := range s.plugins {
		p.Logger = l
		p.UserInfoFinder = uf
	}

	return nil
}

// watchForTerminationSignalToAbort waits for a SIGTERM or SIGINT and closes the rtm's IncomingEvents channel to finish
// the main Run() loop and terminate cleanly. Note that this is meant to run in a go routine given that this is blocking
func (s *Slackscot) watchForTerminationSignalToAbort(rtm *slack.RTM, termination chan<- bool) {
	tSignals := make(chan os.Signal, 1)
	// Register to be notified of termination signals so we can abort
	signal.Notify(tSignals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-tSignals

	s.log.Debugf("Received termination signal [%s], closing RTM's incoming events channel to terminate processing\n", sig)
	termination <- true
}

// attachIdentifiersToPluginActions attaches an action identifier to a plugin action and sets them accordingly
// in the internal state of Slackscot
// The identifiers are generated the following way:
//  - pluginName.c[pluginIndexOfTheCommand] for commands
//  - pluginName.h[pluginIndexOfTheHearAction] for hear actions
//
//  The identifier remains the same for the duration of an execution but might change if the slackscot instance
//  reorders/replaces actions. Since the identifier isn't used for any durable functionality at the moment, this seems
//  adequate. If this ever changes, we might formalize an action identifier that could be generated by users and validated
//  to be unique.
func (s *Slackscot) attachIdentifiersToPluginActions(p *Plugin) {
	if p.Commands != nil {
		for i, c := range p.Commands {
			s.commandsWithID = append(s.commandsWithID, ActionDefinitionWithID{ActionDefinition: c, id: fmt.Sprintf("%s.c[%d]", p.Name, i)})
		}
	}

	if p.HearActions != nil {
		for i, c := range p.HearActions {
			s.hearActionsWithID = append(s.hearActionsWithID, ActionDefinitionWithID{ActionDefinition: c, id: fmt.Sprintf("%s.c[%d]", p.Name, i)})
		}
	}
}

// cacheSelfIdentity gets "our" identity and keeps the selfID and selfName to avoid having to look it up every time
func (s *Slackscot) cacheSelfIdentity(selfInfoFinder selfInfoFinder) {
	s.selfID = selfInfoFinder.GetInfo().User.ID
	s.selfName = selfInfoFinder.GetInfo().User.Name

	s.log.Debugf("Caching self id [%s] and self name [%s]\n", s.selfID, s.selfName)
}

// startActionScheduler creates all ScheduledActionDefinition from all plugins and registers them with the scheduler
// Very importantly, it also starts the scheduler
func (s *Slackscot) startActionScheduler(timeLoc *time.Location, sender RealTimeMessageSender) {
	gocron.ChangeLoc(timeLoc)
	sc := gocron.NewScheduler()

	for _, p := range s.plugins {
		if p.ScheduledActions != nil {
			for _, sa := range p.ScheduledActions {
				j, err := schedule.NewJob(sc, sa.Schedule)
				if err != nil {
					s.log.Debugf("Adding job [%v] to scheduler\n", j)
					j.Do(sa.Action, sender)
				} else {
					s.log.Printf("Error: failed to schedule job for scheduled action [%s]: %v\n", sa, err)
				}
			}
		}
	}

	_, t := sc.NextRun()
	s.log.Debugf("Starting scheduler with first job scheduled at [%s]\n", t)

	// TODO: consider keeping track of the scheduler to stop it if it starts to appear necessary
	<-sc.Start()
}

// processMessageEvent handles high-level processing of all slack message events.
func (s *Slackscot) processMessageEvent(driver chatDriver, msgEvent *slack.MessageEvent) {
	// reply_to is an field set to 1 sent by slack when a sent message has been acknowledged and should be considered
	// officially sent to others. Therefore, we ignore all of those since it's mostly for clients/UI to show status
	isReply := msgEvent.ReplyTo > 0

	s.log.Debugf("Processing event : %v\n", msgEvent)

	if !isReply && msgEvent.Type == "message" {
		slackMessageID := SlackMessageID{channelID: msgEvent.Channel, timestamp: msgEvent.Timestamp}

		if msgEvent.SubType == "message_deleted" {
			s.processDeletedMessage(driver, msgEvent)
		} else {
			if msgEvent.SubType == "message_changed" {
				s.processUpdatedMessage(driver, msgEvent, slackMessageID)
			} else {
				s.processNewMessage(driver, msgEvent, slackMessageID)
			}
		}
	}
}

// processUpdatedMessage processes changed messages. This is a more complicated scenario but slackscot handles it by doing the following:
// 1. If the message isn't present in the triggering message cache, we process it as we would any other regular new message (check if it triggers an action and sends responses accordingly)
// 2. If the message is present in cache, we had pre-existing responses so we handle this by updating responses on a plugin action basis. A plugin action that isn't triggering anymore gets its previous
//    response deleted while a still triggering response will result in a message update. Newly triggered actions will be sent out as new messages.
// 3. The new state of responses replaces the previous one for the triggering message in the cache
func (s *Slackscot) processUpdatedMessage(driver chatDriver, msgEvent *slack.MessageEvent, incomingMessageID SlackMessageID) {
	editedSlackMessageID := SlackMessageID{channelID: msgEvent.Channel, timestamp: msgEvent.SubMessage.Timestamp}

	s.log.Debugf("Updated message: [%s], does cache contain it => [%t]", editedSlackMessageID, s.triggeringMsgToResponse.Contains(editedSlackMessageID))

	if cachedResponses, exists := s.triggeringMsgToResponse.Get(editedSlackMessageID); exists {
		responsesByAction := cachedResponses.(map[string]SlackMessageID)
		newResponseByActionID := make(map[string]SlackMessageID)

		outMsgs := s.routeMessage(combineIncomingMessageToHandle(msgEvent))
		s.log.Debugf("Detected %d existing responses to message [%s]\n", len(responsesByAction), editedSlackMessageID)

		for _, o := range outMsgs {
			// We had a previous response for that same plugin action so edit it instead of posting a new message
			if r, ok := responsesByAction[o.pluginIdentifier]; ok {
				s.log.Debugf("Trying to update response at [%s] with message [%s]\n", r, o.OutgoingMessage.Text)

				rID, err := s.updateExistingMessage(driver, r, o)
				if err != nil {
					s.log.Printf("Unable to update message [%s] to triggering message [%s]: %v\n", r, editedSlackMessageID, err)
				} else {
					// Add the new updated message to the new responses
					newResponseByActionID[o.pluginIdentifier] = rID

					// Remove entries for plugin actions as we process them so that we can detect afterwards if a plugin isn't triggering
					// anymore (to delete those responses).
					delete(responsesByAction, o.pluginIdentifier)
				}
			} else {
				s.log.Debugf("New response triggered to updated message [%s] [%s]: [%s]\n", o.OutgoingMessage.Text, r, o.OutgoingMessage.Text)

				// It's a new message for that action so post it as a new message
				rID, err := s.sendNewMessage(driver, o, incomingMessageID.timestamp)
				if err != nil {
					s.log.Printf("Unable to send new message to updated message [%s]: %v\n", r, err)
				} else {
					// Add the new updated message to the new responses
					newResponseByActionID[o.pluginIdentifier] = rID
				}
			}
		}

		// Delete any previous triggered responses that aren't triggering anymore
		for pa, r := range responsesByAction {
			s.log.Debugf("Deleting previous response [%s] on a now non-triggered plugin action [%s]\n", r, pa)
			driver.DeleteMessage(r.channelID, r.timestamp)
		}

		// Since the updated message now has new responses, update the entry with those or remove if no actions are triggered
		if len(newResponseByActionID) > 0 {
			s.log.Debugf("Updating responses to edited message [%s]\n", editedSlackMessageID)
			s.triggeringMsgToResponse.Add(editedSlackMessageID, newResponseByActionID)
		} else {
			s.log.Debugf("Deleting entry for edited message [%s] since no more triggered response\n", editedSlackMessageID)
			s.triggeringMsgToResponse.Remove(editedSlackMessageID)
		}
	} else {
		outMsgs := s.routeMessage(combineIncomingMessageToHandle(msgEvent))
		s.sendOutgoingMessages(driver, incomingMessageID, outMsgs)
	}
}

// processDeletedMessage handles a deleted message. Slackscot cares about those in order to
// delete any previous responses triggered by that now inexistant message
func (s *Slackscot) processDeletedMessage(deleter messageDeleter, msgEvent *slack.MessageEvent) {
	deletedMessageID := SlackMessageID{channelID: msgEvent.Channel, timestamp: msgEvent.DeletedTimestamp}

	s.log.Debugf("Message deleted: [%s] and cache contains: [%s]", deletedMessageID, s.triggeringMsgToResponse.Keys())

	if existingResponses, exists := s.triggeringMsgToResponse.Get(deletedMessageID); exists {
		byAction := existingResponses.(map[string]SlackMessageID)

		for _, v := range byAction {
			// Delete existing response since the triggering message was deleted
			_, _, err := deleter.DeleteMessage(v.channelID, v.timestamp)
			if err != nil {
				s.log.Printf("Error deleting existing response to triggering message [%s]: %s: %v", deletedMessageID, v, err)
			}
		}

		s.triggeringMsgToResponse.Remove(deletedMessageID)
	}
}

// processNewMessage handles a regular new message and sends any triggered response
func (s *Slackscot) processNewMessage(msgSender messageSender, msgEvent *slack.MessageEvent, incomingMessageID SlackMessageID) {
	outMsgs := s.routeMessage(&msgEvent.Msg)

	s.sendOutgoingMessages(msgSender, incomingMessageID, outMsgs)
}

// sendOutgoingMessages sends out any triggered plugin responses and keeps track of those in the internal cache
func (s *Slackscot) sendOutgoingMessages(sender messageSender, incomingMessageID SlackMessageID, outMsgs []*OutgoingMessage) {
	newResponseByActionID := make(map[string]SlackMessageID)

	for _, o := range outMsgs {
		// Send the message and keep track of our response in cache to be able to update it as needed later
		rID, err := s.sendNewMessage(sender, o, incomingMessageID.timestamp)
		if err != nil {
			s.log.Printf("Unable to send new message triggered by [%s]: %v\n", incomingMessageID, err)
		} else {
			// Add the new updated message to the new responses
			newResponseByActionID[o.pluginIdentifier] = rID
		}
	}

	if len(newResponseByActionID) > 0 {
		s.log.Debugf("Adding responses to triggering message [%s]: %s", incomingMessageID, newResponseByActionID)

		// Add current responses for that triggering message
		s.triggeringMsgToResponse.Add(incomingMessageID, newResponseByActionID)
	}
}

// sendNewMessage sends a new outgoingMsg and waits for the response to return that message's identifier
func (s *Slackscot) sendNewMessage(sender messageSender, o *OutgoingMessage, threadTS string) (rID SlackMessageID, err error) {
	options := []slack.MsgOption{slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionUser(s.selfID), slack.MsgOptionAsUser(true)}
	if s.config.GetBool(config.ThreadedRepliesKey) {
		options = append(options, slack.MsgOptionTS(threadTS))

		if s.config.GetBool(config.BroadcastThreadedRepliesKey) {
			options = append(options, slack.MsgOptionBroadcast())
		}
	}

	channelID, newOutgoingMsgTimestamp, _, err := sender.SendMessage(o.OutgoingMessage.Channel, options...)
	rID = SlackMessageID{channelID: channelID, timestamp: newOutgoingMsgTimestamp}

	return rID, err
}

// updateExistingMessage updates an existing message with the content of a newly triggered OutgoingMessage
func (s *Slackscot) updateExistingMessage(updater messageUpdater, r SlackMessageID, o *OutgoingMessage) (rID SlackMessageID, err error) {
	channelID, newOutgoingMsgTimestamp, _, err := updater.UpdateMessage(r.channelID, r.timestamp, slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionUser(s.selfID), slack.MsgOptionAsUser(true))
	rID = SlackMessageID{channelID: channelID, timestamp: newOutgoingMsgTimestamp}

	return rID, err
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
func (s *Slackscot) routeMessage(m *slack.Msg) (responses []*OutgoingMessage) {
	// Built regex to detect if message was directed at "us"
	r, _ := regexp.Compile("^(<@" + s.selfID + ">|@?" + s.selfName + "):? (.+)")
	matches := r.FindStringSubmatch(m.Text)

	responses = make([]*OutgoingMessage, 0)

	// Ignore messages send by "us"
	if m.User == s.selfID || m.BotID == s.selfID {
		s.log.Debugf("Ignoring message from user [%s] because that's \"us\" [%s]", m.User, s.selfID)

		return responses
	}

	if len(matches) == 3 {
		if s.commandsWithID != nil {
			omsgs := handleCommand(s.defaultAction, s.commandsWithID, matches[2], m, reply)
			if len(omsgs) > 0 {
				responses = append(responses, omsgs...)
			}
		}
	} else if strings.HasPrefix(m.Channel, "D") {
		if s.commandsWithID != nil {
			omsgs := handleCommand(s.defaultAction, s.commandsWithID, m.Text, m, directReply)
			if len(omsgs) > 0 {
				responses = append(responses, omsgs...)
			}
		}
	} else if s.hearActionsWithID != nil {
		omsgs := handleMessage(s.hearActionsWithID, m.Text, m, send)
		if len(omsgs) > 0 {
			responses = append(responses, omsgs...)
		}
	}

	return responses
}

// handleCommand handles a command by trying a match with all known actions. If no match is found, the default action is invoked
// Note that in the case of the default action being executed, the return value is still false to indicate no bot actions were triggered
func handleCommand(defaultAnswer Answerer, actions []ActionDefinitionWithID, content string, m *slack.Msg, rs responseStrategy) (outMsgs []*OutgoingMessage) {
	outMsgs = handleMessage(actions, content, m, rs)
	if len(outMsgs) == 0 {
		response := defaultAnswer(m)

		slackOutMsg := rs(m, response)
		outMsg := OutgoingMessage{OutgoingMessage: slackOutMsg, pluginIdentifier: "default"}
		return []*OutgoingMessage{&outMsg}
	}

	return outMsgs
}

// processMessage loops over all action definitions and invokes its action if the incoming message matches it's regular expression
// Note that more than one action can be triggered during the processing of a single message
func handleMessage(actions []ActionDefinitionWithID, t string, m *slack.Msg, rs responseStrategy) (outMsgs []*OutgoingMessage) {
	outMsgs = make([]*OutgoingMessage, 0)

	for _, action := range actions {
		matches := action.Match(t, m)

		if matches {
			response := action.Answer(m)

			if response != "" {
				slackOutMsg := rs(m, response)
				outMsg := OutgoingMessage{OutgoingMessage: slackOutMsg, pluginIdentifier: action.id}

				outMsgs = append(outMsgs, &outMsg)
			}
		}
	}

	return outMsgs
}

// newOutgoingMessage creates a new slack.OutgoingMessage for a given channelID and text content
func newOutgoingMessage(channelID string, text string) *slack.OutgoingMessage {
	om := slack.OutgoingMessage{
		Type:    "message",
		Channel: channelID,
		Text:    text,
	}

	return &om
}

// reply sends a reply to the user (using @user) who sent the message on the channel it was sent on
func reply(replyToMsg *slack.Msg, response string) *slack.OutgoingMessage {
	return newOutgoingMessage(replyToMsg.Channel, fmt.Sprintf("<@%s>: %s", replyToMsg.User, response))
}

// directReply sends a reply to a direct message (which is internally a channel id for slack). It is essentially
// the same as send but it's kept separate for clarity
func directReply(replyToMsg *slack.Msg, response string) *slack.OutgoingMessage {
	return send(replyToMsg, response)
}

// send creates a message to be sent on the same channel as received (which can be a direct message since
// slack internally uses a channel id for private conversations)
func send(replyToMsg *slack.Msg, response string) *slack.OutgoingMessage {
	return newOutgoingMessage(replyToMsg.Channel, response)
}
