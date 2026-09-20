package database

import (
	"fmt"

	"github.com/m161awm2/velocity-bulletinBE/internal/model"
	"gorm.io/gorm"
)

// Migrate synchronizes the models before deployment, including legacy SQL databases.
func Migrate(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(764219038)").Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable("schema_migrations") {
			var state struct {
				Version int64
				Dirty   bool
			}
			if err := tx.Table("schema_migrations").Take(&state).Error; err != nil {
				return err
			}
			if state.Dirty || state.Version < 1 || state.Version > 2 {
				return fmt.Errorf("unsupported legacy migration state: version=%d dirty=%t", state.Version, state.Dirty)
			}
			if state.Version == 1 {
				// Equivalent to the former second SQL migration, including soft-deleted posts.
				if err := tx.Model(&model.Post{}).Unscoped().Where("category = ?", "NOTICE").UpdateColumn("category", "GENERAL").Error; err != nil {
					return err
				}
				if err := tx.Migrator().DropConstraint(&model.Post{}, "posts_category_check"); err != nil {
					return err
				}
				if err := tx.Migrator().DropTable("likes"); err != nil {
					return err
				}
				for _, column := range []string{"view_count", "like_count"} {
					if err := tx.Migrator().DropColumn(&model.Post{}, column); err != nil {
						return err
					}
				}
			}
		}
		if err := tx.AutoMigrate(&model.User{}, &model.Post{}, &model.PostImage{}, &model.Comment{}); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}
		return tx.Migrator().DropTable("schema_migrations")
	})
}
