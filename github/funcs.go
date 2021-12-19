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
	"sort"
	"strings"
)

func (u *User) SetGH(gh *GitHubClient) {
	if u != nil {
		u.GitHubClient = gh
	}
}

func (l *Label) SetGH(gh *GitHubClient) {
	if l != nil {
		l.GitHubClient = gh
	}
}

func (m *Milestone) SetGH(gh *GitHubClient) {
	if m != nil {
		m.GitHubClient = gh
		m.Creator.SetGH(gh)
	}
}

func (i *Issue) SetGH(gh *GitHubClient) {
	if i != nil {
		i.GitHubClient = gh
		i.User.SetGH(gh)
		for _, l := range i.Labels {
			l.SetGH(gh)
		}
		i.Assignee.SetGH(gh)
		for _, a := range i.Assignees {
			a.SetGH(gh)
		}
		i.Milestone.SetGH(gh)
		i.Closed_By.SetGH(gh)
	}
}

func (c *Comment) SetGH(gh *GitHubClient) {
	if c != nil {
		c.GitHubClient = gh
		c.User.SetGH(gh)
	}
}

func (r *Repository) SetGH(gh *GitHubClient) {
	if r != nil {
		r.GitHubClient = gh
		r.Owner.SetGH(gh)
	}
}

func (o *Organization) SetGH(gh *GitHubClient) {
	if o != nil {
		o.GitHubClient = gh
	}
}

func (e *Enterprise) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
	}
}

func (t *Team) SetGH(gh *GitHubClient) {
	if t != nil {
		t.GitHubClient = gh
	}
}

func (e *Event_Issue_Comment) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		e.Sender.SetGH(gh)
		e.Repository.SetGH(gh)
		e.Organization.SetGH(gh)
		e.Issue.SetGH(gh)
		e.Comment.SetGH(gh)
	}
}

func (e *Event_Issues) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		e.Issue.SetGH(gh)
		e.Assignee.SetGH(gh)
		e.Label.SetGH(gh)
		e.Repository.SetGH(gh)
		e.Organization.SetGH(gh)
		e.Sender.SetGH(gh)
	}
}

func (e *Event_Milestone) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		e.Milestone.SetGH(gh)
		e.Repository.SetGH(gh)
		e.Organization.SetGH(gh)
		e.Sender.SetGH(gh)
	}
}

func (e *Event_Push) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		for _, c := range e.Commits {
			c.SetGH(gh)
		}
		e.Pusher.SetGH(gh)
		e.Repository.SetGH(gh)
		e.Organization.SetGH(gh)
		e.Enterprise.SetGH(gh)
		e.Sender.SetGH(gh)
		e.Head_Commit.SetGH(gh)
	}
}

func (e *Commit) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		e.Author.SetGH(gh)
		e.Committer.SetGH(gh)
	}
}

func (e *MiniUser) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
	}
}

func (e *Project) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		e.Creator.SetGH(gh)
	}
}

func (e *Column) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
	}
}

func (e *Card) SetGH(gh *GitHubClient) {
	if e != nil {
		e.GitHubClient = gh
		e.Creator.SetGH(gh)
	}
}

type GitResponse struct {
	StatusCode int
	Links      map[string]string
	Body       []byte
}

