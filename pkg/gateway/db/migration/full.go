// Copyright 2015-present Oursky Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migration

import (
	"github.com/jmoiron/sqlx"
)

type fullMigration struct {
}

func (r *fullMigration) Version() string { return "48b46babebcb" }

func (r *fullMigration) createTable(tx *sqlx.Tx) error {
	const stmt = `
CREATE TABLE plan (
	id uuid PRIMARY KEY,
	created_at timestamp WITHOUT TIME ZONE NOT NULL,
	updated_at timestamp WITHOUT TIME ZONE NOT NULL,
	name text NOT NULL,
	auth_enabled boolean NOT NULL DEFAULT FALSE
	);

CREATE TABLE app (
	id uuid PRIMARY KEY,
	created_at timestamp WITHOUT TIME ZONE NOT NULL,
	updated_at timestamp WITHOUT TIME ZONE NOT NULL,
	name text NOT NULL,
	plan_id uuid REFERENCES plan(id) NOT NULL,
	UNIQUE (name)
);

CREATE TABLE config (
	id uuid PRIMARY KEY,
	created_at timestamp WITHOUT TIME ZONE NOT NULL,
	updated_at timestamp WITHOUT TIME ZONE NOT NULL,
	config jsonb NOT NULL,
	app_id uuid REFERENCES app(id) NOT NULL
);

CREATE TABLE domain (
	id uuid PRIMARY KEY,
	created_at timestamp WITHOUT TIME ZONE NOT NULL,
	updated_at timestamp WITHOUT TIME ZONE NOT NULL,
	domain text NOT NULL,
	app_id uuid REFERENCES app(id) NOT NULL
);

ALTER TABLE app ADD COLUMN config_id uuid NOT NULL;

ALTER TABLE ONLY app
	ADD CONSTRAINT app_config_id_fkey
	FOREIGN KEY (config_id)
	REFERENCES config(id);
	`
	_, err := tx.Exec(stmt)
	return err
}

func (r *fullMigration) insertSeedData(tx *sqlx.Tx) error {
	stmts := []string{}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (r *fullMigration) Up(tx *sqlx.Tx) error {
	var err error
	if err = r.createTable(tx); err != nil {
		return err
	}

	return r.insertSeedData(tx)
}

func (r *fullMigration) Down(tx *sqlx.Tx) error {
	panic("cannot downgrade from a base revision")
}
