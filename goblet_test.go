package goblet

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func Eq(v, wants interface{}) bool {
	return v == wants
}

type ConnInterface interface {
	Query(string) (string, error)
	Addr() string
}

type Conn struct {
	host string
	port int
}

func (conn *Conn) Addr() string {
	return fmt.Sprintf("%v:%v", conn.host, conn.port)
}

func (conn *Conn) Query(query string) (string, error) {
	return fmt.Sprintf("%v:%v:%v", conn.host, conn.port, query), nil
}

func (conn *Conn) String() string {
	return conn.Addr()
}

func TestSetMethod(t *testing.T) {
	cases := []struct {
		err  error
		defs []Def
	}{
		{
			defs: []Def{
				{
					Name:  "name",
					Value: "bluele",
				},
				{
					Name: "greet",
					Constructor: func(name string) (string, error) {
						return "Hello, " + name, nil
					},
					Refs: Refs{"name"},
				},
			},
		},
		{
			err: errors.New("Constructor's argument is uncompatible error: expected: 0, but actual: 1"),
			defs: []Def{
				{
					Name:  "name",
					Value: "bluele",
				},
				{
					Name: "greet",
					Constructor: func(name string) (string, error) {
						return "Hello, " + name, nil
					},
					Refs: Refs{},
				},
			},
		},
		{
			err: errors.New("Constructor's output parameter count should be 2, actual: 0"),
			defs: []Def{
				{
					Name:  "name",
					Value: "bluele",
				},
				{
					Name:        "greet",
					Constructor: func(name string) {},
					Refs:        Refs{"name"},
				},
			},
		},
	}

	for _, cs := range cases {
		gb := New()
		for _, def := range cs.defs[:len(cs.defs)-1] {
			if err := gb.Set(def); err != nil {
				t.Fatal(err)
			}
		}
		def := cs.defs[len(cs.defs)-1]
		if err := gb.Set(def); cs.err == nil && err != nil {
			t.Fatalf("def: %v, err: %v", def, err)
		} else if cs.err != nil && err == nil {
			t.Fatalf("def: %v should be an error", def)
		} else if cs.err != nil && err != nil {
			if cs.err.Error() != err.Error() {
				t.Fatalf("%v != %v", cs.err.Error(), err.Error())
			}
		}
	}
}

func TestGetMethod(t *testing.T) {
	cases := []struct {
		key   string
		wants interface{}
		cmp   func(interface{}, interface{}) bool
		err   error
		defs  []Def
	}{
		{
			key:   "greet",
			wants: "Hello, bluele",
			cmp:   Eq,
			defs: []Def{
				{
					Name:  "name",
					Value: "bluele",
				},
				{
					Name: "greet",
					Constructor: func(name string) (string, error) {
						return "Hello, " + name, nil
					},
					Refs: Refs{"name"},
				},
			},
		},
		{
			key:   "greet",
			wants: "Hello, bluele",
			cmp:   Eq,
			defs: []Def{
				{
					Name:  "name",
					Value: "bluele",
				},
				{
					Name: "greet",
					Constructor: func(name string) (string, error) {
						return "Hello, " + name, nil
					},
					Refs: Refs{"name"},
				},
			},
		},
		{
			key:   "conn",
			wants: "localhost:8000",
			cmp: func(v, wants interface{}) bool {
				return v.(ConnInterface).Addr() == wants
			},
			defs: []Def{
				{
					Name:  "host",
					Value: "localhost",
				},
				{
					Name:  "port",
					Value: 8000,
				},
				{
					Name: "conn",
					Constructor: func(host string, port int) (*Conn, error) {
						return &Conn{host: host, port: port}, nil
					},
					Refs: Refs{"host", "port"},
				},
			},
		},
		{
			key:   "conn",
			wants: "localhost:8000",
			cmp: func(v, wants interface{}) bool {
				return v.(ConnInterface).Addr() == wants
			},
			defs: []Def{
				{
					Name: "host",
					Constructor: func() (string, error) {
						time.Sleep(time.Millisecond)
						return "localhost", nil
					},
				},
				{
					Name: "port",
					Constructor: func() (int, error) {
						time.Sleep(time.Millisecond)
						return 8000, nil
					},
				},
				{
					Name:  "opt",
					Value: "opt",
				},
				{
					Name: "conn",
					Constructor: func(host string, port int, opt string) (*Conn, error) {
						return &Conn{host: host, port: port}, nil
					},
					Refs: Refs{
						ParallelRefs("host", "port"), "opt",
					},
				},
			},
		},
	}

	for _, cs := range cases {
		gb := New()
		for _, def := range cs.defs {
			if err := gb.Set(def); err != nil {
				t.Fatalf("def: %v, err: %v", def, err)
			}
		}
		v, err := gb.Get(cs.key)
		if cs.err == nil && err != nil {
			t.Fatal(err)
		} else if cs.err != nil && err.Error() != cs.err.Error() {
			t.Fatalf("%v != %v", err.Error(), cs.err.Error())
		} else if err == nil && !cs.cmp(v, cs.wants) {
			t.Fatalf("%v != %v", v, cs.wants)
		}
	}
}

