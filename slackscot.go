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
	"hash"
	"hash/fnv"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
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
	selfBotID      string
	selfName       string
	selfUserPrefix string

	// Runtime configuration options
	namespaceCommands bool

	// Slack options to apply on Run()
	slackOpts []slack.Option

	// Logger
	log *sLogger

	// Resources to close on shutdown
	closers []io.Closer

	// Test mode which defines whether or not the bot reacts to terminationEvents
	testMode bool

	// workerQueues with partition keyed by the hash of the incoming message id
	// so that processing of messages (new, updates and deletes) are handled by
	// the same work queue therefore ensuring correct ordered processing
	// of those events
	workerQueues []chan slack.MessageEvent

	// workerTerminationChans are channels receiving a termination signal for each
	// workerQueue
	workerTerminationChans []chan bool

	// Termination channel
	terminationCh chan bool

	// hash function to direct message processing to partitions
	hasher hash.Hash32
}

// Plugin represents a plugin (its name, action definitions and slackscot injected services)
//
// Set NamespaceCommands to true if the plugin's commands are to be namespaced by slackscot.
// This means that commands will be first checked for a prefix that matches the plugin name.
// Since that's all handled by slackscot, a plugin should be written with matching only
// considering what comes after the namespace. For example, a plugin with name make would have
// a coffee command be something like
//
//   Match: func(m *IncomingMessage) bool {
//       return strings.HasPrefix(m.NormalizedText, "coffee ")
//   },
//   Usage:       "coffee `<when>`",
//   Description: "Make coffee",
//   Answer: func(m *IncomingMessage) *Answer {
//       when := strings.TrimPrefix(m.NormalizedText, "coffee ")
//       return &Answer{Text: fmt.Sprintf("coffee will be reading %s", when))}}
//   }
//
// In this example, if namespacing is enabled, a user would trigger the command with a message such as:
//   <@slackscotID> make coffee in 10 minutes
// Note that the plugin itself doesn't need to concern itself with the namespace in the matching or answering
// as the NormalizedText has been formatted to be stripped of namespacing whether or not that's enabled and slackscot
// will have made sure the namespace matched if enabled.
//
// At runtime, instances of slackscot can request to disregard namespacing with OptionNoPluginNamespacing (for example, to run a single plugin and simplify usage).
type Plugin struct {
	Name string

	NamespaceCommands bool // Set to true for slackscot-managed namespacing of commands where the namespace/prefix to all commands is set to the plugin name

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

	// The slack.Client is injected post-creation. It gives access to all the https://godoc.org/github.com/nlopes/slack#Client.
	// Plugin writers might want to check out https://godoc.org/github.com/nlopes/slack/slacktest to create a slack test server in order
	// to mock a slack server to test plugins using the SlackClient.
	SlackClient *slack.Client
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

// IsMsgModifiable returns true if this slack message id can be used to update/delete the message.
// In practice, ephemeral messages don't have a channel ID and can't be deleted/updated so this would
// be a case where IsMsgModifiable would return false
func (sid SlackMessageID) IsMsgModifiable() bool {
	return sid.channelID != "" && sid.timestamp != ""
}

// String returns the string representation of a SlackMessageID
func (sid SlackMessageID) String() string {
	return fmt.Sprintf("%s-%s", sid.channelID, sid.timestamp)
}

// responseStrategy defines how a slack.OutgoingMessage is generated from an Answer
type responseStrategy func(m *IncomingMessage, answer *Answer) slack.OutgoingMessage

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
	slack.OutgoingMessage

	// Answer from plugins/internal commands
	Answer

	// The identifier of the source of the outgoing message. The format being: <pluginName>.command[<commandIndex>] (for a command) or <pluginName>.hearAction[actionIndex] (for an hear action)
	pluginActionID string
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
	slackClient       *slack.Client
}

// Option defines an option for a Slackscot
type Option func(*Slackscot)

// OptionLog sets a logger for Slackscot
func OptionLog(logger *log.Logger) Option {
	return func(s *Slackscot) {
		s.log.logger = logger
	}
}

// OptionWithSlackOption adds a slack.Option to apply on the slack client
func OptionWithSlackOption(opt slack.Option) Option {
	return func(s *Slackscot) {
		s.slackOpts = append(s.slackOpts, opt)
	}
}

