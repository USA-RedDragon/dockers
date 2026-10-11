//go:build mariadb

package mariadb_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/USA-RedDragon/dockers/images/restic/cmd/restoredb/mariadb"
	"github.com/USA-RedDragon/dockers/images/restic/internal/testutil"
)

const (
	rootPassword    = "root-pw"
	appDB           = "app"
	stagingDB       = appDB + mariadb.StagingSuffix
	oldDB           = appDB + mariadb.OldSuffix
	argDropOld      = "--drop-old"
	argDryRun       = "--dry-run"
	argAddDropTable = "--add-drop-table"
)

var server string

// failAfter names the variable that makes the mariadb shim report a lost
// connection after it has run SQL that holds the variable's value.
const failAfter = "B2BK_TEST_FAIL_AFTER"

// TestMain starts one MariaDB server in Docker for the whole suite and puts a
// mariadb shim on PATH that runs the client inside it, so the suite needs
// nothing but Docker. MARIADB_IMAGE overrides the image.
func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	image := os.Getenv("MARIADB_IMAGE")
	if image == "" {
		image = "mariadb:11"
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	server = "b2bk-restoredb-e2e-" + hex.EncodeToString(suffix)
	if out, err := exec.Command("docker", "run", "-d", "--rm", "--name", server,
		"-e", "MARIADB_ROOT_PASSWORD="+rootPassword, image).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "start %s: %v\n%s", image, err, out)
		return 1
	}
	defer func() { _ = exec.Command("docker", "rm", "-f", "-v", server).Run() }()

	ready := false
	for range 120 {
		if exec.Command("docker", "exec", "-e", "MYSQL_PWD="+rootPassword, server,
			"mariadb", "-h127.0.0.1", "-uroot", "-e", "SELECT 1").Run() == nil {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		fmt.Fprintln(os.Stderr, "the MariaDB server did not come up")
		return 1
	}

	bin, err := os.MkdirTemp("", "b2bk-mariadb-shim")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(bin) }()
	run := "docker exec -i -e MYSQL_PWD " + server + " mariadb \"$@\""
	shim := "#!/bin/sh\n" +
		"if [ -n \"$" + failAfter + "\" ]; then\n" +
		"  sql=$(cat)\n" +
		"  printf '%s\\n' \"$sql\" | " + run + "\n" +
		"  status=$?\n" +
		"  case \"$sql\" in *\"$" + failAfter + "\"*)\n" +
		"    echo 'ERROR 2013 (HY000): Lost connection to server during query' >&2; exit 1;;\n" +
		"  esac\n" +
		"  exit $status\n" +
		"fi\n" +
		"exec " + run + "\n"
	if err := os.WriteFile(filepath.Join(bin, "mariadb"), []byte(shim), 0o700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = os.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return m.Run()
}

func docker(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("docker", slices.Concat([]string{"exec", "-i", "-e", "MYSQL_PWD=" + rootPassword, server},
		args)...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, errOut.String())
	}
	return out.String()
}

func sql(t *testing.T, stmts string) string {
	t.Helper()
	return docker(t, stmts, "mariadb", "-h127.0.0.1", "-uroot", "--batch", "--skip-column-names")
}

// dump takes a dump the way dumpdb mariadb does, with any extra flags.
func dump(t *testing.T, args ...string) string {
	t.Helper()
	base := []string{"mariadb-dump", "-h127.0.0.1", "-uroot", argAddDropTable, "--skip-dump-date",
		"--single-transaction"}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") && !slices.Contains(args, appDB) {
		args = append(args, appDB)
	}
	return docker(t, "", slices.Concat(base, args)...)
}

// fingerprint is a deterministic dump of a database: its tables, rows and
// auto-increment counters.
func fingerprint(t *testing.T, db string) string {
	t.Helper()
	return docker(t, "", "mariadb-dump", "-h127.0.0.1", "-uroot", "--skip-dump-date", "--skip-comments",
		"--triggers", "--routines", "--events", db)
}

func schemas(t *testing.T) []string {
	t.Helper()
	return strings.Fields(sql(t, "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA"))
}

func assertNoStaging(t *testing.T) {
	t.Helper()
	for _, s := range schemas(t) {
		if s == stagingDB || s == oldDB {
			t.Errorf("%s was left behind", s)
		}
	}
}

