// Copyright 2021 Digital China Group Co.,Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import "bytes"

// Config is a config wrapper for postgres dsn string passed to database/sql module
// Supports options according to the pq documentation:
// https://pkg.go.dev/github.com/lib/pq#hdr-Connection_String_Parameters
type Config struct {
	dbname                  string // The name of the database to connect to
	user                    string // The user to sign in as
	password                string // The user's password
	host                    string // The host to connect to. Values that start with / are for unix domain sockets. (default is localhost)
	port                    string // The port to bind to. (default is 5432)
	sslmode                 string // Whether to use SSL (options are disable, required, verify-ca, verify-full)
	fallbackApplicationName string // An application_name to fall back to if one isn't provided.
	connectTimeout          string // Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely.
	sslcert                 string // Cert file location. The file must contain PEM encoded data.
	sslkey                  string // Key file location. The file must contain PEM encoded data.
	sslrootcert             string // The location of the root certificate file. The file must contain PEM encoded data.
}

// NewConfig creates a new Config and sets default values.
func NewConfig() *Config {
	return &Config{}
}

// FormatDSN formats the config into a dsn string which can be used in sql.Open
// For more detail about connection string format:
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
func (cfg *Config) FormatDSN() string {
	var buf bytes.Buffer
	//user = ...
	buf.WriteString("user=" + cfg.user + " ")

	//password = ...
	if len(cfg.password) > 0 {
		buf.WriteString("password=" + cfg.password + " ")
	}

	//host = ...
	buf.WriteString("host=" + cfg.host + " ")

	//port = ...
	buf.WriteString("port=" + cfg.port + " ")

	//dbname = ...
	buf.WriteString("dbname=" + cfg.dbname + " ")

	//sslmode = ...
	buf.WriteString("sslmode=" + cfg.sslmode)

	return buf.String()
}