// OptionNoPluginNamespacing disables plugin command namespacing for this instance. This means
// that namespacing plugin candidates will run without any extra plugin name matching required
// This is useful to simplify command usage for instances running a single plugin
func OptionNoPluginNamespacing() Option {
	return func(s *Slackscot) {
		s.namespaceCommands = false
	}
}

// OptionLogfile sets a logfile for Slackscot while using the other default logging prefix and options
func OptionLogfile(logfile *os.File) Option {
	return func(s *Slackscot) {
		s.log.logger = log.New(logfile, defaultLogPrefix, defaultLogFlag)
		s.slackOpts = append(s.slackOpts, slack.OptionLog(s.log.logger))
	}
}

// OptionTestMode sets the instance in test mode which instructs it to react to a goodbye event to terminate
// its execution. It is meant to be used for testing only and mostly in conjunction with github.com/nlopes/slack/slacktest.
// Very importantly, the termination message must be formed correctly so that the slackscot instance terminates
// correctly for tests to actually terminate.
//
// Here's an example:
//
//  testServer := slacktest.NewTestServer()
//  testServer.Handle("/channels.create", slacktest.Websocket(func(conn *websocket.Conn) {
//      // Trigger a termination on any API call to channels.create
// 	    slacktest.RTMServerSendGoodbye(conn)
//  }))
//  testServer.Start()
//  defer testServer.Stop()
//
//  termination := make(chan bool)
//  s, err := New("BobbyTables", config.NewViperWithDefaults(), OptionWithSlackOption(slack.OptionAPIURL(testServer.GetAPIURL())), OptionTestMode(termination))
//  require.NoError(t, err)
//
//  tp := newTestPlugin()
//  s.RegisterPlugin(tp)
//
//  go s.Run()
//
//  // TODO: Use the testserver to send events and messages and assert your plugin's behavior
//
//  // Send this event to the testServer's websocket. This gets transformed into a
//  // slack.DisconnectedEvent with Cause equal to slack.ErrRTMGoodbye that slackscot will
//  // interpret as a signal to self-terminate
//  testServer.SendToWebsocket("{\"type\":\"goodbye\"}")
//
//  // Wait for slackscot to terminate
//  <-termination
func OptionTestMode(terminationCh chan bool) Option {
	return func(s *Slackscot) {
		s.testMode = true
		s.terminationCh = terminationCh
	}
}

// NewSlackscot creates a new slackscot from an array of plugins and a name
//
// Deprecated: Use New instead. Will be removed in 2.0.0
func NewSlackscot(name string, v *viper.Viper, options ...Option) (s *Slackscot, err error) {
	return New(name, v, options...)
}

// New creates a new slackscot from an array of plugins and a name
func New(name string, v *viper.Viper, options ...Option) (s *Slackscot, err error) {
	s = new(Slackscot)

	s.triggeringMsgToResponse, err = lru.NewARC(v.GetInt(config.ResponseCacheSizeKey))
	if err != nil {
		return nil, err
	}

	s.name = name
	s.config = v
	s.namespaceCommands = true
	s.testMode = false
	s.closers = make([]io.Closer, 0)
	s.defaultAction = func(m *IncomingMessage) *Answer {
		return &Answer{Text: fmt.Sprintf("I don't understand. Ask me for \"%s\" to get a list of things I do", helpPluginName)}
	}
	s.log = NewSLogger(log.New(os.Stdout, defaultLogPrefix, defaultLogFlag), v.GetBool(config.DebugKey))
	s.workerQueues = make([]chan slack.MessageEvent, s.config.GetInt(config.MessageProcessingPartitionCount))
	for i := range s.workerQueues {
		s.workerQueues[i] = make(chan slack.MessageEvent, s.config.GetInt(config.MessageProcessingBufferedMessageCount))
	}
	s.workerTerminationChans = make([]chan bool, s.config.GetInt(config.MessageProcessingPartitionCount))
	for i := range s.workerTerminationChans {
		s.workerTerminationChans[i] = make(chan bool)
	}
	s.hasher = fnv.New32a()

	s.slackOpts = make([]slack.Option, 0)
	s.slackOpts = append(s.slackOpts, slack.OptionDebug(s.config.GetBool(config.DebugKey)))
	s.slackOpts = append(s.slackOpts, slack.OptionLog(log.New(s.log.logger.Writer(), "slack: ", defaultLogFlag)))

	for _, opt := range options {
		opt(s)
	}

	return s, nil
}