func (gh *GitHubClient) Git(method string, url string, body string) (*GitResponse, error) {

	if gh.Token == "" {
		return nil, fmt.Errorf("Missing GitHub Token, perhaps .gitToken is missing?")
	}

	// fmt.Printf("Git: %s %s\n%s\n\n", method, url, body)
	if !strings.HasPrefix(url, "https://") {
		if len(url) > 0 && url[0] != '/' {
			url = "/" + url
		}

		url = fmt.Sprintf("https://%s/api/v3%s", gh.Host, url)
	}

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

	auth := base64.StdEncoding.EncodeToString([]byte("user:" + gh.Token))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	if strings.Contains(url, "projects") || strings.Contains(url, "cards") ||
		strings.Contains(url, "columns") {
		req.Header.Add("Accept", "application/vnd.GitHubClient.inertia-preview+json")
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
	gitResponse.Body = buf

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
func (gh *GitHubClient) GetAll(url string, daItem interface{}) (interface{}, error) {
	daType := reflect.TypeOf(daItem)
	result := reflect.MakeSlice(daType, 0, 0)

	for url != "" {
		var res *GitResponse
		var err error
		if res, err = gh.Git("GET", url, ""); err != nil {
			return nil, err
		}

		// Create a pointer Value to a slice, JSON Unmarshal needs a ptr
		itemsPtr := reflect.New(daType)

		// Create an empty slice Value and make our pointer reference it
		itemsPtr.Elem().Set(reflect.MakeSlice(daType, 0, 0))

		err = json.Unmarshal(res.Body, itemsPtr.Interface())
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

func (gh *GitHubClient) GraphQL(cmd string) (map[string]interface{}, error) {
	buf := []byte{}
	resMap := map[string]interface{}{}

	js := struct {
		Query string `json:"query"`
	}{
		Query: cmd,
	}

	buf, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}

	url := "https://api." + gh.Host + "/graphql"

	req, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		fmt.Printf("GitQL: %s\n", url)
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte("user:" + gh.Token))
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

	if res.StatusCode/100 != 2 {
		return nil,
			fmt.Errorf("GitQL: Error %s: %d %s\nReq Body: %s\n", url,
				res.StatusCode, string(buf), cmd)
	}

	err = json.Unmarshal(buf, &resMap)

	return resMap, err
}

func (gh *GitHubClient) VerifyEvent(req *http.Request, body []byte) bool {
	sig := req.Header.Get("X-HUB-SIGNATURE")

	if len(sig) != 45 || !strings.HasPrefix(sig, "sha1=") {
		return false
	}

	calc := make([]byte, 20)
	hex.Decode(calc, []byte(sig[5:]))

	mac := hmac.New(sha1.New, []byte(gh.Secret))
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
	_, err := issue.Git("POST", issue.URL+"/labels", `{"labels": [ "`+label+`"]}`)
	return err
}

func (issue *Issue) RemoveLabel(label string) error {
	_, err := issue.Git("DELETE", issue.URL+"/labels/"+label, "")
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
	_, err := issue.Git("POST", issue.URL+"/comments", Body(comment))
	return err
}

func (issue *Issue) Close() error {
	_, err := issue.Git("PATCH", issue.URL, `{"state":"closed"}`)
	return err
}

func (issue *Issue) Reopen() error {
	_, err := issue.Git("PATCH", issue.URL, `{"state":"open"}`)
	return err
}

func (issue *Issue) SetBody(body string) error {
	data := Body(body)
	res, err := issue.Git("PATCH", issue.URL, string(data))
	if err != nil {
		return err
	}

	newIssue := Issue{}
	if err = json.Unmarshal(res.Body, &newIssue); err != nil {
		return err
	}
	newIssue.SetGH(issue.GitHubClient)

	// Erase old data and replace with the updated Issue
	*issue = Issue{}
	*issue = newIssue

	return nil
}

func (issue *Issue) Refresh() error {
	newIssue, err := issue.GetIssue(issue.URL)
	if err != nil {
		return err
	}
	*issue = Issue{}
	*issue = *newIssue
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
	_, err := issue.Git("POST", issue.URL+"/assignees", `{"assignees":["`+user+`"]}`)
	return err
}

func (issue *Issue) RemoveAssignee(user string) error {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}
	_, err := issue.Git("DELETE", issue.URL+"/assignees", `{"assignees":["`+user+`"]}`)
	return err
}

func (issue *Issue) SetMilestone(newMile string) error {
	if newMile == "" {
		_, err := issue.Git("PATCH", issue.URL, `{"milestone": null}`)
		return err
	}

	items, err := issue.GetAll(issue.Repository_URL+"/milestones", []*Milestone{})
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

	_, err = issue.Git("PATCH", issue.URL, fmt.Sprintf(`{"milestone": %d}`, mileNum))

	return err
}

func (org *Organization) IsMember(user string) (bool, error) {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}
	res, err := org.Git("GET", org.URL+"/public_members/"+user, "")
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (org *Organization) IsTeamMember(user string, team string) (bool, error) {
	if len(user) > 1 && user[0] == '@' {
		user = user[1:]
	}

	res, err := org.Git("GET", org.URL+"/teams/"+team, "")
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return false, fmt.Errorf("Team %q not found", team)
		}
		return false, err
	}

	daTeam := Team{}
	if err = json.Unmarshal(res.Body, &daTeam); err != nil {
		return false, err
	}
	daTeam.SetGH(org.GitHubClient)

	loc := fmt.Sprintf("/organizations/%d/team/%d/memberships/%s",
		org.ID, daTeam.ID, user)
	res, err = org.Git("GET", loc, "")
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		if res != nil && res.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (repo *Repository) GetLabels() ([]*Label, error) {
	items, err := repo.GetAll(repo.URL+"/labels", []*Label{})
	if err != nil {
		return nil, err
	}
	return items.([]*Label), nil

	labels := items.([]*Label)
	for _, label := range labels {
		label.SetGH(repo.GitHubClient)
	}

	return labels, nil
}

