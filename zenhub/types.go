package zenhub

type ZenHubClient struct {
	URL    string
	Token  string
	Secret string
}

func NewZenHubClient(url string, token string, secret string) *ZenHubClient {
	return &ZenHubClient{
		URL:    url,
		Token:  token,
		Secret: secret,
	}
}

// GET /p1/repositories/:repo_id/issues/:issue_number -> Issue
type Issue struct {
	*ZenHubClient

	Estimate struct {
		Value int `json:"value"`
	} `json:"estimate"`
	Plus_Ones []struct {
		Created_At string `json:"created_at"`
	} `json:"plus_ones"`
	Pipelines []struct {
		Name         string `json:"name"`
		Pipeline_ID  string `json:"pipeline_id"`
		Workspace_ID string `json:"workspace_id"`
	} `json:"pipelines"`
	Is_Epic bool `json:"is_epic"`
}

// GET /p2/repositories/:repo_id/workspaces -> []Workspace
type Workspace struct {
	*ZenHubClient

	Name         string `json:"name"`
	Description  string `json:"description"`
	ID           string `json:"id"`
	Repositories []int  `json:"repositories"`
}

type PipelineIssue struct {
	*ZenHubClient

	Issue_Number int `json:"issue_number"`
	Estimate     struct {
		Value int `json:"value"`
	} `json:"estimate"`
	Position int  `json:"position"`
	Is_Epic  bool `json:"is_epic"`
}

type Pipeline struct {
	*ZenHubClient

	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Issues []*PipelineIssue `json:"issues"`
}

// GET /p2/workspaces/:workspace_id/repositories/:repo_id/board -> Board
type Board struct {
	*ZenHubClient

	Workspace *Workspace
	RepoID    int
	Pipelines []*Pipeline `json:"pipelines"`
}

// GET /p1/repositories/:repo_id/epics  -> []Epic
type RepositoryEpics struct {
	*ZenHubClient

	Epic_Issues []struct {
		Issue_Number int    `json:"issue_number"`
		Repo_ID      int    `json:"repo_id"`
		Issue_URL    string `json:"issue_url"`
	} `json:"epic_issues"`
}

// GET /p1/repositories/:repo_id/epics/:epic_id  -> Epic
type Epic struct {
	*ZenHubClient
}
