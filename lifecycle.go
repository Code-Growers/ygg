package ygg

import "context"

type LifeCycleFunc func(ctx context.Context) error

type lifecycleComponent struct {
	startups []LifeCycleFunc
	run      LifeCycleFunc
	cleanups []LifeCycleFunc
	hooks    []Hook

	contextClose func()
	// Channel to close
	exitChan chan (bool)
}

// Hooks implements Component.
func (cc *lifecycleComponent) Hooks() []Hook {
	return cc.hooks
}

var _ Component = (*lifecycleComponent)(nil)

func NewLifecycleComponent(startups []LifeCycleFunc, run LifeCycleFunc, cleanups []LifeCycleFunc) *lifecycleComponent {
	return &lifecycleComponent{
		startups: startups,
		run:      run,
		cleanups: cleanups,
		hooks:    nil,

		exitChan: make(chan bool, 1),
	}
}

func (cc lifecycleComponent) WithCleanupFunctions(cleanups []LifeCycleFunc) lifecycleComponent {
	c := cc
	c.cleanups = append(c.cleanups, cleanups...)
	return c
}

func (cc lifecycleComponent) WithHooks(hooks []Hook) lifecycleComponent {
	c := cc
	c.hooks = hooks
	return c
}

func (cc *lifecycleComponent) Startup() error {
	ctx := context.TODO()
	for _, startups := range cc.startups {
		err := startups(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cc *lifecycleComponent) Run() error {
	ctx, cancel := context.WithCancel(context.TODO())

	cc.contextClose = cancel

	if cc.run != nil {
		err := cc.run(ctx)
		if err != nil {
			return err
		}
	} else {
		cc.exitChan <- true
		close(cc.exitChan)
		return nil
	}
	return nil
}

func (cc *lifecycleComponent) Close(ctx context.Context) error {
	if cc.contextClose != nil {
		cc.contextClose()
	}
	for _, cleanup := range cc.cleanups {
		err := cleanup(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
