package goblet

import (
	"fmt"
	"reflect"
	"sync"
)

// service is created by Def
type service struct {
	gb     *Goblet
	def    *Def
	value  reflect.Value
	isFunc bool
	group  Group
}

// validate a definition for service
func (sv *service) validate() error {
	if !sv.isFunc {
		return nil
	}

	tmp := sv.value
	for tmp.Kind() == reflect.Ptr {
		tmp = tmp.Elem()
	}
	if tmp.Kind() != reflect.Func {
		return fmt.Errorf("Constructor type should be function.")
	}

	tp := sv.value.Type()
	if sv.def.Refs.Len() != tp.NumIn() {
		return fmt.Errorf("Constructor's argument is uncompatible error: expected: %v, but actual: %v", sv.def.Refs.Len(), tp.NumIn())
	}

	if tp.NumOut() != 2 {
		return fmt.Errorf("Constructor's output parameter count should be 2, actual: %v", tp.NumOut())
	}

	if tp.Out(1).Name() != "error" {
		return fmt.Errorf("%v != %#v", tp.Out(1).Name(), "error")
	}
	return nil
}

type parallelResult struct {
	id    int
	value reflect.Value
	err   error
}

// resolve resolves dependencies for service.
func (sv *service) resolve() ([]reflect.Value, error) {
	length := sv.def.Refs.Len()
	deps := make([]reflect.Value, length)
	i := 0
	for _, ref := range sv.def.Refs {
		switch ref.(type) {
		case string:
			v, err := sv.gb.invoke(ref.(string))
			if err != nil {
				return nil, err
			}
			deps[i] = v
			i++
		case *ParallelReference:
			var wg sync.WaitGroup
			ch := make(chan *parallelResult, ref.(*ParallelReference).Len())
			for j, rs := range ref.(*ParallelReference).refs {
				wg.Add(1)
				go func(id int, rs string) {
					v, err := sv.gb.invoke(rs)
					ch <- &parallelResult{id: id, value: v, err: err}
					wg.Done()
				}(i+j, rs)
			}
			wg.Wait()
			close(ch)
			for result := range ch {
				if result.err != nil {
					return nil, result.err
				}
				deps[result.id] = result.value
			}
			i += ref.(*ParallelReference).Len()
		default:
			return nil, fmt.Errorf("Unknown ref type %T", ref)
		}
	}
	return deps, nil
}
