package main

import (
	"fmt"
	"log"
	"os"

	"github.com/isaacjstriker/gatorapp/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
	fmt.Printf("Config loaded: %v\n", cfg)

	state := &State{
		Config: cfg,
	}

	commands := &Commands{
		handlers: make(map[string]func(*State, Command) error),
	}

	commands.register("login", handlerLogin)

	//Get command-line arguments passed in by the user
	if len(os.Args) < 2 {
		fmt.Println(err)
		os.Exit(1)
	}
	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	cmd := Command{
		name: cmdName,
		args: cmdArgs,
	}

	if err := commands.run(state, cmd); err != nil {
		fmt.Println("Error:", err)
	}
}