const seedSQL = `
DROP DATABASE IF EXISTS app; DROP DATABASE IF EXISTS app__b2bk_restore; DROP DATABASE IF EXISTS app__b2bk_old;
DROP DATABASE IF EXISTS other; DROP DATABASE IF EXISTS app2;
CREATE DATABASE app; CREATE DATABASE other;
CREATE TABLE app.parent (id INT PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE app.child (id INT AUTO_INCREMENT PRIMARY KEY, pid INT,
  FOREIGN KEY (pid) REFERENCES parent (id)) ENGINE=InnoDB;
CREATE TABLE app.keep (id INT PRIMARY KEY, v VARCHAR(20));
CREATE TABLE app.gone (id INT PRIMARY KEY);
CREATE SEQUENCE app.seq;
INSERT INTO app.parent VALUES (1), (2);
INSERT INTO app.child (pid) VALUES (1), (2), (1);
INSERT INTO app.keep VALUES (1, 'backup'), (2, 'line\nbreak');
INSERT INTO app.gone VALUES (7);
SELECT NEXTVAL(app.seq);
CREATE TABLE other.t (id INT PRIMARY KEY);
INSERT INTO other.t VALUES (42);
`

// mutateSQL changes app after the backup: rows change, a table goes away, and
// one appears that the backup never had.
const mutateSQL = `
UPDATE app.keep SET v = 'changed';
INSERT INTO app.parent VALUES (3);
INSERT INTO app.child (pid) VALUES (3);
DROP TABLE app.gone;
CREATE TABLE app.extra (id INT PRIMARY KEY);
INSERT INTO app.extra VALUES (9);
`

func seed(t *testing.T) (backup, backupState string) {
	t.Helper()
	sql(t, seedSQL)
	backupState = fingerprint(t, appDB)
	backup = dump(t)
	sql(t, mutateSQL)
	return backup, backupState
}

func restore(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()
	testutil.ClearEnv(t)
	cmd := mariadb.New()
	cmd.SetContext(context.Background())
	cmd.SetIn(strings.NewReader(input))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(slices.Concat([]string{"-H", "127.0.0.1", "-u", "root", "-p", rootPassword, "-d", appDB}, args))
	err := cmd.Execute()
	t.Logf("restoredb output:\n%s", out.String())
	return out.String(), err
}

// refused runs a restore that must fail with want and leave app exactly as it was.
func refused(t *testing.T, input string, want error, args ...string) string {
	t.Helper()
	before := fingerprint(t, appDB)
	out, err := restore(t, input, args...)
	if want != nil && !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if err == nil {
		t.Fatal("the restore succeeded")
	}
	if !strings.Contains(err.Error(), "was not changed") {
		t.Errorf("the error does not say the live database was left alone: %v", err)
	}
	if after := fingerprint(t, appDB); after != before {
		t.Errorf("the live database changed:\n--- before\n%s\n--- after\n%s", before, after)
	}
	t.Logf("error: %v", err)
	return out + err.Error()
}

func TestMariaDBRestoreSwapsAllTablesAtOnce(t *testing.T) {
	backup, want := seed(t)
	mutated := fingerprint(t, appDB)

	out, err := restore(t, backup)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := fingerprint(t, appDB); got != want {
		t.Errorf("app does not match the backup:\n--- got\n%s\n--- want\n%s", got, want)
	}
	if got := strings.Fields(sql(t, "SHOW TABLES FROM app__b2bk_old")); !slices.Equal(got,
		[]string{"child", "extra", "keep", "parent", "seq"}) {
		t.Errorf("app__b2bk_old holds %v", got)
	}
	if got := sql(t, "SELECT v FROM app__b2bk_old.keep WHERE id = 1"); got != "changed\n" {
		t.Errorf("the replaced rows were not kept: %q", got)
	}
	if slices.Contains(schemas(t), stagingDB) {
		t.Error("the staging database was left behind")
	}
	if !strings.Contains(out, "DROP DATABASE `app__b2bk_old`;") || !strings.Contains(out, "To roll back") {
		t.Errorf("the output does not say how to drop or roll back:\n%s", out)
	}

	refs := sql(t, "SELECT CONSTRAINT_SCHEMA, TABLE_NAME, UNIQUE_CONSTRAINT_SCHEMA FROM "+
		"information_schema.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_SCHEMA LIKE 'app%' ORDER BY 1")
	if refs != "app\tchild\tapp\napp__b2bk_old\tchild\tapp__b2bk_old\n" {
		t.Errorf("foreign keys point across databases:\n%s", refs)
	}

	t.Run("a second restore keeps the rollback copy", func(t *testing.T) {
		out := refused(t, backup, mariadb.ErrOldExists)
		if !strings.Contains(out, "DROP DATABASE `app__b2bk_old`") {
			t.Errorf("the refusal does not say how to drop the rollback copy:\n%s", out)
		}
		if got := fingerprint(t, oldDB); !strings.Contains(got, "'changed'") {
			t.Error("the rollback copy was touched")
		}
	})

	t.Run("the printed rollback restores the replaced tables", func(t *testing.T) {
		var stmts []string
		for line := range strings.SplitSeq(out[strings.Index(out, "To roll back"):], "\n") {
			if s := strings.TrimSpace(line); strings.HasSuffix(s, ";") {
				stmts = append(stmts, s)
			}
		}
		sql(t, strings.Join(stmts, "\n"))
		if got := fingerprint(t, appDB); got != mutated {
			t.Errorf("the rollback did not bring back the replaced tables:\n%s", got)
		}
		sql(t, "DROP DATABASE app__b2bk_old")
	})

	t.Run("--drop-old drops the replaced tables", func(t *testing.T) {
		if _, err := restore(t, backup, argDropOld); err != nil {
			t.Fatalf("restore: %v", err)
		}
		if got := fingerprint(t, appDB); got != want {
			t.Errorf("app does not match the backup")
		}
		assertNoStaging(t)
	})
}

