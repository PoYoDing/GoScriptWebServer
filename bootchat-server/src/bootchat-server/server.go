package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	_ "sync/atomic"
	_ "time"
)

type SqlObject struct {
	db *sql.DB
}

var verbose bool = false

func printLogo() {
	fmt.Println("\n")
	fmt.Println(`,-----.                  ,--.   ,-----.,--.               ,--.`)
	fmt.Println(`|  |) /_  ,---.  ,---. ,-'  '-.'  .--./|  ,---.  ,--,--.,-'  '-.`)
	fmt.Println(`|  .-.  \| .-. || .-. |'-.  .-'|  |    |  .-.  |' ,-.  |'-.  .-'`)
	fmt.Println(`|  '--' /' '-' '' '-' '  |  |  '  '--'\|  | |  |\ '-'  |  |  |`)
	fmt.Println(`'------'  '---'  '---'   '--'   '-----''--' '--' '--'--'  '--'`)
	fmt.Println("\n")
}

func main() {
	printLogo()

	if len(os.Args) > 1 {
		if os.Args[1] == "-v" {
			verbose = true
			fmt.Println("Verbose enabled.")
		}
	}

	dbo, err := open_database(verbose)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	//init_tables(dbo)
	

	sqlHttpHandler := &SqlObject{db: dbo}
	http.HandleFunc("/", sqlHttpHandler.handleConnection)

	log.Printf("Starting server on port %s...", "8443")
	//err = http.ListenAndServeTLS("127.0.0.1:8443", "./etc/server.crt", "./etc/server.key", nil)
	err = http.ListenAndServe("127.0.0.1:8443", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getErrorJson(exception string) string {
	jsonMap := make(map[string]string)

	jsonMap["success"] = "false"
	jsonMap["exception"] = exception

	jsonBytes, _ := json.Marshal(jsonMap)
	return string(jsonBytes)
}

func mapToJsonString(replyMap map[string]string) (string, error) {
	replyBytes, err := json.Marshal(replyMap)
	if err != nil {
		return "", err
	}
	return string(replyBytes), nil
}

func interfaceMapToJsonString(replyMap map[string]interface{}) (string, error) {
	replyBytes, err := json.Marshal(replyMap)
	if err != nil {
		return "", err
	}
	return string(replyBytes), nil
}

func (sqlobject *SqlObject) handleConnection(response http.ResponseWriter, request *http.Request) {
	//fmt.Println(request.URL.Path)

	response.Header().Set("Content-Type", "text/json")

	const errParseStr = "error - can not parse request body."
	const errUnserializeStr = "error - can not unserialize the request."

	//var requestStr string
	var requestBytes []byte
	var postData map[string]interface{}

	requestBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Println(errParseStr)
		fmt.Fprintf(response, getErrorJson(errParseStr))
		return
	}

	//requestStr = string(requestBytes)

	if err := json.Unmarshal(requestBytes, &postData); err != nil {
		log.Println(errUnserializeStr)
		fmt.Fprintf(response, getErrorJson(errUnserializeStr))
		return
	}

	if request, exists := postData["request"]; exists {

		if request == "login" {
			jsonString, _ := mapToJsonString(handleLoginRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)

			if u, exists := postData["username"]; exists {
				set_new_message_flag(sqlobject.db, u.(string), 1)
			}

			return
		}

		if request == "regusr" {
			jsonString, _ := mapToJsonString(handleCreateUserRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "send" {
			jsonString, _ := mapToJsonString(handleSendMessageRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "getmyrow" {
			jsonString, _ := mapToJsonString(handleGetUserRowRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "getinboxstatus" {
			jsonString, _ := mapToJsonString(handleGetInboxStatusRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "getallmsgs" {
			jsonString, _ := interfaceMapToJsonString(handleGetMessagesRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "setnewmsg" {
			jsonString, _ := mapToJsonString(handleSetNewMessageRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "register" {
			jsonString, _ := mapToJsonString(handleRegisterNewUserRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "forgotpass" {
			jsonString, _ := mapToJsonString(handleForgotPasswordRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		if request == "deleteconv" {
			jsonString, _ := mapToJsonString(handleDeleteConvoRequest(sqlobject.db, postData))
			fmt.Fprintf(response, jsonString)
			return
		}

		fmt.Fprintf(response, getErrorJson("unimplemented request"))
		return
	}

	fmt.Fprintf(response, getErrorJson("missing request"))
}

/* ADD REQUESET HANDLERS HERE */
/* ALL HANDLERS MUST RETURN A MAP CONTAINING: { 'success' : Boolean, 'exception' : String (if any) } */

func handleLoginRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	var username string = ""
	var password string = ""

	if u, exists := postData["username"]; exists {
		username, _ = u.(string)
	}

	if p, exists := postData["password"]; exists {
		password, _ = p.(string)
	}

	if !(len(username) > 0 && len(password) > 0) {
		replyMap["exception"] = "unable to get username and/or password from request"
		return replyMap
	}

	if success, _ := verify_user_login(db, username, password); success {
		userRow, err := get_user_row(db, username)
		if err == nil {
			replyMap["success"] = "true"
			replyMap["id"] = userRow["id"]
			replyMap["nickname"] = userRow["nickname"]
			replyMap["gender"] = userRow["gender"]
			replyMap["new_message"] = userRow["new_message"]

			return replyMap
		} else {
			replyMap["exception"] = err.Error()
			return replyMap
		}
	}

	replyMap["exception"] = "unable to login"
	return replyMap
}

func handleCreateUserRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	var username string
	var question string
	var answer string
	var password string

	if u, exists := postData["username"]; exists {
		username, _ = u.(string)
	} else {
		replyMap["exception"] = "missing parameter: username"
		return replyMap
	}

	if q, exists := postData["question"]; exists {
		question, _ = q.(string)
	} else {
		replyMap["exception"] = "missing parameter: security question"
		return replyMap
	}

	if a, exists := postData["answer"]; exists {
		answer, _ = a.(string)
	} else {
		replyMap["exception"] = "missing parameter: security answer"
		return replyMap
	}

	if p, exists := postData["password"]; exists {
		password, _ = p.(string)
	} else {
		replyMap["exception"] = "missing parameter: password"
		return replyMap
	}

	//db *sql.DB, username string, question string, answer string, password string
	err := add_user(db, username, question, answer, password)
	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	replyMap["success"] = "true"
	return replyMap
}

func handleSetNewMessageRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	verifyLoginCredentials := handleLoginRequest(db, postData)
	if verifyLoginCredentials["success"] != "true" {
		replyMap["exception"] = "invalid login"
		return replyMap
	}

	var username string = postData["username"].(string)
	var svalue string = postData["value"].(string)

	value := 1

	if svalue != "1" {
		value = 0
	}

	//value,err := strconv.Atoi(postData["value"].(string))
	//if err != nil{
	//	value = 1
	//}

	err := set_new_message_flag(db, username, value)

	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	if verbose {
		log.Printf("Updating new message flag for %s\n", username)
	}

	replyMap["success"] = "true"
	return replyMap
}

func handleDeleteConvoRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	verifyLoginCredentials := handleLoginRequest(db, postData)
	if verifyLoginCredentials["success"] != "true" {
		replyMap["exception"] = "invalid login"
		return replyMap
	}

	var username string = postData["username"].(string)

	var remove_user string

	if r, exists := postData["remove_user"]; exists {
		remove_user, _ = r.(string)
	} else {
		replyMap["exception"] = "missing parameter"
		return replyMap
	}

	err := delete_convo(db, remove_user, username)
	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	err = set_new_message_flag(db, username, 1)
	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	replyMap["success"] = "true"
	return replyMap
}

func handleGetInboxStatusRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	verifyLoginCredentials := handleLoginRequest(db, postData)
	if verifyLoginCredentials["success"] != "true" {
		replyMap["exception"] = "invalid login"
		return replyMap
	}

	var username string = postData["username"].(string)
	newMsg, err := get_new_message_flag(db, username)

	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	if verbose {
		log.Printf("Getting inbox status flag for %s\n", username)
	}

	replyMap["success"] = "true"
	replyMap["new"] = strconv.Itoa(newMsg)

	set_new_message_flag(db, username, 0)
	return replyMap
}

func handleForgotPasswordRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	var username string
	var question string
	var answer string
	var newpassword string
	var missing bool = false

	if u, exists := postData["username"]; exists {
		username, _ = u.(string)
	} else {
		missing = true
	}

	if q, exists := postData["security_question"]; exists {
		question, _ = q.(string)
	} else {
		missing = true
	}

	if a, exists := postData["security_answer"]; exists {
		answer, _ = a.(string)
	} else {
		missing = true
	}

	if p, exists := postData["newpassword"]; exists {
		newpassword, _ = p.(string)
	} else {
		missing = true
	}

	if missing {
		replyMap["exception"] = "missing parameter(s)"
		return replyMap
	}

	controlRow, err := get_control_user_row(db, username)

	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	if controlRow["security_answer"] != answer || controlRow["security_question"] != question {
		replyMap["exception"] = "invalid question/answer"
		return replyMap
	}

	err = set_password(db, username, newpassword)
	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	replyMap["success"] = "true"
	return replyMap
}

func handleRegisterNewUserRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	var username string
	var password string
	var nickname string
	var question string
	var answer string

	if u, exists := postData["username"]; exists {
		username, _ = u.(string)
		nickname = username
	}

	if p, exists := postData["password"]; exists {
		password = md5Sum(p.(string))
	}

	if n, exists := postData["nickname"]; exists {
		nickname, _ = n.(string)
	}

	if q, exists := postData["question"]; exists {
		question, _ = q.(string)
	}

	if a, exists := postData["answer"]; exists {
		answer, _ = a.(string)
	}

	//TODO check for valid strings or else error

	statement := "INSERT INTO accounts(username,nickname,password,security_question,security_answer) VALUES(?,?,?,?,?)"
	stmt, err := db.Prepare(statement)

	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	_, err = stmt.Exec(username, nickname, password, question, answer)
	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	stmt.Close()

	replyMap["success"] = "true"
	return replyMap
}

func handleGetUserRowRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	verifyLoginCredentials := handleLoginRequest(db, postData)
	if verifyLoginCredentials["success"] != "true" {
		replyMap["exception"] = "invalid credentials: " + verifyLoginCredentials["exception"]
		return replyMap
	}

	userRow, err := get_user_row(db, postData["username"].(string))
	if err == nil {
		replyMap["success"] = "true"
		replyMap["id"] = userRow["id"]
		replyMap["nickname"] = userRow["nickname"]
		replyMap["gender"] = userRow["gender"]
		replyMap["new_message"] = userRow["new_message"]
		return replyMap
	}

	replyMap["exception"] = err.Error()
	return replyMap
}

func handleGetMessagesRequest(db *sql.DB, postData map[string]interface{}) map[string]interface{} {
	replyMap := make(map[string]interface{})
	replyMap["success"] = false

	verifyLoginCredentials := handleLoginRequest(db, postData)
	if verifyLoginCredentials["success"] != "true" {
		replyMap["exception"] = "invalid login"
		return replyMap
	}

	var username string = postData["username"].(string)
	listOfRows, err := get_all_messages(db, username)

	if err != nil {
		replyMap["exception"] = err.Error()
		return replyMap
	}

	if verbose {
		log.Printf("Getting inbox contents for %s\n", username)
	}

	replyMap["success"] = "true"
	replyMap["messages"] = listOfRows
	return replyMap
}

func handleSendMessageRequest(db *sql.DB, postData map[string]interface{}) map[string]string {
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	verifyLoginCredentials := handleLoginRequest(db, postData)
	if verifyLoginCredentials["success"] != "true" {
		replyMap["exception"] = "invalid credentials: " + verifyLoginCredentials["exception"]
		return replyMap
	}

	var to_user string = ""
	var from_user string = postData["username"].(string)
	var message_body string = ""

	if t, exists := postData["to_user"]; exists {
		to_user, _ = t.(string)
	}

	if m, exists := postData["body"]; exists {
		message_body, _ = m.(string)
	}

	if !user_exists(db, to_user) {
		replyMap["exception"] = "receipient does not exist"
		return replyMap
	}

	if len(message_body) < 1 {
		replyMap["exception"] = "can not send empty message"
		return replyMap
	}

	err := send_message(db, to_user, from_user, message_body)
	if err == nil {
		replyMap["success"] = "true"
		return replyMap
	}

	replyMap["exception"] = err.Error()
	return replyMap
}
