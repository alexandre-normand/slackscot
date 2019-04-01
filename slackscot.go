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
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	defaultLogPrefix = "slackscot: "
	defaultLogFlag   = log.Lshortfile | log.LstdFlags
)

// Action types
const (
	commandType    = "command"
	hearActionType = "hearAction"
)

// Slackscot represents what defines a Slack Mascot (mostly, a name and its plugins)
type Slackscot struct {
	name                    string
	config                  *viper.Viper
	defaultAction           Answerer
	plugins                 []*Plugin
	triggeringMsgToResponse *lru.ARCCache

	// Caching self identity used during message processing/filtering
	selfID         string
	selfName       string
	selfUserPrefix string

	// Logger
	log *sLogger
}

// Plugin represents a plugin (its name, action definitions and slackscot injected services)
type Plugin struct {
	Name string

	Commands         []ActionDefinition
	HearActions      []ActionDefinition
	ScheduledActions []ScheduledActionDefinition

	// Those slackscot services are injected post-creation when slackscot is called.
	// A plugin shouldn't rely on those being available during creation
	UserInfoFinder    UserInfoFinder
	Logger            SLogger
	EmojiReactor      EmojiReactor
	FileUploader      FileUploader
	RealTimeMsgSender RealTimeMessageSender
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

// Matcher is the function that determines whether or not an action should be triggered based on a IncomingMessage (which
// includes a slack.Msg and a normalized text content. Note that a match doesn't guarantee that the action should
// actually respond with anything once invoked
type Matcher func(m *IncomingMessage) bool

// Answerer is what gets executed when an ActionDefinition is triggered. To signal the absence of an answer, an action
// should return nil
type Answerer func(m *IncomingMessage) *Answer

// ActionDefinitionWithID holds an action definition along with its identifier string
type ActionDefinitionWithID struct {
	ActionDefinition
	id string
}

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

// ScheduledAction is what gets executed when a ScheduledActionDefinition is triggered (by its ScheduleDefinition)
// In order to do anything, a plugin should define its scheduled actions functions with itself as a receiver
// so the function has access to the injected services
type ScheduledAction func()

// SlackMessageID holds the elements that form a unique message identifier for slack. Technically, slack also uses
// the workspace id as the first part of that unique identifier but since an instance of slackscot only lives within
// a single workspace, that part is left out
type SlackMessageID struct {
	channelID string
	timestamp string
}

// responseStrategy defines how a slack.OutgoingMessage is generated from an Answer
type responseStrategy func(m *IncomingMessage, answer *Answer) *slack.OutgoingMessage

// IncomingMessage holds data for an incoming slack message. In addition to a slack.Msg, it also has
// a normalized text that is the original text stripped from the "<@Mention>" prefix when a message
// is addressed to a slackscot instance. Since commands are usually received either via direct message
// (without @Mention) or on channels with @Mention, the normalized text is useful there to allow plugins
// to have a single version to do Match and Answer against
type IncomingMessage struct {
	// The original slack.Msg text stripped from the "<@Mention>" prefix, if applicable
	NormalizedText string
	slack.Msg
}

// OutgoingMessage holds a plugin generated slack outgoing message along with the plugin identifier
type OutgoingMessage struct {
	*slack.OutgoingMessage

	// Answer from plugins/internal commands
	*Answer

	// The identifier of the source of the outgoing message. The format being: <pluginName>.command[<commandIndex>] (for a command) or <pluginName>.hearAction[actionIndex] (for an hear action)
	pluginActionID string
}

// terminationEvent is an empty struct that is only used for whitebox testing in order to signal slackscot to terminate
// Any such events when executed as part of the normal API would be ignored
type terminationEvent struct {
}

// runDependencies represents all runtime dependencies. Note that they're mostly satisfied by slack.RTM or slack.Client
// but having dependencies used as the smaller interfaces keeps the rest of the code cleaner and easier to test
type runDependencies struct {
	chatDriver        chatDriver
	userInfoFinder    UserInfoFinder
	emojiReactor      EmojiReactor
	fileUploader      FileUploader
	selfInfoFinder    selfInfoFinder
	realTimeMsgSender RealTimeMessageSender
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
	s.defaultAction = func(m *IncomingMessage) *Answer {
		return &Answer{Text: fmt.Sprintf("I don't understand, ask me for \"%s\" to get a list of things I do", helpPluginName)}
	}
	s.log = NewSLogger(log.New(os.Stdout, defaultLogPrefix, defaultLogFlag), v.GetBool(config.DebugKey))

	for _, opt := range options {
		opt(s)
	}

	return s, nil
}

// RegisterPlugin registers a plugin with the Slackscot engine. This should be invoked
// prior to calling Run
func (s *Slackscot) RegisterPlugin(p *Plugin) {
	s.plugins = append(s.plugins, p)
}

// Run starts the Slackscot and loops until the process is interrupted
func (s *Slackscot) Run() (err error) {
	sc := slack.New(
		s.config.GetString(config.TokenKey),
		slack.OptionDebug(s.config.GetBool(config.DebugKey)),
		// TODO: For now, the slackscot logger is propagated to slack for its own logging. This means that the prefix is the same for both. With
		// https://github.com/golang/go/commit/51104cd4d2dab6bdd8bda694c0a9a5613cec3b84 (to be released in 1.12),
		// we should be able to create a new logger to the same file but using a different prefix
		// This is the line to use when 1.12 is officially out:
		//
		// slack.OptionLog(log.New(s.log.logger.Writer(), "slack: ", defaultLogFlag)),
		slack.OptionLog(s.log.logger),
	)

	// This will initiate the connection to the slack RTM and start the reception of messages
	rtm := sc.NewRTM()
	go rtm.ManageConnection()

	// Load time zone location for the scheduler, we just log the error here since we fail to start
	// but we're in a go routine. Hopefully, this should be sufficient for users to figure out the bad
	// configuration
	timeLoc, err := config.GetTimeLocation(s.config)
	if err != nil {
		return err
	}

	// Start scheduling of all plugins' scheduled actions
	go s.startActionScheduler(timeLoc)

	termination := make(chan bool)

	// This is a blocking call so it's running in a goroutine. The way slackscot would usually terminate
	// in a production scenario is by receiving a termination signal which
	go s.runInternal(rtm.IncomingEvents, termination, &runDependencies{chatDriver: sc, userInfoFinder: sc, emojiReactor: sc, fileUploader: NewFileUploader(sc), selfInfoFinder: rtm, realTimeMsgSender: rtm}, true)

	// Wait for termination
	<-termination

	return nil
}

// runInternal handles all incoming events and acts as the main loop. It will essentially
// always process events as long as the process isn't interrupted. Normally, this happens
// by a kill signal being sent and slackscot gets notified and closes the events channel which
// terminates this loop and shuts down gracefully
func (s *Slackscot) runInternal(events <-chan slack.RTMEvent, termination chan<- bool, deps *runDependencies, productionMode bool) {
	// Ensure we send a termination signal on the channel to unblock the main thread and exit
	defer func() {
		termination <- true
	}()

	// Register to receive a notification for a termination signal which will, in turn, send a termination message to the
	// termination channel
	go s.watchForTerminationSignalToAbort(termination)

	// Start by adding the help command now that we know all plugins have been registered
	helpPlugin := newHelpPlugin(s.name, VERSION, s.config, s.plugins)
	s.RegisterPlugin(&helpPlugin.Plugin)

	// Inject services into plugins before starting to process events
	s.injectServicesToPlugins(deps.userInfoFinder, s.log, deps.emojiReactor, deps.fileUploader, deps.realTimeMsgSender)

	for msg := range events {
		switch e := msg.Data.(type) {
		case *slack.ConnectedEvent:
			s.log.Printf("Infos: %v\n", e.Info)
			s.log.Printf("Connection counter: %d\n", e.ConnectionCount)
			s.cacheSelfIdentity(deps.selfInfoFinder)

		case *slack.MessageEvent:
			s.processMessageEvent(deps.chatDriver, e)

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
func (s *Slackscot) injectServicesToPlugins(loadingUserInfoFinder UserInfoFinder, logger SLogger, emojiReactor EmojiReactor, fileUploader FileUploader, msgSender RealTimeMessageSender) (err error) {
	userInfoFinder, err := NewCachingUserInfoFinder(s.config, loadingUserInfoFinder, logger)
	if err != nil {
		return err
	}

	for _, p := range s.plugins {
		p.Logger = logger
		p.UserInfoFinder = userInfoFinder
		p.EmojiReactor = emojiReactor
		p.FileUploader = fileUploader
		p.RealTimeMsgSender = msgSender
	}

	return nil
}

// watchForTerminationSignalToAbort waits for a SIGTERM or SIGINT and sends a termination signal on the termination channel to finish
// the main Run() loop and terminate cleanly. Note that this is meant to run in a go routine given that this is blocking
func (s *Slackscot) watchForTerminationSignalToAbort(termination chan<- bool) {
	tSignals := make(chan os.Signal, 1)
	// Register to be notified of termination signals so we can abort
	signal.Notify(tSignals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-tSignals

	s.log.Debugf("Received termination signal [%s], closing RTM's incoming events channel to terminate processing\n", sig)
	termination <- true
}

// getActionID returns a formatted identifier for an action. It includes the plugin name,
// the action type (command or hear action) and its index within the list of such actions for the plugin
//
// The identifier remains the same for the duration of an execution but might change if the slackscot instance
// reorders/replaces actions. Since the identifier isn't used for any durable functionality at the moment, this seems
// adequate. If this ever changes, we might formalize an action identifier that could be generated by users and validated
// to be unique.
func getActionID(pluginName string, actionType string, index int) (actionID string) {
	return fmt.Sprintf("%s.%s[%d]", pluginName, actionType, index)
}

// cacheSelfIdentity gets "our" identity and keeps the selfID and selfName to avoid having to look it up every time
func (s *Slackscot) cacheSelfIdentity(selfInfoFinder selfInfoFinder) {
	s.selfID = selfInfoFinder.GetInfo().User.ID
	s.selfName = selfInfoFinder.GetInfo().User.Name
	s.selfUserPrefix = fmt.Sprintf("<@%s> ", s.selfID)

	s.log.Debugf("Caching self id [%s], self name [%s] and self prefix [%s]\n", s.selfID, s.selfName, s.selfUserPrefix)
}

// startActionScheduler creates all ScheduledActionDefinition from all plugins and registers them with the scheduler
// Very importantly, it also starts the scheduler
func (s *Slackscot) startActionScheduler(timeLoc *time.Location) {
	gocron.ChangeLoc(timeLoc)
	sc := gocron.NewScheduler()

	for _, p := range s.plugins {
		if p.ScheduledActions != nil {
			for _, sa := range p.ScheduledActions {
				j, err := schedule.NewJob(sc, sa.Schedule)
				if err == nil {
					s.log.Debugf("Adding job [%v] to scheduler\n", j)
					err = j.Do(sa.Action)
				}

				if err != nil {
					s.log.Printf("Error: failed to schedule job for scheduled action ['%s' - %s]: %v\n", sa.Schedule, sa.Description, err)
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

	s.log.Debugf("Processing event: %v", msgEvent)

	if !isReply && msgEvent.Type == "message" {
		if msgEvent.SubType == "message_deleted" {
			s.processDeletedMessage(driver, msgEvent)
		} else {
			if msgEvent.SubType == "message_changed" {
				s.processUpdatedMessage(driver, msgEvent)
			} else if msgEvent.SubType != "message_replied" {
				s.processNewMessage(driver, msgEvent)
			}
		}
	}
}

// processUpdatedMessage processes changed messages. This is a more complicated scenario but slackscot handles it by doing the following:
// 1. If the message isn't present in the triggering message cache, we process it as we would any other regular new message (check if it triggers an action and sends responses accordingly)
// 2. If the message is present in cache, we had pre-existing responses so we handle this by updating responses on a plugin action basis. A plugin action that isn't triggering anymore gets its previous
//    response deleted while a still triggering response will result in a message update. Newly triggered actions will be sent out as new messages.
// 3. The new state of responses replaces the previous one for the triggering message in the cache
func (s *Slackscot) processUpdatedMessage(driver chatDriver, m *slack.MessageEvent) {
	incomingMessageID := SlackMessageID{channelID: m.Channel, timestamp: m.Timestamp}
	editedMsgID := SlackMessageID{channelID: m.Channel, timestamp: m.SubMessage.Timestamp}

	s.log.Debugf("Updated message: [%s], does cache contain it => [%t]", editedMsgID, s.triggeringMsgToResponse.Contains(editedMsgID))

	if cachedResponses, exists := s.triggeringMsgToResponse.Get(editedMsgID); exists {
		s.processUpdatedMessageWithCachedResponses(driver, m, editedMsgID, cachedResponses.(map[string]SlackMessageID))
	} else {
		outMsgs := s.routeMessage(m)

		s.sendOutgoingMessages(driver, incomingMessageID, outMsgs)
	}
}

// processUpdatedMessageWithCachedResponses handles a message update for which we still have cached responses in cache. This is where we take care of deleting responses that are no longer
// triggering the action they're coming from, updating the reactions for still triggering plugin actions as well as sending new reactions for plugin actions that are now triggering
func (s *Slackscot) processUpdatedMessageWithCachedResponses(driver chatDriver, m *slack.MessageEvent, editedMsgID SlackMessageID, cachedResponses map[string]SlackMessageID) {
	newResponseByActionID := make(map[string]SlackMessageID)

	outMsgs := s.routeMessage(m)
	s.log.Debugf("Detected %d existing responses to message [%s]\n", len(cachedResponses), editedMsgID)

	for _, o := range outMsgs {
		// We had a previous response for that same plugin action so edit it instead of posting a new message
		if r, ok := cachedResponses[o.pluginActionID]; ok {
			s.log.Debugf("Trying to update response at [%s] with message [%s]\n", r, o.OutgoingMessage.Text)

			rID, err := s.updateExistingMessage(driver, r, o)
			if err != nil {
				s.log.Printf("Unable to update message [%s] to triggering message [%s]: %v\n", r, editedMsgID, err)
			} else {
				// Add the new updated message to the new responses
				newResponseByActionID[o.pluginActionID] = rID

				// Remove entries for plugin actions as we process them so that we can detect afterwards if a plugin isn't triggering
				// anymore (to delete those responses).
				delete(cachedResponses, o.pluginActionID)
			}
		} else {
			s.log.Debugf("New response triggered to updated message [%s] [%s]: [%s]\n", o.OutgoingMessage.Text, r, o.OutgoingMessage.Text)

			// It's a new message for that action so post it as a new message
			rID, err := s.sendNewMessage(driver, o, editedMsgID.timestamp)
			if err != nil {
				s.log.Printf("Unable to send new message to updated message [%s]: %v\n", r, err)
			} else {
				// Add the new updated message to the new responses
				newResponseByActionID[o.pluginActionID] = rID
			}
		}
	}

	// Delete any previous triggered responses that aren't triggering anymore
	for pa, r := range cachedResponses {
		s.log.Debugf("Deleting previous response [%s] on a now non-triggered plugin action [%s]\n", r, pa)
		driver.DeleteMessage(r.channelID, r.timestamp)
	}

	// Since the updated message now has new responses, update the entry with those or remove if no actions are triggered
	if len(newResponseByActionID) > 0 {
		s.log.Debugf("Updating responses to edited message [%s]\n", editedMsgID)
		s.triggeringMsgToResponse.Add(editedMsgID, newResponseByActionID)
	} else {
		s.log.Debugf("Deleting entry for edited message [%s] since no more triggered response\n", editedMsgID)
		s.triggeringMsgToResponse.Remove(editedMsgID)
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
func (s *Slackscot) processNewMessage(msgSender messageSender, m *slack.MessageEvent) {
	incomingMessageID := SlackMessageID{channelID: m.Channel, timestamp: m.Timestamp}
	outMsgs := s.routeMessage(m)

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
			newResponseByActionID[o.pluginActionID] = rID
		}
	}

	if len(newResponseByActionID) > 0 {
		s.log.Debugf("Adding responses to triggering message [%s]: %s", incomingMessageID, newResponseByActionID)

		// Add current responses for that triggering message
		s.triggeringMsgToResponse.Add(incomingMessageID, newResponseByActionID)
	}
}

// sendNewMessage sends a new outgoingMsg and waits for the response to return that message's identifier
func (s *Slackscot) sendNewMessage(sender messageSender, o *OutgoingMessage, defaultThreadTS string) (rID SlackMessageID, err error) {
	sendOpts := ApplyAnswerOpts(o.Options...)
	options := []slack.MsgOption{slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionUser(s.selfID), slack.MsgOptionAsUser(true)}
	if s.config.GetBool(config.ThreadedRepliesKey) || cast.ToBool(sendOpts[ThreadedReplyOpt]) {
		if threadTS := cast.ToString(sendOpts[ThreadTimestamp]); threadTS != "" {
			options = append(options, slack.MsgOptionTS(threadTS))
		} else {
			options = append(options, slack.MsgOptionTS(defaultThreadTS))
		}

		if s.config.GetBool(config.BroadcastThreadedRepliesKey) || cast.ToBool(sendOpts[BroadcastOpt]) {
			options = append(options, slack.MsgOptionBroadcast())
		}
	}

	// Add any block kit content blocks, if any
	if len(o.ContentBlocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(o.ContentBlocks...))
	}

	channelID, newOutgoingMsgTimestamp, _, err := sender.SendMessage(o.OutgoingMessage.Channel, options...)
	rID = SlackMessageID{channelID: channelID, timestamp: newOutgoingMsgTimestamp}

	return rID, err
}

// updateExistingMessage updates an existing message with the content of a newly triggered OutgoingMessage
func (s *Slackscot) updateExistingMessage(updater messageUpdater, r SlackMessageID, o *OutgoingMessage) (rID SlackMessageID, err error) {
	options := []slack.MsgOption{slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionUser(s.selfID), slack.MsgOptionAsUser(true)}
	// Add any block kit content blocks, if any
	if len(o.ContentBlocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(o.ContentBlocks...))
	}

	channelID, newOutgoingMsgTimestamp, _, err := updater.UpdateMessage(r.channelID, r.timestamp, options...)
	rID = SlackMessageID{channelID: channelID, timestamp: newOutgoingMsgTimestamp}

	return rID, err
}

// normalizeIncomingMessage normalizes a main message event and its sub message to form what would be an intuitive message to process for
// a bot. When it's a regular message (no SubMessage), a copy is returned unchanged. For other cases (like message updates),
// a message with the new updated text (since we're talking about a changed message) along with the channel being the one where the message
// is visible and with the user correctly set to the person who updated/sent the message. We also take the timestamp of the original message to make
// it convenient for plugins using the timestamp to know that they're looking at the same one they've seen before. Regarding this timestamp, we sort of treat
// is like the identifier that it is which would be initialized when first posted.
//
// Essentially, take everything from the main message except for the text, user and timestamp that is set on the SubMessage, if present.
func normalizeIncomingMessage(m *slack.MessageEvent) (normalized *slack.Msg) {
	normalized = new(slack.Msg)
	*normalized = m.Msg

	if m.SubMessage != nil {
		normalized.Text = m.SubMessage.Text
		normalized.User = m.SubMessage.User
		normalized.Timestamp = m.SubMessage.Timestamp
	}
	return normalized
}

// resolveThreadTimestamp returns the proper thread timestamp to use for a new message.
// In the case of a response to a message on a thread, that value would be the original
// thread timestamp. Otherwise, this would be the timestamp of the message responded to.
// The function also returns whether or not the incoming message is a threaded message
// which would indicate that we want any answer to get posted to that thread instead of the
// main channel
func resolveThreadTimestamp(m *slack.Msg) (threadTs string, isThreadedMessage bool) {
	if m.ThreadTimestamp != "" {
		return m.ThreadTimestamp, true
	}

	return m.Timestamp, false
}

// routeMessage handles routing the message to commands or hear actions according to the context
// The rules are the following:
// 	1. If the message is on a channel with a direct mention to us (@name), we route to commands
// 	2. If the message is a direct message to us, we route to commands
// 	3. If the message is on a channel without mention (regular conversation), we route to hear actions
func (s *Slackscot) routeMessage(me *slack.MessageEvent) (responses []*OutgoingMessage) {
	m := normalizeIncomingMessage(me)

	responses = make([]*OutgoingMessage, 0)

	// Ignore messages_replied and messages send by "us"
	if m.User == s.selfID || m.BotID == s.selfID {
		s.log.Debugf("Ignoring message from user [%s] because that's \"us\" [%s]", m.User, s.selfID)

		return responses
	}

	// Try commands or hear actions depending on the format of the message
	if isCommand, isDM := isCommand(m, s.selfUserPrefix); isCommand {
		replyStrategy := reply
		if isDM {
			replyStrategy = directReply
		}

		for _, p := range s.plugins {
			inMsg := s.newIncomingMsgWithNormalizedText(p, m)

			outMsgs := tryPluginActions(p.Name, commandType, p.Commands, inMsg, replyStrategy)
			responses = append(responses, outMsgs...)
		}

		if len(responses) == 0 {
			responses = append(responses, defaultAnswer(s.defaultAction, s.newIncomingMsgWithNormalizedText(nil, m), replyStrategy))
		}
	} else {
		for _, p := range s.plugins {
			inMsg := s.newIncomingMsgWithNormalizedText(p, m)

			outMsgs := tryPluginActions(p.Name, hearActionType, p.HearActions, inMsg, send)
			responses = append(responses, outMsgs...)
		}
	}

	return responses
}

func defaultAnswer(answerDefault Answerer, inMsg *IncomingMessage, rs responseStrategy) (o *OutgoingMessage) {
	answer := answerDefault(inMsg)
	answer.useExistingThreadIfAny(inMsg)

	slackOutMsg := rs(inMsg, answer)

	return newOutMessageForAnswer(slackOutMsg, "default", answer)
}

// newIncomingMsgWithNormalizedText creates a new IncomingMessage and generates the normalized text for plugins
// to have a normalized view of the message regardless of context. This includes having the text stripped of the "<@user>"
// for commands sent via a directed message on a channel
// TODO normalize should handle removing the namespace and user id for bot mentions depending on the case
func (s *Slackscot) newIncomingMsgWithNormalizedText(p *Plugin, m *slack.Msg) (incomingMsg *IncomingMessage) {
	incomingMsg = new(IncomingMessage)
	incomingMsg.NormalizedText = m.Text
	incomingMsg.Msg = *m
	if isCommand, isDM := isCommand(m, s.selfUserPrefix); isCommand && !isDM {
		incomingMsg.NormalizedText = strings.TrimPrefix(m.Text, s.selfUserPrefix)
	}

	return incomingMsg
}

// isCommand returns true if the slack message is to be interpreted as a command rather than a normal message
// subject to be handled by hear actions
func isCommand(m *slack.Msg, selfUserPrefix string) (isCommand bool, isDirectMsg bool) {
	isDirectMsg = strings.HasPrefix(m.Channel, "D")
	return strings.HasPrefix(m.Text, selfUserPrefix) || isDirectMsg, isDirectMsg
}

// useExistingThreadIfAny sets the option on an Answer to reply in the existing thread if there is one
func (a *Answer) useExistingThreadIfAny(m *IncomingMessage) {
	// If the message we're reacting to is happening on an existing thread, make sure we reply on that
	// thread too and avoid the awkward situation of responding on the parent channel
	threadTimestamp, threaded := resolveThreadTimestamp(&m.Msg)
	if threaded {
		a.Options = append(a.Options, AnswerInExistingThread(threadTimestamp))
	}
}

// tryPluginActions loops over all action definitions and invokes its action if the incoming message matches it's regular expression
// Note that more than one action can be triggered during the processing of a single message
func tryPluginActions(pluginName string, actionType string, actions []ActionDefinition, m *IncomingMessage, rs responseStrategy) (outMsgs []*OutgoingMessage) {
	outMsgs = make([]*OutgoingMessage, 0)

	for i, action := range actions {
		matches := action.Match(m)

		if matches {
			answer := action.Answer(m)

			if answer != nil {
				answer.useExistingThreadIfAny(m)
				slackOutMsg := rs(m, answer)

				outMsg := newOutMessageForAnswer(slackOutMsg, getActionID(pluginName, actionType, i), answer)
				outMsgs = append(outMsgs, outMsg)
			}
		}
	}

	return outMsgs
}

// newOutMessageForAnswer creates a new internal OutgoingMessage for the given Answer
func newOutMessageForAnswer(o *slack.OutgoingMessage, id string, answer *Answer) (om *OutgoingMessage) {
	om = new(OutgoingMessage)
	om.OutgoingMessage = o
	om.pluginActionID = id
	om.Answer = answer

	return om
}

// newSlackOutgoingMessage creates a new slack.OutgoingMessage for a given channelID and text content
func newSlackOutgoingMessage(channelID string, text string) *slack.OutgoingMessage {
	om := slack.OutgoingMessage{
		Type:    "message",
		Channel: channelID,
		Text:    text,
	}

	return &om
}

// reply sends a reply to the user (using @user) who sent the message on the channel it was sent on
func reply(replyToMsg *IncomingMessage, answer *Answer) *slack.OutgoingMessage {
	return newSlackOutgoingMessage(replyToMsg.Channel, fmt.Sprintf("<@%s>: %s", replyToMsg.User, answer.Text))
}

// directReply sends a reply to a direct message
func directReply(replyToMsg *IncomingMessage, answer *Answer) *slack.OutgoingMessage {
	// Force a non-threaded reply since we're in a direct conversation. Instead of overriding
	// all existing options, we just add the one to override the threading here
	answer.Options = append(answer.Options, AnswerWithoutThreading())

	return send(replyToMsg, answer)
}

// send creates a message to be sent on the same channel as received (which can be a direct message since
// slack internally uses a channel id for private conversations)
func send(replyToMsg *IncomingMessage, answer *Answer) *slack.OutgoingMessage {
	return newSlackOutgoingMessage(replyToMsg.Channel, answer.Text)
}
