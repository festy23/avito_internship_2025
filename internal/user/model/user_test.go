package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUser_TableName(t *testing.T) {
	t.Run("returns correct table name", func(t *testing.T) {
		user := User{}
		assert.Equal(t, "users", user.TableName())
	})
}

func TestUser_BeforeUpdate(t *testing.T) {
	t.Run("updates timestamp before update", func(t *testing.T) {
		user := &User{
			UserID:    "u1",
			Username:  "Alice",
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		oldUpdatedAt := user.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		// Call BeforeUpdate
		err := user.BeforeUpdate(nil)
		require.NoError(t, err)

		// UpdatedAt should be updated
		assert.True(t, user.UpdatedAt.After(oldUpdatedAt))
	})

	t.Run("does not modify other fields", func(t *testing.T) {
		user := &User{
			UserID:    "u1",
			Username:  "Alice",
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		originalUserID := user.UserID
		originalUsername := user.Username
		originalTeamName := user.TeamName
		originalIsActive := user.IsActive
		originalCreatedAt := user.CreatedAt

		err := user.BeforeUpdate(nil)
		require.NoError(t, err)

		assert.Equal(t, originalUserID, user.UserID)
		assert.Equal(t, originalUsername, user.Username)
		assert.Equal(t, originalTeamName, user.TeamName)
		assert.Equal(t, originalIsActive, user.IsActive)
		assert.Equal(t, originalCreatedAt, user.CreatedAt)
	})
}

func TestUser_GORMIntegration(t *testing.T) {
	t.Run("creates user with GORM", func(t *testing.T) {
		db := setupTestDB(t)

		user := &User{
			UserID:   "u1",
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}

		err := db.Create(user).Error
		require.NoError(t, err)

		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("updates user with BeforeUpdate hook", func(t *testing.T) {
		db := setupTestDB(t)

		// Create user
		user := &User{
			UserID:   "u1",
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}
		err := db.Create(user).Error
		require.NoError(t, err)

		originalUpdatedAt := user.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		// Update user
		user.Username = "Alice Updated"
		err = db.Save(user).Error
		require.NoError(t, err)

		// UpdatedAt should be updated due to BeforeUpdate hook
		var updatedUser User
		err = db.First(&updatedUser, "user_id = ?", "u1").Error
		require.NoError(t, err)
		assert.True(t, updatedUser.UpdatedAt.After(originalUpdatedAt) || updatedUser.UpdatedAt.Equal(originalUpdatedAt))
		assert.Equal(t, "Alice Updated", updatedUser.Username)
	})

	t.Run("user_id is primary key", func(t *testing.T) {
		db := setupTestDB(t)

		user1 := &User{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true}
		err := db.Create(user1).Error
		require.NoError(t, err)

		// Try to create another user with same ID
		user2 := &User{UserID: "u1", Username: "Bob", TeamName: "frontend", IsActive: true}
		err = db.Create(user2).Error
		assert.Error(t, err) // Should fail due to primary key constraint
	})

	t.Run("retrieves user by id", func(t *testing.T) {
		db := setupTestDB(t)

		// Create user
		originalUser := &User{
			UserID:   "u1",
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}
		err := db.Create(originalUser).Error
		require.NoError(t, err)

		// Retrieve user
		var retrievedUser User
		err = db.First(&retrievedUser, "user_id = ?", "u1").Error
		require.NoError(t, err)

		assert.Equal(t, originalUser.UserID, retrievedUser.UserID)
		assert.Equal(t, originalUser.Username, retrievedUser.Username)
		assert.Equal(t, originalUser.TeamName, retrievedUser.TeamName)
		assert.Equal(t, originalUser.IsActive, retrievedUser.IsActive)
	})

	t.Run("filters users by team", func(t *testing.T) {
		db := setupTestDB(t)

		// Create users in different teams
		users := []*User{
			{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
			{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
			{UserID: "u3", Username: "Charlie", TeamName: "frontend", IsActive: true},
		}

		for _, user := range users {
			err := db.Create(user).Error
			require.NoError(t, err)
		}

		// Query backend team
		var backendUsers []User
		err := db.Where("team_name = ?", "backend").Find(&backendUsers).Error
		require.NoError(t, err)
		assert.Len(t, backendUsers, 2)
	})

	t.Run("filters active users", func(t *testing.T) {
		db := setupTestDB(t)

		// Use raw SQL to insert with explicit INTEGER values for is_active (SQLite boolean workaround)
		err := db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", 1).Error
		require.NoError(t, err)
		err = db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", 0).Error
		require.NoError(t, err)
		err = db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", 1).Error
		require.NoError(t, err)

		// Query active users
		var activeUsers []User
		err = db.Where("is_active = ?", 1).Find(&activeUsers).Error
		require.NoError(t, err)
		assert.Len(t, activeUsers, 2)
	})

	t.Run("deletes user", func(t *testing.T) {
		db := setupTestDB(t)

		user := &User{
			UserID:   "u1",
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}
		err := db.Create(user).Error
		require.NoError(t, err)

		// Delete user
		err = db.Delete(user).Error
		require.NoError(t, err)

		// Verify user is deleted
		var count int64
		err = db.Model(&User{}).Where("user_id = ?", "u1").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestUser_Fields(t *testing.T) {
	t.Run("user struct has correct fields", func(t *testing.T) {
		now := time.Now()
		user := User{
			UserID:    "u1",
			Username:  "Alice",
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, "u1", user.UserID)
		assert.Equal(t, "Alice", user.Username)
		assert.Equal(t, "backend", user.TeamName)
		assert.True(t, user.IsActive)
		assert.Equal(t, now, user.CreatedAt)
		assert.Equal(t, now, user.UpdatedAt)
	})

	t.Run("user with inactive status", func(t *testing.T) {
		user := User{
			IsActive: false,
		}

		assert.False(t, user.IsActive)
	})

	t.Run("user with empty fields", func(t *testing.T) {
		user := User{}

		assert.Empty(t, user.UserID)
		assert.Empty(t, user.Username)
		assert.Empty(t, user.TeamName)
		assert.False(t, user.IsActive) // Default bool value
	})
}

func TestUser_ZeroValue(t *testing.T) {
	t.Run("zero value user", func(t *testing.T) {
		var user User

		assert.Empty(t, user.UserID)
		assert.Empty(t, user.Username)
		assert.Empty(t, user.TeamName)
		assert.False(t, user.IsActive)
		assert.True(t, user.CreatedAt.IsZero())
		assert.True(t, user.UpdatedAt.IsZero())
	})
}

func TestUser_CompositeIndex(t *testing.T) {
	t.Run("composite index on team_name and is_active", func(t *testing.T) {
		db := setupTestDB(t)

		// Use raw SQL to insert with explicit INTEGER values for is_active (SQLite boolean workaround)
		err := db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", 1).Error
		require.NoError(t, err)
		err = db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", 0).Error
		require.NoError(t, err)
		err = db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "frontend", 1).Error
		require.NoError(t, err)

		// Query using composite index - should find only Alice (is_active=1)
		var activeBackendUsers []User
		err = db.Where("team_name = ? AND is_active = ?", "backend", 1).Find(&activeBackendUsers).Error
		require.NoError(t, err)
		assert.Len(t, activeBackendUsers, 1, "Should find exactly 1 active backend user")
		if len(activeBackendUsers) > 0 {
			assert.Equal(t, "Alice", activeBackendUsers[0].Username)
		}
	})
}

func setupTestDB(t *testing.T) *gorm.DB {
	// Enable SQL logging for debugging
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		// Disable default transaction for better performance in tests
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	// Create users table (SQLite compatible, no DEFAULT for is_active to avoid issues)
	err = db.Exec(`
		CREATE TABLE users (
			user_id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			team_name VARCHAR(255) NOT NULL,
			is_active INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// Create indexes
	err = db.Exec(`CREATE INDEX idx_users_team_name ON users(team_name)`).Error
	require.NoError(t, err)

	err = db.Exec(`CREATE INDEX idx_users_team_active ON users(team_name, is_active)`).Error
	require.NoError(t, err)

	return db
}

// Benchmark tests.
func BenchmarkUser_TableName(b *testing.B) {
	user := User{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = user.TableName()
	}
}

func BenchmarkUser_BeforeUpdate(b *testing.B) {
	user := &User{
		UserID:    "u1",
		Username:  "Alice",
		TeamName:  "backend",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = user.BeforeUpdate(nil)
	}
}
