# Gator CLI

## Prerequisites

To use the Gator CLI, you will need to have **PostgreSQL** and **Go** installed on your system.

- **PostgreSQL**: Make sure you have a running PostgreSQL server. You can download it from [https://www.postgresql.org/download/](https://www.postgresql.org/download/).
- **Go**: Install Go from [https://golang.org/dl/](https://golang.org/dl/).

## Installation

You can install the Gator CLI using the following command:

```sh
go install github.com/isaacjstriker/gatorapp@latest
```

This will install the `gatorapp` binary to your `$GOPATH/bin` directory.

## Configuration

Before running the program, you need to set up your configuration file.  
Create a file named `.gatorconfig.json` in your home directory with the following content:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

- Replace the `db_url` value with your actual PostgreSQL connection string.
- The `current_user_name` field will be set automatically when you log in or register.

## Running the Program

You can run the CLI using:

```sh
gatorapp <command> [arguments...]
```

Or, if you are running from source:

```sh
go run . <command> [arguments...]
```

## Example Commands

- `register <username>`: Register a new user.
- `login <username>`: Log in as an existing user.
- `addfeed <feed name> <feed url>`: Add a new feed and automatically follow it.
- `feeds`: List all feeds in the database.
- `follow <feed url>`: Follow a feed by its URL.
- `following`: List all feeds you are following.
- `unfollow <feed url>`: Unfollow a feed by its URL.
- `browse [limit]`: Browse recent posts from feeds you follow.

For more commands and details, run:

```sh
gatorapp help
```
