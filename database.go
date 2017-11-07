package main

import (
	"fmt"
	"database/sql"
	"time"
	"log"
	"crypto/md5"
	"encoding/hex"
	"strconv"
	_"github.com/mattn/go-sqlite3"
	_"errors"
)

func md5Sum(str string)(string){
	data := []byte(str)
	csum := md5.Sum(data)
	return hex.EncodeToString(csum[:])
}

func open_database(verboseEnable bool) (*sql.DB, error){
	db, err := sql.Open("sqlite3", "./etc/bootchat.db")
	if err != nil{
		return nil,err
	}
	return db,nil
}

/*
	add_user - Add a new user to the accounts TABLE
	 db sql.DB
	 username string
	 nickname string
	 gender string (max length = 1)
	 question string
	 answer string
	 password string

	 returns (error)
*/
func add_user(db *sql.DB, username string, nickname string, gender string, question string, answer string, password string) (error){
	
	if verbose{
		log.Printf("Creating new user: %s\n", username)
	}
	
	password = md5Sum(password)
	
	statement := "INSERT INTO accounts(username,nickname,gender,security_question,security_answer,password) VALUES(?,?,?,?,?,?)"

	stmt, err := db.Prepare(statement)
	if err != nil{
	 return err
	}

	_, err = stmt.Exec(username, nickname, gender, question, answer, password)
	stmt.Close()

	return err
}


/*
	user_exists - check if a user exists in accounts
	 db *sql.DB
	 username string

	 returns (bool)
*/
func user_exists(db *sql.DB, username string) (bool){
	statement := "SELECT EXISTS(SELECT id FROM accounts WHERE username = ? LIMIT 1)"

	stmt, err := db.Prepare(statement)
	if err != nil {
	 return false
	}

	row, err := stmt.Query(username)
	if err != nil{
	 return false
	}

	var result bool

	row.Next()
	row.Scan(&result)
	row.Close()
	stmt.Close()

	return result
}

/*
	veriy_user_login - checks if a username and password combo is valid
	 db *sql.DB
	 username string
	 password string
	 returns (bool)
*/
func verify_user_login(db *sql.DB, username string, password string) (bool, error){
	
	if verbose{
		log.Printf("Attempting to login user: %s  ", username)
	}
	
	statement := "SELECT id,password FROM accounts WHERE username = ?"
	
	stmt, err := db.Prepare(statement)
	if err != nil{
		if verbose{
			log.Println("Failed (breakpoint 1)")
		}
		stmt.Close()
		return false, err
	}
	
	row, err := stmt.Query(username)
	if err != nil{
		if verbose{
			log.Println("Failed (breakpoint 2)")
		}
		row.Close()
		stmt.Close()
	 return false, err
	}
	
	row.Next()
	
	var id int
	var password_ string
	row.Scan(&id, &password_)
	row.Close()
	stmt.Close()
	
	if password_ == md5Sum(password){
		if verbose{
			log.Println("Success")
		}
	 return true, nil
	}
	
	if verbose{
		log.Println("Failed (breakpoint 3)")
	}
	return false, nil
}

func get_user_row(db *sql.DB, username string) (map[string]string, error){
	statement := "SELECT id,nickname,gender,new_message FROM accounts WHERE username = ?"
	
	stmt,err := db.Prepare(statement)
	
	if err != nil{
		return nil,err
	}
	
	row,err := stmt.Query(username)

	if err != nil{
		return nil,err
	}

	var id int
	var nickname string
	var gender string
	var new_message int

	row.Next()
	
	row.Scan(&id, &nickname, &gender, &new_message)
	row.Close()
	stmt.Close()
	
	userRow := make(map[string]string)
	userRow["id"] = strconv.Itoa(id)
	userRow["nickname"] = nickname
	userRow["gender"] = gender
	userRow["new_message"] = strconv.Itoa(new_message)
	
	if verbose{
		log.Printf("Got user row: %s, %s", nickname, gender);
	}
	
	return userRow, nil
}

