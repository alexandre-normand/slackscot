package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/alexandre-normand/slackscot/store/mocks"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

type userInfoFinder struct {
}

func (u userInfoFinder) GetUserInfo(userID string) (user *slack.User, err error) {
	return &slack.User{ID: userID, RealName: "Bernard Tremblay"}, nil
}

func TestKarmaMatchesAndAnswers(t *testing.T) {
	testCases := []struct {
		text           string
		channel        string
		expectedAnswer string
	}{
		{"creek++", "Cgeneral", "`creek` just gained a level (`creek`: 1)"},
		{"creek--", "Cgeneral", "`creek` just lost a life (`creek`: 0)"},
		{"the creek++", "Cgeneral", "`creek` just gained a level (`creek`: 1)"},
		{"our creek++ is nice", "Cgeneral", "`creek` just gained a level (`creek`: 2)"},
		{"our creek++ is really nice", "Cgeneral", "`creek` just gained a level (`creek`: 3)"},
		{"oceans++", "Cgeneral", "`oceans` just gained a level (`oceans`: 1)"},
		{"oceans++", "Cgeneral", "`oceans` just gained a level (`oceans`: 2)"},
		{"nettle++", "Cgeneral", "`nettle` just gained a level (`nettle`: 1)"},
		{"salmon++", "Cgeneral", "`salmon` just gained a level (`salmon`: 1)"},
		{"salmon++", "Cgeneral", "`salmon` just gained a level (`salmon`: 2)"},
		{"salmon++", "Cgeneral", "`salmon` just gained a level (`salmon`: 3)"},
		{"salmon++", "Cgeneral", "`salmon` just gained a level (`salmon`: 4)"},
		{"salmon+++", "Cgeneral", "`salmon` just gained 2 levels (`salmon`: 6)"},
		{"salmon++++", "Cgeneral", "`salmon` just gained 3 levels (`salmon`: 9)"},
		{"salmon+++++", "Cgeneral", "`salmon` just gained 4 levels (`salmon`: 13)"},
		{"salmon++++++", "Cgeneral", "`salmon` just gained 5 levels (`salmon`: 18)"},
		{"salmon+++++++", "Cgeneral", "`salmon` just gained 5 levels (`salmon`: 23)"},
		{"dams--", "Cgeneral", "`dams` just lost a life (`dams`: -1)"},
		{"dams--", "Cgeneral", "`dams` just lost a life (`dams`: -2)"},
		{"dams---", "Cgeneral", "`dams` just lost 2 lives (`dams`: -4)"},
		{"dams----", "Cgeneral", "`dams` just lost 3 lives (`dams`: -7)"},
		{"dams-----", "Cgeneral", "`dams` just lost 4 lives (`dams`: -11)"},
		{"dams------", "Cgeneral", "`dams` just lost 5 lives (`dams`: -16)"},
		{"dams-------", "Cgeneral", "`dams` just lost 5 lives (`dams`: -21)"},
		{"<@bot> karma", "Cgeneral", ""},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 1)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 2)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 3)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 4)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 5)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 6)"},
		{"<@U21355> ++", "Cgeneral", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 7)"},
		{"thing ++", "Cgeneral", ""},
		{"don't++", "Cgeneral", "`don't` just gained a level (`don't`: 1)"},
		{"under-the-bridge++", "Cgeneral", "`the-bridge` just gained a level (`the-bridge`: 1)"},
		{"Jean-Michel++", "Cgeneral", "`Jean-Michel` just gained a level (`Jean-Michel`: 1)"},
		{"+----------+", "Cgeneral", ""},
		{"---", "Cgeneral", ""},
		{"+++", "Cgeneral", ""},
		{"salmon++", "Coceanlife", "`salmon` just gained a level (`salmon`: 1)"},
		{"<@bot> top 1", "Cother", "Sorry, no recorded karma found :disappointed:"},
		{"dams--", "Coceanlife", "`dams` just lost a life (`dams`: -1)"},
		{"<@bot> reset", "Coceanlife", "karma all cleared :white_check_mark::boom:"},
		{"<@bot> top 1", "Coceanlife", "Sorry, no recorded karma found :disappointed:"},
	}

	// Create a temp file that will serve as an invalid storage path
	tmpdir, err := ioutil.TempDir("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	storer, err := store.NewLevelDB("karmaTest", tmpdir)
	assert.Nil(t, err)
	defer storer.Close()

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(storer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	if assert.NotNil(t, p) {
		for _, tc := range testCases {
			t.Run(tc.text, func(t *testing.T) {
				assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: tc.channel, Text: tc.text}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
					if len(tc.expectedAnswer) > 0 {
						return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], tc.expectedAnswer)
					}

					return assert.Empty(t, answers, "Reaction to [%s] should be empty but wasn't", tc.text)
				})
			})
		}
	}
}

func TestErrorStoringKarmaRecord(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChannel", "thing").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutSiloString", "myLittleChannel", "thing", "1").Return(fmt.Errorf("can't persist"))

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "thing++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})
}

