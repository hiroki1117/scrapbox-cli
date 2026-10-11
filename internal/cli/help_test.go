package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestHelpQuality は AI エージェントや利用者がヘルプだけで使い方を判断できるよう、
// コマンドのヘルプに必要な情報がそろっていることを検査する。
func TestHelpQuality(t *testing.T) {
	root := NewRootCmd(&App{})
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			path := sub.CommandPath()
			if sub.Short == "" {
				t.Errorf("%s: Short is empty", path)
			}
			// 参照系・検索系の末端コマンドには、出力形式の説明と使用例を必須とする。
			if !sub.HasSubCommands() && (inGroup(sub, groupRead, groupSearch)) {
				if !strings.Contains(strings.ToLower(sub.Long), "output") {
					t.Errorf("%s: Long must describe the output", path)
				}
				if sub.Example == "" {
					t.Errorf("%s: Example is empty", path)
				}
			}
			walk(sub)
		}
	}
	walk(root)
}

// inGroup はコマンド自身または祖先がいずれかのグループに属しているかを返す。
func inGroup(c *cobra.Command, groupIDs ...string) bool {
	for p := c; p != nil; p = p.Parent() {
		if slices.Contains(groupIDs, p.GroupID) {
			return true
		}
	}
	return false
}

func TestUnknownSubcommandFails(t *testing.T) {
	for _, group := range []string{"page", "search", "project", "auth", "config"} {
		_, err := runCmd(t, nil, nil, group, "no-such-command")
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("%s: err = %v, want unknown command error", group, err)
		}
	}
}
