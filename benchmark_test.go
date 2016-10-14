package goblet

import "testing"

func getter(key string) (string, error) {
	return key, nil
}

func BenchmarkGobletGet(b *testing.B) {
	gb := New()
	gb.MustSetALL([]Def{
		{
			Name:  "key",
			Value: "ok",
		},
		{
			Name:        "getter",
			Constructor: getter,
			Refs:        []interface{}{"key"},
		},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err := gb.Get("getter")
		if err != nil {
			panic(err)
		}
		if v.(string) != "ok" {
			panic(v)
		}
	}
}

func BenchmarkGobletCall(b *testing.B) {
	gb := New()
	gb.MustSetALL([]Def{
		{
			Name:  "key",
			Value: "ok",
		},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err := gb.Call(getter, []interface{}{"key"})
		if err != nil {
			panic(err)
		}
		if v.(string) != "ok" {
			panic(v)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v, err := getter("ok")
		if err != nil {
			panic(err)
		}
		if v != "ok" {
			panic(v)
		}
	}
}
