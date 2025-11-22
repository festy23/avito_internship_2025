package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	teamModel "github.com/festy23/avito_internship/internal/team/model"
)

type testTeam struct {
	TeamName  string    `gorm:"primaryKey;column:team_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (testTeam) TableName() string {
	return "teams"
}

type testUser struct {
	UserID    string    `gorm:"primaryKey;column:user_id"`
	Username  string    `gorm:"column:username;not null"`
	TeamName  string    `gorm:"column:team_name;not null"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (testUser) TableName() string {
	return "users"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&testTeam{}, &testUser{})
	require.NoError(t, err)

	return db
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)

		team, err := repo.Create(ctx, "payments")

		require.NoError(t, err)
		assert.Equal(t, "payments", team.TeamName)
		assert.False(t, team.CreatedAt.IsZero())
		assert.False(t, team.UpdatedAt.IsZero())

		var dbTeam testTeam
		db.Where("team_name = ?", "payments").First(&dbTeam)
		assert.Equal(t, "payments", dbTeam.TeamName)
	})

	t.Run("duplicate team name", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "payments")

		team, err := repo.Create(ctx, "payments")

		assert.Nil(t, team)
		assert.ErrorIs(t, err, teamModel.ErrTeamExists)
	})

	t.Run("team name with special characters", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)

		team, err := repo.Create(ctx, "team-with-special_chars.123")

		require.NoError(t, err)
		assert.Equal(t, "team-with-special_chars.123", team.TeamName)
	})
}

func TestRepository_GetByName(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		team, err := repo.GetByName(ctx, "backend")

		require.NoError(t, err)
		assert.Equal(t, "backend", team.TeamName)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)

		team, err := repo.GetByName(ctx, "nonexistent")

		assert.Nil(t, team)
		assert.ErrorIs(t, err, teamModel.ErrTeamNotFound)
	})
}

func TestRepository_CreateOrUpdateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("create new user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		user, err := repo.CreateOrUpdateUser(ctx, "backend", "u1", "Alice", true)

		require.NoError(t, err)
		assert.Equal(t, "u1", user.UserID)
		assert.Equal(t, "Alice", user.Username)
		assert.Equal(t, "backend", user.TeamName)
		assert.True(t, user.IsActive)

		var dbUser testUser
		db.Where("user_id = ?", "u1").First(&dbUser)
		assert.Equal(t, "Alice", dbUser.Username)
		assert.True(t, dbUser.IsActive)
	})

	t.Run("update existing user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)

		user, err := repo.CreateOrUpdateUser(ctx, "backend", "u1", "Alice Updated", false)

		require.NoError(t, err)
		assert.Equal(t, "u1", user.UserID)
		assert.Equal(t, "Alice Updated", user.Username)
		assert.Equal(t, "backend", user.TeamName)
		assert.False(t, user.IsActive)

		var dbUser testUser
		db.Where("user_id = ?", "u1").First(&dbUser)
		assert.Equal(t, "Alice Updated", dbUser.Username)
		assert.False(t, dbUser.IsActive)
	})

	t.Run("update user team", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "frontend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)

		user, err := repo.CreateOrUpdateUser(ctx, "frontend", "u1", "Alice", true)

		require.NoError(t, err)
		assert.Equal(t, "u1", user.UserID)
		assert.Equal(t, "frontend", user.TeamName)

		var dbUser testUser
		db.Where("user_id = ?", "u1").First(&dbUser)
		assert.Equal(t, "frontend", dbUser.TeamName)
	})
}

func TestRepository_GetTeamMembers(t *testing.T) {
	ctx := context.Background()

	t.Run("empty team", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		members, err := repo.GetTeamMembers(ctx, "backend")

		require.NoError(t, err)
		assert.Empty(t, members)
	})

	t.Run("multiple members sorted by user_id", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", false)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)

		members, err := repo.GetTeamMembers(ctx, "backend")

		require.NoError(t, err)
		require.Len(t, members, 3)
		assert.Equal(t, "u1", members[0].UserID)
		assert.Equal(t, "Alice", members[0].Username)
		assert.True(t, members[0].IsActive)
		assert.Equal(t, "u2", members[1].UserID)
		assert.Equal(t, "Bob", members[1].Username)
		assert.True(t, members[1].IsActive)
		assert.Equal(t, "u3", members[2].UserID)
		assert.Equal(t, "Charlie", members[2].Username)
		assert.False(t, members[2].IsActive)
	})

	t.Run("non-existent team returns empty list", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)

		members, err := repo.GetTeamMembers(ctx, "nonexistent")

		require.NoError(t, err)
		assert.Empty(t, members)
	})

	t.Run("filters by team correctly", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "frontend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "frontend", true)

		members, err := repo.GetTeamMembers(ctx, "backend")

		require.NoError(t, err)
		require.Len(t, members, 1)
		assert.Equal(t, "u1", members[0].UserID)
	})
}

func TestRepository_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("team name with max length (255 chars)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		longTeamName := ""
		for i := 0; i < 255; i++ {
			longTeamName += "a"
		}

		team, err := repo.Create(ctx, longTeamName)

		require.NoError(t, err)
		assert.Equal(t, longTeamName, team.TeamName)
	})

	t.Run("team name with unicode characters", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		unicodeTeamName := "команда_開発_🚀"

		team, err := repo.Create(ctx, unicodeTeamName)

		require.NoError(t, err)
		assert.Equal(t, unicodeTeamName, team.TeamName)
	})

	t.Run("user_id with SQL special characters", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db)
		specialUserID := "user'; DROP TABLE users; --"
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		user, err := repo.CreateOrUpdateUser(ctx, "backend", specialUserID, "Hacker", true)

		require.NoError(t, err)
		assert.Equal(t, specialUserID, user.UserID)

		// Verify SQL injection didn't work
		var count int64
		db.Table("users").Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

