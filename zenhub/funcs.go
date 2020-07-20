package zenhub

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// https://github.com/ZenHubIO/API

var ZenToken = ""
var ZenHubURL = ""
var ZenSecret = ""

func Zen(method string, url string, body string) (string, error) {
	buf := []byte{}
	if body != "" {
		buf = []byte(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(buf))

	req.Header.Add("X-Authentication-Token", ZenToken)
	req.Header.Add("Content-Type", "application/json")

	// fmt.Printf("*** ZEN: %s %s\n%s\n", method, url, body)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	res, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	buf, _ = ioutil.ReadAll(res.Body)
	// fmt.Printf("    Res: %d %s\n", res.StatusCode, string(buf))
	if res.StatusCode/100 != 2 {
		fmt.Printf("Zen Error:\n--> %s %s\n--> %s\n", method, url, body)
		fmt.Printf("%d %s\n", res.StatusCode, string(buf))
		return "", fmt.Errorf("Error zening: %d %s\n", res.StatusCode, string(buf))
	}
	return string(buf), nil
}

func GetIssue(repoID int, issueNum int) (*Issue, error) {
	url := fmt.Sprintf("%s/p1/repositories/%d/issues/%d", ZenHubURL, repoID,
		issueNum)
	res, err := Zen("GET", url, "")
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err = json.Unmarshal([]byte(res), &issue); err != nil {
		fmt.Printf("json: %s\n", res)
		return nil, err
	}
	return &issue, nil
}

func MakeEpic(repoID int, issueNum int) error {
	url := fmt.Sprintf("%s/p1/repositories/%d/issues/%d/convert_to_epic",
		ZenHubURL, repoID, issueNum)
	_, err := Zen("POST", url, "[]")
	return err
}

// POST /p2/workspaces/:workspace_id/repositories/:repo_id/issues/:issue_number/moves
func SetIssuePipeline(workspaceID string, repoID int, issueNum int, pipelineID string) error {
	url := fmt.Sprintf("%s/p2/workspaces/%s/repositories/%d/issues/%d/moves",
		ZenHubURL, workspaceID, repoID, issueNum)
	body := fmt.Sprintf(`{"pipeline_id":"%s","position":"top"}`, pipelineID)

	res, err := Zen("POST", url, string(body))
	if err != nil {
		err = fmt.Errorf("Error setting pipeline: %s\n%s", err, res)
	}
	return err
}

func SetIssuePipeline2(repoID int, workspace string, issueNum int, pipeline string) error {
	board, err := GetBoard(repoID, workspace)
	if err != nil {
		return err
	}

	for _, p := range board.Pipelines {
		if p.Name == pipeline {
			return SetIssuePipeline(board.Workspace.ID, repoID, issueNum, p.ID)
		}
	}

	return fmt.Errorf("Can't find pipeline %q", pipeline)
}

func GetWorkspaces(repoID int) ([]*Workspace, error) {
	url := fmt.Sprintf("%s/p2/repositories/%d/workspaces", ZenHubURL, repoID)
	res, err := Zen("GET", url, "")
	if err != nil {
		return nil, err
	}

	// [{"name":"Planning","description":null,"id":"5e25e46b8ce0f020d121738b","repositories":[685476,752885]},{"name":"Coligo Broker","description":null,"id":"5e4f33fc8c800b6f2f4e05ec","repositories":[732940,685476]},{"name":"Cross Squad Work Items","description":null,"id":"5eda566f7e176e0c85419a41","repositories":[685476,752885]}]

	workspaces := []*Workspace{}
	if err = json.Unmarshal([]byte(res), &workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

func GetWorkspace(repoID int, workspace string) (*Workspace, error) {
	workspaces, err := GetWorkspaces(repoID)
	if err != nil {
		return nil, err
	}

	for _, w := range workspaces {
		if w.Name == workspace {
			return w, nil
		}
	}

	return nil, nil
}

func (workspace *Workspace) GetBoard(repoID int) (*Board, error) {
	url := fmt.Sprintf("%s/p2/workspaces/%s/repositories/%d/board", ZenHubURL, workspace.ID, repoID)
	res, err := Zen("GET", url, "")
	if err != nil {
		return nil, err
	}

	board := Board{}
	if err = json.Unmarshal([]byte(res), &board); err != nil {
		return nil, err
	}
	return &board, nil

}

func (workspace *Workspace) GetPipeline(repoID int, pipeline string) (*Pipeline, error) {
	board, err := workspace.GetBoard(repoID)
	if err != nil {
		return nil, err
	}

	for _, p := range board.Pipelines {
		if p.Name == pipeline {
			return p, nil
		}
	}
	return nil, nil
}

func GetBoard(repoID int, workspace string) (*Board, error) {
	w, err := GetWorkspace(repoID, workspace)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, fmt.Errorf("Can't find workspace %q", workspace)
	}

	b, err := w.GetBoard(repoID)
	if err != nil {
		return nil, err
	}

	b.Workspace = w
	b.RepoID = repoID

	return b, nil
}

func GetPipeline(repoID int, workspace string, pipeline string) (*Pipeline, error) {
	board, err := GetBoard(repoID, workspace)
	if err != nil {
		return nil, err
	}

	for _, p := range board.Pipelines {
		if p.Name == pipeline {
			return p, nil
		}
	}
	return nil, nil
}

func AddTask(epicRepoID int, epicNum int, taskRepoID int, taskNum int) error {
	url := fmt.Sprintf("%s/p1/repositories/%d/epics/%d/update_issues",
		ZenHubURL, epicRepoID, epicNum)
	body := fmt.Sprintf(`{"add_issues":[{"repo_id":%d,"issue_number":%d}]}`,
		taskRepoID, taskNum)
	_, err := Zen("POST", url, body)
	return err
}