func (repo *Repository) GetIssues(query string) ([]*Issue, error) {
	url := repo.URL + "/issues"
	if query != "" {
		url += "?" + query
	}

	items, err := repo.GetAll(url, []*Issue{})
	if err != nil {
		return nil, err
	}

	issues := items.([]*Issue)
	for _, issue := range issues {
		issue.SetGH(repo.GitHubClient)
	}

	return issues, nil
}

func (repo *Repository) GetMilestones(query string) ([]*Milestone, error) {
	url := repo.URL + "/milestones"
	if query != "" {
		url += "?" + query
	}

	items, err := repo.GetAll(url, []*Milestone{})
	if err != nil {
		return nil, err
	}

	milestones := items.([]*Milestone)
	for _, milestone := range milestones {
		milestone.SetGH(repo.GitHubClient)
	}

	return milestones, nil
}

// /repos/:owner/:repo/issues/:issue_number
func (gh *GitHubClient) GetMilestone(url string) (*Milestone, error) {
	res, err := gh.Git("GET", url, "")
	if err != nil {
		return nil, err
	}

	milestone := Milestone{}
	if err = json.Unmarshal(res.Body, &milestone); err != nil {
		return nil, err
	}
	milestone.SetGH(gh)
	return &milestone, nil
}

func (milestone *Milestone) Refresh() error {
	newMile, err := milestone.GetMilestone(milestone.URL)
	if err != nil {
		return err
	}
	newMile.SetGH(milestone.GitHubClient)
	*milestone = Milestone{}
	*milestone = *newMile

	return nil
}

func (gh *GitHubClient) GetRepository(org string, name string) (*Repository, error) {
	res, err := gh.Git("GET", "/repos/"+org+"/"+name, "")
	if err != nil {
		return nil, err
	}

	repo := Repository{}
	if err = json.Unmarshal(res.Body, &repo); err != nil {
		return nil, err
	}
	repo.SetGH(gh)

	return &repo, nil
}

func (gh *GitHubClient) SetIssueMilestone(org string, repo string, num int, newMile string) (*Issue, error) {
	items, err := gh.GetAll("/repos/"+org+"/"+repo+"/milestones",
		[]*Milestone{})
	if err != nil {
		return nil, err
	}
	milestones := items.([]*Milestone)

	mileNum := -1
	for _, mile := range milestones {
		mile.SetGH(gh)
		if mile.Title == newMile {
			mileNum = mile.Number
		}
	}
	if mileNum == -1 {
		err = fmt.Errorf("Can't find milestone %q\n", newMile)
		return nil, err
	}

	url := fmt.Sprintf("/repos/%s/%s/issues/%d", org, repo, num)
	res, err := gh.Git("PATCH", url, fmt.Sprintf(`{"milestone": %d}`, mileNum))
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err = json.Unmarshal(res.Body, &issue); err != nil {
		return nil, err
	}
	issue.SetGH(gh)

	return &issue, nil
}

func (gh *GitHubClient) GetRepositoryMilestones(org string, repo string) ([]*Milestone, error) {
	items, err := gh.GetAll("/repos/"+org+"/"+repo+"/milestones", []*Milestone{})
	if err != nil {
		return nil, err
	}
	milestones := items.([]*Milestone)
	for _, mile := range milestones {
		mile.SetGH(gh)
	}
	return milestones, nil
}

// /repos/:owner/:repo/issues/:issue_number
func (gh *GitHubClient) GetIssue(url string) (*Issue, error) {
	res, err := gh.Git("GET", url, "")
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err = json.Unmarshal(res.Body, &issue); err != nil {
		return nil, err
	}
	issue.SetGH(gh)

	return &issue, nil
}

func (gh *GitHubClient) GetIssueParts(org string, repo string, num int) (*Issue, error) {
	url := fmt.Sprintf("/repos/%s/%s/issues/%d", org, repo, num)
	return gh.GetIssue(url)
}

func (gh *GitHubClient) GetIssuesParts(org string, repo string, query string) ([]*Issue, error) {
	url := fmt.Sprintf("/repos/%s/%s/issues", org, repo)
	if query != "" {
		url += "?" + query
	}
	items, err := gh.GetAll(url, []*Issue{})
	if err != nil {
		return nil, err
	}

	issues := items.([]*Issue)
	for _, issue := range issues {
		issue.SetGH(gh)
	}
	return issues, nil
}

