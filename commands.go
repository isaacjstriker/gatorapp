package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/isaacjstriker/gatorapp/internal/config"
)

type State struct {
	Config *config.Config
}

type Command struct {
	name	string
	args	[]string
}

type Commands struct {
	handlers	map[string]func(*State, Command) error
}

func handlerLogin(s *State, cmd Command) error {
	if len(cmd.args) == 0 {
		os.Exit(1)
	}

	username := cmd.args[0]
	s.Config.CurrentUsername = username
	fmt.Println("Username set!")

	if username == "" {
		return errors.New("error! username cannot be empty")
	}

	return nil
}

func (c *Commands) run(s *State, cmd Command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		return errors.New("unknown command: " + cmd.name)
	}
	return handler(s, cmd)
}

func (c *Commands) register(name string, f func(*State, Command) error) {
	c.handlers[name] = f
}