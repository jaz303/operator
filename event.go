package operator

import (
	"fmt"
	"reflect"
)

var (
	errorInterface = reflect.TypeOf((*error)(nil)).Elem()
)

type Event interface {
	EventName() string
}

type eventHandler[Tx Transaction] interface {
	Dispatch(op *OpContext[Tx], evt any) error
}

func makeEventHandler[Tx Transaction](eventType reflect.Type, fn any) eventHandler[Tx] {
	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic(fmt.Errorf("event handler type %T is not a function", fn))
	}

	switch val.Type().NumOut() {
	case 0:
		// nothing to do
	case 1:
		outType := val.Type().Out(0)
		if !outType.Implements(errorInterface) {
			panic(fmt.Errorf("event handler return type %s does not implement error", outType))
		}
	default:
		panic(fmt.Errorf("event handler must return 0..1 values"))
	}

	hnd := genericEventHandler[Tx]{
		fn:               val,
		evtParameterType: eventType,
	}

	ix := 0
	switch val.Type().NumIn() {
	case 2:
		ctxType := val.Type().In(0)
		if !reflect.TypeOf(&OpContext[Tx]{}).AssignableTo(ctxType) {
			panic(fmt.Errorf("OpContext[Tx] is not assigned to event handler context parameter %s", ctxType))
		}
		hnd.hasContext = true
		ix++
		fallthrough
	case 1:
		inEvtType := val.Type().In(ix)
		if !eventType.AssignableTo(inEvtType) {
			panic(fmt.Errorf("concrete event type %s is not assignable to event handler parameter %s", eventType, inEvtType))
		}
	default:
		panic(fmt.Errorf("event handler must declare 1..2 parameters"))
	}

	return &hnd
}

type genericEventHandler[Tx Transaction] struct {
	fn               reflect.Value
	evtParameterType reflect.Type
	hasContext       bool
}

func (h *genericEventHandler[Tx]) Dispatch(op *OpContext[Tx], evt any) error {
	args := make([]reflect.Value, 0, 2)

	if h.hasContext {
		args = append(args, reflect.ValueOf(op))
	}

	rEvt := reflect.ValueOf(evt)
	if !rEvt.Type().AssignableTo(h.evtParameterType) {
		return fmt.Errorf("event type %T is not assignable to handler parameter type %s", evt, h.evtParameterType)
	} else {
		args = append(args, rEvt)
	}

	out := h.fn.Call(args)

	if len(out) == 0 || out[0].IsNil() {
		return nil
	} else {
		return out[0].Interface().(error)
	}
}
