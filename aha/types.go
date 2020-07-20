package aha

type Pagination struct {
	Total_Records int
	Total_Pages   int
	Current_Page  int
}

type User struct {
	ID         string
	Name       string
	Email      string
	Created_At string
	Updated_At string
}

type Event struct {
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
	}
	Auditable_Type  string
	Auditable_ID    string
	Associated_Type string
	Associated_ID   string
	Description     string
	Auditable_URL   string
	Changes         []struct {
		Field_Name string
		Value      string
	}
}

type Product struct {
	ID               string
	Reference_Prefix string
	Name             string
	Product_Line     bool
	Created_At       string
}

type Feature struct {
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
	Progress_Sourec string
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
}

type Release struct {
	ID                 string
	Reference_Num      string
	Name               string
	Start_Date         string
	Release_Date       string
	Parking_Lot        bool
	Created_At         string
	Product_ID         string
	Integration_Fields []*Integration
	URL                string
	Resource           string
	Owner              *User
	Project            *Project
}

type Release_Phase struct {
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
	ID             string
	Name           string
	Value          string
	Integration_ID string
	Service_Name   string
	Created_At     string
}

type Attachment struct {
	ID           string
	Download_URL string
	Created_At   string
	Updated_At   string
	Content_Type string
	File_Name    string
	File_Size    int
}

type Requirement struct {
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
	ID       string
	Name     string
	Position int
	Complete bool
	Color    string
}

type Custom_Field struct {
	Key   string
	Name  string
	Value []struct {
		Values map[string]struct {
			Value         int
			Display_Value string
		}
	}
	Type string
}

type Integration struct {
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
	ID               string
	Reference_Prefix string
	Name             string
	Product_Line     bool
	Created_At       string
}
