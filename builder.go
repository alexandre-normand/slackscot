package slackscot

import (
	"github.com/spf13/viper"
	"io"
)

type Builder struct {
	bot *Slackscot
	err error
}

func NewBot(name string, v *viper.Viper, options ...Option) (sb *Builder) {
	sb = new(Builder)
	sb.bot, sb.err = New(name, v, options...)

	return sb
}

func (sb *Builder) WithPlugin(p *Plugin) *Builder {
	if sb.err != nil {
		return sb
	}

	return sb
}

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

func (sb *Builder) Build() Slackscot {
	return *sb.bot
}
