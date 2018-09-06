BEGIN;

CREATE SCHEMA app_config;
SET search_path TO app_config;

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

SET search_path TO DEFAULT;

END;
