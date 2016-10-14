# Goblet

goblet is a lightweight dependency injection framework like guice.

## Example

```go
func main() {
    gb := goblet.New()
    gb.MustSetALL([]goblet.Def{
        {
            Name: "config",
            Constructor: func() (*Config, error) {
                return new(Config), nil
            },
            Singleton: true,
        },
        {
            Name: "app",
            Constructor: func(cfg *Config) (App, error) {
                // MyApp implements App interface
                return &MyApp{cfg}, nil
            },
            Refs: goblet.Refs{"config"}
        },
    })

    ctx := context.Background()
    if _, err := gb.Call(func(app App) (interface{}, error) {
        err := app.Run(ctx)
        return nil, err
    }, goblet.Refs{"app"}); err != nil {
        log.Println(err)
    }
}
```

## Contribution

1. Fork ([https://github.com/bluele/goblet/fork](https://github.com/bluele/goblet/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create new Pull Request

## Author

**Jun Kimura**

* <http://github.com/bluele>
* <junkxdev@gmail.com>
