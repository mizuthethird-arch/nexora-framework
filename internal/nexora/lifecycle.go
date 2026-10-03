package nexora

import "context"

type Lifecycle interface {
	Initialize(context.Context) error
	Start(context.Context) error
	Stop(context.Context) error
	Release(context.Context) error
}

type recordingLifecycle struct {
	name   string
	events *[]string
}

func (r *recordingLifecycle) Initialize(context.Context) error {
	*r.events = append(*r.events, r.name+".initialize")
	return nil
}

func (r *recordingLifecycle) Start(context.Context) error {
	*r.events = append(*r.events, r.name+".start")
	return nil
}

func (r *recordingLifecycle) Stop(context.Context) error {
	*r.events = append(*r.events, r.name+".stop")
	return nil
}

func (r *recordingLifecycle) Release(context.Context) error {
	*r.events = append(*r.events, r.name+".release")
	return nil
}
