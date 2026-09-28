# eye

A super simple live reloader for go programs. it watches the files relevant and
reloads the program when changes happen.

## install

```sh
go install github.com/kyledickey/eye@latest
```

## usage

```sh
eye                                # run the package in the current directory
eye ./cmd/server                   # run a different package
eye ./cmd/server -addr :3000       # pass arguments to the program
eye -- -addr :3000                 # same, for the package in the current dir
eye --ext go,tmpl -w . -w ../lib   # watch more things
eye --debug                        # see everything eye is doing
```

then you see

```
eye ◉ 11:56PM INF watching pkg=./cmd/server dirs=3
eye ◉ 11:56PM INF started took=106m
...
# your app logs
...
eye ◉ 11:57PM INF changed file=cmd/server/main.go
eye ◉ 11:57PM INF reloaded took=98ms
```

files compiled in with `//go:embed` trigger a reload too

| flag           | default                            | what it does                               |
| -------------- | ---------------------------------- | ------------------------------------------ |
| `-w, --watch`  | `.`                                | directories to watch                       |
| `-e, --ext`    | `go,mod,sum`                       | file extensions that trigger a reload      |
| `-i, --ignore` | `vendor,node_modules,testdata,tmp` | directory names to skip                    |
| `-d, --delay`  | `100ms`                            | how long changes must settle before reload |
| `--debug`      | off                                | show debug logs                            |

### license

[MIT](LICENSE)
