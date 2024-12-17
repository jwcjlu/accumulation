package logfile

import (
	"accumulation/pkg/log"
	"context"
	"fmt"
)

type Pipeline struct {
	head *HandlerContext
}

func NewPipeline() *Pipeline {
	return &Pipeline{}
}
func (p *Pipeline) AddHandler(handler Handler) *Pipeline {
	newNode := &HandlerContext{handler: handler}
	if p.head == nil {
		p.head = newNode
	} else {
		p.head.AddLast(newNode)
	}
	return p
}

func (p *Pipeline) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	result, err := p.head.Invoke(ctx, input)
	if err != nil {
		log.Errorf(ctx, "exec head task failure ,err:%v", err)
		p.head.Rollback()
	}
	return result, err
}

type Handler interface {
	Do(ctx context.Context, input interface{}) (interface{}, error)
	Type() TaskType
	Rollback()
}

type HandlerContext struct {
	handler Handler
	next    *HandlerContext
}

func (hc *HandlerContext) AddLast(ctx *HandlerContext) {
	if hc.next == nil {
		hc.next = ctx
	} else {
		hc.next.AddLast(ctx)
	}
}

func (hc *HandlerContext) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	if hc.handler == nil {
		return nil, fmt.Errorf("handler is empty")
	}
	log.Debugf(ctx, "upload log step:%v", hc.handler.Type())
	param, err := hc.handler.Do(ctx, input)
	if err != nil {
		log.Errorf(ctx, "exec task [%v] failure ,err:%v", hc.handler.Type(), err)
		return nil, err
	}
	if hc.next != nil {
		param, err = hc.next.Invoke(ctx, param)
	}
	return param, err
}
func (hc *HandlerContext) Rollback() {
	if hc.handler == nil {
		return
	}
	hc.handler.Rollback()

	if hc.next != nil {
		hc.next.Rollback()
	}
}
