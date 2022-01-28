package gointegration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	mysqlContainerName = "gointegration_test_mysql"
)

var (
	_, b, _, _ = runtime.Caller(0)
	basePath   = filepath.Dir(b)
)

type inspectLogs []struct {
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	NetworkSettings struct {
		Ports struct {
			TCP3306 []struct {
				HostIP   string `json:"HostIp"`
				HostPort string `json:"HostPort"`
			} `json:"3306/tcp"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func (il inspectLogs) getConnectionAndTearDown(t *testing.T, databaseName string) (*sql.DB, func(), error) {
	host := fmt.Sprintf("%s:%s", il[0].NetworkSettings.Ports.TCP3306[0].HostIP, il[0].NetworkSettings.Ports.TCP3306[0].HostPort)

	dbHost := fmt.Sprintf("tcp(%s)", host)

	err := startMysqlConnection(databaseName, dbHost)

	if err != nil {
		return nil, nil, err
	}

	err = startInformationSchemeConnection(dbHost)

	if err != nil {
		return nil, nil, err
	}

	tearDown := func() {
		_ = truncate(informationSchemeConnection, testDatabaseConnection, databaseName)
	}

	return testDatabaseConnection, tearDown, nil
}

type Configs struct {
	DatabaseName string
	SQLFilePath  string
}

func (cf Configs) NewUnit(t *testing.T) (*sql.DB, func(), error) {
	t.Helper()

	_ = os.Setenv("GO_INTEGRATION_TEST_SQL_FILE_PATH", fmt.Sprintf("%s%s", basePath, cf.SQLFilePath))
	mysqlContainerInspectLogs, err := inspectContainerLogs(mysqlContainerName)

	if err != nil {
		return nil, nil, err
	}

	if mysqlContainerInspectLogs[0].State.Status == "running" {
		return mysqlContainerInspectLogs.getConnectionAndTearDown(t, cf.DatabaseName)
	}

	dockerComposePath := basePath + "/pkg/docker-compose.yaml"
	dockerComposeCommand := exec.Command("docker", "compose", "-f", dockerComposePath, "up", "-d")

	if err := dockerComposeCommand.Run(); err != nil {
		return nil, nil, err
	}

	fmt.Println("waiting for docker compose create the dependencies")
	time.Sleep(time.Second * 3)

	mysqlContainerInspectLogs, err = inspectContainerLogs(mysqlContainerName)

	if err != nil {
		return nil, nil, err
	}

	return mysqlContainerInspectLogs.getConnectionAndTearDown(t, cf.DatabaseName)
}

func inspectContainerLogs(containerName string) (inspectLogs, error) {
	var containerLogs inspectLogs
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("docker", "inspect", containerName)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return containerLogs, err
	}

	if err := json.Unmarshal(out.Bytes(), &containerLogs); err != nil {
		return containerLogs, err
	}

	return containerLogs, nil
}

var testDatabaseConnection *sql.DB

func startMysqlConnection(dbName, dbHost string) error {
	dsn := fmt.Sprintf(
		"%v:%v@%v/%v?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=true",
		"root",
		"root",
		dbHost,
		dbName,
	)

	if db, err := sql.Open("mysql", dsn); err != nil {
		return err
	} else {
		db.SetConnMaxLifetime(0)
		testDatabaseConnection = db
		testDatabaseConnection.SetConnMaxIdleTime(time.Minute)

		return nil
	}
}

var informationSchemeConnection *sql.DB

func startInformationSchemeConnection(dbHost string) error {
	dsn := fmt.Sprintf(
		"%v:%v@%v/%v?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=true",
		"root",
		"root",
		dbHost,
		"information_schema",
	)

	if db, err := sql.Open("mysql", dsn); err != nil {
		return err
	} else {
		db.SetConnMaxLifetime(0)
		informationSchemeConnection = db
		informationSchemeConnection.SetConnMaxIdleTime(time.Minute)
		return nil
	}
}

type table struct {
	Name string `db:"TABLE_NAME"`
}

func truncate(informationSchemeConnection, testDatabaseConnection *sql.DB, databaseName string) error {
	if err := informationSchemeConnection.Ping(); err != nil {
		return err
	}

	if err := testDatabaseConnection.Ping(); err != nil {
		return err
	}

	_, err := testDatabaseConnection.Exec("SET FOREIGN_KEY_CHECKS=0")

	if err != nil {
		return err
	}

	rows, err := informationSchemeConnection.Query("SELECT TABLE_NAME FROM TABLES WHERE TABLE_SCHEMA = ?", databaseName)

	if err != nil {
		return err
	}

	for rows.Next() {
		var t = &table{}

		err := rows.Scan(&t.Name)

		if err != nil {
			fmt.Println(err.Error())

			break
		}

		_, err = testDatabaseConnection.Exec(fmt.Sprintf("TRUNCATE %v", t.Name))

		if err != nil {
			fmt.Println(err.Error())

			break
		}
	}

	return nil

}

// Wait for the database to be ready. Wait 100ms longer between each attempt.
func healthCheck(db *sql.DB, t *testing.T) {
	var pingError error

	maxAttempts := 20

	for attempts := 1; attempts <= maxAttempts; attempts++ {
		pingError = db.Ping()

		if pingError == nil {
			break
		}

		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
	}

	if pingError != nil {
		t.Fatalf("database is not ready: %v", pingError)
	}
}
