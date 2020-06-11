package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	_ "github.com/lib/pq"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "root"
	password = "toor"
	dbname   = "postgres"
)

//Showcase all the user details in basic
func GetAllUserHandler(w http.ResponseWriter, r *http.Request){
	sqlStatement:="SELECT * FROM user_details"
	db := getDBConnection()
	defer db.Close()
	if db == nil {
		http.Error(w, "Can't establish DB Connection", http.StatusInternalServerError)
		return
	}
	rows, err := db.Query(sqlStatement)
	defer rows.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var users []User
	for rows.Next(){
		var user User
		var email, phone, meta, linkedInId sql.NullString
		if err := rows.Scan(&user.Id, &user.Name, &email, &phone, &meta, &user.Github.Id, &linkedInId); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if email.Valid{
			user.Email = email.String
		}
		if phone.Valid{
			user.PhoneNumber = phone.String
		}
		if linkedInId.Valid{
			user.LinkedIn.Id = linkedInId.String
		}
		if meta.Valid{
			if err := json.Unmarshal([]byte(meta.String),&user.MetaData); err != nil{
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		users = append(users, user)
	}
	if len(users) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	usersByte, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(usersByte)
	return
}

//showcase per user details
func GetUserDetailHandler(w http.ResponseWriter, r *http.Request){
	id := r.URL.Query().Get("id")
	sqlStatement:="SELECT * FROM user_details WHERE id=$1"
	db := getDBConnection()
	defer db.Close()
	if db == nil {
		http.Error(w, "Can't establish DB Connection", http.StatusInternalServerError)
		return
	}
	row := db.QueryRow(sqlStatement, id)
	var user User
	var email, phone, meta, linkedInId sql.NullString
	if err := row.Scan(&user.Id, &user.Name, &email, &phone, &meta, &user.Github.Id, &linkedInId); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if email.Valid{
		user.Email = email.String
	}
	if phone.Valid{
		user.PhoneNumber = phone.String
	}
	if linkedInId.Valid{
		user.LinkedIn.Id = linkedInId.String
	}
	if meta.Valid{
		if err := json.Unmarshal([]byte(meta.String),&user.MetaData); err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	userByte, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(userByte)
	return
}

//sets Mobile number
func SetMobileNumberHandler(w http.ResponseWriter, r *http.Request){
	if err := r.ParseForm(); err != nil{
		log.Println("Error in parsing form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := r.Form.Get("id")
	phone_number := r.Form.Get("phone_number")
	var phoneNumSqlStr sql.NullString
	if phone_number == ""{
		phoneNumSqlStr.Valid=false
	} else {
		phoneNumSqlStr.Valid = true
		phoneNumSqlStr.String = phone_number
	}
	db := getDBConnection()
	defer db.Close()
	sqlStatement := `UPDATE user_details SET phonenumber=$1 WHERE id=$2;`
	_, err := db.Exec(sqlStatement,phoneNumSqlStr,id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	return
}

//sets user password
func SetUserPasswordHandler(w http.ResponseWriter, r *http.Request){
	if err := r.ParseForm(); err != nil{
		log.Println("Error in parsing form")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	email := r.Form.Get("mail_id")
	password := r.Form.Get("password")
	db := getDBConnection()
	defer db.Close()
	sqlStatement := `SELECT id FROM user_details WHERE mailid=$1;`
	row := db.QueryRow(sqlStatement, email)
	var id int
	if err := row.Scan(&id); err != nil {
		//Cant set a password for the account without mailid
		if err == sql.ErrNoRows {
			http.Error(w, "EmailID Required", http.StatusBadRequest)
			return
		}
	}
	sqlStatement = `SELECT COUNT(*) FROM user_credentials WHERE user_id=$1`
	row = db.QueryRow(sqlStatement, id)
	var count int
	if err := row.Scan(&count); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		sqlStatement = `UPDATE user_credentials SET password=$1 WHERE user_id=$2;`
		_, err := db.Exec(sqlStatement, password, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		sqlStatement = `INSERT INTO user_credentials (user_id, password) values ($1,$2);`
		_, err := db.Exec(sqlStatement, id, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	fmt.Println("Password Saved")
	w.WriteHeader(http.StatusNoContent)
	return
}

//search user with phonenumber
func SearchUserHandler (w http.ResponseWriter, r *http.Request){
	q := r.URL.Query().Get("q")
	basedOn := r.URL.Query().Get("based_on")
	db := getDBConnection()
	defer db.Close()
	switch basedOn {
	case "phone_no":
		sqlStatement := `SELECT * FROM user_details WHERE phonenumber=$1`
		rows, err := db.Query(sqlStatement,q)
		defer rows.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var users []User
		for rows.Next(){
			var user User
			var email, phone, meta, linkedInId sql.NullString
			if err := rows.Scan(&user.Id, &user.Name, &email, &phone, &meta, &user.Github.Id, &linkedInId); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if email.Valid{
				user.Email = email.String
			}
			if linkedInId.Valid{
				user.LinkedIn.Id = linkedInId.String
			}
			if phone.Valid{
				user.PhoneNumber = phone.String
			}
			if meta.Valid{
				if err := json.Unmarshal([]byte(meta.String),&user.MetaData); err != nil{
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			users = append(users, user)
		}
		if len(users) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		usersByte, err := json.Marshal(users)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(usersByte)
		return
	default:
		w.WriteHeader(http.StatusNoContent)
		return
	}
	return
}

//User Authentication
func Authentication (w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch r.Form.Get("medium") {
	case "github":
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken:r.Header.Get("Authorization")})
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		user,_,err := client.Users.Get(ctx,r.Form.Get("username"))
		if err != nil {
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		db:=getDBConnection()
		defer db.Close()
		if db == nil {
			http.Error(w, "Couldn't establish db connection",http.StatusInternalServerError)
			return
		}
		if user.GetEmail() != "" {
			var email, meta sql.NullString
			isFound := true
			sqlStatement :=`SELECT mailid, meta FROM user_details WHERE mailid=$1;`
			row := db.QueryRow(sqlStatement, user.GetEmail())
			err := row.Scan(&email,&meta)
			if err != nil {
				if err == sql.ErrNoRows{
					fmt.Println("No Accounts found")
					isFound = false
				} else {
					panic(err)
					return
				}
			}
			var githubMetadata GithubMeta
			githubMetadata.NoOfFollowers = user.GetFollowers()
			githubMetadata.NoOfFollowing = user.GetFollowing()
			githubMetadata.NoOfPrivateRepos = user.GetTotalPrivateRepos()
			githubMetadata.NoOfPublicRepos = user.GetPublicRepos()
			if isFound {
				var metaData MetaData
				if err := json.Unmarshal([]byte(meta.String), &metaData); err != nil {
					http.Error(w, err.Error(),http.StatusInternalServerError)
					return
				}
				metaData.Github = githubMetadata
				metaDataByte,_ := json.Marshal(metaData)
				sqlStatement = `UPDATE user_details SET meta=$1, github_id=$2 WHERE mailid=$3;`
				_, err = db.Exec(sqlStatement, metaDataByte, user.GetID(), user.GetEmail())
				if err != nil {
					if err.Error()==`pq: duplicate key value violates unique constraint "user_details_github_id_uindex"`{
						delSQLStatement := `DELETE FROM user_details WHERE github_id=$1`;
						_, err = db.Exec(delSQLStatement,user.GetID())
						if err!=nil{
							http.Error(w, err.Error(),http.StatusInternalServerError)
							return
						}
						_, err = db.Exec(sqlStatement, metaDataByte, user.GetID(), user.GetEmail())
						if err!=nil{
							http.Error(w, err.Error(),http.StatusInternalServerError)
							return
						}
					} else {
						http.Error(w, err.Error(),http.StatusInternalServerError)
						return
					}
				}
				fmt.Println("Hurray Updated Meta With Email")
			} else {
				var metaData MetaData
				metaData.Github = githubMetadata
				metaDataByte,_ := json.Marshal(metaData)
				sqlStatement = `INSERT INTO user_details (name,mailid,meta, github_id) VALUES ($1, $2, $3, $4);`
				_, err = db.Exec(sqlStatement, user.GetName(), user.GetEmail(), metaDataByte, user.GetID())
				if err != nil {
					if err.Error()==`pq: duplicate key value violates unique constraint "user_details_github_id_uindex"`{
						delSQLStatement := `DELETE FROM user_details WHERE github_id=$1`;
						_, err = db.Exec(delSQLStatement,user.GetID())
						if err!=nil{
							http.Error(w, err.Error(),http.StatusInternalServerError)
							return
						}
						_, err = db.Exec(sqlStatement, user.GetName(), user.GetEmail(), metaDataByte, user.GetID())
						if err!=nil{
							http.Error(w, err.Error(),http.StatusInternalServerError)
							return
						}
					} else {
						http.Error(w, err.Error(),http.StatusInternalServerError)
						return
					}
				}
				fmt.Println("New user created with Email")
			}
		} else {
			sqlStatement:="SELECT COUNT(*) FROM user_details WHERE github_id=$1"
			var count int
			row := db.QueryRow(sqlStatement, user.GetID())
			err = row.Scan(&count)
			if err != nil {
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return
			}
			var metaData MetaData
			metaData.Github.NoOfFollowers = user.GetFollowers()
			metaData.Github.NoOfFollowing = user.GetFollowing()
			metaData.Github.NoOfPrivateRepos = user.GetTotalPrivateRepos()
			metaData.Github.NoOfPublicRepos = user.GetPublicRepos()
			metaDataByte, err := json.Marshal(metaData)
			if err != nil {
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return
			}
			if count >0 {
				var email, meta sql.NullString
				sqlStatement =`SELECT mailid, meta FROM user_details WHERE github_id=$1;`
				row = db.QueryRow(sqlStatement, user.GetID())
				err := row.Scan(&email, &meta)
				if err != nil {
					http.Error(w, err.Error(),http.StatusInternalServerError)
					return
				}
				if email.Valid == false && user.GetEmail() != "" {
					email.Valid = true
					email.String = user.GetEmail()
					sqlStatement = `UPDATE user_details SET mailid=$1, meta=$2 WHERE github_id=$3;`
					_, err = db.Exec(sqlStatement, email, metaDataByte, user.GetID())
					if err != nil {
						http.Error(w, err.Error(),http.StatusInternalServerError)
						return
					}
					fmt.Println("Hurray Updated wil metadata and email with GitId")
				} else {
					sqlStatement = `UPDATE user_details SET meta=$1 WHERE github_id=$2;`
					_, err = db.Exec(sqlStatement, metaDataByte, user.GetID())
					if err != nil {
						http.Error(w, err.Error(),http.StatusInternalServerError)
						return
					}
					fmt.Println("Hurray Updated metadata with gitid")
				}
			} else {
				sqlStatement = `INSERT INTO user_details (name,meta, github_id) VALUES ($1, $2, $3);`
				_, err = db.Exec(sqlStatement, user.GetName(), metaDataByte, user.GetID())
				if err != nil {
					http.Error(w, err.Error(),http.StatusInternalServerError)
					return
				}
				fmt.Println("Inserted with gitID")
			}
		}
		sqlStatement:=`SELECT id,mailid, name FROM user_details WHERE github_id=$1`
		row := db.QueryRow(sqlStatement, user.GetID())
		var id int
		var mailid, name sql.NullString
		if err = row.Scan(&id,&mailid, &name); err != nil {
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		var authResponse AuthResponse
		authResponse.Id = id
		authResponse.Name = name.String
		authResponse.EmailID = mailid.String
		authResponse.SetPassword = mailid.Valid
		authResByte, err := json.Marshal(authResponse)
		if err != nil{
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(authResByte)
	case "linkedin":
		var email string
		var metadataLinkedIn LinkedInMeta
		details,err:=hitLinkedApi("https://api.linkedin.com/v2/me", "GET", r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		result := gjson.GetMany(details,"id","localizedFirstName", "localizedLastName")
		metadataLinkedIn.Id = result[0].String()
		metadataLinkedIn.LocalizedFirstName = result[1].String()
		metadataLinkedIn.LocalizedLastName = result[2].String()
		emailDetails, err:=hitLinkedApi("https://api.linkedin.com/v2/emailAddress?q=members&projection=(elements*(handle~))", "GET", r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		emailResult := gjson.Get(emailDetails,"elements.0.handle~.emailAddress")
		email = emailResult.String()
		db := getDBConnection()
		defer db.Close()
		if db == nil {
			http.Error(w, "Couldn't establish DB Connection",http.StatusInternalServerError)
			return
		}
		var count, id int
		var metadataSQLString sql.NullString
		sqlStatement:="SELECT COUNT(*), meta,id FROM user_details WHERE mailid=$1 GROUP BY id;"
		row := db.QueryRow(sqlStatement, email)
		if err := row.Scan(&count, &metadataSQLString, &id); err != nil {
			if err == sql.ErrNoRows{
				count=0
			} else {
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return
			}
		}
		fmt.Println(email)
		var metadata MetaData
		if count > 0 {
			if err := json.Unmarshal([]byte(metadataSQLString.String), &metadata); err != nil {
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return
			}
			metadata.LinkedIn = metadataLinkedIn
			metaDataJson, _ := json.Marshal(metadata)
			sqlStatement:=`UPDATE user_details SET linkedin_id=$1, meta=$2 WHERE mailid=$3`
			_, err := db.Exec(sqlStatement,metadataLinkedIn.Id, metaDataJson, email)
			if err != nil {
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return
			}
			fmt.Println("UPDATEEEEEEEEDD")
		} else {
			metadata.LinkedIn = metadataLinkedIn
			metaDataJson, _ := json.Marshal(metadata)
			sqlStatement = `INSERT INTO user_details (name,mailid, meta, linkedin_id) VALUES ($1, $2, $3, $4);`
			_,err := db.Exec(sqlStatement, metadataLinkedIn.LocalizedFirstName, email, metaDataJson, metadataLinkedIn.Id)
			if err != nil{
				http.Error(w, err.Error(),http.StatusInternalServerError)
				return
			}
			fmt.Println("CREATEDDDDDDD")
		}
		var authResponse AuthResponse
		var isSetPassword bool
		sqlStatement=`SELECT id FROM user_details WHERE mailid=$1`
		row = db.QueryRow(sqlStatement, email)
		var userid int
		if err = row.Scan(&userid); err != nil {
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		authResponse.Id = userid
		authResponse.Name = metadataLinkedIn.LocalizedFirstName
		authResponse.EmailID = email
		if email != "" {
			isSetPassword = true
		}
		authResponse.SetPassword = isSetPassword
		authResByte, err := json.Marshal(authResponse)
		if err != nil{
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		w.Write(authResByte)
	case "basicauth":
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(),http.StatusInternalServerError)
			return
		}
		emailid := r.Form.Get("emailid")
		password := r.Form.Get("password")
		var name string
		sqlStatement := `SELECT id, name from user_details where mailid=$1`
		db:=getDBConnection()
		defer db.Close()
		if db == nil {
			http.Error(w, "Couldn't establish DB Connection",http.StatusInternalServerError)
			return
		}
		var id int
		row := db.QueryRow(sqlStatement, emailid)
		if err := row.Scan(&id, &name); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "User Not Found", http.StatusUnauthorized)
				return
			}
		}
		sqlStatement = `SELECT password from user_credentials where user_id=$1`
		var actualPassword string
		row = db.QueryRow(sqlStatement, &id)
		if err:= row.Scan(&actualPassword); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "User Not Found", http.StatusUnauthorized)
				return
			}
		}
		if actualPassword == password {
			var authResponse AuthResponse
			authResponse.Id= id
			authResponse.Name = name
			authResponse.EmailID = emailid
			authResponse.SetPassword = true
			authResByte, err := json.Marshal(authResponse)
			if err != nil {
				http.Error(w, "User Not Found", http.StatusUnauthorized)
				return
			}
			w.Write(authResByte)
		} else {
			http.Error(w, "Bad Credentials", http.StatusUnauthorized)
		}
		return
	default:
		fmt.Println("Invalid Authorization mode")
		http.Error(w, "Invalid Authorised mode",http.StatusBadRequest)
		return
	}
	return
}

func hitLinkedApi(url string, method string, access_token string) (string,error) {
	client := &http.Client{}
	req,_ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", access_token)
	response, err := client.Do(req)
	defer response.Body.Close()
	if err != nil {
		return "", nil
	}
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", nil
	}
	return string(responseData), nil
}

//Get DB Connection
func getDBConnection() *sql.DB{
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+ "password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return db
}



