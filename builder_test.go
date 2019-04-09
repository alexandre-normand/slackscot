package slackscot_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestNewSlackscotWithoutPlugins(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		Build()

	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestNewSlackscotWithSimplePlugin(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPlugin(newPlugin()).
		Build()

	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestNewSlackscotWithPluginAndError(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPluginErr(newPluginWithErr("")).
		Build()

	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestNewSlackscotWithPluginAndErrorSet(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPluginErr(newPluginWithErr("error1")).
		Build()

	require.Error(t, err)
	assert.EqualError(t, err, "error1")
	assert.Nil(t, b)
}

func TestNewSlackscotWithPluginAndManyErrors(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPluginErr(newPluginWithErr("error1")).
		WithPluginErr(newPluginWithErr("error2")).
		WithPlugin(newPlugin()).
		Build()

	require.Error(t, err)
	assert.EqualError(t, err, "error1")
	assert.Nil(t, b)
}

func TestNewSlackscotWithCloserPluginClosingWithError(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPluginCloserErr(newPluginWithErrAndCloser("", CloseTester{errorMsg: "should be called"})).
		Build()

	require.NoError(t, err)
	require.NotNil(t, b)

	err = b.Close()
	assert.EqualError(t, err, "should be called")
}

func TestNewSlackscotWithCloserPluginClosingWithoutError(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPluginCloserErr(newPluginWithErrAndCloser("", CloseTester{errorMsg: ""})).
		Build()

	require.NoError(t, err)
	require.NotNil(t, b)

	err = b.Close()
	assert.NoError(t, err)
}

func TestNewSlackscotWithCloserAndErr(t *testing.T) {
	b, err := slackscot.NewBot("jane", config.NewViperWithDefaults()).
		WithPluginCloserErr(newPluginWithErrAndCloser("error1", CloseTester{})).
		WithPluginCloserErr(newPluginWithErrAndCloser("error2", CloseTester{})).
		Build()

	require.Error(t, err)
	assert.EqualError(t, err, "error1")
	assert.Nil(t, b)
}

// newPlugin returns a new tester plugin
func newPlugin() (p *slackscot.Plugin) {
	p = new(slackscot.Plugin)
	p.Name = "tester"
	p.Commands = []slackscot.ActionDefinition{{
		Match: func(m *slackscot.IncomingMessage) bool {
			return strings.HasPrefix(m.NormalizedText, "make")
		},
		Usage:       "make `<something>`",
		Description: "Have the test bot make something for you",
		Answer: func(m *slackscot.IncomingMessage) *slackscot.Answer {
			return &slackscot.Answer{Text: "Ready"}
		},
	}}

	return p
}

// newPluginWithErr returns the plugin along with an error if errorMsg is not empty
func newPluginWithErr(errorMsg string) (p *slackscot.Plugin, err error) {
	if errorMsg != "" {
		return nil, fmt.Errorf(errorMsg)
	}

	return newPlugin(), nil
}

// newPluginWithErr returns the plugin along with an error if errorMsg is not empty and the closer
func newPluginWithErrAndCloser(errorMsg string, closer io.Closer) (c io.Closer, p *slackscot.Plugin, err error) {
	p, err = newPluginWithErr(errorMsg)

	return closer, p, err
}

// CloseTester is an empty struct that has is a Closer that either doesn't do anything
// or returns the error set on the CloseTester
type CloseTester struct {
	errorMsg string
}

// Close returns the CloseTester error if set, or just returns nil and does nothing otherwise
func (c CloseTester) Close() (err error) {
	if c.errorMsg != "" {
		return fmt.Errorf(c.errorMsg)
	}

	return nil
}
