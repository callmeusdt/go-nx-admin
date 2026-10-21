package commands

import (
	"crypto/rand"
	"log"
	"math/big"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"go-nx-admin/app"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/models"
)

func ResetPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-password <username> [new_password]",
		Short: "重置用户密码；不带新密码时随机生成并只显示一次",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			username := args[0]
			newPassword := ""
			if len(args) >= 2 {
				newPassword = args[1]
			} else {
				generated, err := randomPassword()
				if err != nil {
					log.Fatal("password generation error: ", err)
				}
				newPassword = generated
			}
			if l := len(newPassword); l < 8 || l > 32 {
				log.Fatal("新密码长度须为 8-32 字节")
			}

			cfg, err := config.Load()
			if err != nil {
				log.Fatal(err)
			}
			// A fresh deployment has no data directory yet; SQLite cannot create it.
			for _, d := range app.RequiredDirs(cfg) {
				if err := os.MkdirAll(d, 0o755); err != nil {
					log.Fatalf("mkdir %s error: %v", d, err)
				}
			}
			db, err := app.OpenDatabase(cfg)
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
			if err := db.Model(&user).Updates(map[string]any{"password": string(hashed), "must_change_password": true}).Error; err != nil {
				log.Fatal("update password error: ", err)
			}
			// Drop live sessions: a reset is usually a response to compromise.
			db.Where("user_id = ?", user.ID).Delete(&models.OnlineUser{})
			if len(args) >= 2 {
				log.Printf("密码已重置: %s（下次登录须修改）", username)
			} else {
				log.Printf("密码已重置: %s → %s（仅显示一次，下次登录须修改）", username, newPassword)
			}
		},
	}
}

func randomPassword() (string, error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 20)
	limit := big.NewInt(int64(len(alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		buf[i] = alphabet[n.Int64()]
	}
	return string(buf), nil
}
