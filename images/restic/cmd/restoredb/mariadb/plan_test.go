package mariadb

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	weird       = "we`ird"
	testUser    = "root"
	createTable = "CREATE TABLE `t` (i int);\n"
)

func TestNames(t *testing.T) {
	t.Parallel()
	staging, old, err := Names("app")
	if err != nil || staging != "app__b2bk_restore" || old != "app__b2bk_old" {
		t.Errorf("Names(app) = %q, %q, %v", staging, old, err)
	}
	if _, _, err := Names(strings.Repeat("x", 50)); err != nil {
		t.Errorf("a 50 character name leaves room for the suffix: %v", err)
	}
	if _, _, err := Names(strings.Repeat("x", 51)); !errors.Is(err, ErrNameTooLong) {
		t.Errorf("err = %v, want ErrNameTooLong", err)
	}
	if _, _, err := Names(""); !errors.Is(err, ErrNoDatabase) {
		t.Errorf("err = %v, want ErrNoDatabase", err)
	}
}

func TestQuoteIdent(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		testLive:  "`app`",
		weird:     "`we``ird`",
		"sp ace":  "`sp ace`",
		"`":       "````",
		"a.b":     "`a.b`",
		"tab\tin": "`tab\tin`",
	} {
		if got := QuoteIdent(in); got != want {
			t.Errorf("QuoteIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLiteralIsHex(t *testing.T) {
	t.Parallel()
	if got := literal("a'b\\"); got != "X'6127625c'" {
		t.Errorf("literal = %s", got)
	}
}

func TestRenameStatement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		current, incoming []string
		want              string
	}{
		{
			name:     "tables on both sides",
			current:  []string{"b", "a"},
			incoming: []string{"a", "b"},
			want: "RENAME TABLE `app`.`a` TO `old`.`a`, `app`.`b` TO `old`.`b`, " +
				"`stg`.`a` TO `app`.`a`, `stg`.`b` TO `app`.`b`",
		},
		{
			name:     "a live-only table moves out, a staging-only table moves in",
			current:  []string{"extra", "keep"},
			incoming: []string{"gone", "keep"},
			want: "RENAME TABLE `app`.`extra` TO `old`.`extra`, `app`.`keep` TO `old`.`keep`, " +
				"`stg`.`gone` TO `app`.`gone`, `stg`.`keep` TO `app`.`keep`",
		},
		{
			name:     "an empty live database",
			incoming: []string{"t"},
			want:     "RENAME TABLE `stg`.`t` TO `app`.`t`",
		},
		{
			name:    "an empty dump empties the live database",
			current: []string{"t"},
			want:    "RENAME TABLE `app`.`t` TO `old`.`t`",
		},
		{
			name: "nothing to move",
			want: "",
		},
		{
			name:     "quoted names",
			current:  []string{weird},
			incoming: []string{weird},
			want:     "RENAME TABLE `app`.`we``ird` TO `old`.`we``ird`, `stg`.`we``ird` TO `app`.`we``ird`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RenameStatement("app", "stg", "old", tt.current, tt.incoming); got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

// Every live table is moved out before any staged table moves in: MariaDB
// applies the parts in order, and a foreign key name must stay unique in live.
func TestRenameStatementMovesOutBeforeIn(t *testing.T) {
	t.Parallel()
	got := RenameStatement("app", "stg", "old", []string{"b", "a"}, []string{"a", "c"})
	want := "RENAME TABLE `app`.`a` TO `old`.`a`, `app`.`b` TO `old`.`b`, `stg`.`a` TO `app`.`a`, " +
		"`stg`.`c` TO `app`.`c`"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestDumpComplete(t *testing.T) {
	t.Parallel()
	body := "-- MariaDB dump 10.20\nINSERT INTO `t` VALUES (1);\n/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;\n\n"
	tests := []struct {
		name string
		dump string
		want bool
	}{
		{"--skip-dump-date, as dumpdb runs it", body + "-- Dump completed\n", true},
		{"with the date", body + "-- Dump completed on 2026-10-11  9:15:12\n", true},
		{"trailing blank lines", body + "-- Dump completed\n\n\n", true},
		{"CRLF", strings.ReplaceAll(body+"-- Dump completed\n", "\n", "\r\n"), true},
		{"no trailing newline", body + "-- Dump completed", true},
		{"cut before the trailer", body, false},
		{"cut inside the trailer", body + "-- Dump compl", false},
		{"cut inside a row", body + "INSERT INTO `t` VALUES (1,'-- Dump completed", false},
		{"the trailer, then more", body + "-- Dump completed\nINSERT INTO `t` VALUES (2);\n", false},
		{"a lookalike", body + "-- Dump completedness\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		if got := DumpComplete([]byte(tt.dump)); got != tt.want {
			t.Errorf("%s: DumpComplete = %t, want %t", tt.name, got, tt.want)
		}
	}
}

// TestAllowStatement pins the statement shapes the allowlist takes and the
// kind it gives each. Anything not shaped like a mariadb-dump statement is
// refused, so the cases below are the whole of what a dump may hold.
func TestAllowStatement(t *testing.T) {
	t.Parallel()
	const createFK = "CREATE TABLE `t` (\n  `id` int,\n" +
		"  CONSTRAINT `fk` FOREIGN KEY (`id`) REFERENCES `other`.`p` (`id`)\n)"
	tests := []struct {
		stmt  string
		kind  stmtKind
		arg   string
		shape shape
	}{
		{stmt: "USE `app`", kind: stmtDatabase, arg: testLive},
		{stmt: "use `My``Db`", kind: stmtDatabase, arg: "My`Db"},
		{stmt: "USE\t`app`", kind: stmtDatabase, arg: testLive},
		{stmt: "USE\n`app`", kind: stmtDatabase, arg: testLive},
		{stmt: "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `app` /*!40100 DEFAULT CHARACTER SET utf8mb4 */",
			kind: stmtDatabase, arg: testLive},
		{stmt: "CREATE DATABASE `app`", kind: stmtDatabase, arg: testLive},
		{stmt: "CREATE  DATABASE `app`", kind: stmtDatabase, arg: testLive},
		{stmt: "CREATE SCHEMA IF NOT EXISTS `app`", kind: stmtDatabase, arg: testLive},
		{stmt: "/*!40000 DROP DATABASE IF EXISTS `app`*/", kind: stmtDatabase, arg: testLive},
		{stmt: "DROP\tDATABASE `app`", kind: stmtDatabase, arg: testLive},
		{stmt: "DROP  SCHEMA `app`", kind: stmtDatabase, arg: testLive},
		{stmt: "DROP TABLE IF EXISTS `t`", kind: stmtObject},
		{stmt: "DROP TABLE `t`, `u`", kind: stmtObject},
		{stmt: "DROP SEQUENCE IF EXISTS `s`", kind: stmtPass},
		{stmt: "CREATE TABLE `t` (\n", kind: stmtObject, shape: shapeTableDef},
		{stmt: createFK, kind: stmtObject, shape: shapeTableDef},
		{stmt: "CREATE SEQUENCE `s` start with 5 minvalue 1 increment by 2 cache 1000 nocycle ENGINE=InnoDB",
			kind: stmtObject},
		{stmt: "DO SETVAL(`s`, 2005, 0)", kind: stmtPass},
		{stmt: "SELECT SETVAL(`s`, 1001, 0)", kind: stmtPass},
		{stmt: "LOCK TABLES `t` WRITE", kind: stmtPass},
		{stmt: "LOCK TABLES `t` READ LOCAL, `u` WRITE", kind: stmtPass},
		{stmt: "UNLOCK TABLES", kind: stmtPass},
		{stmt: "/*!40000 ALTER TABLE `t` DISABLE KEYS */", kind: stmtPass},
		{stmt: "/*!40000 ALTER TABLE `t` ENABLE KEYS */", kind: stmtPass},
		{stmt: "INSERT INTO `t` VALUES (1)", kind: stmtObject, shape: shapeValues},
		{stmt: "INSERT INTO `t` (`a`, `b`) VALUES (1,'x')", kind: stmtObject, shape: shapeValues},
		{stmt: "INSERT INTO `t` VALUES", kind: stmtObject, shape: shapeValues},
		{stmt: "COMMIT", kind: stmtPass},
		{stmt: "START TRANSACTION", kind: stmtPass},
		{stmt: "/*!40101 SET NAMES utf8mb4 */", kind: stmtPass},
		{stmt: "SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO'", kind: stmtPass},
		{stmt: "SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0", kind: stmtPass},
		{stmt: "SET AUTOCOMMIT=@OLD_AUTOCOMMIT", kind: stmtPass},
		{stmt: "/*M!999999 enable the sandbox mode */", kind: stmtPass},
		{stmt: "/*!40101 SET @saved_cs_client     = @@character_set_client */", kind: stmtPass},
		{stmt: "DELIMITER ", kind: stmtRefused, arg: objectsTriggers},
		{stmt: "  delimiter //\nCREATE TRIGGER x", kind: stmtRefused, arg: objectsTriggers},
		{stmt: "/*!50001 CREATE ALGORITHM=UNDEFINED */", kind: stmtRefused, arg: objectsViews},
		{stmt: "CREATE DEFINER=`root`@`%` PROCEDURE `p`() SELECT 1", kind: stmtRefused, arg: objectsTriggers},
		{stmt: "CREATE OR REPLACE VIEW `v` AS SELECT 1", kind: stmtRefused, arg: objectsViews},
		{stmt: "DROP VIEW IF EXISTS `v`", kind: stmtRefused, arg: objectsViews},
		{stmt: "CREATE USER `x`", kind: stmtServer, arg: "CREATE USER `x`"},
		{stmt: "GRANT ALL ON *.* TO `x`", kind: stmtServer, arg: "GRANT ALL ON *.* TO `x`"},
		{stmt: "CHANGE MASTER TO MASTER_LOG_POS=4", kind: stmtServer, arg: "CHANGE MASTER TO MASTER_LOG_POS=4"},
		{stmt: "START SLAVE", kind: stmtServer, arg: "START SLAVE"},
		{stmt: "ALTER USER `x` IDENTIFIED BY 'y'", kind: stmtServer, arg: "ALTER USER `x` IDENTIFIED BY 'y'"},
	}
	for _, tt := range tests {
		kind, arg, sh := allowStatement(tt.stmt, true)
		if kind != tt.kind || sh != tt.shape || tt.arg != "" && arg != tt.arg {
			t.Errorf("allowStatement(%q) = %d, %q, %d, want %d, %q, %d",
				tt.stmt, kind, arg, sh, tt.kind, tt.arg, tt.shape)
		}
	}
}

// Everything the allowlist does not know is refused, whatever it would have
// done on the server.
func TestAllowStatementRefusesEverythingElse(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"USE app",
		"USE `a`.`b`",
		"ALTER DATABASE `app` CHARACTER SET latin1",
		"ALTER  DATABASE `app` CHARACTER SET gbk",
		"CREATE OR REPLACE DATABASE `app`",
		"SELECT 1",
		"SELECT * FROM `t`",
		"DO 1",
		"TRUNCATE `t`",
		"DELETE FROM `t`",
		"UPDATE `t` SET `a` = 1",
		"ANALYZE TABLE `t`",
		"OPTIMIZE TABLE `t`",
		"CHECK TABLE `t`",
		"REPAIR TABLE `t`",
		"RENAME TABLE `t` TO `u`",
		"ALTER TABLE `t` ADD COLUMN `b` int",
		"ALTER TABLE `t` DISABLE KEYS, ENABLE KEYS",
		"LOCK TABLES `t`",
		"LOCK TABLES `t` WRITE,",
		"UNLOCK TABLES `t`",
		"INSERT INTO `t` SET `a` = 1",
		"INSERT INTO `t` SELECT * FROM `u`",
		"INSERT INTO `t` VALUES (REPEAT('x',10))",
		"INSERT INTO `t` VALUES (`a`)",
		"INSERT INTO `t` VALUES (1) ON DUPLICATE KEY UPDATE `a` = 1",
		"CREATE TABLE `t` AS SELECT 1",
		"CREATE TABLE `t` LIKE `u`",
		"DROP TABLE `t` CASCADE",
		"SET @a = (SELECT 1)",
		"SET innodb_fake_variable = 1",
		"SET NAMES utf8mb4, @a = (SELECT 1)",
		"DO SETVAL(`s`, (SELECT 1), 0)",
		"SELECT SETVAL(`s`, 1) INTO OUTFILE '/tmp/x'",
		"COMMIT AND CHAIN",
		"START TRANSACTION WITH CONSISTENT SNAPSHOT",
		"(SELECT 1)",
		"1",
	} {
		kind, _, _ := allowStatement(stmt, true)
		if kind != stmtUnsupported && kind != stmtServer {
			t.Errorf("allowStatement(%q) = %d, want it refused", stmt, kind)
		}
	}
}

// A statement cut short by the head is refused unless its shape says the rest
// is data the watchers check.
func TestAllowStatementRefusesACutStatement(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"DROP",
		"DROP TABLE",
		"DROP  DATABASE",
		"CREATE",
		"CREATE TABLE",
		"LOCK TABLES `t` WRITE, ",
		"SET @OLD_SQL_MODE=",
		"USE",
		"INSERT INTO `t` (`a`",
	} {
		if kind, _, _ := allowStatement(stmt, false); kind != stmtUnsupported {
			t.Errorf("allowStatement(%q, false) = %d, want it refused", stmt, kind)
		}
	}
	for _, stmt := range []string{"INSERT INTO `t` VALUES (1", "CREATE TABLE `t` (`a` int"} {
		if kind, _, _ := allowStatement(stmt, false); kind != stmtObject {
			t.Errorf("allowStatement(%q, false) = %d, want it allowed", stmt, kind)
		}
	}
}

func filterString(t *testing.T, dump string) (string, *dumpFilter, error) {
	t.Helper()
	var out strings.Builder
	f := &dumpFilter{}
	err := f.copy(&out, bufio.NewReaderSize(strings.NewReader(dump), 64))
	return out.String(), f, err
}

func TestFilterLeavesOutDatabaseLines(t *testing.T) {
	t.Parallel()
	dump := "/*!40000 DROP DATABASE IF EXISTS `app`*/;\n" +
		"CREATE DATABASE /*!32312 IF NOT EXISTS*/ `app`;\n" +
		"USE `app`;\n" +
		"CREATE TABLE `t` (\n  `id` int\n);\n" +
		"INSERT INTO `t` VALUES (1),(2),(3),(4),(5),(6),(7),(8),(9);\n" +
		"USE `app`;\n" +
		"-- Dump completed\n"
	got, f, err := filterString(t, dump)
	if err != nil {
		t.Fatal(err)
	}
	want := "\n\n\nCREATE TABLE `t` (\n  `id` int\n);\n" +
		"INSERT INTO `t` VALUES (1),(2),(3),(4),(5),(6),(7),(8),(9);\n" +
		"\n-- Dump completed\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if f.database != testLive {
		t.Errorf("database = %q", f.database)
	}
	if !f.complete() {
		t.Errorf("tail = %q", f.tail)
	}
}

// A line longer than the reader's buffer is passed on whole, and text inside
// it that looks like a statement is data.
func TestFilterPassesLongLines(t *testing.T) {
	t.Parallel()
	dump := "INSERT INTO `t` VALUES ('" + strings.Repeat("x", 100) + "\\nUSE `other`;" +
		strings.Repeat("USE `x`;", 20) + "');\n-- Dump completed\n"
	got, f, err := filterString(t, dump)
	if err != nil || got != dump {
		t.Errorf("got %q, %v", got, err)
	}
	if f.database != "" {
		t.Errorf("database = %q", f.database)
	}
}

func TestFilterRefusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, dump string
		want       error
		passed     string
	}{
		{
			name: "two databases",
			dump: "USE `a`;\nCREATE TABLE `t` (i int);\nUSE `b`;\nCREATE TABLE `u` (i int);\n",
			want: ErrMultipleDatabases, passed: "\n" + createTable,
		},
		{
			name: "a database named after the first table",
			dump: "CREATE TABLE `t` (i int);\nUSE `a`;\n",
			want: ErrMultipleDatabases, passed: createTable,
		},
		{
			name: "a trigger",
			dump: "CREATE TABLE `t` (i int);\nDELIMITER ;;\n/*!50003 CREATE*/ /*!50003 TRIGGER x */;;\n",
			want: ErrUnsupportedObjects, passed: createTable,
		},
		{
			name: "a view",
			dump: "/*!50001 DROP VIEW IF EXISTS `v`*/;\n",
			want: ErrUnsupportedObjects,
		},
		{
			name: "ALTER DATABASE",
			dump: "ALTER DATABASE `app` CHARACTER SET latin1 ;\n",
			want: ErrUnsupportedStatement,
		},
		{
			name: "an unquoted USE",
			dump: "USE app;\n",
			want: ErrUnsupportedStatement,
		},
		{
			name: "an indented DELIMITER",
			dump: createTable + "  DELIMITER ;;\n",
			want: ErrUnsupportedObjects, passed: createTable,
		},
		{
			name: "a second statement on a line",
			dump: createTable + "COMMIT; INSERT INTO `other`.`t` VALUES (1);\n",
			want: ErrOtherDatabase, passed: createTable,
		},
		{
			name: "a statement split over lines",
			dump: createTable + "INSERT INTO\n`other`\n.`t` VALUES (1);\n",
			want: ErrOtherDatabase, passed: createTable,
		},
		{
			name: "a comment between the words",
			dump: createTable + "SET/* x */GLOBAL max_connections = 1;\n",
			want: ErrServerStatement, passed: createTable,
		},
		{
			name: "a statement no mariadb-dump writes",
			dump: createTable + "SELECT 1;\n",
			want: ErrUnsupportedStatement, passed: createTable,
		},
		{
			name: "replication",
			dump: createTable + "CHANGE MASTER TO MASTER_LOG_FILE='mysql-bin.000001', MASTER_LOG_POS=4;\n",
			want: ErrServerStatement, passed: createTable,
		},
		{
			name: "an executable comment left open on an earlier line",
			dump: createTable + "/*!40000\n/* */ GRANT ALL ON `mysql`.* TO 'z'@'%'; /* */\nSELECT 1;\n",
			want: ErrServerStatement, passed: createTable,
		},
		{
			name: "a table named past the part read before passing it on",
			dump: createTable + "INSERT" + strings.Repeat(" ", 4200) + "INTO `other`.`t` VALUES (1);\n",
			want: ErrUnsupportedStatement, passed: createTable,
		},
		{
			name: "a table named after a long comment",
			dump: createTable + "INSERT /*" + strings.Repeat("x", 70000) + "*/ INTO `other`.`t` VALUES (1);\n",
			want: ErrUnsupportedStatement, passed: createTable,
		},
		{
			name: "a file written by the server",
			dump: createTable + "SELECT * FROM `t` INTO OUTFILE '/tmp/x';\n",
			want: ErrUnsupportedStatement, passed: createTable,
		},
		{
			name: "a client command",
			dump: createTable + "\\u other\n",
			want: ErrClientCommand, passed: createTable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _, err := filterString(t, tt.dump)
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if got != tt.passed {
				t.Errorf("passed on %q, want %q", got, tt.passed)
			}
		})
	}
}

