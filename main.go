package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/isaacjstriker/devware/games"
	"github.com/isaacjstriker/devware/games/tetris"
	"github.com/isaacjstriker/devware/games/typing"
	"github.com/isaacjstriker/devware/internal/auth"
	"github.com/isaacjstriker/devware/internal/config"
	"github.com/isaacjstriker/devware/internal/database"
	"github.com/isaacjstriker/devware/ui"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection with fallback
	db, err := database.ConnectWithFallback(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to any database: %v", err)
	}
	defer db.Close()

	// Create tables if they don't exist
	if err := db.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// Add test data if database is empty
	if err := db.CreateTestData(); err != nil {
		fmt.Printf("ℹ️  Test data: %v\n", err)
	}

	fmt.Println("✅ Database connected and tables verified.")

	var authManager *auth.CLIAuth

	// Initialize authentication if database is available
	if db != nil {
		authManager = auth.NewCLIAuth(db)
		fmt.Println("🔐 Authentication system initialized.")
	}

	// Main application loop
	for {
		// Build menu items step by step
		var menuItems []ui.MenuItem

		// Always available items
		menuItems = append(menuItems,
			ui.MenuItem{Label: "🎲 Challenge Mode (All Games)", Value: "challenge"},
			ui.MenuItem{Label: "🎯 Typing Speed Challenge", Value: "typing"},
			ui.MenuItem{Label: "🧱 Tetris", Value: "block-stacking"},
		)

		// User-specific items
		if authManager != nil && authManager.GetSession().IsLoggedIn() {
			session := authManager.GetSession().GetCurrentSession()
			userDisplayName := "User"
			if session != nil {
				userDisplayName = session.Username
			}

			menuItems = []ui.MenuItem{
				{Label: fmt.Sprintf("👤 %s", userDisplayName), Value: "user_info"},
				{Label: "🎲 Challenge Mode (All Games)", Value: "challenge"},
				{Label: "🎯 Typing Speed Challenge", Value: "typing"},
				{Label: "🏆 View Leaderboards", Value: "leaderboard"},
				{Label: "🔄 Authentication", Value: "auth"},
			}
		} else if authManager != nil {
			menuItems = append(menuItems,
				ui.MenuItem{Label: "👤 Login / Register", Value: "auth"},
				ui.MenuItem{Label: "🏆 View Leaderboards", Value: "leaderboard"},
			)
		}

		// Always at the end
		menuItems = append(menuItems,
			ui.MenuItem{Label: "⚙️  Settings", Value: "settings"},
			ui.MenuItem{Label: "❌ Exit", Value: "exit"},
		)

		// Create and show menu
		menu := ui.NewMenu("Main Menu - Select Your Adventure", menuItems)
		choice := menu.Show()

		switch choice {
		case "typing":
			typingGame := typing.NewTypingGame()
			if authManager != nil {
				typingGame.Play(db, authManager)
			}

		case "block-stacking":
			tetrisGame := tetris.NewTetris()
			if authManager != nil {
				tetrisGame.Play(db, authManager)
			}

		case "auth":
			if authManager != nil {
				authManager.ShowAuthMenu()
			} else {
				fmt.Println("\n⚠️  Authentication not available (no database connection)")
				fmt.Println("Press Enter to continue...")
				fmt.Scanln()
			}

		case "user_info":
			if authManager != nil && authManager.GetSession().IsLoggedIn() {
				showUserProfile(db, authManager)
			}

		case "leaderboard":
			if db != nil {
				showLeaderboard(db)
			} else {
				fmt.Println("\n⚠️  Leaderboards not available (no database connection)")
				fmt.Println("Press Enter to continue...")
				fmt.Scanln()
			}

		case "settings":
			showSettings(cfg)

		case "exit":
			fmt.Println("\n👋 Thanks for playing Dev Ware!")
			fmt.Println("💝 Come back soon for more games!")
			if authManager != nil && authManager.GetSession().IsLoggedIn() {
				fmt.Printf("🔐 %s will remain logged in for next time.\n", authManager.GetSession().GetCurrentSession().Username)
			}
			return

		case "challenge":
			gameRegistry := games.NewGameRegistry()
			gameRegistry.RegisterGame(typing.NewTypingGame())
			gameRegistry.RegisterGame(tetris.NewTetris())
			// Register other games as you create them

			challengeMode := games.NewChallengeMode(gameRegistry)
			challengeMode.RunChallenge(db, authManager)

		default:
			fmt.Println("Invalid selection. Please try again.")
		}

		// Debug: Show login status
		if authManager != nil && authManager.GetSession().IsLoggedIn() {
			session := authManager.GetSession().GetCurrentSession()
			fmt.Printf("🔐 Debug: Logged in as %s (ID: %d)\n", session.Username, session.UserID)
		} else {
			fmt.Println("❌ Debug: Not logged in")
		}
	}
}

