package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

var GitHubToken = ""
var GitHubURL = ""
var GitHubSecret = "" // used to verify events are from github

type GitResponse struct {
	StatusCode int
	Links      map[string]string
	Body       string
}

func Git(method string, url string, body string) (*GitResponse, error) {
	gitResponse := GitResponse{
		Links: map[string]string{},
	}

	buf := []byte{}
	if body != "" {
		buf = []byte(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(buf))
	if err != nil {
		fmt.Printf("Git: %s %s\n", method, url)
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte("user:" + GitHubToken))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	if strings.Contains(url, "projects") || strings.Contains(url, "cards") ||
		strings.Contains(url, "columns") {
		req.Header.Add("Accept", "application/vnd.github.inertia-preview+json")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	res, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf, _ = ioutil.ReadAll(res.Body)

	gitResponse.StatusCode = res.StatusCode
	gitResponse.Body = string(buf)

	if res.StatusCode/100 != 2 {
		// fmt.Printf("Git Error:\n--> %s %s\n--> %s\n", method, url, body)
		// fmt.Printf("%d %s\n", res.StatusCode, string(buf))
		return &gitResponse,
			fmt.Errorf("Github: Error %s: %d %s\nReq Body: %s\n", url,
				res.StatusCode, string(buf), body)
	}

	// Link: <https://.../issues?page=2>; rel="next",
	//   <https://issues?page=2>; rel="last"
	if links := res.Header["Link"]; len(links) > 0 {
		links = strings.Split(links[0], ",")
		for _, link := range links {
			parts := strings.Split(link, ";")
			for i, part := range parts {
				parts[i] = strings.TrimSpace(part)
			}
			if len(parts) == 2 && strings.HasPrefix(parts[1], `rel="`) {
				key := parts[1][5 : len(parts[1])-1]
				val := parts[0][1 : len(parts[0])-1] // trim <>
				gitResponse.Links[key] = val
			}
		}
	}

	return &gitResponse, nil
}

// daItem is an empty slice of the resource type to return (e.g. []*Issue{})
func GetAll(url string, daItem interface{}) (interface{}, error) {
	daType := reflect.TypeOf(daItem)
	result := reflect.MakeSlice(daType, 0, 0)

	for url != "" {
		var res *GitResponse
		var err error
		if res, err = Git("GET", url, ""); err != nil {
			return nil, err
		}

		// Create a pointer Value to a slice, JSON Unmarshal needs a ptr
		itemsPtr := reflect.New(daType)

		// Create an empty slice Value and make our pointer reference it
		itemsPtr.Elem().Set(reflect.MakeSlice(daType, 0, 0))

		err = json.Unmarshal([]byte(res.Body), itemsPtr.Interface())
		if err != nil {
			return nil, err
		}

		// Re-get the pointer Value of the slice since it may have moved,
		// then append it to the result set
		result = reflect.AppendSlice(result, itemsPtr.Elem())

		url = res.Links["next"]
	}

	return result.Interface(), nil
}

func VerifyEvent(req *http.Request, body []byte) bool {
	sig := req.Header.Get("X-HUB-SIGNATURE")

	if len(sig) != 45 || !strings.HasPrefix(sig, "sha1=") {
		return false
	}

	calc := make([]byte, 20)
	hex.Decode(calc, []byte(sig[5:]))

	mac := hmac.New(sha1.New, []byte(GitHubSecret))
	mac.Write(body)

	return hmac.Equal(calc, mac.Sum(nil))
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

func (issue *Issue) HasLabel(label string) bool {
	for _, l := range issue.Labels {
		if strings.EqualFold(l.Name, label) {
			return true
		}
	}
	return false
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

func (issue *Issue) SetBody(body string) error {
	data := Body(body)
	res, err := Git("PATCH", issue.URL, string(data))
	if err != nil {
		return err
	}

	newIssue := Issue{}
	if err = json.Unmarshal([]byte(res.Body), &newIssue); err != nil {
		return err
	}
	*issue = newIssue

	return nil
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

func (issue *Issue) SetMilestone(newMile string) error {
	if newMile == "" {
		_, err := Git("PATCH", issue.URL, `{"milestone": null}`)
		return err
	}

	items, err := GetAll(issue.Repository_URL+"/milestones", []*Milestone{})
	if err != nil {
		return err
	}
	milestones := items.([]*Milestone)

	mileNum := -1
	for _, mile := range milestones {
		if mile.Title == newMile {
			mileNum = mile.Number
		}
	}
	if mileNum == -1 {
		err = fmt.Errorf("Can't find milestone %q\n", newMile)
		return err
	}

	_, err = Git("PATCH", issue.URL, fmt.Sprintf(`{"milestone": %d}`, mileNum))

	return err
}

func (org *Organization) IsMember(user string) (bool, error) {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}
	res, err := Git("GET", org.URL+"/public_members/"+user, "")
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (repo *Repository) GetLabels() ([]*Label, error) {
	items, err := GetAll(repo.URL+"/labels", []*Label{})
	if err != nil {
		return nil, err
	}
	return items.([]*Label), nil
}

func (repo *Repository) GetIssues(query string) ([]*Issue, error) {
	url := repo.URL + "/issues"
	if query != "" {
		url += "?" + query
	}

	items, err := GetAll(url, []*Issue{})
	if err != nil {
		return nil, err
	}
	return items.([]*Issue), nil
}

func (repo *Repository) GetMilestones(query string) ([]*Milestone, error) {
	url := repo.URL + "/milestones"
	if query != "" {
		url += "?" + query
	}

	items, err := GetAll(url, []*Milestone{})
	if err != nil {
		return nil, err
	}
	return items.([]*Milestone), nil
}

// Static methods

func GetRepository(org string, name string) (*Repository, error) {
	res, err := Git("GET", GitHubURL+"/repos/"+org+"/"+name, "")
	if err != nil {
		return nil, err
	}

	repo := Repository{}
	if err = json.Unmarshal([]byte(res.Body), &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

func SetIssueMilestone(org string, repo string, num int, newMile string) (*Issue, error) {
	items, err := GetAll(GitHubURL+"/repos/"+org+"/"+repo+"/milestones",
		[]*Milestone{})
	if err != nil {
		return nil, err
	}
	milestones := items.([]*Milestone)

	mileNum := -1
	for _, mile := range milestones {
		if mile.Title == newMile {
			mileNum = mile.Number
		}
	}
	if mileNum == -1 {
		err = fmt.Errorf("Can't find milestone %q\n", newMile)
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", GitHubURL, org, repo, num)
	res, err := Git("PATCH", url, fmt.Sprintf(`{"milestone": %d}`, mileNum))
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err = json.Unmarshal([]byte(res.Body), &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func GetRepositoryMilestones(org string, repo string) ([]*Milestone, error) {
	items, err := GetAll(GitHubURL+"/repos/"+org+"/"+repo+"/milestones",
		[]*Milestone{})
	if err != nil {
		return nil, err
	}
	return items.([]*Milestone), nil
}

// /repos/:owner/:repo/issues/:issue_number
func GetIssue(url string) (*Issue, error) {
	res, err := Git("GET", url, "")
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err = json.Unmarshal([]byte(res.Body), &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func GetIssueParts(org string, repo string, num int) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", GitHubURL, org, repo, num)
	return GetIssue(url)
}

func GetIssuesParts(org string, repo string, query string) ([]*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues", GitHubURL, org, repo)
	if query != "" {
		url += "?" + query
	}
	items, err := GetAll(url, []*Issue{})
	if err != nil {
		return nil, err
	}
	return items.([]*Issue), nil
}

func GetMilestones(org string, repo string, query string) ([]*Milestone, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/milestones", GitHubURL, org, repo)
	if query != "" {
		url += "?" + query
	}
	items, err := GetAll(url, []*Milestone{})
	if err != nil {
		return nil, err
	}
	return items.([]*Milestone), nil
}

func GetRepositoryTeams(org string, repo string) ([]*Team, error) {
	items, err := GetAll(GitHubURL+"/repos/"+org+"/"+repo+"/teams", []*Label{})
	if err != nil {
		return nil, err
	}
	return items.([]*Team), nil
}

func IsUserInOrganization(org string, user string) (bool, error) {
	_, err := Git("GET", GitHubURL+"/orgs/"+org+"/public_members/"+user, "")
	if err != nil {
		return false, err
	}
	return true, nil
}

// Github Data Manipulation

type GitDataEntry struct {
	Label   string
	Entries []string
}

type GitData struct {
	Body []string
	Data []*GitDataEntry
}

func (gd *GitData) AddData(label string, text string) {
	var gde *GitDataEntry

	for _, gde = range gd.Data {
		if gde.Label == label {
			break
		}
		gde = nil
	}

	if gde == nil {
		gde = &GitDataEntry{
			Label: label,
		}
		gd.Data = append(gd.Data, gde)
	}

	for _, val := range gde.Entries {
		if val == text {
			return
		}
	}

	gde.Entries = append(gde.Entries, text)
}

func (gd *GitData) DeleteData(label string, text string) bool {
	res := false
	for j, gde := range gd.Data {
		if gde.Label == label {
			for i, val := range gde.Entries {
				if val == text {
					gde.Entries = append(gde.Entries[:i], gde.Entries[i+1:]...)
					res = true
				}
			}
			if len(gde.Entries) == 0 || text == "" {
				gde.Entries = nil
				gd.Data = append(gd.Data[:j], gd.Data[j+1:]...)
				res = true
			}
			return res
		}
	}
	return false
}

func (gd *GitData) HasData(label string, text string) bool {
	var gde *GitDataEntry

	for _, gde = range gd.Data {
		if gde.Label == label {
			break
		}
		gde = nil
	}

	if gde == nil {
		return false
	}

	for _, val := range gde.Entries {
		if val == text {
			return true
		}
	}

	return false
}

func (gd *GitData) SetData(label string, text string) {
	var gde *GitDataEntry

	for _, gde = range gd.Data {
		if gde.Label == label {
			break
		}
		gde = nil
	}

	if gde == nil {
		gde = &GitDataEntry{
			Label: label,
		}
		gd.Data = append(gd.Data, gde)
	}

	gde.Entries = []string{text}
}

func (issue *Issue) GetData() *GitData {
	data := &GitData{}

	lines := strings.Split(issue.Body, "\n")
	for _, line := range lines {
		// line = strings.TrimSpace(line)

		// **_Title_**: text
		i := strings.Index(line, "_**: ")
		if i < 2 || !strings.HasPrefix(line, "**_") {
			data.Body = append(data.Body, line)
			continue
		}

		label, text := "", ""

		label = strings.TrimSpace(line[3:i])
		text = strings.TrimSpace(line[i+5:])

		data.AddData(label, text)
	}

	// Remove trailing "---" || "" in Body
	for len(data.Body) > 0 {
		line := strings.TrimSpace(data.Body[len(data.Body)-1])

		if line == "---" || line == "" {
			data.Body = data.Body[:len(data.Body)-1]
			continue
		}
		break
	}

	return data
}

func (issue *Issue) AddData(label string, text string) error {
	data := issue.GetData()
	data.AddData(label, text)
	return issue.SetGitData(data)
}

func (issue *Issue) DeleteData(label string, text string) error {
	data := issue.GetData()
	if data.DeleteData(label, text) {
		return issue.SetGitData(data)
	}
	return nil
}

func (issue *Issue) HasData(label string, text string) bool {
	data := issue.GetData()
	return data.HasData(label, text)
}

func (issue *Issue) SetData(label string, text string) error {
	data := issue.GetData()
	data.SetData(label, text)
	return issue.SetGitData(data)
}

func (issue *Issue) SetGitData(data *GitData) error {
	body := ""

	// Remove trailing "---" || "" in Body
	for len(data.Body) > 0 {
		line := strings.TrimSpace(data.Body[len(data.Body)-1])

		if line == "---" || line == "" {
			data.Body = data.Body[:len(data.Body)-1]
			continue
		}
		break
	}

	for _, line := range data.Body {
		body += line + "\n"
	}

	if len(data.Data) > 0 {
		body += "\n---\n"

		for _, gde := range data.Data {
			for _, entry := range gde.Entries {
				body += fmt.Sprintf("**_%s_**: %s\n", gde.Label, entry)
			}
		}
	}

	return issue.SetBody(body)
}
