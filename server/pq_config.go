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

// ConfigPG is a config wrapper for postgres dsn string passed to database/sql module
// Supports options according to the pq documentation:
// https://pkg.go.dev/github.com/lib/pq#hdr-Connection_String_Parameters
type ConfigPG struct {
	dbname                    string // The name of the database to connect to
	user                      string // The user to sign in as
	password                  string // The user's password
	host                      string // The host to connect to. Values that start with / are for unix domain sockets. (default is localhost)
	port                      string // The port to bind to. (default is 5432)
	sslmode                   string // Whether to use SSL (options are disable, required, verify-ca, verify-full)
	fallback_application_name string // An application_name to fall back to if one isn't provided.
	connect_timeout           string // Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely.
	sslcert                   string // Cert file location. The file must contain PEM encoded data.
	sslkey                    string // Key file location. The file must contain PEM encoded data.
	sslrootcert               string // The location of the root certificate file. The file must contain PEM encoded data.
}

// NewConfigPG creates a new ConfigPG and sets default values.
func NewConfigPG() *ConfigPG {
	return &ConfigPG{
	}
}

// FormatDSNPG formats the configPG into a dsn string which can be used in sql.Open
// For more detail about connection string format:
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
func (cfg *ConfigPG) FormatDSNPG() string {
	var buf bytes.Buffer
	//user = ...
	buf.WriteString("user=" + cfg.user + " ")
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
