package aha

// https://www.aha.io/api

type AhaClient struct {
	URL    string
	Token  string
	Secret string // used to verify events are from Aha
}

func NewAhaClient(url string, token string, secret string) *AhaClient {
	return &AhaClient{
		URL:    url,
		Token:  token,
		Secret: secret,
	}
}

type Pagination struct {
	Total_Records int
	Total_Pages   int
	Current_Page  int
}

type User struct {
	*AhaClient

	ID         string
	Name       string
	Email      string
	Created_At string
	Updated_At string
}

type Event struct {
	*AhaClient

	Event string
	Audit struct {
		ID           string
		Audit_Action string
		Created_At   string
		Interesting  bool
		User         *User
		Contributors []struct {
			User *User
		}
		Auditable_Type  string
		Auditable_ID    string
		Associated_Type string
		Associated_ID   string
		Description     string
		Auditable_URL   string
		Changes         []struct {
			Field_Name string
			Value      interface{}
		}
	}
}

type Product struct {
	*AhaClient

	ID                string
	Reference_Prefix  string
	Name              string
	Product_Line      bool
	Product_Line_Type interface{}
	Created_At        string
	Updated_At        string
	Description       struct {
		ID          string
		Body        string
		Created_At  string
		Attachments []interface{}
	}
	URL                string
	Resource           string
	Children           []interface{}
	Custom_Fields      []interface{}
	Screen_Definitions []struct {
		ID                       string
		Screenable_Type          string
		Name                     string
		Custom_Field_Definitions []struct {
			ID       string
			Key      string
			Position int
			Name     string
			Type     string
			API_Type string
			Required bool
			Options  []struct {
				ID    string
				Label string
			}
		}
	}
}

type Feature struct {
	*AhaClient

	ID              string
	Name            string
	Reference_Num   string
	Position        int
	Score           int
	Created_At      string
	Updated_At      string
	Start_Date      string
	Due_Date        string
	Product_ID      string
	Progress        interface{}
	Progress_Source string
	Workflow_Kind   struct {
		ID   string
		Name string
	}
	Workflow_Status *Workflow_Status
	Description     struct {
		ID          string
		Body        string
		Created_At  string
		Attachments []*Attachment
	}
	Attachments              []*Attachment
	Integration_Fields       []*Integration_Field
	URL                      string
	Resource                 string
	Release                  *Release
	MasterFeature            *Feature
	Belongs_To_Release_Phase *Release_Phase
	Epic                     *Feature
	Created_By_User          *User
	Assign_To_User           *User
	Requirements             []*Requirement
	// Initiative
	// Goals
	Comment_Count int
	// Score_Facts
	Tags      []string
	Full_Tags []struct {
		ID    string
		Name  string
		Color string
	}
	Custom_Fields       []*Custom_Field
	Custom_Object_Links []struct {
		Key         string
		Name        string
		Record_Type string
		Record_IDs  []string
	}
	// Feature_Links
	// Feature_Only_Original_Estimate
	// Feature_Only_Remaining_Estimate
	// Feature_Only_Work_Done

	Product *Product
}

type Epic Feature

type Release struct {
	*AhaClient

	ID                 string         `json:"id,omitempty"`
	Reference_Num      string         `json:"reference_num,omitempty"`
	Name               string         `json:"name,omitempty"`
	Start_Date         string         `json:"start_date,omitempty"`
	Release_Date       string         `json:"release_date,omitempty"`
	Parking_Lot        bool           `json:"parking_lot,omitempty"`
	Created_At         string         `json:"created_at,omitempty"`
	Product_ID         string         `json:"product_id,omitempty"`
	Integration_Fields []*Integration `json:"integration_fields,omitempty"`
	URL                string         `json:"url,omitempty"`
	Resource           string         `json:"resource,omitempty"`
	Owner              *User          `json:"owner,omitempty"`
	Project            *Project       `json:"project,omitempty"`

	Product *Product
}

type Release_Phase struct {
	*AhaClient

	ID              string
	Name            string
	Start_On        string
	End_On          string
	Type            string
	Release_ID      string
	Created_At      string
	Updated_At      string
	Progress        interface{}
	Progress_Source string
	Description     struct {
		ID          string
		Body        string
		Created_At  string
		Attachments []*Attachment
	}
}

type Integration_Field struct {
	*AhaClient

	ID             string
	Name           string
	Value          string
	Integration_ID string
	Service_Name   string
	Created_At     string
}

type Attachment struct {
	*AhaClient

	ID           string
	Download_URL string
	Created_At   string
	Updated_At   string
	Content_Type string
	File_Name    string
	File_Size    int
}

type Requirement struct {
	*AhaClient

	ID              string
	Name            string
	Reference_Num   string
	Position        int
	Created_At      string
	Updated_At      string
	Release_ID      string
	Workflow_Status *Workflow_Status
	URL             string
	Resource        string
	Description     struct {
		ID          string
		Body        string
		Created_At  string
		Attachments []*Attachment
	}
	Feature            *Feature
	Assigned_To_User   *User
	Created_By_User    *User
	Attachments        []*Attachment
	Custom_Fields      []*Custom_Field
	Integration_Fields []*Integration_Field
}

type Workflow_Status struct {
	*AhaClient

	ID       string
	Name     string
	Position int
	Complete bool
	Color    string
}

type Custom_Field struct {
	*AhaClient

	Key   string
	Name  string
	Value interface{}
	Type  string
}

type Custom_Object_Record struct {
	*AhaClient

	ID                  string
	Product_ID          string
	Key                 string
	Created_At          string
	Updated_At          string
	Custom_Fields       []*Custom_Field
	Custom_Object_Links []struct {
		Key         string
		Name        string
		Record_Type string
		Record_IDs  []string
	}
}

type Integration struct {
	*AhaClient

	ID             string
	Service_Name   string
	Name           string
	Enabled        bool
	Callback_Token string
	Created_At     string
	Updated_At     string
	URL            string
	Resource       string
	Owner          *User
}

type Project struct {
	*AhaClient

	ID               string
	Reference_Prefix string
	Name             string
	Product_Line     bool
	Created_At       string
}
