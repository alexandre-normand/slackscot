package plugins_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/botservices"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/alexandre-normand/slackscot/v2/plugins"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

func TestNewKarmaWithMissingStoragePathConfig(t *testing.T) {
	pc := viper.New()

	_, err := plugins.NewKarma(pc)
	assert.NotNil(t, err)
}

func TestNewKarmaWithInvalidStoragePath(t *testing.T) {
	// Create a temp file that will serve as an invalid storage path
	tmpfile, err := ioutil.TempFile("", "test")
	assert.Nil(t, err)
	defer os.Remove(tmpfile.Name()) // clean up

	pc := viper.New()
	pc.Set(config.StoragePathKey, tmpfile.Name())

	_, err = plugins.NewKarma(pc)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "Opening [karma] db failed")
	}
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
		{"karma worst 2", map[string]bool{"h[0]": false, "c[0]": false, "c[1]": true}, map[string]string{"c[1]": "Here are the 2 worst things: \n```-2   dams\n1    nettle\n```\n"}},
	}

	// Create a temp file that will serve as an invalid storage path
	tmpdir, err := ioutil.TempDir("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	pc := viper.New()
	pc.Set(config.StoragePathKey, tmpdir)

	k, err := plugins.NewKarma(pc)
	assert.Nil(t, err)

	// Attach the logger
	var b strings.Builder
	k.BotServices = new(botservices.BotServices)
	k.BotServices.Logger = log.New(&b, "", 0)

	if assert.NotNil(t, k) {
		defer k.Close()

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

	for i, h := range k.Plugin.HearActions {
		id := fmt.Sprintf("h[%d]", i)

		msg := slack.Msg{Text: text}
		m := h.Match(text, &msg)
		matches[id] = m

		if m {
			answers[id] = h.Answer(&msg)
		}
	}

	for i, c := range k.Plugin.Commands {
		id := fmt.Sprintf("c[%d]", i)

		msg := slack.Msg{Text: text}
		m := c.Match(text, &msg)
		matches[id] = m

		if m {
			answers[id] = c.Answer(&msg)
		}
	}

	return matches, answers
}
