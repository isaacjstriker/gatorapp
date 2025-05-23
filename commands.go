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

func handlerReset(s *State, cmd Command) error {

	err := s.Queries.DelUsers(context.Background())
	if err != nil {
		fmt.Printf("failed to delete users: %s", err)
		os.Exit(1)
	}
	fmt.Println("Users deleted successfully")
	return nil
}

func handlerUsers(s *State, cmd Command) error {
	users, err := s.Queries.GetUsers(context.Background())
	if err != nil {
		fmt.Printf("failed to fetch users: %s", err)
		os.Exit(1)
	}
	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}
	fmt.Println("Users:")
	for _, user := range users {
		if user.Name == s.Config.CurrentUsername {
			fmt.Printf("* %s (current)\n", user.Name)
		}
		fmt.Printf("* %s\n", user.Name)
	}
	return nil
}

func handlerAgg(s *State, cmd Command) error {
	feedURL := "https://www.wagslane.dev/index.xml"
	feed, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	fmt.Printf("%+v\n", feed)
	return nil
}
