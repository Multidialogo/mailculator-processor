package shellexec

import "os/exec"

type Command struct {
	base *exec.Cmd
}

func (cmd *Command) Execute() error {
	return cmd.base.Run()
}

type CommandFactory struct{}

func NewCommandFactory() *CommandFactory {
	return &CommandFactory{}
}

func (cmd *CommandFactory) New(command string) *Command {
	return &Command{exec.Command("sh", "-c", command)}
}
