package utils

import (
	"fmt"
	"strings"
)

// MySQLDSNInfo stores parsed information from a MySQL DSN.
type MySQLDSNInfo struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
	Params   string
}

// ParseMySQLDSN parses DSN strings in the format user:pass@tcp(host:port)/dbname?params.
func ParseMySQLDSN(dsn string) (*MySQLDSNInfo, error) {
	const marker = "@tcp("
	idx := strings.Index(dsn, marker)
	if idx == -1 {
		return nil, fmt.Errorf("unsupported DSN format: %s", dsn)
	}
	cred := dsn[:idx]
	rest := dsn[idx+len(marker):]
	end := strings.Index(rest, ")")
	if end == -1 {
		return nil, fmt.Errorf("invalid DSN, missing closing parenthesis")
	}
	hostPort := rest[:end]
	path := rest[end+1:]
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid DSN, missing database segment")
	}
	path = path[1:]

	dbName := path
	params := ""
	if q := strings.Index(path, "?"); q >= 0 {
		dbName = path[:q]
		params = path[q+1:]
	}

	user := cred
	password := ""
	if sep := strings.Index(cred, ":"); sep >= 0 {
		user = cred[:sep]
		password = cred[sep+1:]
	}

	host := hostPort
	port := "3306"
	if sep := strings.LastIndex(hostPort, ":"); sep >= 0 {
		host = hostPort[:sep]
		port = hostPort[sep+1:]
	}

	if dbName == "" {
		return nil, fmt.Errorf("database name missing in DSN")
	}

	return &MySQLDSNInfo{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,
		Params:   params,
	}, nil
}

// BuildMySQLDSN builds a DSN from MySQLDSNInfo, optionally overriding the database name.
func BuildMySQLDSN(info *MySQLDSNInfo, dbName string) string {
	if dbName == "" {
		dbName = "/"
	} else {
		dbName = "/" + dbName
	}
	if info.Params != "" {
		if strings.Contains(dbName, "?") {
			dbName += "&" + info.Params
		} else {
			dbName += "?" + info.Params
		}
	}

	userPart := info.User
	if info.Password != "" {
		userPart += ":" + info.Password
	}

	host := info.Host
	port := info.Port
	if port == "" {
		port = "3306"
	}

	return fmt.Sprintf("%s@tcp(%s:%s)%s", userPart, host, port, dbName)
}
