// Package configlite provides an abstraction of application configuration values
// stored in a sqlite database. Several application can then share a single configuration database.
//
//nolint:all
package configlite

import (
	"database/sql"
	"fmt"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db *sql.DB
}

var ErrConfigNotFound = fmt.Errorf("configuration value not found")

func DefaultConfigurationFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/.config.db"
	}

	return path.Join(home, ".config.db")
}

func New(databaseName string) (*Repository, error) {
	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		return nil, fmt.Errorf("cannot open database configuration: %s - %w", databaseName, err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("cannot run database schema migrations: %s - %w", databaseName, err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Close() {
	r.db.Close()
	r.db = nil
}

func (r *Repository) GetApps() ([]string, error) {
	rows, err := r.db.Query(`SELECT name FROM applications`)
	if err != nil {
		return []string{}, fmt.Errorf("cannot query the database: %w", err)
	}
	defer rows.Close()

	apps := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return []string{}, fmt.Errorf("cannot scan application table row: %w", err)
		}
		apps = append(apps, name)
	}
	if err := rows.Err(); err != nil {
		return []string{}, fmt.Errorf("cannot browse application table: %w", err)
	}

	return apps, nil
}

func (r *Repository) GetConfigs(applicationName string) (map[string]string, error) {
	rows, err := r.db.Query(`
		SELECT configuration_name, configuration_value
		FROM configurations
		WHERE application_name = ?`, applicationName)
	if err != nil {
		return nil,
			fmt.Errorf("cannot get configurations from database: %s - %w", applicationName, err)
	}
	defer rows.Close()
	configs := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("cannot scan single config: %s - %w", applicationName, err)
		}
		configs[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf("cannot iterate over all configurations: %s - %w", applicationName, err)
	}
	return configs, nil
}

func (r *Repository) GetConfig(applicationName, configName string) (string, error) {
	rows, err := r.db.Query(`
		SELECT configuration_value
		FROM configurations
		WHERE application_name = ?
			AND configuration_name = ?`, applicationName, configName)
	if err != nil {
		return "",
			fmt.Errorf("cannot get configuration from database: (%s, %s) - %w",
				applicationName, configName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return "",
				fmt.Errorf("cannot scan configuration value from database: (%s, %s) - %w",
					applicationName, configName, err)
		}
		return value, nil
	}
	if err := rows.Err(); err != nil {
		return "",
			fmt.Errorf("cannot browse configuration value: (%s, %s) - %w",
				applicationName, configName, err)
	}

	return "", fmt.Errorf("%w: (%s, %s)", ErrConfigNotFound, applicationName, configName)
}

func (r *Repository) MustRegisterApplication(applicationName string) error {
	_, err := r.db.Exec(`INSERT INTO applications (name) VALUES (?)`, applicationName)
	return err
}

func (r *Repository) RegisterApplication(applicationName string) error {
	_, err := r.db.Exec(
		`INSERT INTO applications (name) VALUES (?) ON CONFLICT DO NOTHING`, applicationName)
	return err
}

func (r *Repository) UpsertConfig(applicationName, configName, configValue string) error {
	_, err := r.db.Exec(`
		INSERT INTO configurations (application_name, configuration_name, configuration_value)
		VALUES (?1, ?2, ?3)
		ON CONFLICT (application_name, configuration_name) DO
		UPDATE SET configuration_value = ?3`,
		applicationName,
		configName,
		configValue)
	return err
}

func (r *Repository) DeleteConfig(applicationName, configName string, likePattern bool) error {
	query := func() string {
		if likePattern {
			return `DELETE FROM configurations
						WHERE application_name = ?
							AND configuration_name LIKE ?`
		}
		return `DELETE FROM configurations
		WHERE application_name = ?
			AND configuration_name = ?`
	}
	result, err := r.db.Exec(query(), applicationName, configName)
	if err != nil {
		return fmt.Errorf("cannot delete a specific config (%s, %s): %w",
			applicationName, configName, err)
	}
	if numRows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot check the number of rows affected by the delete: %w", err)
	} else if numRows == 0 {
		return fmt.Errorf("unexpected number of affected rows: %d", numRows)
	}
	return nil
}