func TestInvalidSelfKarma(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{User: "U123", Channel: "myLittleChannel", Text: "<@U123>++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "*Attributing yourself karma is frown upon* :face_with_raised_eyebrow:") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.EphemeralAnswerToOpt, Value: "U123"})
	})
}

func TestInvalidStoredKarmaShouldResetValue(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChannel", "thing").Return("abc", nil)
	mockStorer.On("PutSiloString", "myLittleChannel", "thing", "1").Return(nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "thing++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "`thing` just gained a level (`thing`: 1)")
	})
}

func TestErrorGettingList(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{}, fmt.Errorf("can't load karma"))

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> top 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the top [1] things for you. If you must know, this happened: can't load karma")
	})
}

func TestErrorGettingKarmaWhenResetting(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{}, fmt.Errorf("can't load karma"))

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> reset"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get delete karma for channel [myLittleChannel] for you. If you must know, this happened: can't load karma")
	})
}

func TestErrorDeletingKarmaWhenResetting(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "abc"}, nil)
	mockStorer.On("DeleteSiloString", "myLittleChannel", "thing").Return(fmt.Errorf("can't delete"))

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> reset"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get delete karma for channel [myLittleChannel] for you. If you must know, this happened: can't delete")
	})
}

func TestErrorGettingGlobalList(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{}, fmt.Errorf("can't load karma"))

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "otherChan", Text: "<@bot> global top 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the global top [1] things for you. If you must know, this happened: can't load karma")
	})
}

func TestInvalidStoredKarmaValuesOnTopList(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "abc"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> top 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the top [1] things for you. If you must know, this happened: strconv.Atoi: parsing \"abc\": invalid syntax")
	})
}

func TestInvalidSingleStoredKarmaValuesOnGlobalTopList(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": map[string]string{"thing": "abc"}, "myOtherChannel": map[string]string{"thing": "1"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "otherChannel", Text: "<@bot> global top 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the global top [1] things for you. If you must know, this happened: strconv.Atoi: parsing \"abc\": invalid syntax")
	})
}

func TestInvalidSingleStoredKarmaValuesOnGlobalWorstList(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": map[string]string{"thing": "1"}, "myOtherChannel": map[string]string{"thing": "abc"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "otherChannel", Text: "<@bot> global worst 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the global worst [1] things for you. If you must know, this happened: strconv.Atoi: parsing \"abc\": invalid syntax")
	})
}

func TestLessItemsThanRequestedTopCountReturnsAllInOrder(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "1", "bird": "2"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> top 3"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• bird `2`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• thing `1`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestGlobalTopFormattingAndKarmaMerging(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": map[string]string{"thing": "1", "@someone": "3"}, "myOtherChannel": map[string]string{"thing": "4"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global top 2"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":leaves::leaves::leaves::trophy: *Global Top* :trophy::leaves::leaves::leaves:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• thing `5`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `3`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestTopFormatting(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "-10", "@someone": "3", "birds": "9", "@alf": "10"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> top 4"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@alf> `10`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• birds `9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `3`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• thing `-10`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestTopListingWithoutRequestedCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "-10", "@someone": "3", "birds": "9", "mountains": "8", "rivers": "9", "@alf": "10"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> top"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@alf> `10`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• birds `9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• rivers `9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• mountains `8`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `3`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestGlobalTopListingWithoutRequestedCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": map[string]string{"thing": "-10", "@someone": "3", "birds": "9", "mountains": "8", "rivers": "9", "@alf": "10"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global top"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":leaves::leaves::leaves::trophy: *Global Top* :trophy::leaves::leaves::leaves:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@alf> `10`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• birds `9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• rivers `9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• mountains `8`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `3`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestGlobalWorstFormattingAndKarmaMerging(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": map[string]string{"thing": "-4", "@someone": "-2"}, "myOtherChannel": map[string]string{"thing": "1"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global worst 2"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Global Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• thing `-3`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `-2`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestWorstFormatting(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "-10", "@someone": "3", "birds": "9", "@alf": "10"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> worst 4"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• thing `-10`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `3`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• birds `9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@alf> `10`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestGlobalWorstListingWithoutRequestedCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": map[string]string{"thing": "10", "@someone": "-3", "birds": "-9", "mountains": "-8", "rivers": "-9", "@alf": "-10"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global worst"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Global Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@alf> `-10`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• rivers `-9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• birds `-9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• mountains `-8`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `-3`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}

func TestWorstListingWithoutRequestedCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "10", "@someone": "-3", "birds": "-9", "mountains": "-8", "rivers": "-9", "@alf": "-10"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> worst"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@alf> `-10`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• rivers `-9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• birds `-9`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• mountains `-8`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• <@someone> `-3`", false, false), nil, nil),
		}, answers[0].ContentBlocks)
	})
}

func TestLessItemsThanRequestedWorstCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("ScanSilo", "myLittleChannel").Return(map[string]string{"thing": "1", "bird": "2"}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> worst 3"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "") && assert.Equal(t, []slack.Block{
			*slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", ":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• thing `1`", false, false), nil, nil),
			*slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "• bird `2`", false, false), nil, nil)}, answers[0].ContentBlocks)
	})
}
