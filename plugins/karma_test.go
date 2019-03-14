package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/store"
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
		expectedAnswer string
	}{
		{"creek++", "`creek` just gained a level (`creek`: 1)"},
		{"creek--", "`creek` just lost a life (`creek`: 0)"},
		{"the creek++", "`creek` just gained a level (`creek`: 1)"},
		{"our creek++ is nice", "`creek` just gained a level (`creek`: 2)"},
		{"our creek++ is really nice", "`creek` just gained a level (`creek`: 3)"},
		{"oceans++", "`oceans` just gained a level (`oceans`: 1)"},
		{"oceans++", "`oceans` just gained a level (`oceans`: 2)"},
		{"nettle++", "`nettle` just gained a level (`nettle`: 1)"},
		{"salmon++", "`salmon` just gained a level (`salmon`: 1)"},
		{"salmon++", "`salmon` just gained a level (`salmon`: 2)"},
		{"salmon++", "`salmon` just gained a level (`salmon`: 3)"},
		{"salmon++", "`salmon` just gained a level (`salmon`: 4)"},
		{"dams--", "`dams` just lost a life (`dams`: -1)"},
		{"dams--", "`dams` just lost a life (`dams`: -2)"},
		{"<@bot> karma", ""},
		{"<@bot> karma top 2", "Here are the top 2 things: \n```4    salmon\n3    creek\n```\n"},
		{"<@bot> karma worst 2", "Here are the worst 2 things: \n```-2   dams\n1    nettle\n```\n"},
		{"<@U21355>++", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 1)"},
		{"<@U21355>++", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 2)"},
		{"<@U21355>++", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 3)"},
		{"<@U21355>++", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 4)"},
		{"<@U21355>++", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 5)"},
		{"<@U21355>++", "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 6)"},
		{"<@bot> karma top 1", "Here are the top 1 things: \n```6    Bernard Tremblay\n```\n"},
		{"don't++", "`don't` just gained a level (`don't`: 1)"},
		{"under-the-bridge++", "`the-bridge` just gained a level (`the-bridge`: 1)"},
		{"Jean-Michel++", "`Jean-Michel` just gained a level (`Jean-Michel`: 1)"},
		{"+----------+", ""},
		{"---", ""},
		{"+++", ""},
		{"<@bot> karma worst", ""},
		{"<@bot> karma top", ""},
	}

	// Create a temp file that will serve as an invalid storage path
	tmpdir, err := ioutil.TempDir("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	storer, err := store.NewLevelDB("karmaTest", tmpdir)
	assert.Nil(t, err)
	defer storer.Close()

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(storer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	if assert.NotNil(t, k) {
		for _, tc := range testCases {
			t.Run(tc.text, func(t *testing.T) {
				assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: tc.text}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
					if len(tc.expectedAnswer) > 0 {
						return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], tc.expectedAnswer)
					}

					return assert.Empty(t, answers)
				})
			})
		}
	}
}

func TestErrorStoringKarmaRecord(t *testing.T) {
	mockStorer := &mockStorer{}

	mockStorer.On("GetString", "thing").Return("", fmt.Errorf("not found"))
	mockStorer.On("PutString", "thing", "1").Return(fmt.Errorf("can't persist"))

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(mockStorer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: "thing++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Empty(t, answers)
	})
}

func TestInvalidStoredKarmaShouldResetValue(t *testing.T) {
	mockStorer := &mockStorer{}

	mockStorer.On("GetString", "thing").Return("abc", nil)
	mockStorer.On("PutString", "thing", "1").Return(nil)

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(mockStorer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: "thing++"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "`thing` just gained a level (`thing`: 1)")
	})
}

func TestErrorGettingList(t *testing.T) {
	mockStorer := &mockStorer{}

	mockStorer.On("Scan").Return(map[string]string{}, fmt.Errorf("can't load karma"))

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(mockStorer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: "<@bot> karma top 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the top [1] things for you. If you must know, this happened: can't load karma")
	})
}

func TestInvalidStoredKarmaValuesOnTopList(t *testing.T) {
	mockStorer := &mockStorer{}

	mockStorer.On("Scan").Return(map[string]string{"thing": "abc"}, nil)

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(mockStorer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: "<@bot> karma top 1"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Sorry, I couldn't get the top [1] things for you. If you must know, this happened: strconv.Atoi: parsing \"abc\": invalid syntax")
	})
}

func TestLessItemsThanRequestedTopCountReturnsAllInOrder(t *testing.T) {
	mockStorer := &mockStorer{}

	mockStorer.On("Scan").Return(map[string]string{"thing": "1", "bird": "2"}, nil)

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(mockStorer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: "<@bot> karma top 3"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Here are the top 2 things: \n```2    bird\n1    thing\n```\n")
	})
}

func TestLessItemsThanRequestedWorstCount(t *testing.T) {
	mockStorer := &mockStorer{}

	mockStorer.On("Scan").Return(map[string]string{"thing": "1", "bird": "2"}, nil)

	var userInfoFinder userInfoFinder
	k := plugins.NewKarma(mockStorer)
	k.UserInfoFinder = userInfoFinder

	assertplugin := assertplugin.New(t, "bot")

	assertplugin.AnswersAndReacts(&k.Plugin, &slack.Msg{Text: "<@bot> karma worst 3"}, func(t *testing.T, answers []*slackscot.Answer, emojis []string) bool {
		return assert.Len(t, answers, 1) && assertanswer.HasText(t, answers[0], "Here are the worst 2 things: \n```1    thing\n2    bird\n```\n")
	})
}
