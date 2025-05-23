package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/isaacjstriker/gatorapp/internal/config"
	"github.com/isaacjstriker/gatorapp/internal/database"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
	fmt.Printf("Config loaded: %v\n", cfg)

	// Open database connection
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Verify connection is valid
	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	queries := database.New(db)

	state := &State{
		Config:  cfg,
		DB:      db,
		Queries: queries,
	}

	commands := &Commands{
		handlers: make(map[string]func(*State, Command) error),
	}

	// Initialize commands
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("reset", handlerReset)
	commands.register("users", handlerUsers)
	commands.register("agg", handlerAgg)

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
