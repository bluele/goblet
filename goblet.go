package goblet

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	// TagName is a reference name for struct tag
	TagName = "goblet"
)

// Goblet is a DI container and service locator.
type Goblet struct {
	mu       sync.RWMutex
	services map[string]*service
	cache    *cache
}

// New returns a new Goblet object.
func New() *Goblet {
	return &Goblet{
		services: make(map[string]*service),
		cache:    newCache(),
	}
}

// Def is definition for Object injected
type Def struct {
	Name string // Definition name

	Value       Value       // Value for object
	Constructor Constructor // Constructor for definition

	Refs        Refs // Refs is depended on this definition(Constructor)
	Singleton   bool // boolean for Singleton object
	Immediately bool // boolean for evaluate this definition immediately
}

// Constructor supports func(<Dependencies>) (interface{}, error)
type Constructor interface{}

// Value is a type for single value.
type Value interface{}

func (gb *Goblet) createService(def *Def) (*service, error) {
	if def.Name == "" {
		return nil, ErrEmptyName
	}

	var err error
	sv := new(service)
	sv.gb = gb
	sv.def = def

	if def.Constructor == nil {
		sv.value = reflect.ValueOf(def.Value)
	} else {
		sv.value = reflect.ValueOf(def.Constructor)
		sv.isFunc = true
	}

	err = sv.validate()
	if err != nil {
		return nil, err
	}
	return sv, nil
}

// Set sets a new definition for object.
func (gb *Goblet) Set(def Def) error {
	sv, err := gb.createService(&def)
	if err != nil {
		return err
	}
	gb.mu.Lock()
	gb.services[def.Name] = sv
	gb.mu.Unlock()
	if def.Immediately {
		if _, err := gb.Get(def.Name); err != nil {
			return err
		}
	}
	return nil
}

// MustSetALL sets new definitions for object.
// If any definitions are invalid, this method will panic.
func (gb *Goblet) MustSetALL(defs []Def) {
	for _, def := range defs {
		if err := gb.Set(def); err != nil {
			panic(err)
		}
	}
}

// Get gets a object by provided name
func (gb *Goblet) Get(name string) (interface{}, error) {
	return rvToI(gb.invoke(name))
}

// Inject resolve a dependencies for specified struct field
func (gb *Goblet) Inject(obj interface{}) error {
	rv := reflect.ValueOf(obj)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%v should be ptr type", obj)
	}
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("Not expected type %v", rv.Kind())
	}

	tp := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fd := rv.Field(i)
		tag := tp.Field(i).Tag.Get(TagName)
		if fd.CanSet() && len(tag) > 0 {
			v, err := gb.invoke(tag)
			if err != nil {
				return err
			}
			fd.Set(v)
		}
	}
	return nil
}

// Call resolves a dependencies for specified function, and executes it.
func (gb *Goblet) Call(handler Constructor, refs Refs) (interface{}, error) {
	sv, err := gb.createService(&Def{
		Name:        "_call", // dummy name
		Constructor: handler,
		Refs:        refs,
	})
	if err != nil {
		return nil, err
	}
	return rvToI(gb.evalute(sv))
}

// MustCall resolves a dependencies for specified function, and executes it.
// If it fails, this method will panic.
func (gb *Goblet) MustCall(handler Constructor, refs Refs) interface{} {
	v, err := gb.Call(handler, refs)
	if err != nil {
		panic(err)
	}
	return v
}

func rvToI(v reflect.Value, err error) (interface{}, error) {
	nilV := reflect.Value{}
	if v == nilV {
		return nil, err
	}
	if v.CanInterface() {
		return v.Interface(), err
	}
	return nil, err
}

func (gb *Goblet) invoke(name string) (reflect.Value, error) {
	gb.mu.RLock()
	sv, ok := gb.services[name]
	gb.mu.RUnlock()
	if !ok {
		return reflect.Value{}, ErrKeyNotFound
	}

	if !sv.def.Singleton {
		return gb.evalute(sv)
	}

	if record, ok := gb.cache.get(name); ok {
		return record.value, record.err
	}

	iv, err := sv.group.Do(name, func() (interface{}, error) {
		if record, ok := gb.cache.get(name); ok {
			return record.value, record.err
		}
		return gb.evalute(sv)
	})
	return iv.(reflect.Value), err
}

func (gb *Goblet) evalute(sv *service) (reflect.Value, error) {
	deps, err := sv.resolve()
	if err != nil {
		return reflect.Value{}, err
	}

	if !sv.isFunc {
		return sv.value, nil
	}

	var v reflect.Value
	result := sv.value.Call(deps)
	if result[1].IsNil() {
		v = result[0]
	} else {
		v, err = result[0], result[1].Interface().(error)
	}
	if sv.def.Singleton {
		sv.gb.cache.set(sv.def.Name, &cacheRecord{value: v, err: err})
	}
	return v, err
}
