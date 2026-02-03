package operator

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testEvent struct {
	Val int
}

func (e *testEvent) EventName() string { return "testEvent" }

var _ Event = &testEvent{}
var eventType = reflect.TypeOf(&testEvent{})

func TestEventHandler_StdlibContext_Event(t *testing.T) {
	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ctx context.Context, ev *testEvent) {
		assert.Equal(t, 123, ev.Val)
		assert.Equal(t, opCtx, ctx)
	})

	assert.Nil(t, hnd.Dispatch(opCtx, &testEvent{
		Val: 123,
	}))
}

func TestEventHandler_OpContext_Event(t *testing.T) {
	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ctx *OpContext[*TxTest], ev *testEvent) {
		assert.Equal(t, 456, ev.Val)
		assert.Equal(t, opCtx, ctx)
	})

	assert.Nil(t, hnd.Dispatch(opCtx, &testEvent{
		Val: 456,
	}))
}

func TestEventHandler_Event(t *testing.T) {
	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ev *testEvent) {
		assert.Equal(t, 789, ev.Val)
	})

	assert.Nil(t, hnd.Dispatch(opCtx, &testEvent{
		Val: 789,
	}))
}

func TestEventHandler_StdlibContext_Any(t *testing.T) {
	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ctx context.Context, ev any) {
		assert.Equal(t, 123, ev.(*testEvent).Val)
		assert.Equal(t, opCtx, ctx)
	})

	assert.Nil(t, hnd.Dispatch(opCtx, &testEvent{
		Val: 123,
	}))
}

func TestEventHandler_OpContext_Any(t *testing.T) {
	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ctx *OpContext[*TxTest], ev any) {
		assert.Equal(t, 456, ev.(*testEvent).Val)
		assert.Equal(t, opCtx, ctx)
	})

	assert.Nil(t, hnd.Dispatch(opCtx, &testEvent{
		Val: 456,
	}))
}

func TestEventHandler_Any(t *testing.T) {
	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ev any) {
		assert.Equal(t, 789, ev.(*testEvent).Val)
	})

	assert.Nil(t, hnd.Dispatch(opCtx, &testEvent{
		Val: 789,
	}))
}

func TestErrorReturn(t *testing.T) {
	err := errors.New("test error")

	opCtx := &OpContext[*TxTest]{
		Context: context.Background(),
	}

	hnd := makeEventHandler[*TxTest](eventType, func(ev any) error {
		return err
	})

	assert.Equal(t, err, hnd.Dispatch(opCtx, &testEvent{
		Val: 789,
	}))
}
