package models

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tediscript/gostarterkit/internal/config"
	"github.com/tediscript/gostarterkit/internal/database"
)

func setupTestDBWithUsers(t *testing.T) (*database.Database, *UserRepository, func()) {
	t.Helper()

	// Create a temporary database file
	tmpDir := t.TempDir()
	dbFile := tmpDir + "/test.db"

	// Create test config
	cfg := &config.Config{}
	cfg.SQLite.DBFile = dbFile
	cfg.SQLite.MaxOpenConnections = 5
	cfg.SQLite.MaxIdleConnections = 2
	cfg.SQLite.ConnectionMaxLifetimeSeconds = 300

	// Create database
	db, err := database.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create users table
	ctx := context.Background()
	_, err = db.Exec(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create user repository
	repo := NewUserRepository(db)

	// Cleanup function
	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
		// Remove database files
		os.Remove(dbFile)
		os.Remove(dbFile + "-wal")
		os.Remove(dbFile + "-shm")
	}

	return db, repo, cleanup
}

func TestCreateUser(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful user creation", func(t *testing.T) {
		user := &User{
			Username: "testuser",
			Email:    "test@example.com",
		}

		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set")
		}

		// Re-fetch user to get database-set timestamps
		retrieved, err := repo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
		}

		if retrieved.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}

		if retrieved.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		user1 := &User{
			Username: "duplicate",
			Email:    "user1@example.com",
		}
		user2 := &User{
			Username: "duplicate",
			Email:    "user2@example.com",
		}

		err := repo.Create(ctx, user1)
		if err != nil {
			t.Fatalf("Failed to create first user: %v", err)
		}

		err = repo.Create(ctx, user2)
		if err == nil {
			t.Error("Expected error for duplicate username")
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		user1 := &User{
			Username: "user1",
			Email:    "duplicate@example.com",
		}
		user2 := &User{
			Username: "user2",
			Email:    "duplicate@example.com",
		}

		err := repo.Create(ctx, user1)
		if err != nil {
			t.Fatalf("Failed to create first user: %v", err)
		}

		err = repo.Create(ctx, user2)
		if err == nil {
			t.Error("Expected error for duplicate email")
		}
	})
}

func TestGetUserByID(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("successful retrieval", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.ID != user.ID {
			t.Errorf("Expected ID %d, got %d", user.ID, retrieved.ID)
		}

		if retrieved.Username != user.Username {
			t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
		}

		if retrieved.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 999)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}

		if err.Error() != "user not found" {
			t.Errorf("Expected 'user not found' error, got: %v", err)
		}
	})
}

func TestGetUserByUsername(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("successful retrieval", func(t *testing.T) {
		retrieved, err := repo.GetByUsername(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.ID != user.ID {
			t.Errorf("Expected ID %d, got %d", user.ID, retrieved.ID)
		}

		if retrieved.Username != user.Username {
			t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := repo.GetByUsername(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

func TestGetUserByEmail(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("successful retrieval", func(t *testing.T) {
		retrieved, err := repo.GetByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.ID != user.ID {
			t.Errorf("Expected ID %d, got %d", user.ID, retrieved.ID)
		}

		if retrieved.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

func TestUpdateUser(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	t.Run("successful update", func(t *testing.T) {
		user.Username = "updateduser"
		user.Email = "updated@example.com"

		err := repo.Update(ctx, user)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		// Verify update
		retrieved, err := repo.GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}

		if retrieved.Username != "updateduser" {
			t.Errorf("Expected username 'updateduser', got '%s'", retrieved.Username)
		}

		if retrieved.Email != "updated@example.com" {
			t.Errorf("Expected email 'updated@example.com', got '%s'", retrieved.Email)
		}
	})

	t.Run("update non-existent user", func(t *testing.T) {
		nonExistentUser := &User{
			ID:       999,
			Username: "nonexistent",
			Email:    "nonexistent@example.com",
		}

		err := repo.Update(ctx, nonExistentUser)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

func TestDeleteUser(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("successful deletion", func(t *testing.T) {
		err := repo.Delete(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		// Verify deletion
		_, err = repo.GetByID(ctx, user.ID)
		if err == nil {
			t.Error("Expected error when getting deleted user")
		}
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		err := repo.Delete(ctx, 999)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

func TestListUsers(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple users
	users := []User{
		{Username: "user1", Email: "user1@example.com"},
		{Username: "user2", Email: "user2@example.com"},
		{Username: "user3", Email: "user3@example.com"},
	}

	for i := range users {
		if err := repo.Create(ctx, &users[i]); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	t.Run("list all users", func(t *testing.T) {
		retrieved, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(retrieved) != 3 {
			t.Errorf("Expected 3 users, got %d", len(retrieved))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		retrieved, err := repo.List(ctx, 2, 0)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 users, got %d", len(retrieved))
		}

		retrieved, err = repo.List(ctx, 2, 2)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(retrieved) != 1 {
			t.Errorf("Expected 1 user, got %d", len(retrieved))
		}
	})

	t.Run("empty result", func(t *testing.T) {
		retrieved, err := repo.List(ctx, 10, 100)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(retrieved) != 0 {
			t.Errorf("Expected 0 users, got %d", len(retrieved))
		}
	})
}

func TestCountUsers(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("count empty database", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 users, got %d", count)
		}
	})

	// Create multiple users
	users := []User{
		{Username: "user1", Email: "user1@example.com"},
		{Username: "user2", Email: "user2@example.com"},
		{Username: "user3", Email: "user3@example.com"},
	}

	for i := range users {
		if err := repo.Create(ctx, &users[i]); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	t.Run("count with users", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 users, got %d", count)
		}
	})
}

func TestConcurrentCRUD(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Perform concurrent CRUD operations
	done := make(chan bool)
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Create user
			user := &User{
				Username: "user" + string(rune('0'+id%10)),
				Email:    "user" + string(rune('0'+id%10)) + "@example.com",
			}
			if err := repo.Create(ctx, user); err != nil {
				// Ignore duplicate errors in concurrent scenario
				return
			}

			// Get user
			_, err := repo.GetByID(ctx, user.ID)
			if err != nil {
				t.Errorf("Failed to get user: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestLargeQuery(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	// Create many users with unique emails
	numUsers := 100
	for i := 0; i < numUsers; i++ {
		user := &User{
			Username: fmt.Sprintf("user%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
		}
		if err := repo.Create(ctx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	t.Run("list many users", func(t *testing.T) {
		retrieved, err := repo.List(ctx, numUsers, 0)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(retrieved) != numUsers {
			t.Errorf("Expected %d users, got %d", numUsers, len(retrieved))
		}
	})

	t.Run("count many users", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		if count != numUsers {
			t.Errorf("Expected %d users, got %d", numUsers, count)
		}
	})
}

func TestUnicodeCharacters(t *testing.T) {
	_, repo, cleanup := setupTestDBWithUsers(t)
	defer cleanup()

	ctx := context.Background()

	user := &User{
		Username: "用户名",
		Email:    "测试@example.com",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user with unicode: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user with unicode: %v", err)
	}

	if retrieved.Username != user.Username {
		t.Errorf("Expected username '%s', got '%s'", user.Username, retrieved.Username)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email '%s', got '%s'", user.Email, retrieved.Email)
	}
}
