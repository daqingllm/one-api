package model

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
	"time"
)

var DB *gorm.DB
var LOG_DB *gorm.DB

func CreateRootAccountIfNeed() error {
	var user User
	//if user.Status != util.UserStatusEnabled {
	if err := DB.First(&user).Error; err != nil {
		logger.SysLog("no user exists, creating a root user for you: username is root, password is 123456")
		hashedPassword, err := common.Password2Hash("123456")
		if err != nil {
			return err
		}
		rootUser := User{
			Username:    "root",
			Password:    hashedPassword,
			Role:        RoleRootUser,
			Status:      UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: random.GetUUID(),
			Quota:       500000000000000,
		}
		DB.Create(&rootUser)
		if config.InitialRootToken != "" {
			logger.SysLog("creating initial root token as requested")
			token := Token{
				Id:             1,
				UserId:         rootUser.Id,
				Key:            config.InitialRootToken,
				Status:         TokenStatusEnabled,
				Name:           "Initial Root Token",
				CreatedTime:    helper.GetTimestamp(),
				AccessedTime:   helper.GetTimestamp(),
				ExpiredTime:    -1,
				RemainQuota:    500000000000000,
				UnlimitedQuota: true,
			}
			DB.Create(&token)
		}
	}
	return nil
}

func chooseDB(envName string) (*gorm.DB, error) {
	if os.Getenv(envName) != "" {
		dsn := os.Getenv(envName)
		if strings.HasPrefix(dsn, "postgres://") {
			// Use PostgreSQL
			logger.SysLog("using PostgreSQL as database")
			common.UsingPostgreSQL = true
			return gorm.Open(postgres.New(postgres.Config{
				DSN:                  dsn,
				PreferSimpleProtocol: true, // disables implicit prepared statement usage
			}), &gorm.Config{
				PrepareStmt: true, // precompile SQL
			})
		}
		// Use MySQL
		logger.SysLog("using MySQL as database")
		common.UsingMySQL = true

		if os.Getenv("CA_FILE") != "" {
			caFile := os.Getenv("CA_FILE")
			tlsConf := createTLSConf(caFile)
			err := mysql_driver.RegisterTLSConfig("custom", &tlsConf)
			if err != nil {
				log.Printf("Error %s when RegisterTLSConfig\n", err)
				return nil, err
			}
		}

		return gorm.Open(mysql.Open(dsn), &gorm.Config{
			PrepareStmt: true, // precompile SQL
		})
	}
	// Use SQLite
	logger.SysError("SQL_DSN not set!")
	return nil, errors.New("SQL_DSN not set")
}

// path to cert-files hard coded
// Most of this is copy pasted from the internet
// and used without much reflection
func createTLSConf(caFile string) tls.Config {

	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("Failed to append PEM.")
	}
	//clientCert := make([]tls.Certificate, 0, 1)

	//certs, err := tls.LoadX509KeyPair("cert/client-cert.pem", "cert/client-key.pem")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//clientCert = append(clientCert, certs)

	return tls.Config{
		RootCAs: rootCertPool,
		//Certificates:       clientCert,
	}
}

func InitDB(envName string) (db *gorm.DB, err error) {
	db, err = chooseDB(envName)
	if err == nil {
		if config.DebugSQLEnabled {
			db = db.Debug()
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(env.Int("SQL_MAX_IDLE_CONNS", 100))
		sqlDB.SetMaxOpenConns(env.Int("SQL_MAX_OPEN_CONNS", 1000))
		sqlDB.SetConnMaxLifetime(time.Second * time.Duration(env.Int("SQL_MAX_LIFETIME", 60)))

		if !config.IsMasterNode {
			return db, err
		}
		if common.UsingMySQL {
			_, _ = sqlDB.Exec("DROP INDEX idx_channels_key ON channels;") // TODO: delete this line when most users have upgraded
		}
		logger.SysLog("database migration started")
		err = db.AutoMigrate(&Channel{})
		if err != nil {
			return nil, err
		}
		err = db.AutoMigrate(&Token{})
		if err != nil {
			return nil, err
		}
		err = db.AutoMigrate(&User{})
		if err != nil {
			return nil, err
		}
		err = db.AutoMigrate(&Option{})
		if err != nil {
			return nil, err
		}
		err = db.AutoMigrate(&Redemption{})
		if err != nil {
			return nil, err
		}
		err = db.AutoMigrate(&Ability{})
		if err != nil {
			return nil, err
		}
		err = db.AutoMigrate(&Log{})
		if err != nil {
			return nil, err
		}
		logger.SysLog("database migrated")
		return db, err
	} else {
		logger.FatalLog(err)
	}
	return db, err
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Close()
	return err
}

func CloseDB() error {
	if LOG_DB != DB {
		err := closeDB(LOG_DB)
		if err != nil {
			return err
		}
	}
	return closeDB(DB)
}
