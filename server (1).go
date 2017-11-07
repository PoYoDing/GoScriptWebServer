package main

import (
	 "fmt"
	 "io/ioutil"
	 "net/http"
	 "log"
	 "encoding/json"
	 "os"
	 "database/sql"
	 _"github.com/mattn/go-sqlite3"
	 _"sync/atomic"
	 _"time"
)

type SqlObject struct{
	db *sql.DB
}

var verbose bool = false

func printLogo(){
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
	
	if len(os.Args) > 1{
		if os.Args[1] == "-v"{
			verbose = true
			fmt.Println("Verbose enabled.")
		}
	}
	
	dbo, err := open_database(verbose)
	if err != nil{
	 fmt.Println(err.Error())
	 os.Exit(1)
	}

	//print_accounts(dbo)

	sqlHttpHandler := &SqlObject{db : dbo}
	http.HandleFunc("/", sqlHttpHandler.handleConnection)
	
	log.Printf("Starting server on port %s...", os.Getenv(`PORT`))
	//err = http.ListenAndServeTLS(os.Getenv(`IP`) + ":" + os.Getenv(`PORT`), "./etc/server.crt", "./etc/server.key", nil)
	err = http.ListenAndServe(os.Getenv(`IP`) + ":" + os.Getenv(`PORT`), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getErrorJson(exception string)(string){
	jsonMap := make(map[string]string)

	jsonMap["success"] = "false"
	jsonMap["exception"] = exception

	jsonBytes,_ := json.Marshal(jsonMap)
	return string(jsonBytes)
}

func mapToJsonString(replyMap map[string]string) (string, error){
	replyBytes,err := json.Marshal(replyMap)
	if err != nil{
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

	if err := json.Unmarshal(requestBytes, &postData); err != nil{
	 log.Println(errUnserializeStr)
	 fmt.Fprintf(response, getErrorJson(errUnserializeStr))
	 return
	}

	if request, exists := postData["request"]; exists {

	 if request == "login"{
		jsonString,_  := mapToJsonString(handleLoginRequest(sqlobject.db, postData))
		fmt.Fprintf(response, jsonString)
		return
	 }
	 
	 if request == "send"{
	 	jsonString,_ := mapToJsonString(handleSendMessageRequest(sqlobject.db, postData))
	 	fmt.Fprintf(response, jsonString)
	 	return
	 }
	 
	 if request == "getmyrow"{
	 	jsonString,_ := mapToJsonString(handleGetUserRowRequest(sqlobject.db, postData))
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

func handleLoginRequest(db *sql.DB, postData map[string]interface{}) (map[string]string){
	replyMap := make(map[string]string)
	replyMap["success"] = "false"

	var username string = ""
	var password string = ""

	if u,exists := postData["username"]; exists{
	 username,_ = u.(string)
	}

	if p,exists := postData["password"]; exists{
	 password,_ = p.(string)
	}

	if !(len(username) > 0 && len(password) > 0){
	 replyMap["exception"] = "unable to get username and/or password from request"
	 return replyMap
	}

	if success,_ := verify_user_login(db, username, password); success {
	 userRow,err := get_user_row(db, username)
	 if err == nil{
	 	replyMap["success"] = "true"
	 	replyMap["id"] = userRow["id"]
	 	replyMap["nickname"] = userRow["nickname"]
	 	replyMap["gender"] = userRow["gender"]
	 	replyMap["new_message"] = userRow["new_message"]
	 	return replyMap
	 }else{
	 	replyMap["exception"] = err.Error()
	 	return replyMap
	 }
	}

	replyMap["exception"] = "unable to login"
	return replyMap
}

func handleGetUserRowRequest(db *sql.DB, postData map[string]interface{}) (map[string]string){
	replyMap := make(map[string]string)
	replyMap["success"] = "false"
	
	verifyLoginCredentials := handleLoginRequest(db, postData);
	if verifyLoginCredentials["success"] != "true"{
		replyMap["exception"] = "invalid credentials: " + verifyLoginCredentials["exception"]
		return replyMap
	}
	
	userRow, err := get_user_row(db, postData["username"].(string))
	if err == nil{
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

func handleSendMessageRequest(db *sql.DB, postData map[string]interface{}) (map[string]string){
	replyMap := make(map[string]string)
	replyMap["success"] = "false"
	
	verifyLoginCredentials := handleLoginRequest(db, postData);
	if verifyLoginCredentials["success"] != "true"{
		replyMap["exception"] = "invalid credentials: " + verifyLoginCredentials["exception"]
		return replyMap
	}
	
	var to_user string = ""
	var from_user string = postData["username"].(string)
	var message_body string = ""
	
	if t,exists := postData["to_user"]; exists{
		to_user,_ = t.(string)
	}
	
	if m,exists:= postData["body"]; exists{
		message_body,_ = m.(string)
	}
	
	
	if !user_exists(db, to_user){
		replyMap["exception"] = "receipient does not exist"
		return replyMap
	}
	
	if len(message_body) < 1{
		replyMap["exception"] = "can not send empty message"
		return replyMap
	}

	err := send_message(db, to_user, from_user, message_body)
	if err == nil{
		replyMap["success"] = "true"
		return replyMap
	}
	
	replyMap["exception"] = err.Error()
	return replyMap
}