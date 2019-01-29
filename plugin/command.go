package plugin

import (
	"strings"
)

type Command struct {
	Name string
	Args []string
}

func (p *BasePlugin) ParseCmdsFromMsg(msg string, onlyOneArg bool) []*Command {
	cmds := make([]*Command, 0)
	rows := strings.Split(msg, "\n")
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if !strings.HasPrefix(row, "/") {
			continue
		}

		row = strings.TrimLeft(row, "/")
		var arrs []string
		if onlyOneArg {
			arrs = strings.SplitN(row, " ", 2)
		} else {
			arrs = strings.Split(row, " ")
		}

		arrs2 := make([]string, 0)
		for _, v := range arrs {
			if v != "" {
				arrs2 = append(arrs2, strings.TrimSpace(v))
			}
		}
		if len(arrs2) == 0 {
			continue
		}

		cmd := &Command{
			Name: arrs2[0],
			Args: arrs2[1:],
		}
		cmds = append(cmds, cmd)
	}
	return cmds
}
