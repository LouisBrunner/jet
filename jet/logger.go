package jet

type Logger interface {
	Errorf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

func (me *app) LogDebugf(format string, v ...interface{}) {
	if me.opts.Logger != nil {
		me.opts.Logger.Debugf(format, v...)
	}
}

func (me *app) LogErrorf(format string, v ...interface{}) {
	if me.opts.Logger != nil {
		me.opts.Logger.Errorf(format, v...)
	}
}
