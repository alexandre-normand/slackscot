package slackscot

import (
	"github.com/alexandre-normand/slackscot/config"
	"github.com/spf13/viper"
	"io"
)

// Builder holds a slackscot instance to build
type Builder struct {
	bot *Slackscot
	err error
}

// NewBot returns a new Builder used to set up a new slackscot
func NewBot(name string, v *viper.Viper, options ...Option) (sb *Builder) {
	sb = new(Builder)
	sb.bot, sb.err = New(name, v, options...)

	return sb
}

// PluginInstantiator creates a new instance of a plugin given a PluginConfig
type PluginInstantiator func(c *config.PluginConfig) (p *Plugin, err error)

// CloserPluginInstantiator creates a new instance of a closer plugin given a PluginConfig
type CloserPluginInstantiator func(c *config.PluginConfig) (closer io.Closer, p *Plugin, err error)

// WithPlugin adds a plugin to the slackscot instance
func (sb *Builder) WithPlugin(p *Plugin) *Builder {
	if sb.err != nil {
		return sb
	}

	return sb
}

// WithPlugin adds a plugin that has a creation function returning (Plugin, error) to the slackscot instance
func (sb *Builder) WithPluginErr(p *Plugin, err error) *Builder {
	if sb.err == nil && err != nil {
		sb.err = err
	}

	if sb.err != nil {
		return sb
	}

	sb.bot.RegisterPlugin(p)

	return sb
}

// WithPlugin adds a plugin that has a creation function returning (io.Closer, Plugin, error) to the slackscot instance
func (sb *Builder) WithPluginCloserErr(closer io.Closer, p *Plugin, err error) *Builder {
	if sb.err == nil && err != nil {
		sb.err = err
	}

	if sb.err != nil {
		return sb
	}

	sb.bot.RegisterPlugin(p)

	if closer != nil {
		sb.bot.closers = append(sb.bot.closers, closer)
	}

	return sb
}

// WithConfigurablePluginErr adds a plugin to the slackscot instance by first checking and
// getting its configuration
func (sb *Builder) WithConfigurablePluginErr(name string, newInstance PluginInstantiator) *Builder {
	if pc, exists := sb.loadPluginConfig(name); exists {
		return sb.WithPluginErr(newInstance(pc))
	}

	return sb
}

// WithConfigurablePluginCloser adds a closer plugin to the slackscot instance by first checking and
// getting its configuration
func (sb *Builder) WithConfigurablePluginCloserErr(name string, newInstance CloserPluginInstantiator) *Builder {
	if pc, exists := sb.loadPluginConfig(name); exists {
		return sb.WithPluginCloserErr(newInstance(pc))
	}

	return sb
}

// loadPluginConfig gets the plugin configuration and returns it along with its validity. If false,
// there is no config for the plugin
func (sb *Builder) loadPluginConfig(name string) (pc *config.PluginConfig, valid bool) {
	if sb.err != nil {
		return nil, false
	}

	pc, err := config.GetPluginConfig(sb.bot.config, name)
	if err != nil {
		sb.err = err
		return nil, false
	}

	return pc, true
}

// Build returns the built slackscot instance. If there was an error during
// setup, the error is returned along with a nil slackscot
func (sb *Builder) Build() (s *Slackscot, err error) {
	if sb.err != nil {
		return nil, sb.err
	}

	return sb.bot, sb.err
}
