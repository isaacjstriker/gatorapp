package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/isaacjstriker/gatorapp/internal/config"
	"github.com/isaacjstriker/gatorapp/internal/database"
)

type State struct {
	Config  *config.Config
	DB      *sql.DB
	Queries *database.Queries
}

type Command struct {
	name string
	args []string
}

type Commands struct {
	handlers map[string]func(*State, Command) error
}

func handlerLogin(s *State, cmd Command) error {
	if len(cmd.args) == 0 {
		fmt.Println("No username provided")
		os.Exit(1)
	}

	username := cmd.args[0]
	_, err := s.Queries.GetUser(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("User %s does not exist\n", username)
			os.Exit(1)
		}
		fmt.Printf("Error checking user: %s\n", username)
		os.Exit(1)
	}
	s.Config.CurrentUsername = username

	if err := config.Write(*s.Config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("You have been logged in")
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

func handlerRegister(s *State, cmd Command) error {
	if len(cmd.args) < 1 || cmd.args[0] == "" {
		return fmt.Errorf("not a valid name")
	}

	name := cmd.args[0]

	// Check if a user exists
	_, err := s.Queries.GetUser(context.Background(), name)
	if err == nil {
		fmt.Printf("User '%s' already exists", name)
		os.Exit(1)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fmt.Printf("Error checking user: %s", err)
		os.Exit(1)
	}

	id := uuid.New()
	now := time.Now()

	_, err = s.Queries.CreateUser(context.Background(), database.CreateUserParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
	})
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	s.Config.CurrentUsername = name
	if err := config.Write(*s.Config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("User '%s' successfully registered!", name)

	log.Printf("[DEBUG] Registered user: ID=%s, Name=%s, CreatedAt=%s, UpdatedAt=%s\n",
		id.String(), name, now.Format(time.RFC3339), now.Format(time.RFC3339))

	return nil
}