func TestMariaDBRefusesATruncatedDump(t *testing.T) {
	backup, _ := seed(t)
	lastLine := strings.LastIndex(strings.TrimRight(backup, "\n"), "\n")
	for name, cut := range map[string]string{
		"mid-line":           backup[:len(backup)/2],
		"at a line boundary": backup[:strings.LastIndex(backup[:len(backup)/2], "\n")+1],
		"just the trailer":   backup[:lastLine+1],
	} {
		t.Run(name, func(t *testing.T) {
			refused(t, cut, mariadb.ErrIncomplete)
			assertNoStaging(t)
		})
	}
}

func TestMariaDBStopsAtAFailingStatement(t *testing.T) {
	backup, _ := seed(t)
	at := strings.Index(backup, "UNLOCK TABLES;\n") + len("UNLOCK TABLES;\n")

	t.Run("the filter refuses what is not a dump statement", func(t *testing.T) {
		out := refused(t, backup[:at]+"THIS IS NOT SQL;\n"+backup[at:], mariadb.ErrUnsupportedStatement)
		if !strings.Contains(out, "THIS IS NOT SQL") {
			t.Errorf("the refusal does not name the statement:\n%s", out)
		}
		assertNoStaging(t)
	})

	t.Run("the server's own error on a statement the filter takes", func(t *testing.T) {
		out := refused(t, backup[:at]+"INSERT INTO `no_such_table` VALUES (1);\n"+backup[at:], nil)
		if !strings.Contains(out, "ERROR 1146") {
			t.Errorf("the client's own error is missing:\n%s", out)
		}
		assertNoStaging(t)
	})
}

func TestMariaDBRefusesWhatRenameCannotMove(t *testing.T) {
	backup, _ := seed(t)

	t.Run("a trigger on a live table", func(t *testing.T) {
		sql(t, "CREATE TRIGGER app.keep_bi BEFORE INSERT ON app.keep FOR EACH ROW SET NEW.v = UPPER(NEW.v)")
		defer sql(t, "DROP TRIGGER app.keep_bi")
		refused(t, backup, mariadb.ErrUnsupportedObjects)
		if slices.Contains(schemas(t), stagingDB) {
			t.Error("the staging database was created before the refusal")
		}
	})

	t.Run("a live view", func(t *testing.T) {
		sql(t, "CREATE VIEW app.v AS SELECT id FROM app.keep")
		defer sql(t, "DROP VIEW app.v")
		refused(t, backup, mariadb.ErrUnsupportedObjects)
	})

	for name, setup := range map[string]string{
		"a view in the dump":    "CREATE VIEW app.v AS SELECT id FROM app.keep",
		"a trigger in the dump": "CREATE TRIGGER app.keep_bi BEFORE INSERT ON app.keep FOR EACH ROW SET NEW.v = 'x'",
		"a routine in the dump": "CREATE PROCEDURE app.p() SELECT 1",
	} {
		t.Run(name, func(t *testing.T) {
			sql(t, setup)
			withObject := dump(t, "--routines", appDB)
			sql(t, "DROP VIEW IF EXISTS app.v; DROP TRIGGER IF EXISTS app.keep_bi; DROP PROCEDURE IF EXISTS app.p")
			refused(t, withObject, mariadb.ErrUnsupportedObjects)
			assertNoStaging(t)
		})
	}
}

