package ygg

type HookFunc func() error

type hookWrapper struct {
	tp  HookType
	run HookFunc
}

var _ Hook = (*hookWrapper)(nil)

func (h *hookWrapper) Run() error {
	return h.run()
}

func (h *hookWrapper) Type() HookType {
	return h.tp
}

func BeforeStartupHook(f HookFunc) Hook {
	return &hookWrapper{
		tp:  HookTypeBeforeStartup,
		run: f,
	}
}

func AfterStartupHook(f HookFunc) Hook {
	return &hookWrapper{
		tp:  HookTypeAfterStartup,
		run: f,
	}
}

func BeforRunHook(f HookFunc) Hook {
	return &hookWrapper{
		tp:  HookTypeBeforeRun,
		run: f,
	}
}

func AfterRunHook(f HookFunc) Hook {
	return &hookWrapper{
		tp:  HookTypeAfterRun,
		run: f,
	}
}

func BeforeCloseHook(f HookFunc) Hook {
	return &hookWrapper{
		tp:  HookTypeBeforeClose,
		run: f,
	}
}
