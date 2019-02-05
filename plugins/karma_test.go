package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

type userInfoFinder struct {
}

func (u userInfoFinder) GetUserInfo(userID string) (user *slack.User, err error) {
	return &slack.User{ID: userID, RealName: "Bernard Tremblay"}, nil
}

func TestKarmaMatchesAndAnswers(t *testing.T) {
	testCases := []struct {
		text            string
		expectedMatches map[string]bool
		expectedAnswers map[string]string
	}{
		{"creek++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`creek` just gained a level (`creek`: 1)"}},
		{"creek--", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`creek` just lost a life (`creek`: 0)"}},
		{"the creek++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`creek` just gained a level (`creek`: 1)"}},
		{"our creek++ is nice", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`creek` just gained a level (`creek`: 2)"}},
		{"our creek++ is really nice", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`creek` just gained a level (`creek`: 3)"}},
		{"oceans++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`oceans` just gained a level (`oceans`: 1)"}},
		{"oceans++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`oceans` just gained a level (`oceans`: 2)"}},
		{"nettle++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`nettle` just gained a level (`nettle`: 1)"}},
		{"salmon++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`salmon` just gained a level (`salmon`: 1)"}},
		{"salmon++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`salmon` just gained a level (`salmon`: 2)"}},
		{"salmon++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`salmon` just gained a level (`salmon`: 3)"}},
		{"salmon++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`salmon` just gained a level (`salmon`: 4)"}},
		{"dams--", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`dams` just lost a life (`dams`: -1)"}},
		{"dams--", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`dams` just lost a life (`dams`: -2)"}},
		{"karma", map[string]bool{"h[0]": false, "c[0]": false, "c[1]": false}, make(map[string]string)},
		{"karma top 2", map[string]bool{"h[0]": false, "c[0]": true, "c[1]": false}, map[string]string{"c[0]": "Here are the top 2 things: \n```4    salmon\n3    creek\n```\n"}},
		{"karma worst 2", map[string]bool{"h[0]": false, "c[0]": false, "c[1]": true}, map[string]string{"c[1]": "Here are the worst 2 things: \n```-2   dams\n1    nettle\n```\n"}},
		{"<@U21355>++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 1)"}},
		{"<@U21355>++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 2)"}},
		{"<@U21355>++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 3)"}},
		{"<@U21355>++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 4)"}},
		{"<@U21355>++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 5)"}},
		{"<@U21355>++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`Bernard Tremblay` just gained a level (`Bernard Tremblay`: 6)"}},
		{"karma top 1", map[string]bool{"h[0]": false, "c[0]": true, "c[1]": false}, map[string]string{"c[0]": "Here are the top 1 things: \n```6    Bernard Tremblay\n```\n"}},
		{"don't++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`don't` just gained a level (`don't`: 1)"}},
		{"under-the-bridge++", map[string]bool{"h[0]": true, "c[0]": false, "c[1]": false}, map[string]string{"h[0]": "`under-the-bridge` just gained a level (`under-the-bridge`: 1)"}},
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

	if assert.NotNil(t, k) {
		// Attach the logger
		var b strings.Builder
		k.Logger = slackscot.NewSLogger(log.New(&b, "", 0), false)

		for _, tc := range testCases {
			t.Run(tc.text, func(t *testing.T) {
				matches, answers := drivePlugin(tc.text, k)
				assert.Equal(t, tc.expectedMatches, matches)
				assert.Equal(t, tc.expectedAnswers, answers)
			})
		}
	}
}

func drivePlugin(text string, k *plugins.Karma) (matches map[string]bool, answers map[string]string) {
	matches = make(map[string]bool)
	answers = make(map[string]string)

	for i, h := range k.HearActions {
		id := fmt.Sprintf("h[%d]", i)

		msg := slackscot.IncomingMessage{NormalizedText: text, Msg: slack.Msg{Text: text}}
		m := h.Match(&msg)
		matches[id] = m

		if m {
			answers[id] = h.Answer(&msg).Text
		}
	}

	for i, c := range k.Commands {
		id := fmt.Sprintf("c[%d]", i)

		msg := slackscot.IncomingMessage{NormalizedText: text, Msg: slack.Msg{Text: text}}
		m := c.Match(&msg)
		matches[id] = m

		if m {
			answers[id] = c.Answer(&msg).Text
		}
	}

	return matches, answers
}
