package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTeam_TableName(t *testing.T) {
	t.Run("returns correct table name", func(t *testing.T) {
		team := Team{}
		assert.Equal(t, "teams", team.TableName())
	})
}

func TestTeam_BeforeUpdate(t *testing.T) {
	t.Run("updates timestamp before update", func(t *testing.T) {
		team := &Team{
			TeamName:  "backend",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		oldUpdatedAt := team.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		// Call BeforeUpdate
		err := team.BeforeUpdate(nil)
		require.NoError(t, err)

		// UpdatedAt should be updated
		assert.True(t, team.UpdatedAt.After(oldUpdatedAt))
	})

	t.Run("does not modify other fields", func(t *testing.T) {
		team := &Team{
			TeamName:  "frontend",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		originalTeamName := team.TeamName
		originalCreatedAt := team.CreatedAt

		err := team.BeforeUpdate(nil)
		require.NoError(t, err)

		assert.Equal(t, originalTeamName, team.TeamName)
		assert.Equal(t, originalCreatedAt, team.CreatedAt)
	})
}

func TestTeam_GORMIntegration(t *testing.T) {
	t.Run("creates team with GORM", func(t *testing.T) {
		db := setupTestDB(t)

		team := &Team{
			TeamName: "backend",
		}

		err := db.Create(team).Error
		require.NoError(t, err)

		assert.NotZero(t, team.CreatedAt)
		assert.NotZero(t, team.UpdatedAt)
	})

	t.Run("updates team with BeforeUpdate hook", func(t *testing.T) {
		db := setupTestDB(t)

		// Create team
		team := &Team{
			TeamName: "backend",
		}
		err := db.Create(team).Error
		require.NoError(t, err)

		originalUpdatedAt := team.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		// Update team
		team.TeamName = "backend" // No actual change
		err = db.Save(team).Error
		require.NoError(t, err)

		// UpdatedAt should be updated due to BeforeUpdate hook
		var updatedTeam Team
		err = db.First(&updatedTeam, "team_name = ?", "backend").Error
		require.NoError(t, err)
		assert.True(t, updatedTeam.UpdatedAt.After(originalUpdatedAt) || updatedTeam.UpdatedAt.Equal(originalUpdatedAt))
	})

	t.Run("team name is primary key", func(t *testing.T) {
		db := setupTestDB(t)

		team1 := &Team{TeamName: "backend"}
		err := db.Create(team1).Error
		require.NoError(t, err)

		// Try to create another team with same name
		team2 := &Team{TeamName: "backend"}
		err = db.Create(team2).Error
		assert.Error(t, err) // Should fail due to primary key constraint
	})

	t.Run("retrieves team by name", func(t *testing.T) {
		db := setupTestDB(t)

		// Create team
		originalTeam := &Team{TeamName: "backend"}
		err := db.Create(originalTeam).Error
		require.NoError(t, err)

		// Retrieve team
		var retrievedTeam Team
		err = db.First(&retrievedTeam, "team_name = ?", "backend").Error
		require.NoError(t, err)

		assert.Equal(t, originalTeam.TeamName, retrievedTeam.TeamName)
		assert.Equal(t, originalTeam.CreatedAt.Unix(), retrievedTeam.CreatedAt.Unix())
	})

	t.Run("deletes team", func(t *testing.T) {
		db := setupTestDB(t)

		team := &Team{TeamName: "backend"}
		err := db.Create(team).Error
		require.NoError(t, err)

		// Delete team
		err = db.Delete(team).Error
		require.NoError(t, err)

		// Verify team is deleted
		var count int64
		err = db.Model(&Team{}).Where("team_name = ?", "backend").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestTeam_Fields(t *testing.T) {
	t.Run("team struct has correct fields", func(t *testing.T) {
		now := time.Now()
		team := Team{
			TeamName:  "backend",
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, "backend", team.TeamName)
		assert.Equal(t, now, team.CreatedAt)
		assert.Equal(t, now, team.UpdatedAt)
	})

	t.Run("team with empty name", func(t *testing.T) {
		team := Team{
			TeamName: "",
		}

		assert.Empty(t, team.TeamName)
	})

	t.Run("team with long name", func(t *testing.T) {
		longName := string(make([]byte, 255))
		for i := range longName {
			longName = string(append([]byte(longName[:i]), 'a'))
		}

		team := Team{
			TeamName: longName[:255],
		}

		assert.Len(t, team.TeamName, 255)
	})
}

func TestTeam_ZeroValue(t *testing.T) {
	t.Run("zero value team", func(t *testing.T) {
		var team Team

		assert.Empty(t, team.TeamName)
		assert.True(t, team.CreatedAt.IsZero())
		assert.True(t, team.UpdatedAt.IsZero())
	})
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create teams table
	err = db.Exec(`
		CREATE TABLE teams (
			team_name VARCHAR(255) PRIMARY KEY,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}

// Benchmark tests.
func BenchmarkTeam_TableName(b *testing.B) {
	team := Team{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = team.TableName()
	}
}

func BenchmarkTeam_BeforeUpdate(b *testing.B) {
	team := &Team{
		TeamName:  "backend",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = team.BeforeUpdate(nil)
	}
}