func TestMariaDBKeepsADatabasesDumpOutOfTheNamedDatabase(t *testing.T) {
	seed(t)
	named := dump(t, "--add-drop-database", "--databases", appDB)
	for _, line := range []string{"DROP DATABASE IF EXISTS `app`", "CREATE DATABASE", "USE `app`;"} {
		if !strings.Contains(named, line) {
			t.Fatalf("the --databases dump has no %q line", line)
		}
	}

	t.Run("a failure after the USE line leaves the named database alone", func(t *testing.T) {
		at := strings.Index(named, "UNLOCK TABLES;\n") + len("UNLOCK TABLES;\n")
		refused(t, named[:at]+"THIS IS NOT SQL;\n"+named[at:], nil)
		assertNoStaging(t)
	})

	t.Run("the dump restores into the database given", func(t *testing.T) {
		want := fingerprint(t, appDB)
		sql(t, "CREATE DATABASE app2; CREATE TABLE app2.junk (id INT)")
		cmdArgs := []string{"-d", "app2", argDropOld}
		out, err := restore(t, named, cmdArgs...)
		if err != nil {
			t.Fatalf("restore: %v", err)
		}
		if got := fingerprint(t, "app2"); got != want {
			t.Errorf("app2 does not match the dump of app:\n%s", got)
		}
		if got := fingerprint(t, appDB); got != want {
			t.Error("app, which the dump names, was changed")
		}
		if !strings.Contains(out, "name `app`; they were left out") {
			t.Errorf("the output does not say the database lines were left out:\n%s", out)
		}
	})

	t.Run("a dump of two databases is refused", func(t *testing.T) {
		otherBefore := fingerprint(t, "other")
		refused(t, dump(t, "--databases", appDB, "other"), mariadb.ErrMultipleDatabases)
		if fingerprint(t, "other") != otherBefore {
			t.Error("the other database was changed")
		}
		assertNoStaging(t)
	})
}

func TestMariaDBDryRunDropsStagingWithoutSwapping(t *testing.T) {
	backup, _ := seed(t)
	sql(t, "CREATE DATABASE app__b2bk_restore COMMENT '"+mariadb.SchemaComment+"'; "+
		"CREATE TABLE app__b2bk_restore.leftover (id INT)")
	before := fingerprint(t, appDB)
	out, err := restore(t, backup, argDryRun)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if fingerprint(t, appDB) != before {
		t.Error("the dry run changed the live database")
	}
	if !strings.Contains(out, "RENAME TABLE `app`.`child` TO `app__b2bk_old`.`child`") {
		t.Errorf("the dry run does not show the swap:\n%s", out)
	}
	assertNoStaging(t)

	refused(t, backup[:len(backup)/2], mariadb.ErrIncomplete, argDryRun)
}

func TestMariaDBRestoreUserPrivileges(t *testing.T) {
	backup, want := seed(t)
	sql(t, `DROP USER IF EXISTS 'limited'@'%';
CREATE USER 'limited'@'%' IDENTIFIED BY 'limited-pw';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, LOCK TABLES, REFERENCES ON app.* TO 'limited'@'%';`)
	defer sql(t, "DROP USER 'limited'@'%'")
	as := []string{"-u", "limited", "-p", "limited-pw"}

	out := refused(t, backup, mariadb.ErrPrivileges, as...)
	if !strings.Contains(out, "grant ALL on both") && !strings.Contains(out, "Access denied") {
		t.Errorf("the refusal does not explain the privileges:\n%s", out)
	}
	assertNoStaging(t)

	sql(t, "GRANT ALL ON `app\\_\\_b2bk\\_restore`.* TO 'limited'@'%';\n"+
		"GRANT ALL ON `app\\_\\_b2bk\\_old`.* TO 'limited'@'%';")
	if _, err := restore(t, backup, slices.Concat(as, []string{argDropOld})...); err != nil {
		t.Fatalf("restore with the documented grants: %v", err)
	}
	if fingerprint(t, appDB) != want {
		t.Error("app does not match the backup")
	}
	assertNoStaging(t)
}

// The dump runs in binary mode, so a client command that would switch
// databases reaches the server as text and fails instead.
func TestMariaDBDoesNotRunClientCommands(t *testing.T) {
	backup, _ := seed(t)
	otherBefore := fingerprint(t, "other")
	at := strings.Index(backup, "UNLOCK TABLES;\n") + len("UNLOCK TABLES;\n")
	refused(t, backup[:at]+"\\u other\n"+backup[at:], nil)
	if fingerprint(t, "other") != otherBefore {
		t.Error("the dump wrote into the database a client command named")
	}
	assertNoStaging(t)
}