type handler func(http.ResponseWriter, *http.Request)

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(w, r)
}

func TestCallMethod(t *testing.T) {
	var mockHandler = func(w http.ResponseWriter, r *http.Request, conn *Conn) {
		q, err := conn.Query("select")
		if err != nil {
			t.Fatal(err)
		}
		if q != "localhost:8000:select" {
			t.Fatalf("%v != %v", q, "localhost:8000:select")
		}
	}

	var err error
	var called bool

	gb := New()
	err = gb.Set(Def{
		Name:  "conn",
		Value: new(Conn),
	})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	_, err = gb.Call(func(conn *Conn) (handler, error) {
		called = true
		mux.Handle("/", handler(func(w http.ResponseWriter, r *http.Request) {
			mockHandler(w, r, conn)
		}))
		return nil, nil
	}, Refs{"conn"})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("Inner function should be called")
	}
}

type model struct {
	Conn *Conn `goblet:"conn"`
}

func TestInjectMethod(t *testing.T) {
	cases := []struct {
		key  string
		obj  *model
		cmp  func(*model) bool
		err  error
		defs []Def
	}{
		{
			key: "conn",
			obj: &model{},
			cmp: func(m *model) bool {
				return m.Conn.Addr() == "localhost:8000"
			},
			defs: []Def{
				{
					Name:  "host",
					Value: "localhost",
				},
				{
					Name:  "port",
					Value: 8000,
				},
				{
					Name: "conn",
					Constructor: func(host string, port int) (*Conn, error) {
						return &Conn{host: host, port: port}, nil
					},
					Refs: Refs{"host", "port"},
				},
			},
		},
	}

	for _, cs := range cases {
		gb := New()
		gb.MustSetALL(cs.defs)
		if err := gb.Inject(cs.obj); err != nil {
			t.Error(err)
		}
		if !cs.cmp(cs.obj) {
			t.Errorf("invalid object: %v", cs.obj)
		}
	}

}

// func TestCallMethod(t *testing.T) {
// 	cases := []struct {
// 		wants interface{}
// 		cmp   func(interface{}, interface{}) bool
// 		args  []interface{}
// 		err   error
// 		defs  []Def
// 	}{
// 		{
// 			wants: "localhost:8000:select",
// 			cmp:   Eq,
// 			args:  []interface{}{"select"},
// 			defs: []Def{
// 				{
// 					Name:  "host",
// 					Value: "localhost",
// 				},
// 				{
// 					Name:  "port",
// 					Value: 8000,
// 				},
// 				{
// 					Name: "conn",
// 					Constructor: func(host string, port int) (*Conn, error) {
// 						return &Conn{
// 							host: host,
// 							port: port,
// 						}, nil
// 					},
// 					Refs: []interface{}{"host", "port"},
// 				},
// 				{
// 					Name: "query",
// 					Constructor: func(conn *Conn, query string) (string, error) {
// 						return conn.Query(query)
// 					},
// 					Refs:       []interface{}{"conn"},
// 					IsCallable: true,
// 				},
// 			},
// 		},
// 		{
// 			args: []interface{}{"select"},
// 			err:  errors.New("Constructor error"),
// 			defs: []Def{
// 				{
// 					Name: "conn",
// 					Constructor: func() (*Conn, error) {
// 						return nil, errors.New("Constructor error")
// 					},
// 				},
// 				{
// 					Name: "query",
// 					Constructor: func(conn *Conn, query string) (string, error) {
// 						return conn.Query(query)
// 					},
// 					Refs:       []interface{}{"conn"},
// 					IsCallable: true,
// 				},
// 			},
// 		},
// 		{
// 			err: ErrCannotCall,
// 			defs: []Def{
// 				{
// 					Name: "conn",
// 					Constructor: func() (*Conn, error) {
// 						return new(Conn), nil
// 					},
// 				},
// 				{
// 					Name: "query",
// 					Constructor: func(conn *Conn) (string, error) {
// 						return conn.Query("select")
// 					},
// 					Refs: []interface{}{"conn"},
// 				},
// 			},
// 		},
// 	}
//
// 	for _, cs := range cases {
// 		gb := New()
// 		for _, def := range cs.defs {
// 			if err := gb.Set(def); err != nil {
// 				t.Fatalf("def: %v, err: %v", def, err)
// 			}
// 		}
//
// 		v, err := gb.Call("query", cs.args...)
// 		if cs.err == nil && err != nil {
// 			t.Fatal(err)
// 		} else if cs.err != nil && err.Error() != cs.err.Error() {
// 			t.Fatalf("%v != %v", err.Error(), cs.err.Error())
// 		} else if err == nil && !cs.cmp(v, cs.wants) {
// 			t.Fatalf("%v != %v", v, cs.wants)
// 		}
// 	}
// }
