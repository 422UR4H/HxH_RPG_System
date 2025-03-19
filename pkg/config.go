package pgfs

import "fmt"

type Config struct {
	DbName        string `conf:"env:PG_DB_NAME,required"`
	DbUser        string `conf:"env:PG_DB_USER,required"`
	DbPass        string `conf:"env:PG_DB_PASS,required,mask"`
	DbHost        string `conf:"env:PG_DB_HOST,default:localhost"`
	DbPort        string `conf:"env:PG_DB_PORT,default:5432"`
	DbSSLMode     string `conf:"env:PG_DB_SSLMODE,default:require"`
	DbPoolMinSize int32  `conf:"env:PG_DB_POOL_MIN_SIZE,default:2"`
	DbPoolMaxSize int32  `conf:"env:PG_DB_POOL_MAX_SIZE,default:10"`
}

func (c *Config) ConnString() string {
	if c.DbSSLMode == "" {
		c.DbSSLMode = "disable"
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_min_conns=%d&pool_max_conns=%d",
		c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName,
		c.DbSSLMode, c.DbPoolMinSize, c.DbPoolMaxSize,
	)
}