// A transaction holding a live table makes the swap wait for its metadata
// lock. After lock_wait_timeout the RENAME fails as a whole.
func TestMariaDBSwapGivesUpOnAHeldLock(t *testing.T) {
	backup, _ := seed(t)
	holder := exec.Command("docker", "exec", "-i", "-e", "MYSQL_PWD="+rootPassword, server,
		"mariadb", "-h127.0.0.1", "-uroot")
	holder.Stdin = strings.NewReader("START TRANSACTION; SELECT * FROM app.keep; SELECT SLEEP(300);\n")
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		sql(t, "SELECT CONCAT('KILL ', ID, ';') FROM information_schema.PROCESSLIST WHERE INFO LIKE 'SELECT SLEEP(300)%'"+
			" INTO @k; EXECUTE IMMEDIATE @k")
		_ = holder.Wait()
	}()
	time.Sleep(2 * time.Second)

	out := refused(t, backup, nil)
	if !strings.Contains(out, "Lock wait timeout exceeded") {
		t.Errorf("the swap did not time out on the lock:\n%s", out)
	}
	assertNoStaging(t)
}

// One RENAME TABLE names every table four times. With hundreds of tables it is
// longer than the 128 KiB Linux allows a single argument, so it must not be
// passed to the client on its command line.
func TestMariaDBSwapsManyTables(t *testing.T) {
	sql(t, seedSQL)
	var create strings.Builder
	for i := range 700 {
		name := fmt.Sprintf("table_with_a_rather_long_name_number_%04d", i)
		fmt.Fprintf(&create, "CREATE TABLE app.%s (id INT PRIMARY KEY); INSERT INTO app.%s VALUES (%d);\n", name, name, i)
	}
	sql(t, create.String())
	want := fingerprint(t, appDB)
	backup := dump(t)
	sql(t, mutateSQL)

	for _, args := range [][]string{{argDryRun}, {argDropOld}} {
		if _, err := restore(t, backup, args...); err != nil {
			t.Fatalf("restore %v: %v", args, err)
		}
	}
	if got := fingerprint(t, appDB); got != want {
		t.Error("app does not match the backup")
	}
	assertNoStaging(t)
}

// A database whose name starts with a dash must not reach the client as an
// option.
func TestMariaDBRestoresADatabaseNamedLikeAnOption(t *testing.T) {
	backup, _ := seed(t)
	mutated := fingerprint(t, appDB)
	sql(t, "DROP DATABASE IF EXISTS `-app`; CREATE DATABASE `-app`")
	defer sql(t, "DROP DATABASE IF EXISTS `-app`")
	if _, err := restore(t, backup, "--database=-app", argDropOld); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := sql(t, "SELECT COUNT(*) FROM `-app`.parent; SELECT v FROM `-app`.keep WHERE id = 1"); got != "2\nbackup\n" {
		t.Errorf("-app holds %q", got)
	}
	if fingerprint(t, appDB) != mutated {
		t.Error("app was changed")
	}
}

// Only the staging database may change while the dump loads: a statement that
// names another database, or the server itself, is refused before it runs.
func TestMariaDBKeepsTheDumpInsideStaging(t *testing.T) {
	maxConnections := sql(t, "SELECT @@GLOBAL.max_connections")
	for name, inject := range map[string]string{
		"another database":                 "INSERT INTO `other`.`t` VALUES (43);\n",
		"the live database":                "INSERT INTO `app`.`keep` VALUES (99, 'live');\n",
		"a second statement on a line":     "SELECT 1; INSERT INTO other.t VALUES (44);\n",
		"a global variable":                "SET GLOBAL max_connections = 99;\n",
		"a global variable after another":  "SET @a = 1, GLOBAL max_connections = 99;\n",
		"a qualified table in a versioned": "/*!40000 ALTER TABLE `other`.`t` DISABLE KEYS */;\n",
		"a multi-table UPDATE":             "UPDATE `child`, `other`.`t` SET `other`.`t`.`id` = 44;\n",
		"a DELETE from a joined table":     "DELETE `t` FROM `other`.`t`;\n",
	} {
		t.Run(name, func(t *testing.T) {
			backup, _ := seed(t)
			at := strings.Index(backup, "UNLOCK TABLES;\n") + len("UNLOCK TABLES;\n")
			otherBefore := fingerprint(t, "other")
			refused(t, backup[:at]+inject+backup[at:], nil)
			if fingerprint(t, "other") != otherBefore {
				t.Error("the other database was changed")
			}
			if got := sql(t, "SELECT @@GLOBAL.max_connections"); got != maxConnections {
				t.Errorf("max_connections is %s, was %s", got, maxConnections)
				sql(t, "SET GLOBAL max_connections = "+maxConnections)
			}
			assertNoStaging(t)
		})
	}
}

