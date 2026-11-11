package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/DamonZS/Jianmu/internal/agent"
)

// `jianmu claude-desktop launcher [add|remove]` is the isolated Claude
// Desktop (#5671, baranwang on X), as `jianmu claude launcher` is
// claude-jianmu: add makes ~/Applications/Claude Jianmu.app, which starts
// Claude Desktop on jianmu in a folder of its own, and moves a Desktop on
// jianmu there, so the user's own Claude.app stays signed in with
// Anthropic; remove puts it back. The GUI's switch on Claude Desktop's
// row does the same (agent.SetDesktopIsolated). macOS only.

const claudeDesktopLauncherUsage = `jianmu claude-desktop launcher [add|remove]
          Claude Jianmu.app (macOS): Claude Desktop on jianmu in a folder of
          its own, beside your own Claude Desktop signed in with Anthropic
`

// claudeDesktopLaunchCmd is `jianmu claude-desktop launcher …`; ok false
// for another form.
func claudeDesktopLaunchCmd(args []string) (ok bool, err error) {
	if len(args) < 2 || args[0] != "claude-desktop" && args[0] != "claude-app" || args[1] != "launcher" {
		return false, nil
	}
	rest := args[2:]
	if len(rest) > 0 && slices.Contains([]string{"-h", "--help", "help"}, rest[0]) {
		fmt.Print(claudeDesktopLauncherUsage)
		return true, nil
	}
	return true, claudeDesktopLauncher(rest)
}

// setDesktopIsolated is agent.SetDesktopIsolated; tests set it.
var setDesktopIsolated = agent.SetDesktopIsolated

// desktopIsolatedHere is agent.DesktopIsolatedHere; tests set it.
var desktopIsolatedHere = agent.DesktopIsolatedHere

func claudeDesktopLauncher(args []string) error {
	what := "status"
	if len(args) > 0 {
		what = args[0]
	}
	if len(args) > 1 {
		return fmt.Errorf("jianmu claude-desktop launcher takes one word, not %q", args[1])
	}
	if what != "status" && what != "add" && what != "remove" {
		return fmt.Errorf("jianmu claude-desktop launcher: add, remove or nothing, not %q", what)
	}
	st, ok := desktopIsolatedHere()
	if !ok {
		return errors.New("Claude Desktop runs in a folder of its own on macOS only")
	}
	home, _ := os.UserHomeDir()
	switch what {
	case "add":
		p, err := setDesktopIsolated(home, true)
		if err != nil {
			return err
		}
		fmt.Println("Claude Jianmu:", p)
		fmt.Println("Open it to use Claude Desktop on jianmu; your own Claude Desktop stays signed in with Anthropic.")
		return nil
	case "remove":
		was := st.Launcher
		if _, err := setDesktopIsolated(home, false); err != nil {
			return err
		}
		if was != "" {
			fmt.Println("removed", was)
		}
		return nil
	}
	if st.On && st.Launcher != "" {
		fmt.Println(st.Launcher)
		return nil
	}
	fmt.Println("no Claude Jianmu app: run `jianmu claude-desktop launcher add`")
	return nil
}
