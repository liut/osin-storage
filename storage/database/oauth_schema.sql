
BEGIN;

CREATE SCHEMA IF NOT EXISTS oauth;

CREATE TABLE IF NOT EXISTS oauth.client
(
	id serial,
	code varchar(80) NOT NULL UNIQUE, -- client_id
	secret varchar(40) NOT NULL,
	redirect_uri varchar(255) NOT NULL DEFAULT '',
	meta jsonb NOT NULL DEFAULT '{}'::jsonb,
	created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS oauth.access
(
	id serial,
	client_id varchar(120) NOT NULL, -- client.code
	authorize_code varchar(140) NOT NULL,
	access_token varchar(240) NOT NULL UNIQUE,
	refresh_token varchar(240) NOT NULL DEFAULT '',
	previous varchar(240) NOT NULL DEFAULT '',
	expires_in int NOT NULL DEFAULT 86400,
	scopes varchar(255) NOT NULL DEFAULT '',
	redirect_uri varchar(255) NOT NULL DEFAULT '',
	extra jsonb NOT NULL DEFAULT '{}'::jsonb,
	is_frozen BOOLEAN NOT NULL DEFAULT false,
	created timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS oauth.refresh
(
	token varchar(240) NOT NULL UNIQUE,
	access varchar(240) NOT NULL ,
	created timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (token)
);

CREATE TABLE IF NOT EXISTS oauth.authorize
(
	id serial,
	code varchar(140) NOT NULL,
	client_id varchar(120) NOT NULL, -- client.code
	redirect_uri varchar(255) NOT NULL DEFAULT '',
	expires_in int NOT NULL DEFAULT 86400,
	scopes varchar(255) NOT NULL DEFAULT '',
	state varchar(255) NOT NULL DEFAULT '',
	extra jsonb NOT NULL DEFAULT '{}'::jsonb,
	created timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (code),
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS oauth.client_user_authorized
(
	id serial,
	client_id varchar(120) NOT NULL, -- client.code
	username varchar(120) NOT NULL DEFAULT '',
	created timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (client_id, username),
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS oauth.scopes
(
	id serial,
	name varchar(64) NOT NULL, -- ascii code
	label varchar(120) NOT NULL,
	description varchar(255) NOT NULL DEFAULT '',
	is_default BOOLEAN  NOT NULL DEFAULT false,
	UNIQUE (name),
	PRIMARY KEY (id)
);

END;