func TestParseBatch(t *testing.T) {
	t.Parallel()
	rows := parseBatch("a\tb\\tc\nd\\\\e\tf\\ng\n\n")
	if len(rows) != 2 || rows[0][0] != "a" || rows[0][1] != "b\tc" || rows[1][0] != `d\e` || rows[1][1] != "f\ng" {
		t.Errorf("rows = %q", rows)
	}
	if rows := parseBatch(""); len(rows) != 0 {
		t.Errorf("empty output gave %q", rows)
	}
}

func TestClientError(t *testing.T) {
	t.Parallel()
	stderr := "WARNING: option --ssl-verify-server-cert is disabled\nERROR 1044 (42000): Access denied\n"
	if got := clientError(stderr); got != "ERROR 1044 (42000): Access denied" {
		t.Errorf("got %q", got)
	}
	if got := clientError("something else\n"); got != "something else" {
		t.Errorf("got %q", got)
	}
}

func TestClientArgs(t *testing.T) {
	t.Parallel()
	c := &client{host: "db", port: "3306", user: testUser, extra: []string{"--skip-ssl"}}
	got := strings.Join(c.args("--batch"), " ")
	want := "--host=db --port=3306 --user=root --default-character-set=utf8mb4 --batch --skip-ssl"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// fakeClient puts a mariadb on PATH that saves its arguments and stdin in dir,
// and returns a client that runs it.
func fakeClient(t *testing.T) (c *client, dir string) {
	t.Helper()
	dir = t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >" + dir + "/args\ncat >" + dir + "/stdin\n"
	if err := os.WriteFile(filepath.Join(dir, "mariadb"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &client{host: "db", port: "3306", user: testUser, stdout: io.Discard, stderr: io.Discard}, dir
}

func saved(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// SQL goes to the client on stdin: a swap of many tables is longer than Linux
// allows a single argument.
func TestClientSendsSQLOnStdin(t *testing.T) {
	c, dir := fakeClient(t)
	stmt := "RENAME TABLE " + strings.Repeat("`app`.`a_long_table_name` TO `app__b2bk_old`.`a_long_table_name`, ", 4000)
	if err := c.execute(context.Background(), stmt); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if args := saved(t, dir, "args"); strings.Contains(args, "RENAME") {
		t.Errorf("the SQL is on the command line: %.200s", args)
	}
	if stdin := saved(t, dir, "stdin"); !strings.Contains(stdin, stmt) {
		t.Errorf("stdin holds %d bytes, not the statement", len(stdin))
	}
}

// A database named like an option is passed as --database=, never on its own.
func TestClientLoadNamesTheDatabaseAsAnOption(t *testing.T) {
	c, dir := fakeClient(t)
	if err := c.load(context.Background(), "-app__b2bk_restore", func(io.Writer) error { return nil }); err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(saved(t, dir, "args")), "\n")
	if !slices.Contains(args, "--database=-app__b2bk_restore") || slices.Contains(args, "-app__b2bk_restore") {
		t.Errorf("args = %q", args)
	}
}

// A name qualified with a database is refused wherever it is in a statement,
// except in CREATE TABLE, where mariadb-dump writes them for a foreign key to
// another database and for a sequence default, and nothing runs.
func TestFilterRefusesNamesInOtherDatabases(t *testing.T) {
	t.Parallel()
	rows := strings.Repeat("(1,'a.b -- `x`.`y`'),", 400)
	for _, stmt := range []string{
		"UPDATE `t`, `other`.`u` SET `other`.`u`.`x`=1",
		"DELETE `other`.`u` FROM `other`.`u`",
		"DELETE `t`, `u` FROM `t` JOIN `other`.`u`",
		"DELETE `t` FROM `other`.`t`",
		"ALTER TABLE `t` EXCHANGE PARTITION p0 WITH TABLE `other`.`u`",
		"INSERT INTO `t` SELECT * FROM `other`.`u`",
		"ALTER TABLE `t` RENAME `other`.`u`",
		"DO NEXTVAL(`other`.`s`)",
		"DO other.f()",
		"ANALYZE TABLE `other`.`u`",
		"SELECT `other` . `f`()",
		"SELECT `other`/* x */.`f`()",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrOtherDatabase) {
			t.Errorf("%.80q: err = %v", stmt, err)
		}
		rest := stmt[strings.LastIndex(stmt, "other")-1:]
		if !strings.HasPrefix(got, createTable) || strings.Contains(got, rest) {
			t.Errorf("%.80q: passed on %.200q", stmt, got)
		}
	}
	if _, _, err := filterString(t, "CREATE TABLE `t2` AS SELECT 1;\n"); !errors.Is(err, ErrUnsupportedStatement) {
		t.Errorf("CREATE TABLE ... SELECT: err = %v", err)
	}
	// A name past the head of a long INSERT is caught by the value check as it
	// streams, after the head of the INSERT has gone to the client. Nothing
	// runs: load kills the client before it sees the end of its input.
	long := createTable + "INSERT INTO `t` VALUES " + rows + "(`other`.`f`());\n"
	got, _, err := filterString(t, long)
	if !errors.Is(err, ErrUnsupportedStatement) || !strings.HasPrefix(got, createTable) ||
		strings.Contains(got, "`other`") {
		t.Errorf("a name past the head: err = %v, passed %.80q", err, got)
	}
	for _, stmt := range []string{
		"CREATE TABLE `d` (\n  `id` int(11) DEFAULT nextval(`app`.`s`),\n" +
			"  CONSTRAINT `fk` FOREIGN KEY (`id`) REFERENCES `other`.`p` (`id`)\n) ENGINE=InnoDB",
		"SET @@SESSION.SQL_LOG_BIN = 0",
		"/*!40101 SET @saved_cs_client     = @@character_set_client */",
		"INSERT INTO `a.b` VALUES (1.5,'a.b',0x2E,.5,-1.2345e30,_binary 'x.y',X'2E')," + rows + "(2,NULL)",
		"DO SETVAL(`s`, 1001, 0)",
	} {
		if _, _, err := filterString(t, stmt+";\n"); err != nil {
			t.Errorf("%.80q: err = %v", stmt, err)
		}
	}
}

// A refused dump stops the client before it sees the end of its input: the
// client would take that as the end of the statement it was reading, and run
// it.
func TestClientLoadKillsTheClientWhenTheDumpIsRefused(t *testing.T) {
	c, dir := fakeClient(t)
	script := "#!/bin/sh\ndata=$(cat)\necho \"$data\" >" + dir + "/eof\n"
	if err := os.WriteFile(filepath.Join(dir, "mariadb"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	refused := errors.New("refused")
	err := c.load(context.Background(), "app__b2bk_restore", func(w io.Writer) error {
		_, _ = io.WriteString(w, "INSERT INTO `t` VALUES (1)")
		return refused
	})
	if !errors.Is(err, refused) {
		t.Errorf("err = %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, statErr := os.Stat(filepath.Join(dir, "eof")); statErr == nil {
		t.Error("the client read to the end of its input")
	}
}

// MariaDB reads a word led by a digit, such as 1db, as a name unless it is all
// digits, so 1db.t names a table in database 1db.
func TestFilterRefusesADatabaseNameLedByADigit(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"INSERT INTO 1db.t VALUES (1)",
		"UPDATE `t` SET `c` = (SELECT c FROM 9x.t2)",
		"DO `other`.1f()",
		"DO 1db . `f`()",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrOtherDatabase) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
	if _, _, err := filterString(t, "INSERT INTO `t` VALUES (1.5e3,0x1F,1e5,.5,1.,12.34,-0.0001);\n"); err != nil {
		t.Errorf("numbers: err = %v", err)
	}
}

// A MERGE table, or one whose engine reaches another server or database,
// writes past the staging database without naming a qualified table in the
// statement that writes.
func TestFilterRefusesEnginesThatReachOutside(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"CREATE TABLE `m` (`id` INT) ENGINE=MRG_MyISAM UNION=(`other`.`t`) INSERT_METHOD=LAST",
		"CREATE TABLE `m` (`id` INT) ENGINE = merge UNION = (`t1`)",
		"CREATE TABLE `m` (`id` INT) UNION=(`t1`)",
		"CREATE TABLE `f` (`id` INT) ENGINE=FEDERATED CONNECTION='mysql://u@h/other/t'",
		"CREATE TABLE `c` (`id` INT) ENGINE=CONNECT TABLE_TYPE=MYSQL DBNAME='other' TABNAME='t'",
		"CREATE TABLE `s` (`id` INT) ENGINE=SPIDER COMMENT='wrapper \"mysql\", table \"t\"'",
		"CREATE TABLE `q` (`id` INT) ENGINE='MRG_MyISAM'",
		"CREATE TABLE `q` (`id` INT) ENGINE=`MERGE`",
		"/*!40000 ALTER TABLE `t` ENGINE=MERGE */",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrOtherDatabase) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
	for _, stmt := range []string{
		"CREATE TABLE `t2` (\n  `id` int(11) DEFAULT nextval(`app`.`s`),\n" +
			"  CONSTRAINT `fk` FOREIGN KEY (`id`) REFERENCES `other`.`p` (`id`)\n" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci",
		"CREATE TABLE `t3` (`id` int) ENGINE=MyISAM",
		"CREATE TABLE `t4` (`union` int, `connection` int) ENGINE=Aria PAGE_CHECKSUM=1",
		"CREATE SEQUENCE `s` start with 1 minvalue 1 maxvalue 9223372036854775806 increment by 1 cache 1000 " +
			"nocycle ENGINE=InnoDB",
	} {
		if _, _, err := filterString(t, stmt+";\n"); err != nil {
			t.Errorf("%q: err = %v", stmt, err)
		}
	}
}

// sql_mode decides how quotes read, so it may only be set to a literal the
// filter can check, or back to the mode a dump saved at its start.
func TestFilterFollowsSQLMode(t *testing.T) {
	t.Parallel()
	const save = "/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;\n"
	const restore = "/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;\n"
	for _, dump := range []string{
		"SET @m='ANSI_QUOTES';\nSET sql_mode=@m;\n",
		"SET sql_mode=X'414E53495F51554F544553';\n",
		"SET sql_mode=0x414E53495F51554F544553;\n",
		"SET SESSION sql_mode=CONCAT('AN', 'SI_QUOTES');\n",
		"SET @@session.sql_mode=@m;\n",
		"SET LOCAL sql_mode := @m;\n",
		"SET sql_mode=@never_saved;\n",
		save + "SET @OLD_SQL_MODE='ANSI_QUOTES';\n" + restore,
		save + "SELECT 'ANSI_QUOTES' INTO @OLD_SQL_MODE;\n" + restore,
		save + "DO @OLD_SQL_MODE:='ANSI_QUOTES';\n" + restore,
		save + "SET @a=(@OLD_SQL_MODE:='ANSI_QUOTES');\n" + restore,
		save + "INSERT INTO `t` VALUES (@`OLD_SQL_MODE`:='ANSI_QUOTES');\n" + restore,
	} {
		if _, _, err := filterString(t, createTable+dump); !errors.Is(err, ErrUnsupportedStatement) {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
	for _, dump := range []string{
		save + goodDump + restore,
		"SET sql_mode=DEFAULT;\nSET @@SESSION.sql_mode='';\nSET sql_mode = 'STRICT_ALL_TABLES,NO_ZERO_DATE';\n",
		save + "SET @saved_cs_client = @@character_set_client;\n" + restore,
	} {
		if _, _, err := filterString(t, dump); err != nil {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
}

// The last word of a statement is only finished by the semicolon that ends it,
// so an engine named there has to be seen too.
func TestFilterRefusesAnEngineNamedAtTheEndOfAStatement(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"ALTER TABLE `t` ENGINE=MERGE",
		"ALTER TABLE `t` ENGINE = mrg_myisam",
		"CREATE TABLE `t` (`i` int) COMMENT='wrapper \"mysql\", srv \"s\", database \"other\"' ENGINE=SPIDER",
		"CREATE TABLE `t` (`i` int) ENGINE=FEDERATED",
		"CREATE TABLE `t` (`i` int) ENGINE=CONNECT",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrOtherDatabase) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
	if _, _, err := filterString(t, "CREATE TABLE `t` (`i` int) ENGINE=Aria;\n"); err != nil {
		t.Errorf("err = %v", err)
	}
}

// A dump that changes the default engine makes a CREATE TABLE with no ENGINE
// of its own use it, so only the engines a real dump needs are allowed.
func TestFilterRefusesADefaultEngineItCannotCheck(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"SET default_storage_engine=SPIDER",
		"SET SESSION default_storage_engine='CONNECT'",
		"SET @@default_storage_engine=MRG_MyISAM",
		"SET LOCAL default_tmp_storage_engine=FEDERATED",
		"SET SESSION enforce_storage_engine=FEDERATED",
		"SET storage_engine=MERGE",
		"SET default_storage_engine=@e",
		"SET default_storage_engine=DEFAULT",
		"SET @a=1, default_storage_engine=SPIDER",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrUnsupportedStatement) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
	for _, stmt := range []string{
		"SET default_storage_engine=InnoDB",
		"SET SESSION default_storage_engine='Aria'",
		"SET default_tmp_storage_engine=MyISAM",
	} {
		if _, _, err := filterString(t, stmt+";\n"); err != nil {
			t.Errorf("%q: err = %v", stmt, err)
		}
	}
}

// The client charset decides how the server reads the bytes of the dump: in a
// charset where 0x5c can end a character, such as gbk, the server ends a
// string where the filter does not. Only charsets that cannot do that are
// allowed.
func TestFilterFollowsTheClientCharset(t *testing.T) {
	t.Parallel()
	const saveCS = "/*!40101 SET @saved_cs_client = @@character_set_client */;\n"
	for _, stmt := range []string{
		"SET NAMES gbk",
		"SET names big5",
		"SET NAMES sjis COLLATE sjis_bin",
		"SET NAMES cp932",
		"SET NAMES gb18030",
		"SET NAMES euckr",
		"SET CHARACTER SET gbk",
		"SET CHARSET gbk",
		"SET character_set_client = gbk",
		"SET @@session.character_set_client = 'gbk'",
		"SET NAMES @c",
		"SET character_set_client = @never_saved",
		"SET NAMES CONCAT('g', 'bk')",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrUnsupportedStatement) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
	for _, dump := range []string{
		"SET NAMES utf8mb4;\n",
		"/*!40101 SET NAMES utf8mb4 */;\n",
		"SET NAMES utf8mb3;\nSET NAMES utf8;\nSET NAMES latin1;\nSET NAMES ascii;\nSET NAMES binary;\n",
		"SET NAMES 'utf8mb4' COLLATE 'utf8mb4_uca1400_ai_ci';\n",
		"SET CHARACTER SET utf8mb4;\n",
		"SET character_set_client = utf8mb4;\nSET character_set_results = gbk;\n",
		saveCS + "/*!40101 SET character_set_client = utf8mb4 */;\n" +
			"/*!40101 SET character_set_client = @saved_cs_client */;\n",
		saveCS + "/*!40101 SET CHARACTER_SET_CLIENT=@saved_cs_client */;\n",
	} {
		if _, _, err := filterString(t, dump); err != nil {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
	// A variable that held the saved charset but was then changed is refused.
	dump := saveCS + "SET @saved_cs_client='gbk';\n" + "SET character_set_client = @saved_cs_client;\n"
	if _, _, err := filterString(t, dump); !errors.Is(err, ErrUnsupportedStatement) {
		t.Errorf("a re-assigned variable: err = %v", err)
	}
}

// MariaDB takes NAMES and CHARACTER SET anywhere in a SET list, so every item
// of a SET is checked, and one the filter does not understand is refused.
func TestFilterChecksEveryItemOfASet(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"SET @a=1, NAMES gbk",
		"SET autocommit=1, NAMES gbk",
		"SET @a=1, CHARACTER SET gbk",
		"SET @a=1, CHARSET gbk",
		"SET NAMES utf8mb4, NAMES gbk",
		"SET @a=1, SESSION CHARACTER SET big5",
		"SET @a=1, TRANSACTION ISOLATION LEVEL READ COMMITTED",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrUnsupportedStatement) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
	for _, stmt := range []string{
		"SET @a=1, NAMES utf8mb4",
		"SET NAMES utf8mb4 COLLATE utf8mb4_uca1400_ai_ci, @b=2",
		"SET autocommit=0, unique_checks=0, foreign_key_checks=0",
	} {
		if _, _, err := filterString(t, stmt+";\n"); err != nil {
			t.Errorf("%q: err = %v", stmt, err)
		}
	}
}

// DEFAULT is the server's configured client charset, which the filter cannot
// see, so it is refused for the charset; mariadb-dump never writes it.
func TestFilterRefusesTheDefaultCharset(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"SET NAMES DEFAULT",
		"SET CHARACTER SET DEFAULT",
		"SET character_set_client = DEFAULT",
		"SET @@SESSION.character_set_client = 'default'",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if !errors.Is(err, ErrUnsupportedStatement) || got != createTable {
			t.Errorf("%q: err = %v, passed on %q", stmt, err, got)
		}
	}
}

// A statement is read as tokens, never as a text prefix, so whitespace in a
// database statement does not hide it, and a long one is not passed on
// unchecked when its head is cut short.
func TestFilterRefusesDatabaseStatementsWhateverTheirSpacing(t *testing.T) {
	t.Parallel()
	for _, stmt := range []string{
		"USE\tapp",
		"USE\napp",
		"DROP  DATABASE `app`",
		"DROP\tDATABASE `app`",
		"DROP\nSCHEMA `app`",
		"CREATE  DATABASE `app`",
		"ALTER  DATABASE `app` CHARACTER SET gbk",
		"ALTER\tDATABASE `app` CHARACTER SET gbk",
	} {
		got, f, err := filterString(t, createTable+stmt+";\n")
		if err == nil && f.database == "" {
			t.Errorf("%q: passed on %q", stmt, got)
		}
		if got != createTable {
			t.Errorf("%q: passed on %q", stmt, got)
		}
	}
	for _, stmt := range []string{
		"DROP" + strings.Repeat(" ", 5000) + "DATABASE `app`",
		"CREATE" + strings.Repeat(" ", 5000) + "USER `x`",
		"DROP" + strings.Repeat(" ", 5000) + "TABLE `t`",
		"SET" + strings.Repeat(" ", 5000) + "GLOBAL `x`=1",
	} {
		got, _, err := filterString(t, createTable+stmt+";\n")
		if err == nil || got != createTable {
			t.Errorf("%.40q: err = %v, passed on %q", stmt, err, got)
		}
	}
}

// The server skips an executable comment whose version is above its own, and
// reads what is inside without minding quotes, so the restore refuses a quote

// The server takes -- followed by a control byte as a comment where the client

// or a semicolon there: it could not tell where such a statement ends.
func TestFilterRefusesQuotesInASkippableComment(t *testing.T) {
	t.Parallel()
	for _, dump := range []string{
		"/*!999999 SELECT '*/ SELECT 1; -- ' */;\n",
		"/*M!999999 x `y` */;\n",
		"/*!100600 SET @a='x' */;\n",
		"/*!999999 ; */;\n",
	} {
		if _, _, err := filterString(t, createTable+dump); !errors.Is(err, ErrUnsupportedStatement) {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
	// A comment every supported server runs is read as the statement it holds.
	for _, dump := range []string{
		"/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;\n",
		"/*!40103 SET TIME_ZONE='+00:00' */;\n",
		"/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;\n",
		"/*M!999999\\- enable the sandbox mode */ \n/*!40101 SET NAMES utf8mb4 */;\n",
	} {
		if _, _, err := filterString(t, dump); err != nil {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
}

// does not, so the restore refuses it rather than read the dump differently.
func TestFilterRefusesADashCommentWithAControlByte(t *testing.T) {
	t.Parallel()
	for _, dump := range []string{
		"CREATE TABLE `u` (`a` int --\x01\n) ENGINE=InnoDB;\n",
		"CREATE TABLE `u` (`a` int --\x7f x\n) ENGINE=InnoDB;\n",
		"INSERT INTO `t` VALUES (1) --\x01 x\n;\n",
	} {
		if _, _, err := filterString(t, createTable+dump); !errors.Is(err, ErrUnsupportedStatement) {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
	for _, dump := range []string{
		"-- a comment\nINSERT INTO `t` VALUES (1);\n",
		"--\nINSERT INTO `t` VALUES (1);\n",
		"-- ------------------------------\n",
		"INSERT INTO `t` VALUES (1) -- trailing\n;\n",
		"INSERT INTO `t` VALUES ('a--b');\n",
	} {
		if _, _, err := filterString(t, dump); err != nil {
			t.Errorf("%q: err = %v", dump, err)
		}
	}
}

func TestFilterChecksTheValuesThatCrossTheHead(t *testing.T) {
	t.Parallel()
	const insert = "INSERT INTO `t` VALUES (1"
	for _, size := range []int{16, 64, 4095, 4096, 4097, 64 << 10} {
		for _, tail := range []string{",NOW()", ",(@a:=1)", ",SLEEP(1)"} {
			for _, at := range []int{headSize - 40, headSize - 3, headSize - 1, headSize, headSize + 1, headSize + 7,
				holdSize + 5} {
				pad := at - len(insert)
				body := insert + strings.Repeat(",1", pad/2) + strings.Repeat(" ", pad%2) + tail + ");\n"
				dump := "CREATE TABLE `t` (\n  `id` int\n);\n" + body + "-- Dump completed\n"
				var out strings.Builder
				f := &dumpFilter{}
				err := f.copy(&out, bufio.NewReaderSize(strings.NewReader(dump), size))
				if !errors.Is(err, ErrUnsupportedStatement) {
					t.Errorf("reader %d, %s at %d: err = %v, want it refused", size, tail, at, err)
				}
			}
		}
	}
	dump := "CREATE TABLE `t` (\n  `id` int\n);\n" +
		"INSERT INTO `t` VALUES (1" + strings.Repeat(",12345", 1000) + ",NULL,-1.5e3,0x1F,'x');\n"
	for _, size := range []int{16, 64, 4095, 4096, 4097, 64 << 10} {
		var out strings.Builder
		if err := (&dumpFilter{}).copy(&out, bufio.NewReaderSize(strings.NewReader(dump), size)); err != nil {
			t.Errorf("reader %d: a plain value list across the head was refused: %v", size, err)
		}
	}
}

func TestFilterRefusesTableFilesOutsideTheDataDirectory(t *testing.T) {
	t.Parallel()
	for _, option := range []string{"DATA DIRECTORY='/var/lib/x'", "INDEX DIRECTORY='/var/lib/x'"} {
		dump := "CREATE TABLE `t` (\n  `id` int\n) ENGINE=InnoDB " + option + ";\n"
		if _, _, err := filterString(t, dump); err == nil {
			t.Errorf("%s: a table placed outside the data directory passed", option)
		}
	}
	if _, _, err := filterString(t, "CREATE TABLE `t` (\n  `directory` int\n) ENGINE=InnoDB;\n"); err != nil {
		t.Errorf("a column named directory was refused: %v", err)
	}
}