func (gh *GitHubClient) GetMilestones(org string, repo string, query string) ([]*Milestone, error) {
	url := fmt.Sprintf("/repos/%s/%s/milestones", org, repo)
	if query != "" {
		url += "?" + query
	}
	items, err := gh.GetAll(url, []*Milestone{})
	if err != nil {
		return nil, err
	}
	milestones := items.([]*Milestone)
	for _, mile := range milestones {
		mile.SetGH(gh)
	}
	return milestones, nil
}

func (gh *GitHubClient) GetRepositoryTeams(org string, repo string) ([]*Team, error) {
	items, err := gh.GetAll("/repos/"+org+"/"+repo+"/teams", []*Label{})
	if err != nil {
		return nil, err
	}
	teams := items.([]*Team)
	for _, team := range teams {
		team.SetGH(gh)
	}
	return teams, nil
}

func (gh *GitHubClient) IsUserInOrganization(org string, user string) (bool, error) {
	_, err := gh.Git("GET", "/orgs/"+org+"/public_members/"+user, "")
	if err != nil {
		return false, err
	}
	return true, nil
}

// Github Data Manipulation

type GitData struct {
	Body []string
	Data [][2]string
}

type ByLabel [][2]string

func (a ByLabel) Len() int      { return len(a) }
func (a ByLabel) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLabel) Less(i, j int) bool {
	if a[i][0] == a[j][0] {
		return a[i][1] < a[j][1]
	}
	return a[i][0] < a[j][0]
}

func (gd *GitData) AddData(label string, text string) {
	for _, entry := range gd.Data {
		if entry[0] == label && entry[1] == text {
			return
		}
	}

	gd.Data = append(gd.Data, [2]string{label, text})
}

func (gd *GitData) DeleteData(label string, text string) bool {
	res := false
	for i, entry := range gd.Data {
		if entry[0] == label && (text == "" || entry[1] == text) {
			gd.Data = append(gd.Data[:i], gd.Data[i+1:]...)
			res = true
		}
	}
	if len(gd.Data) == 0 {
		gd.Data = nil
	}

	return res
}

func (gd *GitData) HasData(label string, text string) bool {
	for _, entry := range gd.Data {
		if entry[0] == label && entry[1] == text {
			return true
		}
	}
	return false
}

func (gd *GitData) SetData(label string, text string) {
	gd.DeleteData(label, "")
	gd.AddData(label, text)
}

func (issue *Issue) GetGitData() *GitData {
	return ParseForGitData(issue.Body)
}

func ParseForGitData(comment string) *GitData {
	data := &GitData{}

	lines := strings.Split(comment, "\n")
	for _, line := range lines {
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

func (issue *Issue) GetData(label string) []string {
	data := issue.GetGitData()

	var res []string = nil
	for _, entry := range data.Data {
		if entry[0] == label {
			res = append(res, entry[1])
		}
	}

	return res
}

func (issue *Issue) GetSingleData(label string) string {
	data := issue.GetGitData()

	for _, entry := range data.Data {
		if entry[0] == label {
			return entry[1]
		}
	}

	return ""
}

func (issue *Issue) AddData(label string, text string) error {
	data := issue.GetGitData()
	data.AddData(label, text)
	return issue.SetGitData(data)
}

func (issue *Issue) DeleteData(label string, text string) error {
	data := issue.GetGitData()
	if data.DeleteData(label, text) {
		return issue.SetGitData(data)
	}
	return nil
}

func (issue *Issue) HasData(label string, text string) bool {
	data := issue.GetGitData()
	return data.HasData(label, text)
}

func (issue *Issue) SetData(label string, text string) error {
	if d := issue.GetData(label); len(d) == 1 && d[0] == text {
		return nil
	}
	data := issue.GetGitData()
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

		sort.Sort(ByLabel(data.Data))

		for _, entry := range data.Data {
			body += fmt.Sprintf("**_%s_**: %s\n", entry[0], entry[1])
		}
	}

	return issue.SetBody(body)
}

func (issue *Issue) GetRepository() (*Repository, error) {
	res, err := issue.Git("GET", issue.Repository_URL, "")
	if err != nil {
		return nil, err
	}

	repo := Repository{}
	if err = json.Unmarshal(res.Body, &repo); err != nil {
		return nil, err
	}
	repo.SetGH(issue.GitHubClient)

	return &repo, nil
}

func (repo *Repository) GetProjects() ([]*Project, error) {
	url := repo.URL + "/projects"

	items, err := repo.GetAll(url, []*Project{})
	if err != nil {
		return nil, err
	}

	projects := items.([]*Project)
	for _, project := range projects {
		project.SetGH(repo.GitHubClient)
	}

	return projects, nil
}

func (repo *Repository) GetProject(name string) (*Project, error) {
	projects, err := repo.GetProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.Name == name {
			return project, nil
		}
	}

	return nil, nil
}

