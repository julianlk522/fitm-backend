package dbtest

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/julianlk522/fitm/db"
	_ "github.com/mattn/go-sqlite3"
)

func SetupTestDB() error {
	log.Print("preparing test DB client")

	TestClient, err := sql.Open("sqlite-spellfix1", "file::memory:?cache=shared")
	if err != nil {
		return fmt.Errorf("could not open in-memory DB: %s", err)
	}

	var sql_dump_path, db_dir string

	test_data_path := os.Getenv("FITM_TEST_DATA_PATH")
	if test_data_path == "" {
		log.Printf("$FITM_TEST_DATA_PATH not set, using default path")
		_, dbtest_file, _, _ := runtime.Caller(0)
		dbtest_dir := filepath.Dir(dbtest_file)
		db_dir = filepath.Join(dbtest_dir, "../db")
	} else {
		log.Print("using $FITM_TEST_DATA_PATH")
		db_dir = test_data_path
	}
	sql_dump_path = filepath.Join(db_dir, "fitm_test.db.sql")

	sql_dump, err := os.ReadFile(sql_dump_path)
	if err != nil {
		return err
	} else if _, err = TestClient.Exec(string(sql_dump)); err != nil {
		return err
	}

	// verify in-memory DB loaded test data
	var link_id string
	if err = TestClient.QueryRow("SELECT id FROM Links WHERE id = '1';").Scan(&link_id); err != nil {
		return fmt.Errorf("in-memory DB did not receive dump data: %s", err)
	}
	log.Printf("verified test DB dump data loaded")

	// verify in-memory DB has spellfix1
	if _, err = TestClient.Exec(`SELECT word, rank FROM global_cats_spellfix;`); err != nil {
		return err
	}

	db.Client = TestClient
	log.Print("switched to test DB client")

	return nil
}
