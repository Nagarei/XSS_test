package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type MySQLConnectionEnv struct {
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
}

type ModelComment struct {
	ID       int    `db:"id"`
	Name     string `db:"name"`
	Comment  string `db:"comment"`
	Admitted int    `db:"admitted"`
}

func NewMySQLConnectionEnv() *MySQLConnectionEnv {
	return &MySQLConnectionEnv{
		Host:     getEnv("MYSQL_HOST", "127.0.0.1"),
		Port:     getEnv("MYSQL_PORT", "3306"),
		User:     getEnv("MYSQL_USER", "user"),
		DBName:   getEnv("MYSQL_DBNAME", "xsstest"),
		Password: getEnv("MYSQL_PASS", "password"),
	}
}

func (mc *MySQLConnectionEnv) ConnectDB() (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true&loc=Asia%%2FTokyo", mc.User, mc.Password, mc.Host, mc.Port, mc.DBName)
	return sqlx.Open("mysql", dsn)
}

func initDB() error {
	db.MustExec("DROP TABLE IF EXISTS comments")
	db.MustExec(`
CREATE TABLE comments (
  id bigint AUTO_INCREMENT,
  name CHAR(36) NOT NULL,
  comment VARCHAR(255) NOT NULL,
  admitted tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY(id)
) ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4;`)

	db.MustExec(`INSERT INTO comments (name, comment, admitted) VALUES ("admin", "I like it", 1)`)
	db.MustExec(`INSERT INTO comments (name, comment, admitted) VALUES ("user1", "I don't like it", 0)`)

	return nil
}