/*
	delete_user - deletes a user from the account TABLE
	 db *sql.DB
	 username string

	 returns (error)
*/
func delete_user(db *sql.DB, username string) (error){
	statement := "DELETE FROM accounts WHERE username = ?"

	stmt,err := db.Prepare(statement)
	if err != nil{
	 return err
	}

	_,err = stmt.Exec(username)
	stmt.Close()
	return err
}

/*
	print_accounts - dumps the account table to stdout
	 db *sql.DB
	 returns (error)
*/
func print_accounts(db *sql.DB) (error){
	statement := "SELECT id,username,nickname,password FROM accounts"

	stmt,err := db.Prepare(statement)
	if err != nil{
	 return err
	}

	rows, _ := stmt.Query()

	var id int
	var account string
	var nickname string
	var password string

	fmt.Println("-------------------------------------\n")
	for rows.Next() {
	 rows.Scan(&id, &account, &nickname, &password)
	 fmt.Printf("User ID: %d\n\tUsername: %s\n\tNickname: %s\n\tPassword: %s\n", id, account, nickname, password)
	}
	fmt.Println("-------------------------------------\n")
	
	rows.Close()
	stmt.Close()
	return nil
}

func set_new_message_flag(db *sql.DB, username string, value int)(error){
	statement := "UPDATE accounts SET new_message = ? WHERE username = ?"
	
	stmt,err := db.Prepare(statement)
	if err != nil{
		return err
	}
	
	_, err = stmt.Exec(value, username)
	stmt.Close()
	
	return err
}

func get_new_message_flag(db *sql.DB, username string)(int,error){
	statement := "SELECT new_message FROM accounts WHERE username = ?"
	
	stmt,err := db.Prepare(statement)
	if err != nil{
		return 0,err
	}
	
	row, err := stmt.Query(username)
	if err != nil{
		stmt.Close()
		return 0,err
	}
	
	var new_message int
	row.Next()
	row.Scan(&new_message)
	row.Close()
	stmt.Close()
	
	return new_message, nil
}

func send_message(db *sql.DB, to_user string, from_user string, body string)(error){
	statement := "INSERT INTO messages(to_user,from_user,body,time) VALUES(?,?,?,?)"
	
	if verbose{
		log.Printf("Sending message to %s from %s...", to_user, from_user)
	}
	
	stmt,err := db.Prepare(statement)
	if err != nil{
		if verbose{
			log.Printf("Failed: %s", err.Error())
		}
		return err
	}
	
	_,err = stmt.Exec(to_user, from_user, body, time.Now().String())
	stmt.Close()
	
	if err == nil{
		err = set_new_message_flag(db, to_user, 1)
		if verbose{
			log.Println("Success")
		}
	}
	
	return err
}

/*
func init_tables(db *sql.DB) (error){

	db.Exec("DROP TABLE IF EXISTS accounts")
	statement := "CREATE TABLE IF NOT EXISTS accounts (\n"
	statement += "id INTEGER PRIMARY KEY AUTOINCREMENT,\n"
	statement += "username VARCHAR(32) UNIQUE,\n"
	statement += "nickname VARCHAR(32) UNIQUE,\n"
	statement += "gender CHARACTER(1),\n"
	statement += "picture TEXT,\n"
	statement += "security_question VARCHAR(256),\n"
	statement += "security_answer VARCHAR(256),\n"
	statement += "password VARCHAR(128)\n"
	statement += ");"

	db.Exec(statement)

	db.Exec("DROP TABLE IF EXISTS messages")
	statement = "CREATE TABLE IF NOT EXISTS messages (\n"
	statement += "id INTEGER PRIMARY KEY AUTOINCREMENT,\n"
	statement += "to_user VARCHAR(32),\n"
	statement += "from_user VARCHAR(32),\n"
	statement += "body VARCHAR(10000),\n"
	statement += "time VARCHAR(32)\n"
	statement += ");"

	db.Exec(statement)

	return nil
}
*/
