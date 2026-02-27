package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	listChannels := flag.Bool("list-channels", false, "query Buffer API for available channels and print their IDs")
	dryRun := flag.Bool("dry-run", false, "compose posts and print them without sending to Buffer")
	flag.Parse()

	// 1. Load config
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// 2. Validate token (required for all operations)
	if cfg.Buffer.Token == "" {
		return fmt.Errorf("buffer token not set; configure buffer.token in config.yaml or set BUFFER_TOKEN env var")
	}

	if *listChannels {
		return runListChannels(cfg)
	}

	// 3. Validate channels configured
	if len(cfg.Channels) == 0 {
		return fmt.Errorf("no channels configured; add at least one channel to config.yaml")
	}

	// 3. Discover + parse RSS feed from cwd
	feed, err := discoverAndParseFeed()
	if err != nil {
		return err
	}
	if len(feed.Channel.Items) == 0 {
		return fmt.Errorf("RSS feed contains no items")
	}

	// 4. Validate production URL
	latest := &feed.Channel.Items[0]
	if err := validateProductionURL(latest); err != nil {
		return err
	}

	// 5. Prepare Buffer client (not used in dry-run)
	var client *BufferClient
	if !*dryRun {
		client = newBufferClient(cfg.Buffer.Token)
	}
	ctx := context.Background()

	// 6. Post to each channel
	anyFailed := false
	for _, ch := range cfg.Channels {
		charLimit := ch.CharLimit
		if charLimit <= 0 {
			charLimit = cfg.Post.CharLimit
		}

		text, err := buildPostText(cfg.Post.Template, latest, charLimit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] failed to render post: %v\n", ch.Name, err)
			anyFailed = true
			continue
		}

		if *dryRun {
			sep := "─"
			bar := fmt.Sprintf("%s %s (%d chars) %s", sep+sep+sep, ch.Name, len([]rune(text)), sep+sep+sep)
			fmt.Printf("\n%s\n%s\n", bar, text)
			continue
		}

		postID, err := client.CreatePost(ctx, ch.ID, text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] failed to create post: %v\n", ch.Name, err)
			anyFailed = true
			continue
		}

		fmt.Printf("[%s] posted successfully (id: %s)\n", ch.Name, postID)
	}

	if *dryRun {
		fmt.Println()
		return nil
	}

	// 7. Exit non-zero if any channel failed
	if anyFailed {
		return fmt.Errorf("one or more channels failed")
	}
	return nil
}

func runListChannels(cfg *Config) error {
	ctx := context.Background()
	client := newBufferClient(cfg.Buffer.Token)

	orgs, err := client.GetOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("fetching organizations: %w", err)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("no organizations found for this token")
	}

	org := orgs[0]
	fmt.Printf("Organization: %s (id: %s, owner: %s)\n\n", org.Name, org.ID, org.OwnerEmail)

	channels, err := client.GetChannels(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("fetching channels: %w", err)
	}
	if len(channels) == 0 {
		fmt.Println("No channels found.")
		return nil
	}

	fmt.Printf("%-30s %-15s %-40s %s\n", "NAME / DISPLAY NAME", "SERVICE", "ID", "QUEUE PAUSED")
	fmt.Printf("%-30s %-15s %-40s %s\n", "-------------------", "-------", "--", "------------")
	for _, ch := range channels {
		label := ch.DisplayName
		if label == "" {
			label = ch.Name
		}
		paused := "no"
		if ch.IsQueuePaused {
			paused = "yes"
		}
		fmt.Printf("%-30s %-15s %-40s %s\n", label, ch.Service, ch.ID, paused)
	}
	return nil
}
