package postgres

import (
	"fmt"
)

func ConnectionBuilder(host string, port int, user, password, dbName, sslMode string) string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		password,
		dbName,
		sslMode,
	)

	return dsn
}
