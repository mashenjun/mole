package collector

import (
	"gorm.io/gorm/logger"
	"runtime"
	"time"

	"github.com/go-sql-driver/mysql"
	gormysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	// following value is just for cli acccess

	MaxDialTimeout   = 30000 // millisecond
	MaxReadTimeout   = 30000 // millisecond
	MaxWriteTimeout  = 30000 // millisecond
	MaxOpenConn      = 128
	MaxIdleConn      = 16
	MaxLifecycleConn = 300 // in second
)

// MysqlConfig defines config for gorm dialer
type MysqlConfig struct {
	DSN          string        `yaml:"dsn" json:"dsn"`
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	MaxOpenConns int           `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxLifeConns int           `yaml:"max_life_conns" json:"max_life_conns"`
}

// FillWithDefaults apply default values for field with invalid value.
func (c *MysqlConfig) FillWithDefaults() {
	if c == nil {
		return
	}
	maxCPU := runtime.NumCPU()

	if c.DialTimeout <= 0 || c.DialTimeout > time.Duration(MaxDialTimeout*maxCPU) {
		c.DialTimeout = MaxDialTimeout
	}

	if c.ReadTimeout <= 0 || c.ReadTimeout > time.Duration(MaxReadTimeout*maxCPU) {
		c.ReadTimeout = MaxReadTimeout
	}

	if c.WriteTimeout <= 0 || c.WriteTimeout > time.Duration(MaxWriteTimeout*maxCPU) {
		c.WriteTimeout = MaxWriteTimeout
	}

	if c.MaxOpenConns <= 0 || c.MaxOpenConns > MaxOpenConn*maxCPU {
		c.MaxOpenConns = MaxOpenConn
	}

	if c.MaxIdleConns <= 0 || c.MaxIdleConns > MaxIdleConn*maxCPU {
		c.MaxIdleConns = MaxIdleConn
	}

	if c.MaxLifeConns <= 0 || c.MaxLifeConns > MaxLifecycleConn*maxCPU {
		c.MaxLifeConns = MaxLifecycleConn
	}
}

func (c *MysqlConfig) formatDSN() (string, error) {
	dsn, err := mysql.ParseDSN(c.DSN)
	if err != nil {
		return "", err
	}

	// adjust timeout of DSN
	if dsn.Timeout <= 0 {
		dsn.Timeout = c.DialTimeout * time.Millisecond
	}
	if dsn.ReadTimeout <= 0 {
		dsn.ReadTimeout = c.ReadTimeout * time.Millisecond
	}
	if dsn.WriteTimeout <= 0 {
		dsn.WriteTimeout = c.WriteTimeout * time.Millisecond
	}
	return dsn.FormatDSN(), nil
}

func (c *MysqlConfig) getDialector() (gorm.Dialector, error) {
	dsn, err := c.formatDSN()
	if err != nil {
		return nil, err
	}
	return gormysql.Open(dsn), nil
}

func Dial(config *MysqlConfig) (*gorm.DB, error) {
	config.FillWithDefaults()
	d, err := config.getDialector()
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(d, &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, err
	}
	connPool, err := db.DB()
	if err != nil {
		return nil, err
	}
	if config.MaxOpenConns > 0 {
		connPool.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		connPool.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.MaxLifeConns > 0 {
		connPool.SetConnMaxLifetime(time.Duration(config.MaxLifeConns) * time.Second)
	}
	err = connPool.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}