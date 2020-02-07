package plugins_test

import (
	"encoding/json"
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/alexandre-normand/slackscot/store/mocks"
	"github.com/alexandre-normand/slackscot/test/assertanswer"
	"github.com/alexandre-normand/slackscot/test/assertplugin"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{"<@bot> karma", "Cgeneral", ""},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 1)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 2)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 3)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 4)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 5)"},
		{"<@U21355>++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 6)"},
		{"<@U21355> ++", "Cgeneral", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 7)"},
		{"<@U21355>+++", "Cgeneral", "`Bernard Tremblay` just gained 2 karma points (`Bernard Tremblay`: 9)"},
		{"<@U21355>++++", "Cgeneral", "`Bernard Tremblay` just gained 3 karma points (`Bernard Tremblay`: 12)"},
		{"<@U21355>+++++", "Cgeneral", "`Bernard Tremblay` just gained 4 karma points (`Bernard Tremblay`: 16)"},
		{"<@U21355>++++++", "Cgeneral", "`Bernard Tremblay` just gained 5 karma points (`Bernard Tremblay`: 21)"},
		{"<@U21355>+++++++", "Cgeneral", "`Bernard Tremblay` just gained 5 karma points (`Bernard Tremblay`: 26)"},
		{"+----------+", "Cgeneral", ""},
		{"---", "Cgeneral", ""},
		{"+++", "Cgeneral", ""},
		{"<@U21355>++", "Coceanlife", "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 1)"},
		{"<@bot> top 1", "Cother", "Sorry, no recorded karma found :disappointed:"},
		{"<@U21355>--", "Coceanlife", "`Bernard Tremblay` just lost karma (`Bernard Tremblay`: 0)"},
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

	if assert.NotNil(t, p) {
		for _, tc := range testCases {
			t.Run(tc.text, func(t *testing.T) {
				assertplugin := assertplugin.New(t, "bot")
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

	mockStorer.On("GetSiloString", "myLittleChannel", "@U21355").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutSiloString", "myLittleChannel", "@U21355", "1").Return(fmt.Errorf("can't persist"))

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@U21355>++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
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

	assertplugin.AnswersAndReacts(p, &slack.Msg{User: "U21355", Channel: "myLittleChannel", Text: "<@U21355>++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "*Attributing yourself karma is frown upon* :face_with_raised_eyebrow:") && assertanswer.HasOptions(t, answers[0], assertanswer.ResolvedAnswerOption{Key: slackscot.EphemeralAnswerToOpt, Value: "U21355"})
	})
}

func TestInvalidStoredKarmaShouldResetValue(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GetSiloString", "myLittleChannel", "@U21355").Return("abc", nil)
	mockStorer.On("PutSiloString", "myLittleChannel", "@U21355", "1").Return(nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@U21355>++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "`Bernard Tremblay` just gained karma (`Bernard Tremblay`: 1)")
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

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": {"thing": "abc"}, "myOtherChannel": {"thing": "1"}}, nil)

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

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": {"thing": "1"}, "myOtherChannel": {"thing": "abc"}}, nil)

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
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• bird `2`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• thing `1`\"}}]", string(render))
	})
}

func TestGlobalTopFormattingAndKarmaMerging(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": {"thing": "1", "@someone": "3"}, "myOtherChannel": {"thing": "4"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global top 2"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":leaves::leaves::leaves::trophy: *Global Top* :trophy::leaves::leaves::leaves:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• thing `5`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `3`\"}}]", string(render))
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
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@alf\\u003e `10`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• birds `9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `3`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• thing `-10`\"}}]", string(render))
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
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":leaves::leaves::leaves::trophy: *Top* :trophy::leaves::leaves::leaves:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@alf\\u003e `10`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• birds `9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• rivers `9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• mountains `8`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `3`\"}}]", string(render))
	})
}

func TestGlobalTopListingWithoutRequestedCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": {"thing": "-10", "@someone": "3", "birds": "9", "mountains": "8", "rivers": "9", "@alf": "10"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global top"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":leaves::leaves::leaves::trophy: *Global Top* :trophy::leaves::leaves::leaves:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@alf\\u003e `10`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• birds `9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• rivers `9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• mountains `8`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `3`\"}}]", string(render))
	})
}

func TestGlobalWorstFormattingAndKarmaMerging(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": {"thing": "-4", "@someone": "-2"}, "myOtherChannel": {"thing": "1"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global worst 2"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Global Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• thing `-3`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `-2`\"}}]", string(render))
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
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• thing `-10`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `3`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• birds `9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@alf\\u003e `10`\"}}]", string(render))
	})
}

func TestGlobalWorstListingWithoutRequestedCount(t *testing.T) {
	mockStorer := &mocks.Storer{}
	defer mockStorer.AssertExpectations(t)

	mockStorer.On("GlobalScan").Return(map[string]map[string]string{"myLittleChannel": {"thing": "10", "@someone": "-3", "birds": "-9", "mountains": "-8", "rivers": "-9", "@alf": "-10"}}, nil)

	var userInfoFinder userInfoFinder
	p := plugins.NewKarma(mockStorer)
	p.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(p, &slack.Msg{Channel: "myLittleChannel", Text: "<@bot> global worst"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Global Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@alf\\u003e `-10`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• rivers `-9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• birds `-9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• mountains `-8`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `-3`\"}}]", string(render))
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
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@alf\\u003e `-10`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• rivers `-9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• birds `-9`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• mountains `-8`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• \\u003c@someone\\u003e `-3`\"}}]", string(render))
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
		require.Len(t, answers, 1)

		render, err := json.Marshal(answers[0].ContentBlocks)
		require.NoError(t, err)

		return assertanswer.HasText(t, answers[0], "") && assert.Equal(t, "[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\":fallen_leaf::fallen_leaf::fallen_leaf::space_invader: *Worst* :space_invader::fallen_leaf::fallen_leaf::fallen_leaf:\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• thing `1`\"}},{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"• bird `2`\"}}]", string(render))
	})
}
