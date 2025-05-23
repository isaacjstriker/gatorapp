package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"strings"
	"strconv"

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

type UserHandler func(s *State, user database.User, cmd Command) error

// Middleware fucntion, allowing us to skip verification in each function
func requireLogin(handler UserHandler) func(s *State, cmd Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.Queries.GetUser(context.Background(), s.Config.CurrentUsername)
		if err != nil {
			fmt.Printf("could not find current username: %s\n", err)
			os.Exit(1)
		}
		return handler(s, user, cmd)
	}
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
	if len(cmd.args) < 1 || len(cmd.args) > 2 {
		return fmt.Errorf("usage: %v <time_between_reqs>", cmd.name)
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	log.Printf("Collecting feeds every %s...", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)

	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func scrapeFeeds(s *State) {
	feed, err := s.Queries.GetNextFeedToFetch(context.Background())
	if err != nil {
		log.Println("Couldn't get next feeds to fetch", err)
		return
	}
	log.Println("Found a feed to fetch!")
	scrapeFeed(s.Queries, feed)
}

func scrapeFeed(db *database.Queries, feed database.Feed) {
	_, err := db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		log.Printf("Couldn't mark feed %s fetched: %v", feed.Name, err)
		return
	}

	feedData, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		log.Printf("Couldn't collect feed %s: %v", feed.Name, err)
		return
	}
	for _, item := range feedData.Channel.Item {
		publishedAt := sql.NullTime{}
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			publishedAt = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}

		_, err = db.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			FeedID:    feed.ID,
			Title:     item.Title,
			Description: sql.NullString{
				String: item.Description,
				Valid:  true,
			},
			Url:         item.Link,
			PublishedAt: publishedAt,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue
			}
			log.Printf("Couldn't create post: %v", err)
			continue
		}
	}
	log.Printf("Feed %s collected, %v posts found", feed.Name, len(feedData.Channel.Item))
}

func handlerAddFeed(s *State, user database.User, cmd Command) error {
	if len(cmd.args) < 2 {
		fmt.Println("Usage: addfeed <name> <url>")
		os.Exit(1)
	}
	feedName := cmd.args[0]
	feedURL := cmd.args[1]

	user, err := s.Queries.GetUser(context.Background(), s.Config.CurrentUsername)
	if err != nil {
		fmt.Printf("could not find current username: %s", err)
		os.Exit(1)
	}

	id := uuid.New()
	now := time.Now()
	feed, err := s.Queries.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Name:      feedName,
		Url:       feedURL,
		UserID:    user.ID,
	})
	if err != nil {
		fmt.Printf("could not create feed: %s", err)
		os.Exit(1)
	}

	followID := uuid.New()
	_, err = s.Queries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        followID,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		fmt.Printf("could not create feed follow: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Feed created:\n")
	fmt.Printf("ID: %s\n", feed.ID)
	fmt.Printf("Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)
	fmt.Printf("UserID: %s\n", feed.UserID)
	fmt.Printf("CreatedAt: %s\n", feed.CreatedAt.Format(time.RFC3339))
	fmt.Printf("UpdatedAt: %s\n", feed.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("LastFetchedAt: %v\n", feed.LastFetchedAt.Time)

	return nil
}

func handlerFeeds(s *State, cmd Command) error {
	feeds, err := s.Queries.GetFeedsWithUser(context.Background())
	if err != nil {
		fmt.Printf("failed to fetch feeds: %s\n", err)
		os.Exit(1)
	}
	if len(feeds) == 0 {
		fmt.Println("No feeds found")
		os.Exit(1)
	}
	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("Name: %s\nURL: %s\nCreated by: %s\n\n", feed.Name, feed.Url, feed.UserName)
	}
	return nil
}

func handlerFollow(s *State, user database.User, cmd Command) error {
	if len(cmd.args) < 1 {
		fmt.Println("Usage: follow <feed_url>")
		os.Exit(1)
	}
	feedURL := cmd.args[0]

	user, err := s.Queries.GetUser(context.Background(), s.Config.CurrentUsername)
	if err != nil {
		fmt.Printf("could not find current username: %s", err)
		os.Exit(1)
	}

	feed, err := s.Queries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		fmt.Printf("could not find feed with url %s: %s\n", feedURL, err)
		os.Exit(1)
	}

	id := uuid.New()
	now := time.Now()
	follow, err := s.Queries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		fmt.Printf("could not create feed follow: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Now following feed '%s' as user '%s'\n", follow.FeedName, follow.UserName)
	return nil
}

func handlerFollowing(s *State, user database.User, cmd Command) error {
	user, err := s.Queries.GetUser(context.Background(), s.Config.CurrentUsername)
	if err != nil {
		fmt.Printf("could not find current user: %s\n", err)
		os.Exit(1)
	}

	follows, err := s.Queries.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		fmt.Printf("could not fetch followed feeds: %s\n", err)
		os.Exit(1)
	}

	if len(follows) == 0 {
		fmt.Println("You are not following any feeds.")
		os.Exit(0)
	}

	fmt.Println("Feeds you are following:")
	for _, follow := range follows {
		fmt.Printf("- %s\n", follow.FeedName)
	}
	return nil
}

func handlerUnfollow(s *State, user database.User, cmd Command) error {
	if len(cmd.args) < 1 {
		fmt.Println("Usage: unfollow <feed_url>")
		os.Exit(1)
	}
	feedURL := cmd.args[0]

	err := s.Queries.DelFeedFollow(context.Background(), database.DelFeedFollowParams{
		UserID: user.ID,
		Url:    feedURL,
	})
	if err != nil {
		fmt.Printf("could not unfollow feed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unfollowed feed with URL: %s\n", feedURL)
	return nil
}

func handlerBrowse(s *State, user database.User, cmd Command) error {
	limit := 2
	if len(cmd.args) == 1 {
		if specifiedLimit, err := strconv.Atoi(cmd.args[0]); err == nil {
			limit = specifiedLimit
		} else {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	posts, err := s.Queries.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("couldn't get posts for user: %w", err)
	}

	fmt.Printf("Found %d posts for user %s:\n", len(posts), user.Name)
	for _, post := range posts {
		fmt.Printf("%s from %s\n", post.PublishedAt.Time.Format("Mon Jan 2"), post.FeedName)
		fmt.Printf("--- %s ---\n", post.Title)
		fmt.Printf("    %v\n", post.Description.String)
		fmt.Printf("Link: %s\n", post.Url)
		fmt.Println("=====================================")
	}

	return nil
}