// Close closes all closers of this slackscot. The first error that occurs
// during a Close is returned but regardless, all closers are attempted
// to be closed
func (s *Slackscot) Close() (err error) {
	for _, c := range s.closers {
		if err == nil {
			err = c.Close()
		} else {
			c.Close()
		}
	}

	return err
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
		s.slackOpts...,
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

	// runInternal is blocking call so it's running in a goroutine. The way slackscot would usually terminate
	// in a production scenario is by its process getting killed which would result in a last message sent on the termination channel
	if s.terminationCh != nil {
		// Start the main processing and send the termination to the externally defined termination channel (so a test can block and wait for processing after sending all of its test messages)
		go s.runInternal(rtm.IncomingEvents, &runDependencies{chatDriver: sc, userInfoFinder: sc, emojiReactor: sc, fileUploader: NewFileUploader(sc), selfInfoFinder: rtm, realTimeMsgSender: rtm, slackClient: sc})
	} else {
		// This is production and the lifecycle is managed here so we create the termination channel and wait for the termination signal
		s.terminationCh = make(chan bool)

		go s.runInternal(rtm.IncomingEvents, &runDependencies{chatDriver: sc, userInfoFinder: sc, emojiReactor: sc, fileUploader: NewFileUploader(sc), selfInfoFinder: rtm, realTimeMsgSender: rtm, slackClient: sc})

		// Wait for termination
		<-s.terminationCh
	}

	return nil
}

// runInternal handles all incoming events and acts as the main loop. It will essentially
// always process events as long as the process isn't interrupted. Normally, this happens
// by a kill signal being sent and slackscot gets notified and closes the events channel which
// terminates this loop and shuts down gracefully
func (s *Slackscot) runInternal(events <-chan slack.RTMEvent, deps *runDependencies) {
	// Ensure we send a termination signal on the channel to unblock the main thread and exit
	defer func() {
		s.terminationCh <- true
	}()

	// Register to receive a notification for a termination signal which will, in turn, send a termination message to the
	// termination channel
	go s.watchForTerminationSignalToAbort()

	// Start by adding the help command now that we know all plugins have been registered
	helpPlugin := s.newHelpPlugin(VERSION)
	s.RegisterPlugin(&helpPlugin.Plugin)

	// Inject services into plugins before starting to process events
	s.injectServicesToPlugins(deps.userInfoFinder, s.log, deps.emojiReactor, deps.fileUploader, deps.realTimeMsgSender, deps.slackClient)

	// start all worker go routines
	for i := range s.workerQueues {
		go s.processMessages(deps.chatDriver, s.workerQueues[i], s.workerTerminationChans[i])
	}

	for msg := range events {
		switch e := msg.Data.(type) {
		case *slack.ConnectedEvent:
			s.log.Printf("Infos: %v\n", e.Info)
			s.log.Printf("Connection counter: %d\n", e.ConnectionCount)
			err := s.cacheSelfIdentity(deps.selfInfoFinder, deps.userInfoFinder)
			if err != nil {
				s.log.Printf("Error getting self identity: %s", err.Error())
				return
			}

		case *slack.MessageEvent:
			s.dispatchMessageEvent(e)

		case *slack.LatencyReport:
			s.log.Printf("Current latency: %v\n", e.Value)

		case *slack.RTMError:
			s.log.Printf("Error: %s\n", e.Error())

		case *slack.InvalidAuthEvent:
			s.log.Printf("Invalid credentials\n")
			return

		case *slack.DisconnectedEvent:
			if s.testMode && e.Cause != nil && e.Cause == slack.ErrRTMGoodbye {
				s.log.Printf("Received termination event in test mode, terminating\n")
				// Close all processing queues and wait for the terminations
				for _, wq := range s.workerQueues {
					close(wq)
				}

				// Wait for all workers to terminate processing
				for _, tc := range s.workerTerminationChans {
					<-tc
				}

				return
			}
		default:
			// Ignoring other messages
		}
	}
}

// injectServicesToPlugins assembles/creates the services and injects them in all plugins
func (s *Slackscot) injectServicesToPlugins(loadingUserInfoFinder UserInfoFinder, logger SLogger, emojiReactor EmojiReactor, fileUploader FileUploader, msgSender RealTimeMessageSender, slackClient *slack.Client) (err error) {
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
		p.SlackClient = slackClient
	}

	return nil
}

