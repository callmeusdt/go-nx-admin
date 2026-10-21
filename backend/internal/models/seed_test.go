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
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("qwer1234")) == nil {
		t.Fatal("default admin still uses the published password")
	}
	if !user.MustChangePassword {
		t.Fatal("default admin must change the initial password")
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

// A legacy unique index on user_id (one session per user) is dropped on
// upgrade; the current plain index of the same name must be left alone.
func TestLegacyOnlineUserIndexIsReplacedOnlyWhenUnique(t *testing.T) {
	const name = "idx_admin_online_users_user_id"
	uniqueOf := func(db *gorm.DB) (found, unique bool) {
		idxs, err := db.Migrator().GetIndexes(&OnlineUser{})
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range idxs {
			if i.Name() == name {
				u, _ := i.Unique()
				return true, u
			}
		}
		return false, false
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if found, unique := uniqueOf(db); !found || unique {
		t.Fatalf("fresh schema: found=%v unique=%v, want a plain index", found, unique)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if found, _ := uniqueOf(db); !found {
		t.Fatal("the plain index was dropped by a re-run")
	}

	// Simulate the old schema.
	if err := db.Exec("DROP INDEX " + name).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX " + name + " ON admin_online_users(user_id)").Error; err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if found, unique := uniqueOf(db); !found || unique {
		t.Fatalf("after upgrade: found=%v unique=%v, want the unique index replaced by a plain one", found, unique)
	}
}
