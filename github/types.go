package github

// https://docs.github.com/en/developers/webhooks-and-events/webhook-events-and-payloads

type GitHubClient struct {
	Host   string
	Token  string
	Secret string // used to verify events are from github
}

func NewGitHubClient(host string, token string, secret string) *GitHubClient {
	return &GitHubClient{
		Host:   host,
		Token:  token,
		Secret: secret,
	}
}

type User struct {
	*GitHubClient

	Login               string
	ID                  int
	Node_ID             string
	Avator_URL          string
	Gravatar_ID         string
	URL                 string
	HTML_URL            string
	Followers_URL       string
	Following_URL       string
	Gists_URL           string
	Starred_URL         string
	Subscriptions_URL   string
	Organizations_URL   string
	Repos_URL           string
	Events_URL          string
	Received_Events_URL string
	Type                string
	Site_Admin          bool
}

type Label struct {
	*GitHubClient

	ID          int
	Node_ID     string
	URL         string
	Name        string
	Description string
	Color       string
	Default     bool
}

type Milestone struct {
	*GitHubClient

	ID            int
	URL           string
	HTML_URL      string
	Labels_URL    string
	Node_ID       string
	Number        int
	State         string
	Title         string
	Description   string
	Creator       *User
	Open_Issues   int
	Closed_Issues int
	Created_At    string
	Updated_At    string
	Closed_At     string
	Due_On        string
}

type Issue struct {
	*GitHubClient

	ID                 int
	Node_ID            string
	URL                string
	Repository_URL     string
	Labels_URL         string
	Comments_URL       string
	Events_URL         string
	HTML_URL           string
	Number             int
	State              string
	Title              string
	Body               string
	User               *User
	Labels             []*Label
	Assignee           *User
	Assignees          []*User
	Milestone          *Milestone
	Locked             bool
	Active_Lock_Reason string
	Comments           int
	Pull_Request       struct {
		URL       string
		HTML_URL  string
		Diff_URL  string
		Patch_URL string
	}
	Closed_At  string
	Updated_At string
	Closed_By  *User
}

type Comment struct {
	*GitHubClient

	URL                string
	HTML_URL           string
	Issue_URL          string
	ID                 int
	Node_ID            string
	User               *User
	Created_At         string
	Updated_At         string
	Author_Association string
	Body               string
}

type Repository struct {
	*GitHubClient

	ID                int
	Node_ID           string
	Name              string
	Full_Name         string
	Private           bool
	Owner             *User
	HTML_URL          string
	Description       string
	Fork              bool
	URL               string
	Forks_URL         string
	Keys_URL          string
	Collaborators_URL string
	Hooks_URL         string
	Teams_URL         string
	Issue_Events_URL  string
	Events_URL        string
	Assignees_URL     string
	Branches_URL      string
	Tags_URL          string
	Blobs_URL         string
	Git_tags_URL      string
	Git_refs_URL      string
	Trees_URL         string
	Statuses_URL      string
	Languages_URL     string
	Stargazers_URL    string
	Contributors_URL  string
	Subscribers_URL   string
	Subscription_URL  string
	Commits_URL       string
	Git_Commits_URL   string
	Comments_URL      string
	Issue_Comment_URL string
	Contents_URL      string
	Compare_URL       string
	Merges_URL        string
	Archive_URL       string
	Downloads_URL     string
	Issues_URL        string
	Pulls_URL         string
	Milestones_URL    string
	Notifications_URL string
	Labels_URL        string
	Releases_URL      string
	Deployments_URL   string
	Created_At        interface{} // string
	Updated_At        string
	Pushed_At         interface{} // string
	Git_URL           string
	SSH_URL           string
	Clone_URL         string
	SVN_URL           string
	Homepage          string
	Size              int
	Stargazers_count  int
	Watchers_Count    int
	Language          string
	Has_Issues        bool
	Has_Projects      bool
	Has_Downloads     bool
	Has_Wiki          bool
	Has_Pages         bool
	Forks_Count       int
	Mirror_URL        string
	Archived          bool
	Disabled          bool
	Open_Issue_Count  int
	License           struct {
		Key     string
		Name    string
		SPDX_ID string
		URL     string
		Node_ID string
	}
	Forks          int
	Open_Isses     int
	Watchers       int
	Default_Branch string
}

type Organization struct {
	*GitHubClient

	ID                 int
	Login              string
	Node_ID            string
	URL                string
	Repos_URL          string
	Event_URL          string
	Hooks_URL          string
	Issues_URL         string
	Members_URL        string
	Public_Members_URL string
	Avatar_URL         string
	Description        string
}

type Enterprise struct {
	*GitHubClient

	ID          int
	Slug        string
	Name        string
	Node_ID     string
	Avatar_URL  string
	Description string
	Website_URL string
	HTML_URL    string
	Created_At  string
	Updated_At  string
}

type Team struct {
	*GitHubClient

	ID               int
	Node_ID          string
	URL              string
	HTML_URL         string
	Name             string
	Slug             string
	Description      string
	Privacy          string
	Permission       string
	Members_URL      string
	Repositories_URL string
	Parent           string
}

type Event_Issue_Comment struct {
	*GitHubClient

	Action       string
	Sender       *User
	Repository   *Repository
	Organization *Organization

	Changes struct { // only for "edited" actions
		Body struct {
			From string
		}
	}
	Issue   *Issue
	Comment *Comment
}

type Event_Issues struct {
	*GitHubClient

	Action  string
	Issue   *Issue
	Changes struct { // only for "edited" actions
		Title struct {
			From string
		}
		Body struct {
			From string
		}
	}
	Assignee     *User
	Label        *Label
	Repository   *Repository
	Organization *Organization
	// Installation
	Sender *User
}

type Event_Milestone struct {
	*GitHubClient

	Action    string
	Milestone *Milestone
	Changes   struct {
		Description struct {
			From string
		}
		Due_On struct {
			From string
		}
		Title struct {
			From string
		}
	}
	Repository   *Repository
	Organization *Organization
	Installation struct {
	}
	Sender *User
}

type Event_Pull_Request struct {
	*GitHubClient
}

type Event_Push struct {
	*GitHubClient

	Ref          string
	Before       string
	After        string
	Commits      []*Commit
	Pusher       *MiniUser
	Repository   *Repository
	Organization *Organization
	Enterprise   *Enterprise
	Installation struct{}
	Sender       *User
	Created      bool
	Deleted      bool
	Forced       bool
	Compare      string
	Head_Commit  *Commit
}

type Commit struct {
	*GitHubClient

	ID        string // SHA of commit
	Tree_ID   string
	Distinct  bool
	Message   string
	Timestamp string // ISO8601
	URL       string
	Author    *MiniUser
	Committer *MiniUser
	Added     []string
	Removed   []string
	Modified  []string
}

type MiniUser struct {
	*GitHubClient

	Name     string // Git name
	Email    string // Git email
	Username string
}
