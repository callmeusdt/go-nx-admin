package commands

import (
	"log"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/models"
)

func ResetPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-password <username> [new_password]",
		Short: "重置用户密码，默认重置为 admin123",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			username := args[0]
			newPassword := "admin123"
			if len(args) >= 2 {
				newPassword = args[1]
			}

			cfg := config.Load()
			var db *gorm.DB
			var err error
			switch cfg.Database.Driver {
			case "mysql":
				db, err = gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{})
			case "postgres":
				db, err = gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
			default:
				db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
			}
			if err != nil {
				log.Fatal("db connect error: ", err)
			}
			models.AutoMigrate(db)

			var user models.User
			if err := db.Where("username = ?", username).First(&user).Error; err != nil {
				log.Fatalf("用户不存在: %s", username)
			}
			hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Fatal("password hash error: ", err)
			}
			db.Model(&user).Update("password", string(hashed))
			log.Printf("密码已重置: %s → %s", username, newPassword)
		},
	}
}
