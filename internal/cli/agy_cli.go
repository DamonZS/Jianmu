package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DamonZS/Jianmu/internal/agent"
	"github.com/DamonZS/Jianmu/internal/proc"
)

// `jianmu agy run [args…]` is what the function agy jianmu puts in the
// user's shells runs (agent/agy.go): the agy on PATH, given jianmu's
// gateway, key, model and settings file (--settings, jianmu-settings.json
// beside agy's own settings.json) while that file names one of jianmu's
// models, and as it is otherwise, remote control's too (it needs agy's
// Google sign-in). Every argument is agy's, --help too. `jianmu agy
// env` prints those variables and the model, one NAME=value a line and
// nothing off jianmu, for PowerShell's function.

const agyRunUsage = `jianmu agy run [args…]   start agy, on jianmu while its model is one of jianmu's

The agy on your PATH starts with GEMINI_API_KEY, GOOGLE_GEMINI_BASE_URL and
--settings <jianmu's file> and --model <its model> (not when you give
--model yourself, nor for remote-control, which needs your Google sign-in); every argument is
agy's. Off jianmu's models it starts as it is. The function agy jianmu puts
in your shell's profile runs this.

jianmu agy env           the variables and model it starts agy with, a NAME=value
                         a line; nothing while agy isn't on jianmu
`

// agyCmd is `jianmu agy run|env …`; ok false for another form.
func agyCmd(args []string) (ok bool, err error) {
	if len(args) < 2 || args[0] != "agy" && args[0] != "antigravity-cli" {
		return false, nil
	}
	switch args[1] {
	case "run":
		return true, agyRun(args[2:])
	case "env":
		if len(args) > 2 {
			if slices.Contains([]string{"-h", "--help", "help"}, args[2]) {
				fmt.Print(agyRunUsage)
				return true, nil
			}
			return true, fmt.Errorf("jianmu agy env takes nothing after it, not %q", args[2])
		}
		env, model := agent.AgyStart()
		if model == "" {
			return true, nil
		}
		for _, kv := range env {
			fmt.Println(kv)
		}
		fmt.Println("settings=" + agent.AgySettings())
		fmt.Println("model=" + model)
		return true, nil
	}
	return false, nil
}

// agyExec starts the program bin as agy with args and env, in place of
// jianmu where the system can (agy_exec_*.go); tests set it.
var agyExec = agyExecOS

// agyLook is the agy on PATH; tests set it.
var agyLook = func() (string, error) { return proc.Look("agy") }

func agyRun(args []string) error {
	bin, err := agyLook()
	if err != nil {
		return errors.New("agy isn't installed, or isn't on your PATH")
	}
	if self, err := os.Executable(); err == nil && sameProgram(self, bin) {
		return errors.New("the agy on your PATH is jianmu itself; install Antigravity CLI")
	}
	env := os.Environ()
	start, model := agent.AgyStart()
	// remote control needs the Google sign-in, which a session on an API
	// key (jianmu's models) doesn't have: it runs as agy is on its own
	if slices.ContainsFunc(args, agent.AgyRemote) {
		model = ""
	}
	if model != "" {
		env = slices.DeleteFunc(env, func(kv string) bool {
			k, _, _ := strings.Cut(kv, "=")
			return strings.EqualFold(k, "GEMINI_API_KEY") || strings.EqualFold(k, "GOOGLE_GEMINI_BASE_URL")
		})
		env = append(env, start...)
		if !slices.ContainsFunc(args, modelFlag) {
			args = append([]string{"--model", model}, args...)
		}
		args = append([]string{"--settings", agent.AgySettings()}, args...)
	}
	return agyExec(bin, args, env)
}

// modelFlag says a is agy's --model, which it takes as -model too.
func modelFlag(a string) bool {
	a, _, _ = strings.Cut(a, "=")
	return a == "--model" || a == "-model"
}

func sameProgram(a, b string) bool {
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(sa, sb)
}
