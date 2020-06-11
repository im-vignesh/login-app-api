package main

type User struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	MetaData `json:"metadata"`
}

type MetaData struct {
	Github GithubMeta `json:"github,omitempty"`
	LinkedIn LinkedInMeta `json:"linkedin,omitempty"`
}

type GithubMeta struct {
	Id int `json:"id"`
	NoOfFollowers int `json:"no_of_followers"`
	NoOfFollowing int `json:"no_of_following"`
	NoOfPublicRepos int `json:"no_of_public_repos"`
	NoOfPrivateRepos int `json:"no_of_private_repos"`
}

type LinkedInMeta struct {
	Id string `json:"id"`
	LocalizedFirstName string `json:"localized_first_name"`
	LocalizedLastName string `json:"localized_last_name"`
}

type AuthResponse struct {
	Id int `json:"id"`
	Name string `json:"name"`
	EmailID string `json:"email_id"`
	SetPassword bool `json:"set_password"`
}