// restoredb drops a staging or rollback database only when it made it.
func TestMariaDBKeepsAStagingNameItDidNotCreate(t *testing.T) {
	for name, setup := range map[string]string{
		"staging":         "CREATE DATABASE app__b2bk_restore; CREATE TABLE app__b2bk_restore.precious (id INT)",
		"rollback, empty": "CREATE DATABASE app__b2bk_old",
	} {
		t.Run(name, func(t *testing.T) {
			backup, _ := seed(t)
			sql(t, setup)
			defer sql(t, "DROP DATABASE IF EXISTS app__b2bk_restore; DROP DATABASE IF EXISTS app__b2bk_old")
			before := strings.Fields(sql(t, "SELECT CONCAT(TABLE_SCHEMA, '.', TABLE_NAME) FROM information_schema.TABLES "+
				"WHERE TABLE_SCHEMA LIKE 'app\\_\\_b2bk%' ORDER BY 1"))
			schemasBefore := schemas(t)
			for _, args := range [][]string{nil, {argDryRun}} {
				refused(t, backup, mariadb.ErrForeignSchema, args...)
			}
			after := strings.Fields(sql(t, "SELECT CONCAT(TABLE_SCHEMA, '.', TABLE_NAME) FROM information_schema.TABLES "+
				"WHERE TABLE_SCHEMA LIKE 'app\\_\\_b2bk%' ORDER BY 1"))
			if !slices.Equal(before, after) || !slices.Equal(schemasBefore, schemas(t)) {
				t.Errorf("tables %v became %v, databases %v became %v", before, after, schemasBefore, schemas(t))
			}
		})
	}
}

// A connection lost after the server ran the swap reports an error for a swap
// that happened. restoredb looks at the tables again and reports it as done.
func TestMariaDBSwapAppliedDespiteAnError(t *testing.T) {
	backup, want := seed(t)
	t.Setenv(failAfter, "RENAME TABLE `app`.")
	out, err := restore(t, backup)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !strings.Contains(out, "the swap was applied") {
		t.Errorf("the output does not say the swap went through:\n%s", out)
	}
	if got := fingerprint(t, appDB); got != want {
		t.Error("app does not match the backup")
	}
	if got := sql(t, "SELECT v FROM app__b2bk_old.keep WHERE id = 1"); got != "changed\n" {
		t.Errorf("the replaced rows were not kept: %q", got)
	}
	sql(t, "DROP DATABASE app__b2bk_old")
}

// A foreign key name is unique per database. When a schema change moved one
// from table to table, the swap must move every live table out before any
// staged table moves in, or the constraint would be in the live database twice
// partway through the RENAME TABLE.
func TestMariaDBSwapsAForeignKeyThatMovedTable(t *testing.T) {
	sql(t, `DROP DATABASE IF EXISTS app; DROP DATABASE IF EXISTS app__b2bk_restore; DROP DATABASE IF EXISTS app__b2bk_old;
CREATE DATABASE app;
CREATE TABLE app.p (id INT PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE app.a (id INT, CONSTRAINT fk1 FOREIGN KEY (id) REFERENCES app.p (id)) ENGINE=InnoDB;
CREATE TABLE app.b (id INT) ENGINE=InnoDB;
INSERT INTO app.p VALUES (1); INSERT INTO app.a VALUES (1);`)
	want := fingerprint(t, appDB)
	backup := dump(t)
	sql(t, `ALTER TABLE app.a DROP FOREIGN KEY fk1;
ALTER TABLE app.b ADD CONSTRAINT fk1 FOREIGN KEY (id) REFERENCES app.p (id);`)

	if _, err := restore(t, backup, argDropOld); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := fingerprint(t, appDB); got != want {
		t.Errorf("app does not match the backup:\n%s", got)
	}
}

