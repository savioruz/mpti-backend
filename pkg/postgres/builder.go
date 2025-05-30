package postgres

import (
	"fmt"
)

func ConnectionBuilder(host string, port int, user, password, dbName, sslMode, timezone string) string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host,
		port,
		user,
		password,
		dbName,
		sslMode,
		timezone,
	)

	return dsn
}
