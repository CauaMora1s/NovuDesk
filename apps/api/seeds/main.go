package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "novudesk"),
		getEnv("DB_USER", "novudesk"),
		getEnv("DB_PASSWORD", "novudesk"),
		getEnv("DB_SSL_MODE", "disable"),
	)

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		log.Fatal("connect:", err)
	}
	defer db.Close()

	ctx := context.Background()

	log.Println("seeding development data...")

	// ─── System roles ──────────────────────────────────────────
	systemRoles := []struct{ id, name string }{
		{uuid.NewString(), "owner"},
		{uuid.NewString(), "admin"},
		{uuid.NewString(), "agent"},
		{uuid.NewString(), "viewer"},
	}

	roleIDs := map[string]string{}
	for _, r := range systemRoles {
		roleIDs[r.name] = r.id
		db.ExecContext(ctx,
			`INSERT INTO roles (id, name, is_system_role) VALUES ($1, $2, TRUE)
			 ON CONFLICT DO NOTHING`, r.id, r.name)
	}

	// Assign all permissions to owner and admin roles
	db.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT $1, id FROM permissions
		ON CONFLICT DO NOTHING`, roleIDs["owner"])

	db.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT $1, id FROM permissions WHERE key != 'organization:manage_settings'
		ON CONFLICT DO NOTHING`, roleIDs["admin"])

	// Agent permissions
	agentPerms := []string{
		"tickets:create", "tickets:view", "tickets:update_any",
		"tickets:assign", "tickets:change_status", "tickets:set_priority",
		"comments:create_public", "comments:create_internal",
		"teams:view", "users:view",
	}
	for _, perm := range agentPerms {
		db.ExecContext(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT $1, id FROM permissions WHERE key = $2
			ON CONFLICT DO NOTHING`, roleIDs["agent"], perm)
	}

	// Viewer gets read-only
	db.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT $1, id FROM permissions WHERE key IN ('tickets:view','teams:view','users:view')
		ON CONFLICT DO NOTHING`, roleIDs["viewer"])

	// ─── Demo organization ──────────────────────────────────────
	orgID := uuid.NewString()
	db.ExecContext(ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, 'Acme Corp', 'acme')
		 ON CONFLICT DO NOTHING`, orgID)

	// ─── Demo users ─────────────────────────────────────────────
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)

	ownerID := uuid.NewString()
	db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, locale)
		 VALUES ($1, 'admin@acme.com', $2, 'Admin User', 'pt')
		 ON CONFLICT DO NOTHING`, ownerID, string(hash))

	agentID := uuid.NewString()
	db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, locale)
		 VALUES ($1, 'agent@acme.com', $2, 'Agent User', 'pt')
		 ON CONFLICT DO NOTHING`, agentID, string(hash))

	// ─── Organization memberships ───────────────────────────────
	db.ExecContext(ctx,
		`INSERT INTO organization_members (org_id, user_id, role_id)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, orgID, ownerID, roleIDs["owner"])

	db.ExecContext(ctx,
		`INSERT INTO organization_members (org_id, user_id, role_id)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, orgID, agentID, roleIDs["agent"])

	// ─── Demo team ──────────────────────────────────────────────
	teamID := uuid.NewString()
	db.ExecContext(ctx,
		`INSERT INTO teams (id, org_id, name, description)
		 VALUES ($1, $2, 'Suporte Técnico', 'Time responsável pelo suporte técnico')
		 ON CONFLICT DO NOTHING`, teamID, orgID)

	db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		teamID, agentID)

	// ─── Demo SLA policy ─────────────────────────────────────────
	slaID := uuid.NewString()
	db.ExecContext(ctx,
		`INSERT INTO sla_policies (id, org_id, name, response_hours, resolution_hours)
		 VALUES ($1, $2, 'Padrão', 4, 24) ON CONFLICT DO NOTHING`, slaID, orgID)

	// ─── Demo tickets ────────────────────────────────────────────
	tickets := []struct {
		title, description, status, priority string
		number                               int
	}{
		{"Sistema fora do ar", "O sistema principal está inacessível desde às 09h.", "open", "urgent", 1},
		{"Erro ao gerar relatório PDF", "Ao clicar em exportar PDF, retorna erro 500.", "open", "high", 2},
		{"Atualizar dados cadastrais", "Preciso alterar o endereço de cobrança.", "pending", "normal", 3},
		{"Dúvida sobre fatura", "Não entendi o item X da minha fatura.", "resolved", "low", 4},
		{"Configurar integração Slack", "Como configuro as notificações do Slack?", "open", "normal", 5},
	}

	for _, t := range tickets {
		id := uuid.NewString()
		db.ExecContext(ctx,
			`INSERT INTO tickets (id, org_id, number, title, description, status, priority, assignee_id, team_id, sla_policy_id)
			 VALUES ($1, $2, $3, $4, $5, $6::ticket_status, $7::ticket_priority, $8, $9, $10)
			 ON CONFLICT DO NOTHING`,
			id, orgID, t.number, t.title, t.description, t.status, t.priority, agentID, teamID, slaID)
	}

	log.Println("seed complete.")
	log.Println()
	log.Println("─── Login credentials ──────────────────────────")
	log.Println("  URL:      http://localhost:5173")
	log.Println("  Org slug: acme")
	log.Println("  Owner:    admin@acme.com  / password123")
	log.Println("  Agent:    agent@acme.com  / password123")
	log.Println("────────────────────────────────────────────────")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