// mariadb-dump qualifies a name with a database in CREATE TABLE for a foreign
// key to another database. That still restores. (A sequence default, which it
// qualifies too, cannot be restored from a mariadb-dump even by hand: its
// LOCK TABLES leaves the sequence unlocked, error 1100.)
func TestMariaDBRestoresQualifiedNamesInCreateTable(t *testing.T) {
	sql(t, seedSQL+`
CREATE TABLE other.p (id INT PRIMARY KEY) ENGINE=InnoDB;
INSERT INTO other.p VALUES (5);
CREATE TABLE app.d (pid INT, FOREIGN KEY (pid) REFERENCES other.p (id)) ENGINE=InnoDB;
INSERT INTO app.d (pid) VALUES (5);`)
	want := fingerprint(t, appDB)
	backup := dump(t)
	if !strings.Contains(backup, "REFERENCES `other`.`p`") {
		t.Fatal("the dump has no foreign key to other")
	}
	sql(t, mutateSQL)
	if _, err := restore(t, backup, argDropOld); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := fingerprint(t, appDB); got != want {
		t.Error("app does not match the backup")
	}
}

// More ways a crafted dump could reach past the staging database: a database
// name led by a digit, a MERGE table over another database's table, and an
// sql_mode that makes double quotes name things, set where the filter cannot
// read it.
func TestMariaDBKeepsCraftedDumpsInsideStaging(t *testing.T) {
	const setup = "DROP DATABASE IF EXISTS `1db`; CREATE DATABASE `1db`; CREATE TABLE `1db`.t (i INT);\n" +
		"CREATE TABLE other.mt (id INT) ENGINE=MyISAM;"
	for name, tt := range map[string]struct {
		inject string
		want   error
	}{
		"a database name led by a digit": {"INSERT INTO 1db.t VALUES (1);\n", mariadb.ErrOtherDatabase},
		"a MERGE table over another database": {"CREATE TABLE `m` (`id` INT) ENGINE=MRG_MyISAM " +
			"UNION=(`other`.`mt`) INSERT_METHOD=LAST;\nINSERT INTO `m` VALUES (42);\n", mariadb.ErrOtherDatabase},
		"an sql_mode from a variable": {"SET @m='ANSI_QUOTES';\nSET sql_mode=@m;\n" +
			"INSERT INTO \"other\".\"mt\" VALUES (43);\n", mariadb.ErrUnsupportedStatement},
		"an sql_mode from a hex literal": {"SET sql_mode=X'414E53495F51554F544553';\n" +
			"INSERT INTO \"other\".\"mt\" VALUES (44);\n", mariadb.ErrUnsupportedStatement},
		"an engine named at the end of a statement": {"ALTER TABLE `keep` ENGINE=MERGE;\n",
			mariadb.ErrOtherDatabase},
		"NAMES after another SET item":         {"SET @a=1, NAMES gbk;\n", mariadb.ErrUnsupportedStatement},
		"NAMES after a system variable":        {"SET autocommit=1, NAMES gbk;\n", mariadb.ErrUnsupportedStatement},
		"CHARACTER SET after another SET item": {"SET @a=1, CHARACTER SET gbk;\n", mariadb.ErrUnsupportedStatement},
		"CHARSET after another SET item":       {"SET @a=1, CHARSET gbk;\n", mariadb.ErrUnsupportedStatement},
		"a second NAMES":                       {"SET NAMES utf8mb4, NAMES gbk;\n", mariadb.ErrUnsupportedStatement},
		"a default engine": {"SET default_storage_engine=MRG_MyISAM;\n" +
			"CREATE TABLE `m2` (`id` INT) COMMENT='wrapper \"mysql\", database \"other\"';\n",
			mariadb.ErrUnsupportedStatement},
		"a multibyte client charset": {"SET NAMES gbk;\n" +
			"INSERT INTO `keep` VALUES (47, '\xbf\\' ; INSERT INTO `other`.`mt` VALUES (48); -- ');\n",
			mariadb.ErrUnsupportedStatement},
	} {
		t.Run(name, func(t *testing.T) {
			backup, _ := seed(t)
			sql(t, setup)
			defer sql(t, "DROP DATABASE IF EXISTS `1db`")
			at := strings.Index(backup, "UNLOCK TABLES;\n") + len("UNLOCK TABLES;\n")
			otherBefore, digitBefore := fingerprint(t, "other"), fingerprint(t, "1db")
			refused(t, backup[:at]+tt.inject+backup[at:], tt.want)
			if fingerprint(t, "other") != otherBefore || fingerprint(t, "1db") != digitBefore {
				t.Error("a database outside staging was changed")
			}
			assertNoStaging(t)
		})
	}
}

