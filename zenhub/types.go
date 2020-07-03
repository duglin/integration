package zenhub

// GET /p1/repositories/:repo_id/issues/:issue_number -> zenIssue
type ZenIssue struct {
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

// GET /p2/repositories/:repo_id/workspaces -> []ZenWorkspace
type ZenWorkspace struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ID           string `json:"id"`
	Repositories []int  `json:"repositories"`
}

// GET /p2/workspaces/:workspace_id/repositories/:repo_id/board -> ZenBoard
type ZenBoard struct {
	Pipelines []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Issues []struct {
			Issue_Number int `json:"issue_number"`
			Estimate     struct {
				Value int `json:"value"`
			} `json:"estimate"`
			Position int  `json:"position"`
			Is_Epic  bool `json:"is_epic"`
		} `json:"issues"`
	} `json:"pipelines"`
}

// GET /p1/repositories/:repo_id/epics  -> []zenEpic
type ZenRepositoryEpics struct {
	Epic_Issues []struct {
		Issue_Number int    `json:"issue_number"`
		Repo_ID      int    `json:"repo_id"`
		Issue_URL    string `json:"issue_url"`
	} `json:"epic_issues"`
}

// GET /p1/repositories/:repo_id/epics/:epic_id  -> zenEpic
type ZenEpic struct {
}