func showUserProfile(db *database.DB, authManager *auth.CLIAuth) {
	session := authManager.GetSession().GetCurrentSession()
	if session == nil {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("👤 User Profile: %s\n", session.Username)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📧 Email: %s\n", session.Email)
	fmt.Printf("🆔 User ID: %d\n", session.UserID)

	// Get typing game stats
	if stats, err := db.GetUserStats(session.UserID, "typing"); err == nil {
		fmt.Println("\n🎯 Typing Game Statistics:")
		fmt.Printf("   🏆 Best Score: %d points\n", stats.BestScore)
		fmt.Printf("   🎮 Games Played: %d\n", stats.GamesPlayed)
		fmt.Printf("   📊 Average Score: %.1f points\n", stats.AvgScore)
		fmt.Printf("   ⏰ Last Played: %s\n", stats.LastPlayed.Format("2006-01-02 15:04"))
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}

func showLeaderboard(db *database.DB) {
	fmt.Println(strings.Repeat("🏆", 25))
	fmt.Println("         LEADERBOARDS")
	fmt.Println(strings.Repeat("🏆", 25))

	// Game selection menu
	fmt.Println("\nSelect game to view leaderboard:")
	fmt.Println("1. 🎯 Typing Speed Challenge")
	fmt.Println("2. 🧱 Tetris")
	fmt.Println("3. 📊 All Games Combined")
	fmt.Print("\nEnter choice (1-3): ")

	var choice string
	fmt.Scanln(&choice)

	var gameType string
	var gameTitle string

	switch choice {
	case "1":
		gameType = "typing"
		gameTitle = "🎯 Typing Speed Challenge"
	case "2":
		gameType = "tetris"
		gameTitle = "🧱 Tetris"
	case "3":
		showAllGamesLeaderboard(db)
		return
	default:
		fmt.Println("Invalid choice, showing typing leaderboard...")
		gameType = "typing"
		gameTitle = "🎯 Typing Speed Challenge"
	}

	showGameLeaderboard(db, gameType, gameTitle)
}

func showGameLeaderboard(db *database.DB, gameType, gameTitle string) {
	fmt.Printf("\n%s\n", gameTitle)
	fmt.Println(strings.Repeat("═", 60))

	entries, err := db.GetLeaderboard(gameType, 15)
	if err != nil {
		fmt.Printf("❌ Error loading leaderboard: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("📝 No scores recorded yet! Be the first to play!")
		return
	}

	// Display headers based on game type
	if gameType == "tetris" {
		fmt.Printf("%-4s %-15s %-8s %-8s %-8s %-12s\n",
			"Rank", "Player", "Score", "Lines", "Level", "Last Played")
	} else {
		fmt.Printf("%-4s %-15s %-8s %-8s %-8s %-12s\n",
			"Rank", "Player", "Score", "Avg", "Games", "Last Played")
	}
	fmt.Println(strings.Repeat("─", 60))

	for i, entry := range entries {
		rank := fmt.Sprintf("#%d", i+1)
		if i < 3 {
			medals := []string{"🥇", "🥈", "🥉"}
			rank = medals[i]
		}

		username := truncateString(entry.Username, 15)

		if gameType == "tetris" {
			// For Tetris, show lines and level from metadata
			lines := "N/A"
			level := "N/A"
			// You'd parse metadata here if available

			fmt.Printf("%-4s %-15s %-8d %-8s %-8s %-12s\n",
				rank, username, entry.BestScore, lines, level,
				entry.LastPlayed.Format("Jan 02"))
		} else {
			fmt.Printf("%-4s %-15s %-8d %-8.1f %-8d %-12s\n",
				rank, username, entry.BestScore, entry.AvgScore, entry.GamesPlayed,
				entry.LastPlayed.Format("Jan 02"))
		}
	}

	showLeaderboardStats(entries, gameType)
}

func showAllGamesLeaderboard(db *database.DB) {
	fmt.Println("\n📊 ALL GAMES COMBINED LEADERBOARD")
	fmt.Println(strings.Repeat("═", 70))

	// This would require a more complex query to combine scores across games
	// For now, show separate sections

	games := []struct {
		gameType string
		title    string
		emoji    string
	}{
		{"typing", "Typing Speed Challenge", "🎯"},
		{"tetris", "Tetris", "🧱"},
	}

	for _, game := range games {
		fmt.Printf("\n%s %s - Top 5\n", game.emoji, game.title)
		fmt.Println(strings.Repeat("─", 40))

		entries, err := db.GetLeaderboard(game.gameType, 5)
		if err != nil {
			fmt.Printf("❌ Error loading %s leaderboard: %v\n", game.title, err)
			continue
		}

		if len(entries) == 0 {
			fmt.Println("   No scores yet!")
			continue
		}

		for i, entry := range entries {
			fmt.Printf("   %d. %s - %d pts\n",
				i+1, truncateString(entry.Username, 12), entry.BestScore)
		}
	}
}

// Helper function to show interesting leaderboard statistics
func showLeaderboardStats(entries []database.LeaderboardEntry, gameType string) {
	if len(entries) == 0 {
		return
	}

	// Calculate some interesting stats
	totalGames := 0
	totalScore := 0
	for _, entry := range entries {
		totalGames += entry.GamesPlayed
		totalScore += entry.BestScore
	}

	avgBestScore := float64(totalScore) / float64(len(entries))

	fmt.Println("\n📈 Community Stats:")
	fmt.Printf("   🎮 Total Games Played: %d\n", totalGames)
	fmt.Printf("   👥 Active Players: %d\n", len(entries))
	fmt.Printf("   📊 Average Best Score: %.1f\n", avgBestScore)

	if len(entries) > 0 {
		fmt.Printf("   🏆 Highest Score: %d (by %s)\n", entries[0].BestScore, entries[0].Username)
	}
}

// Helper function to truncate long usernames
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

func showSettings(cfg *config.Config) {
	fmt.Println("\n⚙️  SETTINGS")
	fmt.Println("============")
	fmt.Printf("App Name: %s\n", cfg.AppName)
	fmt.Printf("Debug Mode: %t\n", cfg.Debug)

	if cfg.DatabaseURL == "" || cfg.DatabaseURL == "disabled" {
		fmt.Println("Database: Disabled (Offline Mode)")
	} else {
		fmt.Printf("Database: Connected (%s)\n", cfg.DatabaseURL)
	}

	fmt.Printf("Server: %s:%d\n", cfg.ServerHost, cfg.ServerPort)

	fmt.Println("\n💡 Tip: Create a .env file to customize these settings")
	fmt.Println("(Copy .env.example and modify as needed)")

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
}
