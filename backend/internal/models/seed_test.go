package models

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedCreatesDefaultAdminWithoutMFA(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db); err != nil {
		t.Fatal(err)
	}

	var user User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.Status != 1 {
		t.Fatalf("default admin status = %d, want 1", user.Status)
	}
	if user.MFAEnabled {
		t.Fatal("default admin MFA is enabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("qwer1234")); err != nil {
		t.Fatal("default admin password does not match qwer1234")
	}

	var users int64
	if err := db.Model(&User{}).Count(&users).Error; err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatalf("seed created %d users, want 1", users)
	}

	if err := Seed(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&User{}).Count(&users).Error; err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatalf("repeat seed created %d users, want 1", users)
	}
}
