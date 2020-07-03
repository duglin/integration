package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

var GitToken = ""
var GitHubURL = ""

func Git(method string, url string, body string) (string, error) {
	buf := []byte{}
	if body != "" {
		buf = []byte(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte("dug:" + GitToken))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	buf, _ = ioutil.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		// fmt.Printf("Git Error:\n--> %s %s\n--> %s\n", method, url, body)
		// fmt.Printf("%d %s\n", res.StatusCode, string(buf))
		return "", fmt.Errorf("Error gitting: %d %s\n", res.StatusCode, string(buf))
	}
	return string(buf), nil
}

func Body(str string) string {
	body := struct {
		Body string `json:"body"`
	}{
		Body: str,
	}
	buf, _ := json.Marshal(body)
	return string(buf)
}

func (issue *Issue) AddLabel(label string) error {
	_, err := Git("POST", issue.URL+"/labels", `{"labels": [ "`+label+`"]}`)
	return err
}

func (issue *Issue) RemoveLabel(label string) error {
	_, err := Git("DELETE", issue.URL+"/labels/"+label, "")
	return err
}

func (issue *Issue) AddComment(comment string) error {
	_, err := Git("POST", issue.URL+"/comments", Body(comment))
	return err
}

func (issue *Issue) Close() error {
	_, err := Git("PATCH", issue.URL, `{"state":"closed"}`)
	return err
}

func (issue *Issue) Reopen() error {
	_, err := Git("PATCH", issue.URL, `{"state":"open"}`)
	return err
}

func (issue *Issue) IsAssignee(user string) bool {
	for _, assignee := range issue.Assignees {
		if strings.EqualFold(user, assignee.Login) {
			return true
		}
	}
	return false
}

func (issue *Issue) AddAssignee(user string) error {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}
	_, err := Git("POST", issue.URL+"/assignees", `{"assignees":["`+user+`"]}`)
	return err
}

func (issue *Issue) RemoveAssignee(user string) error {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}
	_, err := Git("DELETE", issue.URL+"/assignees", `{"assignees":["`+user+`"]}`)
	return err
}

func (org *Organization) IsMember(user string) (bool, error) {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}
	_, err := Git("GET", org.URL+"/public_members/"+user, "")
	if err != nil {
		if strings.Index(err.Error(), " 404 ") > 0 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Static methods

func GetRepository(org string, name string) (*Repository, error) {
	res, err := Git("GET", GitHubURL+"/repos/"+org+"/"+name, "")
	if err != nil {
		return nil, err
	}

	repo := Repository{}
	if err = json.Unmarshal([]byte(res), &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// /repos/:owner/:repo/issues/:issue_number
func GetIssue(org string, repo string, num int) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", GitHubURL, org, repo, num)
	res, err := Git("GET", url, "")
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err = json.Unmarshal([]byte(res), &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func GetRepositoryTeams(org string, repo string) ([]Team, error) {
	res, err := Git("GET", GitHubURL+"/repos/"+org+"/"+repo+"/teams", "")
	if err != nil {
		return nil, err
	}

	var teams []Team
	if err = json.Unmarshal([]byte(res), &teams); err != nil {
		return nil, err
	}

	return teams, nil
}

func IsUserInOrganization(org string, user string) (bool, error) {
	_, err := Git("GET", GitHubURL+"/orgs/"+org+"/public_members/"+user, "")
	if err != nil {
		return false, err
	}
	return true, nil
}
