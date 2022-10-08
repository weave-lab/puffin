# Puffin 
Puffin abstracts Go's [os/exec go package](https://pkg.go.dev/os/exec).
It can be used replace calls to the underlying shell with calls to go functions.
This can be great for testing or for use cases where a simulated shell would be preferred over a real one.

# Quick Start
Puffin is designed to be easy to incorporate into your existing codebase.
The following code is an example of how to refactor your code to use puffin.

```go
package main

import (
    "log"
    "os/exec"
)

func main() {
    clean, err := branchIsClean()
    if err != nil {
        log.Fatalln(err)
    }

    log.Printf("clean: %v\n", clean)
}

func branchIsClean() (bool, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    status, err := cmd.Output()
    if err != nil {
        return false, err
    }

    return len(status) == 0, nil
}
```

This code can be re-written to use Puffin instead of Go's os/exec package.

```go
package main

import (
    "log"

    "github.com/weave-lab/puffin"
)

func main() {
    clean, err := branchIsClean(puffin.NewOsExec())
    if err != nil {
        log.Fatalln(err)
    }

    log.Printf("clean: %v\n", clean)
}

func branchIsClean(exec puffin.Exec) (bool, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    status, err := cmd.Output()
    if err != nil {
        return false, err
    }

    return len(status) == 0, nil
}

```

A few things to note about the refactored code.
1) We've shadowed the exec package name with a `puffin.Exec` argument.
    This ensures that any code that was using the `os/exec` package previously will now use puffin instead.
2) Previously, this code was using a package dependency `os/exec`.
    Now, however, the `puffin.Exec` dependency is being injected into the function.
3) `puffin.Exec` is an interface.
    `puffin.NewOsExec` is one implementation of that interface (there are others as well).
    This specific implementation behaves in the same way as the `os/exec` package.
    This means that while our code has changed, it's behavior has not.

The nice thing about this refactor is now we can write unit test for the `branchIsClean` function.
Take the following code as an example.

```go
package main

import (
    "testing"

    "github.com/weave-lab/puffin"
)

func Test_branchIsClean(t *testing.T) {
    type args struct {
        exec puffin.Exec
    }
    tests := []struct {
        name    string
        args    args
        want    bool
        wantErr bool
    }{
        {
            "is clean",
            args{
                puffin.NewFuncExec(puffin.NewHandlerMux(
                    func(cmd *puffin.FuncCmd) int {
                        return 0
                    },
                )),
            },
            true,
            false,
        },
        {
            "is dirty",
            args{
                puffin.NewFuncExec(puffin.NewHandlerMux(
                    func(cmd *puffin.FuncCmd) int {
                        cmd.Stdout().Write([]byte("M README.md"))
                        return 0
                    },
                )),
            },
            false,
            false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := branchIsClean(tt.args.exec)
            if (err != nil) != tt.wantErr {
                t.Errorf("branchIsClean() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("branchIsClean() got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Notice that this code uses `puffin.NewFuncExec` rather than `puffin.NewOSExec`.
`puffin.NewFuncExec` also implements the `puffin.Exec` interface and is a "simulated" shell.
It uses a `puffin.Mux`, that routes command calls to go functions, which then run mock shell commands.
This allows exec commands, like `git`, to have consistent, easily to program behaviors which is great for writing tests around code that includes exec commands.
Just remember, the point of these tests is to test the behavior of the code surrounding the exec commands, not the behavior the commands themselves.

# The Exec Interface
The exec interface can be used as a drop in replacement for the `os/exec` package.
It exposes most of the same functions as that package and can often be used to shadow the `os/exec` package name.
```go
package main

import (
    "os/exec"

    "github.com/weave-lab/puffin"
)

func example(exec puffin.Exec) {
    // this now uses puffin rather than the os/exec import
    exec.Command("new", "command")
}
```

The major differences between `puffin.Exec` and `os/exec` are as follows.

1) `Command` and `CommandContext` return a `puffin.Cmd` rather than an `exec.Cmd`
2) Puffin does not contain any alternative to the `exec.Error` or the `exec.ExitError` types.
   in fact, puffin returns `exec.Error` and `exec.ExitError` types wherever possible in an attempt to prevent existing error checking code from being broken.

# The Cmd Interface
The Cmd interface is the main type provided by puffin.
It is designed to abstract the [exec.Cmd](https://pkg.go.dev/os/exec#Cmd) type and in many cases is a simple drop in replacement for that type.
There are a few key differences to be aware of however, mostly dude to the fact that interfaces in to are derived based on behavior only.

1) public fields from `exec.Cmd` including 
   `Path`, `Args`, `Env`, `Dir`, `Stdin`, `Stdout`, `ExtraFiles`, `SysProcAttr`, `Process`, and `Err`
   must be accessed using getter and setter methods with `puffin.Cmd`.
   This is because interfaces in Go can not export public fields like structs can.
2) setting `SysProcAttr`, `Process`, `ProcessState`, and `Err` is not possible on a `puffin.Cmd`
   the way it is for an `exec.Cmd` as setters for these fields are not included in the interface.
   This was done to reduce the size of this interface which is already quite large.

This means that code which sets cmd members such as `Args` or `Dir` must instead use the `SetArgs` or `SetDir` functions.
The following code provides an example.
```go
package main 

import "os/exec"

func branchIsClean(dir string) (bool, error) {
    cmd := exec.Command("git")
    cmd.Args = append(cmd.Args, "status", "--porcelain")
    cmd.Dir = dir

    status, err := cmd.Output()
    if err != nil {
        return false, err
    }

    return len(status) == 0, nil
}
```

This code must be changed slightly to use the `Args()`, `SetArgs()`, and `SetDir` methods rather than modifing the cmd directly

```go
package main 

import "github.com/weave-lab/puffin"

func branchIsClean(exec puffin.Exec, dir string) (bool, error) {
    cmd := exec.Command("git")
    cmd.SetArgs(append(cmd.Args(), "status", "--porcelain"))
    cmd.SetDir(dir)

    status, err := cmd.Output()
    if err != nil {
        return false, err
    }

    return len(status) == 0, nil
}
```