// watchForTerminationSignalToAbort waits for a SIGTERM or SIGINT and sends a termination signal on the termination channel to finish
// the main Run() loop and terminate cleanly. Note that this is meant to run in a go routine given that this is blocking
func (s *Slackscot) watchForTerminationSignalToAbort() {
	tSignals := make(chan os.Signal, 1)
	// Register to be notified of termination signals so we can abort
	signal.Notify(tSignals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-tSignals

	s.log.Debugf("Received termination signal [%s], closing RTM's incoming events channel to terminate processing\n", sig)
	s.terminationCh <- true
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
func (s *Slackscot) cacheSelfIdentity(selfInfoFinder selfInfoFinder, userInfoFinder UserInfoFinder) (err error) {
	s.selfID = selfInfoFinder.GetInfo().User.ID
	s.selfName = selfInfoFinder.GetInfo().User.Name
	user, err := userInfoFinder.GetUserInfo(s.selfID)
	if err != nil {
		return err
	}
	s.selfBotID = user.Profile.BotID
	s.selfUserPrefix = fmt.Sprintf("<@%s> ", s.selfID)

	s.log.Debugf("Caching self id [%s], self name [%s], self bot ID [%s] and self prefix [%s]\n", s.selfID, s.selfName, s.selfBotID, s.selfUserPrefix)
	return nil
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

// processMessages processes messages from a queue and sends a termination signal on terminationChan when done
func (s *Slackscot) processMessages(driver chatDriver, queue chan slack.MessageEvent, terminationChan chan bool) {
	for msg := range queue {
		// reply_to is an field set to 1 sent by slack when a sent message has been acknowledged and should be considered
		// officially sent to others. Therefore, we ignore all of those since it's mostly for clients/UI to show status
		isReply := msg.ReplyTo > 0

		s.log.Debugf("Processing event: %v", msg)

		if !isReply && msg.Type == "message" {
			if msg.SubType == "message_deleted" {
				s.processDeletedMessage(driver, &msg)
			} else {
				if msg.SubType == "message_changed" {
					s.processUpdatedMessage(driver, &msg)
				} else if msg.SubType != "message_replied" {
					s.processNewMessage(driver, &msg)
				}
			}
		}
	}

	terminationChan <- true
}

// dispatchMessageEvent handles high-level processing of all slack message events.
func (s *Slackscot) dispatchMessageEvent(msgEvent *slack.MessageEvent) {
	msgID := SlackMessageID{channelID: msgEvent.Channel, timestamp: msgEvent.Timestamp}

	partition := s.partitionIndexForMsgID(msgID, len(s.workerQueues))

	s.log.Debugf("Dispatching message [%s] to partition [%d]", msgID, partition)
	s.workerQueues[partition] <- *msgEvent
}

// partitionIndexForMsgID returns the partition index for a given message ID
func (s *Slackscot) partitionIndexForMsgID(msgID SlackMessageID, maxValueInclusive int) (partition int) {
	s.hasher.Reset()
	s.hasher.Write([]byte(msgID.channelID))
	res := s.hasher.Sum([]byte(msgID.timestamp))

	s.log.Debugf("Hashing returned [%v] for [%s]", res, msgID)
	// TODO truncate to an int that is between 0 and maxValue
	return 0
}

// getAgeOriginalMsg returns the age of an updated message as defined by the time elapsed between the message
// update (from the time of the current event) and the original message. If there's no previous message, the
// age is 0.
func getAgeOriginalMsg(m *slack.MessageEvent) (age time.Duration, err error) {
	updatedTime, err := strconv.ParseFloat(m.Timestamp, 64)
	if err != nil {
		return time.Duration(0), err
	}

	originalTime, err := strconv.ParseFloat(m.SubMessage.Timestamp, 64)
	if err != nil {
		return time.Duration(0), err
	}

	ageInSeconds := updatedTime - originalTime
	return time.Duration(int64(ageInSeconds)) * time.Second, nil
}

// processUpdatedMessage processes changed messages. This is a more complicated scenario but slackscot handles it by doing the following:
// 1. If the message age is older than the config.MaxAgeHandledMessages threshold, the message update is ignored
// 2. If the message isn't present in the triggering message cache, we process it as we would any other regular new message (check if it triggers an action and sends responses accordingly)
// 3. If the message is present in cache, we had pre-existing responses so we handle this by updating responses on a plugin action basis. A plugin action that isn't triggering anymore gets its previous
//    response deleted while a still triggering response will result in a message update. Newly triggered actions will be sent out as new messages.
// 4. The new state of responses replaces the previous one for the triggering message in the cache
func (s *Slackscot) processUpdatedMessage(driver chatDriver, m *slack.MessageEvent) {
	incomingMessageID := SlackMessageID{channelID: m.Channel, timestamp: m.Timestamp}
	editedMsgID := SlackMessageID{channelID: m.Channel, timestamp: m.SubMessage.Timestamp}

	maxAgeThreshold := s.config.GetDuration(config.MaxAgeHandledMessages)
	msgAge, err := getAgeOriginalMsg(m)
	if err != nil {
		s.log.Printf("Unable to determine max age for message [%v]: %s", m, err.Error())
		return
	}

	if msgAge > maxAgeThreshold {
		s.log.Debugf("Updated message: [%s] has an age of [%s] but the max age for handled messages is [%s]. Skipping...", editedMsgID, msgAge, maxAgeThreshold)
		return
	}

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
			} else if rID.IsMsgModifiable() {
				// Add the new updated message to the new responses if it can be modified later
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
func (s *Slackscot) sendOutgoingMessages(sender messageSender, incomingMessageID SlackMessageID, outMsgs []OutgoingMessage) {
	newResponseByActionID := make(map[string]SlackMessageID)

	for _, o := range outMsgs {
		// Send the message and keep track of our response in cache to be able to update it as needed later
		rID, err := s.sendNewMessage(sender, o, incomingMessageID.timestamp)
		if err != nil {
			s.log.Printf("Unable to send new message triggered by [%s]: %v\n", incomingMessageID, err)
		} else if rID.IsMsgModifiable() {
			// Add the new updated message to the new responses if it's one that can be modified later
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
func (s *Slackscot) sendNewMessage(sender messageSender, o OutgoingMessage, defaultThreadTS string) (rID SlackMessageID, err error) {
	s.log.Printf("Sending new message: %s", o.OutgoingMessage.Text)
	sendOpts := ApplyAnswerOpts(o.Options...)
	options := []slack.MsgOption{slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionAsUser(true)}
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

	// Add ephemeral option if present
	if userID, ok := sendOpts[EphemeralAnswerToOpt]; ok {
		options = append(options, slack.MsgOptionPostEphemeral(userID))
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
func (s *Slackscot) updateExistingMessage(updater messageUpdater, r SlackMessageID, o OutgoingMessage) (rID SlackMessageID, err error) {
	options := []slack.MsgOption{slack.MsgOptionText(o.OutgoingMessage.Text, false), slack.MsgOptionAsUser(true)}
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
func (s *Slackscot) routeMessage(me *slack.MessageEvent) (responses []OutgoingMessage) {
	m := normalizeIncomingMessage(me)

	responses = make([]OutgoingMessage, 0)

	// Ignore messages_replied and messages send by "us"
	if m.User == s.selfID || m.BotID == s.selfID || m.BotID == s.selfBotID {
		s.log.Debugf("Ignoring message from user [%s] / bot ID [%s] because that's \"us\" (userID: [%s], botID: [%s]", m.User, m.BotID, s.selfID, s.selfBotID)

		return responses
	}

	// Try commands or hear actions depending on the format of the message
	if isCommand, isDM := isCommand(m, s.selfUserPrefix); isCommand {
		replyStrategy := reply
		if isDM {
			replyStrategy = directReply
		}

		for _, p := range s.plugins {
			matchedNamespace, inMsg := s.newCmdInMsgWithNormalizedText(p, m)

			if matchedNamespace {
				outMsgs := tryPluginActions(p.Name, commandType, p.Commands, inMsg, replyStrategy)
				responses = append(responses, outMsgs...)
			}
		}

		// Use default answer if this was a message formatted as a command for which we didn't have any answer to
		if len(responses) == 0 {
			responses = append(responses, defaultAnswer(s.defaultAction, s.newIncomingMsgWithNormalizedText(m), replyStrategy))
		}
	} else {
		for _, p := range s.plugins {
			inMsg := s.newIncomingMsgWithNormalizedText(m)

			outMsgs := tryPluginActions(p.Name, hearActionType, p.HearActions, inMsg, send)
			responses = append(responses, outMsgs...)
		}
	}

	return responses
}

// defaultAnswer returns the answer by invocation of the default action
func defaultAnswer(answerDefault Answerer, inMsg *IncomingMessage, rs responseStrategy) (o OutgoingMessage) {
	answer := answerDefault(inMsg)
	answer.useExistingThreadIfAny(inMsg)

	slackOutMsg := rs(inMsg, answer)

	return newOutMessageForAnswer(slackOutMsg, "default", *answer)
}

// newCmdInMsgWithNormalizedText creates a new IncomingMessage for a command and generates the normalized text for plugins
// to have a normalized view of the message regardless of context. For commands part of a Plugin with NamespaceCommands,
// the normalized text removes the namespace if the proper namespace is found. If not, matchedNamespace is false
// and the normalized text is the same as what newIncomingMsgWithNormalizedText would return
func (s *Slackscot) newCmdInMsgWithNormalizedText(p *Plugin, m *slack.Msg) (matchedNamespace bool, inMsg *IncomingMessage) {
	inMsg = s.newIncomingMsgWithNormalizedText(m)
	matchedNamespace = true

	if p != nil && s.namespaceCommands && p.NamespaceCommands {
		namespacePrefix := fmt.Sprintf("%s ", p.Name)
		if matchedNamespace = strings.HasPrefix(inMsg.NormalizedText, namespacePrefix); matchedNamespace {
			inMsg.NormalizedText = strings.TrimPrefix(inMsg.NormalizedText, namespacePrefix)
		}
	}

	return matchedNamespace, inMsg
}

// newIncomingMsgWithNormalizedText creates a new IncomingMessage and generates the normalized text for plugins
// to have a normalized view of the message regardless of context. This includes having the text stripped of the "<@user>"
// for commands sent via a directed message on a channel
func (s *Slackscot) newIncomingMsgWithNormalizedText(m *slack.Msg) (inMsg *IncomingMessage) {
	inMsg = new(IncomingMessage)
	inMsg.NormalizedText = m.Text
	inMsg.Msg = *m
	if isCommand, isDM := isCommand(m, s.selfUserPrefix); isCommand && !isDM {
		inMsg.NormalizedText = strings.TrimPrefix(m.Text, s.selfUserPrefix)
	}

	return inMsg
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
func tryPluginActions(pluginName string, actionType string, actions []ActionDefinition, m *IncomingMessage, rs responseStrategy) (outMsgs []OutgoingMessage) {
	outMsgs = make([]OutgoingMessage, 0)

	for i, action := range actions {
		matches := action.Match(m)

		if matches {
			answer := action.Answer(m)

			if answer != nil {
				answer.useExistingThreadIfAny(m)
				slackOutMsg := rs(m, answer)

				outMsg := newOutMessageForAnswer(slackOutMsg, getActionID(pluginName, actionType, i), *answer)
				outMsgs = append(outMsgs, outMsg)
			}
		}
	}

	return outMsgs
}

// newOutMessageForAnswer creates a new internal OutgoingMessage for the given Answer
func newOutMessageForAnswer(o slack.OutgoingMessage, id string, answer Answer) (om OutgoingMessage) {
	return OutgoingMessage{OutgoingMessage: o, pluginActionID: id, Answer: answer}
}

// newSlackOutgoingMessage creates a new slack.OutgoingMessage for a given channelID and text content
func newSlackOutgoingMessage(channelID string, text string) slack.OutgoingMessage {
	return slack.OutgoingMessage{
		Type:    "message",
		Channel: channelID,
		Text:    text,
	}
}

// reply sends a reply to the user (using @user) who sent the message on the channel it was sent on
func reply(replyToMsg *IncomingMessage, answer *Answer) slack.OutgoingMessage {
	return newSlackOutgoingMessage(replyToMsg.Channel, fmt.Sprintf("<@%s>: %s", replyToMsg.User, answer.Text))
}

// directReply sends a reply to a direct message
func directReply(replyToMsg *IncomingMessage, answer *Answer) slack.OutgoingMessage {
	// Force a non-threaded reply since we're in a direct conversation. Instead of overriding
	// all existing options, we just add the one to override the threading here
	answer.Options = append(answer.Options, AnswerWithoutThreading())

	return send(replyToMsg, answer)
}

// send creates a message to be sent on the same channel as received (which can be a direct message since
// slack internally uses a channel id for private conversations)
func send(replyToMsg *IncomingMessage, answer *Answer) slack.OutgoingMessage {
	return newSlackOutgoingMessage(replyToMsg.Channel, answer.Text)
}