func (repo *Repository) GetFile(path string) ([]byte, error) {
	url := fmt.Sprintf("%s/contents/%s", repo.URL, path)

	res, err := repo.Git("GET", url, "")
	if res != nil && res.StatusCode == 404 {
		return nil, fmt.Errorf("File not found: %s", path)
	}

	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

// Find all cards for an issue in a project
func (issue *Issue) GetProjectCards(name string) ([]*Card, error) {
	repo, err := issue.GetRepository()
	if err != nil {
		return nil, err
	}
	proj, err := repo.GetProject(name)
	if err != nil {
		return nil, err
	}
	cols, err := proj.GetColumns()
	if err != nil {
		return nil, err
	}

	var result []*Card

	for _, col := range cols {
		cards, _ := col.GetCards()
		for _, card := range cards {
			if card.Content_URL == issue.URL {
				result = append(result, card)
			}
		}
	}

	return result, nil
}

func (issue *Issue) AddToProject(name string) error {
	repo, err := issue.GetRepository()
	if err != nil {
		return err
	}
	proj, err := repo.GetProject(name)
	if err != nil {
		return err
	}
	col, err := proj.GetColumn("Under Review")
	if err != nil {
		return err
	}

	data := fmt.Sprintf(`{"note":null,"content_id":%d,"content_type":"Issue"}`,
		issue.ID)

	res, err := repo.Git("POST", col.Cards_URL, data)
	if res.Body != nil {
		gitErr := struct {
			Message string
			Errors  []struct {
				Resource string
				Code     string
				Field    string
				Message  string
			}
			Documentation_URL string
		}{}

		err := json.Unmarshal(res.Body, &gitErr)
		if err == nil {
			if gitErr.Errors != nil {
				if gitErr.Errors[0].Message == "Project already has the associated issue" {
					return nil
				}
			}
		}
	}

	return err
}

func (issue *Issue) RemoveFromProject(name string) error {
	cards, err := issue.GetProjectCards(name)
	if err != nil {
		return err
	}

	for _, card := range cards {
		card.Delete()
	}

	return nil
}

func (card *Card) Delete() error {
	_, err := card.Git("DELETE", card.URL, "")
	return err
}

func (issue *Issue) MoveToRepository(repoName string) error {
	oldRepo, err := issue.GetRepository()
	if err != nil {
		return err
	}

	newRepo, err := issue.GitHubClient.GetRepository(oldRepo.Owner.Login, repoName)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`
mutation {
  transferIssue( input:{
    issueId : "%s",
	repositoryId : "%s"
  }) {
    issue { number }
  }
}
`, issue.Node_ID, newRepo.Node_ID)

	res, err := issue.GraphQL(cmd)
	if err != nil {
		return err
	}

	if res["errors"] != nil {
		return fmt.Errorf("GraphQL Error: %#v\n", res["errors"])
	}

	return nil
}

func (project *Project) GetColumns() ([]*Column, error) {
	items, err := project.GetAll(project.Columns_URL, []*Column{})
	if err != nil {
		return nil, err
	}

	columns := items.([]*Column)
	for _, column := range columns {
		column.SetGH(project.GitHubClient)
	}

	return columns, nil
}

func (project *Project) GetColumn(name string) (*Column, error) {
	items, err := project.GetAll(project.Columns_URL, []*Column{})
	if err != nil {
		return nil, err
	}

	columns := items.([]*Column)
	for _, column := range columns {
		if column.Name == name {
			column.SetGH(project.GitHubClient)
			return column, nil
		}
	}

	return nil, nil
}

func (project *Project) GetCards() ([]*Card, error) {
	cols, err := project.GetColumns()
	if err != nil {
		return nil, err
	}

	var result []*Card

	for _, col := range cols {
		cards, _ := col.GetCards()
		for _, card := range cards {
			result = append(result, card)
		}
	}

	return result, nil
}

func (column *Column) GetCards() ([]*Card, error) {
	items, err := column.GetAll(column.Cards_URL, []*Card{})
	if err != nil {
		return nil, err
	}

	cards := items.([]*Card)
	for _, card := range cards {
		card.SetGH(column.GitHubClient)
	}

	return cards, nil
}
