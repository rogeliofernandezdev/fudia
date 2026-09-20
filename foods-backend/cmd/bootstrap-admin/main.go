package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/foods-platform/foods-backend/internal/platform/config"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const adminEmail = "admin@foods.local"

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		exit("load configuration", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx)
	if err != nil {
		exit("connect database", err)
	}
	defer db.Close()

	var existing string
	err = db.QueryRow(ctx, `SELECT id FROM users WHERE lower(email)=lower($1) LIMIT 1`, adminEmail).Scan(&existing)
	if err == nil {
		fmt.Printf("administrator already exists\nemail: %s\n", adminEmail)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		exit("check administrator", err)
	}

	password, err := temporaryPassword()
	if err != nil {
		exit("generate password", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		exit("hash password", err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		exit("begin bootstrap", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var organizationID, locationID, userID, roleID string
	err = tx.QueryRow(ctx, `INSERT INTO organizations(legal_name,trade_name,tax_id) VALUES('Foods Restaurante Demo S.A.C.','Foods Restaurante','00000000000') ON CONFLICT(tax_id) DO UPDATE SET trade_name=EXCLUDED.trade_name RETURNING id`).Scan(&organizationID)
	if err != nil {
		exit("create organization", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Local principal','PRINCIPAL','Por configurar') ON CONFLICT(organization_id,code) DO UPDATE SET name=EXCLUDED.name RETURNING id`, organizationID).Scan(&locationID)
	if err != nil {
		exit("create location", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'Administrador Foods',$3) RETURNING id`, organizationID, adminEmail, string(hash)).Scan(&userID)
	if err != nil {
		exit("create administrator", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Administrador',ARRAY['*']) ON CONFLICT(organization_id,name) DO UPDATE SET permissions=EXCLUDED.permissions RETURNING id`, organizationID).Scan(&roleID)
	if err != nil {
		exit("create role", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, userID, roleID, locationID); err != nil {
		exit("assign role", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'bootstrap','user',$3,'{"source":"bootstrap-admin"}')`, organizationID, locationID, userID); err != nil {
		exit("audit bootstrap", err)
	}
	if err = tx.Commit(ctx); err != nil {
		exit("commit bootstrap", err)
	}
	fmt.Printf("administrator created\nemail: %s\ntemporary password: %s\n", adminEmail, password)
}

func temporaryPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "F!" + base64.RawURLEncoding.EncodeToString(raw), nil
}
func exit(step string, err error) { fmt.Fprintf(os.Stderr, "%s: %v\n", step, err); os.Exit(1) }