// richSQL builds a database with the shapes real apps use, so the allowlist is
// checked against what mariadb-dump writes for them.
const richSQL = `
DROP DATABASE IF EXISTS app; CREATE DATABASE app CHARACTER SET utf8mb4;
CREATE TABLE app.parent (id INT PRIMARY KEY,
  code VARCHAR(20) CHARACTER SET latin1 COLLATE latin1_bin) ENGINE=InnoDB;
CREATE TABLE app.child (
  id BIGINT AUTO_INCREMENT PRIMARY KEY, pid INT, payload BLOB, doc JSON,
  amount DECIMAL(10,2) DEFAULT 0.00, dbl DOUBLE, flt FLOAT, bits BIT(8), yr YEAR,
  tm TIME(6), st SET('x','y'), en ENUM('p','q'),
  gen INT AS (pid * 2) VIRTUAL, stored VARCHAR(10) AS (CONCAT('a', pid)) PERSISTENT,
  ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  note TEXT COMMENT 'it''s a note; really -- yes',
  CONSTRAINT ck CHECK (amount >= 0), KEY idx_pid (pid), UNIQUE KEY uq (pid, amount),
  FULLTEXT KEY ft (note),
  CONSTRAINT fk FOREIGN KEY (pid) REFERENCES parent (id) ON DELETE CASCADE
) ENGINE=InnoDB ROW_FORMAT=DYNAMIC COMMENT='table comment' AUTO_INCREMENT=100;
CREATE TABLE app.part (id INT, d DATE, PRIMARY KEY (id, d)) ENGINE=InnoDB
  PARTITION BY RANGE (YEAR(d)) (PARTITION p0 VALUES LESS THAN (2020),
  PARTITION p1 VALUES LESS THAN MAXVALUE);
CREATE TABLE app.aria (id INT) ENGINE=Aria;
CREATE SEQUENCE app.s1 START WITH 5 INCREMENT BY 2;
INSERT INTO app.parent VALUES (1,'a'),(2,'b');
INSERT INTO app.child (pid,payload,doc,amount,dbl,flt,bits,yr,tm,st,en,note) VALUES
 (1,0x00FF0A5C27,'{"k":[1,2]}',12.34,1.2345678901234e30,1.5e-3,b'10101010',2026,
  '12:34:56.789012','x,y','q','line\nbreak'),
 (2,_binary 'bin\\data',NULL,1.50,-1e-10,-0.5,255,0,'-00:00:01','','p','quote '' and -- dash'),
 (1,REPEAT('x',3000),'{}',0,0,0,0,0,'00:00:00',NULL,NULL,NULL);
INSERT INTO app.part VALUES (1,'2019-01-01'),(2,'2021-06-06');
SELECT NEXTVAL(app.s1);
`

// Every dump mariadb-dump writes of that database still restores: the
// allowlist takes the statement shapes it writes and no others.
func TestMariaDBRestoresEveryRealDumpShape(t *testing.T) {
	sql(t, richSQL)
	want := fingerprint(t, appDB)
	for name, args := range map[string][]string{
		"as dumpdb takes it":           nil,
		argAddDropTable:                {argAddDropTable},
		"--databases":                  {argAddDropTable, "--databases", appDB},
		"--skip-extended-insert":       {"--skip-extended-insert"},
		"--complete-insert --hex-blob": {"--complete-insert", "--hex-blob"},
		"--no-autocommit":              {"--no-autocommit"},
		"--quick --order-by-primary":   {"--quick", "--order-by-primary"},
		"--insert-ignore":              {"--insert-ignore"},
		"--skip-quote-names":           {"--skip-quote-names"},
	} {
		t.Run(name, func(t *testing.T) {
			backup := dump(t, args...)
			sql(t, "UPDATE app.parent SET code = 'changed'")
			if _, err := restore(t, backup, argDropOld); err != nil {
				t.Fatalf("restore: %v", err)
			}
			if got := fingerprint(t, appDB); got != want {
				t.Errorf("app does not match the backup:\n%s", firstDiff(want, got))
			}
		})
	}
}

// firstDiff names the first line where two dumps differ.
func firstDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := range max(len(w), len(g)) {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("line %d is %.120q, want %.120q", i+1, gl, wl)
		}
	}
	return "no difference"
}

// A dump taken with --no-create-info holds no tables, only rows, so the swap
// would move every live table out and bring nothing in. The restore fails
// before the swap instead of emptying the database.
func TestMariaDBRefusesADataOnlyDump(t *testing.T) {
	sql(t, richSQL)
	before := fingerprint(t, appDB)
	out, err := restore(t, dump(t, "--no-create-info"))
	if err == nil {
		t.Error("the restore succeeded on a dump that holds no tables")
	}
	t.Logf("error: %v\n%s", err, out)
	if after := fingerprint(t, appDB); after != before {
		t.Errorf("app changed: %s", firstDiff(before, after))
	}
	assertNoStaging(t)
}
