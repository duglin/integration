package aha

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type AhaResponse struct {
	StatusCode int
	Body       string
	PageInfo   Pagination
}

func (ac *AhaClient) Aha(method string, url string, body string) (*AhaResponse, error) {
	// defer fmt.Printf("\n")
	// fmt.Printf("%s %s", method, url)
	ahaResponse := AhaResponse{}

	if ac.Token == "" {
		return nil, fmt.Errorf("Missing Aha Token, perhaps .ahaToken is missing?")
	}

	buf := []byte{}
	if body != "" {
		buf = []byte(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+ac.Token)
	req.Header.Add("Content-Type", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	res, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf, _ = ioutil.ReadAll(res.Body)

	ahaResponse.StatusCode = res.StatusCode
	// fmt.Printf(" - %d", res.StatusCode)

	if len(buf) > 0 {
		rawMap := map[string]json.RawMessage{} // interface{}{}
		err = json.Unmarshal([]byte(buf), &rawMap)
		if err != nil {
			return &ahaResponse, err
		}

		if info, ok := rawMap["pagination"]; ok {
			err = json.Unmarshal(info, &ahaResponse.PageInfo)
			if err != nil {
				return &ahaResponse, err
			}

			for k, v := range rawMap {
				if k == "pagination" {
					continue
				}
				ahaResponse.Body = string(v)
			}
		} else {
			ahaResponse.Body = string(buf)
		}
	}

	// fmt.Printf("\n\n\nGET: %s\n%s\n", url, string(buf))
	if res.StatusCode/100 != 2 {
		// fmt.Printf("Aha Error:\n--> %s %s\n--> %s\n", method, url, body)
		// fmt.Printf("%d %s\n", res.StatusCode, string(buf))
		return &ahaResponse,
			fmt.Errorf("Aha: Error %s: %d %s\nReq Body: %s\n", url,
				res.StatusCode, string(buf), body)
	}

	return &ahaResponse, nil
}

func (ac *AhaClient) GetAll(daURL string, daItem interface{}) (interface{}, error) {
	size := 0 // unlimited

	URL, err := url.Parse(daURL)
	if err != nil {
		return nil, err
	}
	if len(URL.RawQuery) == 0 {
		daURL += "?"
	}

	if size != 0 {
		daURL = fmt.Sprintf("%s&size=%d", daURL, size)
	}

	oldURL := daURL
	daType := reflect.TypeOf(daItem)
	result := reflect.MakeSlice(daType, 0, 0)

	for daURL != "" {
		var res *AhaResponse
		var err error
		if res, err = ac.Aha("GET", daURL, ""); err != nil {
			return nil, err
		}

		// Create a pointer Value to a slice, JSON Unmarshal needs a ptr
		itemsPtr := reflect.New(daType)

		// Create an empty slice Value and make our pointer reference it
		itemsPtr.Elem().Set(reflect.MakeSlice(daType, 0, 0))

		err = json.Unmarshal([]byte(res.Body), itemsPtr.Interface())
		if err != nil {
			// fmt.Printf("%#v\n", res.Body)
			return nil, err
		}

		// Re-get the pointer Value of the slice since it may have moved,
		// then append it to the result set
		result = reflect.AppendSlice(result, itemsPtr.Elem())

		if res.PageInfo.Current_Page != res.PageInfo.Total_Pages {
			daURL = fmt.Sprintf("%s&page=%d", oldURL, res.PageInfo.Current_Page+1)
			if size > 0 {
				daURL += fmt.Sprintf("&per_page=%d", size)
			}
		} else {
			daURL = ""
		}
	}

	return result.Interface(), nil
}

func SprintfJSON(obj interface{}) string {
	res, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return ""
	}
	return string(res)
}

func (product *Product) GetFeatures() ([]*Feature, error) {
	items, err := product.GetAll(product.AhaClient.URL+"/api/v1/products/"+product.ID+"/features?fields=*",
		[]*Feature{})
	if err != nil {
		return nil, err
	}

	features := items.([]*Feature)

	for _, f := range features {
		f.AhaClient = product.AhaClient
		f.Product = product
	}

	return features, err
}

func (product *Product) GetFeaturesByReleaseName(name string) ([]*Feature, error) {
	rel, err := product.GetReleaseByName(name)
	if err != nil {
		return nil, fmt.Errorf("Can't find Aha release %q: %s", name, err)
	}
	if rel == nil {
		return nil, fmt.Errorf("Can't find Aha release %q", name)
	}

	items, err := product.GetAll(product.AhaClient.URL+"/api/v1/releases/"+rel.ID+"/features?fields=*",
		[]*Feature{})
	if err != nil {
		return nil, err
	}

	features := items.([]*Feature)

	for _, f := range features {
		f.AhaClient = product.AhaClient
		f.Product = product
	}

	return features, err
}

func (product *Product) GetFeatureByID(id string) (*Feature, error) {
	res, err := product.Aha("GET",
		product.AhaClient.URL+"/api/v1/features/"+id, "")
	if err != nil {
		return nil, err
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return nil, err
	}

	f.Feature.AhaClient = product.AhaClient
	f.Feature.Product = product

	return &f.Feature, err
}

func (product *Product) CreateFeature(title string, relName string, desc string) (*Feature, error) {
	rel, err := product.GetReleaseByName(relName)
	if err != nil {
		return nil, fmt.Errorf("Can't find Aha release %q: %s", relName, err)
	}
	if rel == nil {
		return nil, fmt.Errorf("Can't find Aha release %q", relName)
	}

	buf, _ := json.Marshal(title)
	title = string(buf)

	buf, _ = json.Marshal(desc)
	desc = string(buf)

	data := fmt.Sprintf(`{"feature":{"name":%s,`+
		`"description":%s,`+
		`"workflow_kind":"new",`+
		`"workflow_status":{"name":"%s"}}}`,
		title, desc, "Under consideration")

	res, err := product.Aha("POST",
		product.AhaClient.URL+"/api/v1/releases/"+rel.Reference_Num+
			"/features", data)
	if err != nil {
		return nil, fmt.Errorf("Error creating Aha feature: %s", err)
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return nil, err
	}

	f.Feature.AhaClient = product.AhaClient
	f.Feature.Product = product

	return &f.Feature, nil
}

func (product *Product) GetReleases() ([]*Release, error) {
	items, err := product.GetAll(
		product.AhaClient.URL+"/api/v1/products/"+product.ID+"/releases?fields=*",
		[]*Release{})
	if err != nil {
		return nil, err
	}

	rels := items.([]*Release)

	for _, r := range rels {
		r.AhaClient = product.AhaClient
		r.Product = product
	}

	return rels, err
}

func (product *Product) GetReleaseByID(id string) (*Release, error) {
	res, err := product.Aha("GET",
		product.AhaClient.URL+"/api/v1/releases/"+id, "")
	if err != nil {
		return nil, err
	}

	r := struct{ Release Release }{}
	err = json.Unmarshal([]byte(res.Body), &r)
	if err != nil {
		return nil, err
	}

	r.Release.AhaClient = product.AhaClient
	r.Release.Product = product

	return &r.Release, err
}

func (product *Product) GetReleaseByName(name string) (*Release, error) {
	rels, err := product.GetReleases()
	if err != nil {
		return nil, err
	}

	for _, r := range rels {
		if r.Name == name {
			r.AhaClient = product.AhaClient
			return r, nil
		}
	}

	return nil, nil
}

func (product *Product) CreateReleaseIfNeeded(name string, date string) error {
	ahaRelease, err := product.GetReleaseByName(name)
	if ahaRelease != nil || err != nil {
		return err
	}

	data := Release{
		// Product_ID:   product.Reference_Num,
		Name:         name,
		Release_Date: date,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = product.Aha("POST",
		product.AhaClient.URL+"/api/v1/products/"+product.ID+"/releases",
		string(body))

	return err
}

func (product *Product) CreateRelease(name string, date string) error {
	data := Release{
		// Product_ID:   product.Reference_Num,
		Name:         name,
		Release_Date: date,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = product.Aha("POST",
		product.AhaClient.URL+"/api/v1/products/"+product.ID+"/releases",
		string(body))

	return err
}

/*
func (product *Product) GetEpic(id string) (*Epic, error) {
	res, err := product.Aha("GET",
		product.AhaClient.URL+"/api/v1/epics/"+id, "")
	if err != nil {
		return nil, err
	}

	e := struct{ Epic Epic }{}
	err = json.Unmarshal([]byte(res.Body), &e)
	if err != nil {
		return nil, err
	}

	e.Epic.AhaClient = product.AhaClient
	e.Epic.Product = product

	return &e.Epic, err
}
*/

func (product *Product) GetCustomObjectRecord(id string) (*Custom_Object_Record, error) {
	// "{\"custom_object_record\":{\"id\":\"6880577663870072105\",\"product_id\":\"6424448796653305601\",\"key\":\"customer_2\",\"created_at\":\"2020-10-06T18:35:26.188Z\",\"updated_at\":\"2020-10-07T19:30:08.034Z\",\"custom_fields\":[{\"key\":\"customer_2_name\",\"name\":\"Name\",\"value\":\"Gartner - B8\",\"type\":\"string\"},{\"key\":\"customer_2_contact\",\"name\":\"Primary customer contact\",\"value\":\"Brett Walters\",\"type\":\"string\"},{\"key\":\"customer_2_phone\",\"name\":\"Phone number\",\"value\":\"\",\"type\":\"string\"},{\"key\":\"customer_2_email

	product.Aha("GET",
		product.AhaClient.URL+"/api/v1/products/"+product.ID+"/custom_objects/customer_2/records", "")
	product.Aha("GET",
		product.AhaClient.URL+"/api/v1/products/"+product.ID+"/custom_objects/public_cloud_customer_from_list/records", "")
	product.Aha("GET",
		product.AhaClient.URL+"/api/v1/products/"+product.ID+"/custom_objects/public_cloud_customer_from_list", "")
	product.Aha("GET",
		product.AhaClient.URL+"/api/v1/custom_object_records/public_cloud_customer_from_list", "")
	res, _ := product.Aha("GET",
		product.AhaClient.URL+"/api/v1/custom_object_records/6858965262405902740", "")
	if res.Body != "" {
		fmt.Printf("%s\n", res.Body)
	}

	res, err := product.Aha("GET",
		product.AhaClient.URL+"/api/v1/custom_object_records/"+id, "")
	if err != nil {
		return nil, err
	}

	record := struct{ Custom_Object_Record *Custom_Object_Record }{}
	err = json.Unmarshal([]byte(res.Body), &record)
	if err != nil {
		return nil, err
	}

	record.Custom_Object_Record.AhaClient = product.AhaClient
	return record.Custom_Object_Record, err
}

func (feature *Feature) Refresh() error {
	f, err := feature.Product.GetFeatureByID(feature.ID)
	if err != nil {
		return err
	}

	f.AhaClient = feature.AhaClient
	f.Product = feature.Product
	*feature = *f
	return nil
}

func (feature *Feature) Delete() (bool, error) {
	res, err := feature.Aha("DELETE",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, "")
	if err == nil {
		return true, nil
	}

	if res.StatusCode == 404 {
		return true, nil
	}

	return false, fmt.Errorf("Error deleting feature %q: %s",
		feature.Reference_Num, err)
}

func (feature *Feature) SetReleaseByID(id string) error {
	buf, _ := json.Marshal(id)
	id = string(buf)

	body := fmt.Sprintf(`{"feature":{"release":%s}}`, id)
	_, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error moving Feature %q to release %q",
			feature.Reference_Num, id, err)
	}
	return err
}

func (feature *Feature) SetReleaseByName(name string) error {
	rel, err := feature.Product.GetReleaseByName(name)
	if err != nil {
		return fmt.Errorf("Can't find Aha release %q: %s", name, err)
	}
	if rel == nil {
		return fmt.Errorf("Can't find Aha release %q", name)
	}

	return feature.SetReleaseByID(rel.Reference_Num)
}

func (feature *Feature) GetGitURL() (string, error) {
	for _, c := range feature.Custom_Fields {
		if c.Key == "ghe_url" && c.Type == "url" {
			url, ok := c.Value.(string)
			if ok {
				return url, nil
			}
			return "", fmt.Errorf("GHEURL isn't a url: %#v\n", c)
		}
	}
	return "", nil
}

func (feature *Feature) SetGitURL(url string) error {
	buf, _ := json.Marshal(url)
	url = string(buf)

	body := `{"feature":{"custom_fields":{"ghe_url":%s}}}`
	body = fmt.Sprintf(body, url)

	_, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error setting Aha feature(%s) GitURL: %s",
			feature.Reference_Num, url)
	}

	return err
}

func (feature *Feature) SetName(name string) error {
	buf, _ := json.Marshal(name)
	name = string(buf)

	body := fmt.Sprintf(`{"feature":{"name":%s}}`, name)

	_, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error updating Aha feature(%s) title: %s",
			feature.Reference_Num, name)
	}

	return err
}

func (feature *Feature) SetStatus(status string) error {
	buf, _ := json.Marshal(status)
	status = string(buf)

	body := fmt.Sprintf(`{"feature":{"workflow_status":{"name":%s}}}`, status)

	_, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error updating Aha feature(%s) status: %s",
			feature.Reference_Num, status)
	}

	return err
}

func (feature *Feature) SetDueDate(date string) error {
	buf, _ := json.Marshal(date)
	date = string(buf)

	body := fmt.Sprintf(`{"feature":{"due_date":%s}}`, date)

	_, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error updating Aha feature(%s) end_date: %s",
			feature.Reference_Num, date)
	}

	return err
}

func (feature *Feature) HasTag(tag string) bool {
	for _, t := range feature.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (feature *Feature) AddTag(tag string) error {
	for _, t := range feature.Tags {
		if t == tag {
			return nil
		}
	}

	feature.Tags = append(feature.Tags, tag)
	buf, _ := json.Marshal(feature.Tags)
	body := fmt.Sprintf(`{"feature":{"tags":%s}}`, string(buf))

	res, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		return fmt.Errorf("Error adding tag %q: %s", tag, err)
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return err
	}

	f.Feature.AhaClient = feature.AhaClient
	f.Feature.Product = feature.Product
	*feature = f.Feature
	return nil
}

func (feature *Feature) RemoveTag(tag string) error {
	found := false
	for i, t := range feature.Tags {
		if t == tag {
			found = true
			feature.Tags = append(feature.Tags[:i], feature.Tags[i+1:]...)
			break
		}
	}
	if !found {
		return nil
	}

	buf, _ := json.Marshal(feature.Tags)
	body := fmt.Sprintf(`{"feature":{"tags":%s}}`, string(buf))

	res, err := feature.Aha("PUT",
		feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		return fmt.Errorf("Error removing tag %q: %s", tag, err)
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return err
	}

	f.Feature.AhaClient = feature.AhaClient
	f.Feature.Product = feature.Product
	*feature = f.Feature
	return nil
}

func (feature *Feature) GetCustomField(name string) (string, bool) {
	for _, c := range feature.Custom_Fields {
		if c.Name == name {
			if c.Type == "url" {
				return c.Value.(string), true
			} else if c.Type == "note" {
				return strings.TrimSpace(c.Value.(string)), true
			}
			break
		}
	}
	return "", false
}
func (feature *Feature) HasCustomFieldValue(name, value string) bool {
	// fmt.Printf("%v -> hasCust: %s %s\n", feature.Reference_Num, name, value)
	for _, sd := range feature.Product.Screen_Definitions {
		if sd.Screenable_Type != "Feature" {
			continue
		}

		for _, cfd := range sd.Custom_Field_Definitions {
			// Allow Name to be the real name or the key
			if cfd.Name != name && cfd.Key != name {
				continue
			}
			// Use the 'key' from this point on
			key := cfd.Key

			if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::UrlField") {
				if cfd.API_Type == "url" {
					val := "..."
					return val == value
				} else {
					fmt.Printf("Unsupported cfd: %s looking for %s", cfd.API_Type, name)
					return false
				}
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::LinkMany") {
				if cfd.API_Type == "array" {
					// Find 'option' that has the key
					ID := ""
					for _, opt := range cfd.Options {
						if opt.Label == value {
							ID = opt.ID
							break
						}
					}
					if ID == "" {
						fmt.Printf("1- Can't find %s/%q as a valid option\n", name, value)
						return false
					}

					// Get existing values
					for _, col := range feature.Custom_Object_Links {
						if col.Key == key {
							for _, rec := range col.Record_IDs {
								if rec == ID {
									// Already there, so just exit
									return true
								}
							}
							break
						}
					}
					return false
				} else {
					fmt.Printf("Unsupported cfd-link: %s", cfd.API_Type)
					return false
				}
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::SelectConstant") {
				if cfd.API_Type == "string" {
					for _, cf := range feature.Custom_Fields {
						if cf.Key == key {
							return cf.Value == value
						}
					}
					return false
				}
				break
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::SelectMultipleConstant") {
				if cfd.API_Type == "array" {
					for _, cf := range feature.Custom_Fields {
						if cf.Name == name {
							if cf.Value != nil {
								values, ok := cf.Value.([]interface{})
								if !ok {
									fmt.Printf("Can't convert '%#v') to []interface{}string", cf.Value)
									return false
								}
								for _, val := range values {
									v, ok := val.(string)
									if !ok {
										fmt.Printf("Can't convert '%#v') to string", v)
										return false
									}
									if val == value {
										// Already there, so just exit
										return true
									}
								}
							}
							break
						}
					}
					return false
				}
				break

			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::NoteField") {
				if cfd.API_Type == "note" {
					for _, cf := range feature.Custom_Fields {
						if cf.Key == key {
							return cf.Value == value
						}
					}
					return false
				}
				break
			}
		}
	}
	return false
}

func (feature *Feature) AddCustomFieldValue(name, value string) error {
	fmt.Printf("Feature.addfield %q.%q - %q\n", feature.Reference_Num, name, value)
	/*
	   "Custom_Object_Links": [
	     {
	       "Key": "public_cloud_customer_from_list",
	       "Name": "Public Cloud Customer/Program",
	       "Record_Type": "CustomObjectRecord",
	       "Record_IDs": [
	         "6880577663870072105"
	       ]
	     }
	   ],
	*/

	body := ""

	for _, sd := range feature.Product.Screen_Definitions {
		if sd.Screenable_Type != "Feature" {
			continue
		}

		for _, cfd := range sd.Custom_Field_Definitions {
			// Allow Name to be the real name or the key
			if cfd.Name != name && cfd.Key != name {
				continue
			}
			// Use the 'key' from this point on
			key := cfd.Key

			if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::UrlField") {
				if cfd.API_Type == "url" {
					body = `{"feature":{"custom_fields":{"%s":"%s"}}}`
					body = fmt.Sprintf(body, key, value)
				} else {
					return fmt.Errorf("Unsupported cfd: %s", cfd.API_Type)
				}
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::LinkMany") {
				if cfd.API_Type == "array" {
					// Find 'option' that has the key
					ID := ""
					for _, opt := range cfd.Options {
						if opt.Label == value {
							ID = opt.ID
							break
						}
					}
					if ID == "" {
						return fmt.Errorf("3- Can't find %s/%q as a valid option\n", name, value)
					}

					values := []string{}

					// Get existing values
					for _, col := range feature.Custom_Object_Links {
						if col.Key == key {
							for _, rec := range col.Record_IDs {
								if rec == ID {
									// Already there, so just exit
									return nil
								}
								values = append(values, rec)
							}
							break
						}
					}

					values = append(values, ID)

					req := struct {
						Feature struct {
							Custom_Object_Links map[string][]string `json:"custom_object_links"`
						} `json:"feature"`
					}{}
					req.Feature.Custom_Object_Links = map[string][]string{}
					req.Feature.Custom_Object_Links[key] = values
					buf, _ := json.MarshalIndent(req, "", "  ")
					body = string(buf)
				} else {
					return fmt.Errorf("Unsupported cfd-link: %s", cfd.API_Type)
				}
				break
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::SelectConstant") {
				if cfd.API_Type == "string" {
					// Find 'option' that has the key
					ID := ""
					for _, opt := range cfd.Options {
						if opt.Label == value {
							ID = opt.ID
							break
						}
					}
					if ID == "" {
						return fmt.Errorf("4- Can't find %s/%q as a valid option\n", name, value)
					}

					// Get existing values
					for _, col := range feature.Custom_Object_Links {
						if col.Key == key {
							for _, rec := range col.Record_IDs {
								if rec == ID {
									// Already there, so just exit
									return nil
								}
							}
							break
						}
					}

					req := struct {
						Feature struct {
							Custom_Fields map[string]string `json:"custom_fields"`
						} `json:"feature"`
					}{}
					req.Feature.Custom_Fields = map[string]string{}
					req.Feature.Custom_Fields[key] = ID
					buf, _ := json.MarshalIndent(req, "", "  ")
					body = string(buf)
				}
				break
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::SelectMultipleConstant") {
				if cfd.API_Type == "array" {
					values := []string{}

					// Get existing values
					for _, cf := range feature.Custom_Fields {
						if cf.Name == name {
							if cf.Value != nil {
								vals, ok := cf.Value.([]interface{})
								if !ok {
									return fmt.Errorf("Can't convert '%#v') to []interface{}string", cf.Value)
								}
								for _, val := range vals {
									v, ok := val.(string)
									if !ok {
										return fmt.Errorf("Can't convert '%#v') to string", v)
									}
									if v == value {
										// Already there, so just exit
										fmt.Printf("Already there\n")
										return nil
									} else {
										values = append(values, v)
									}
								}
							}
							break
						}
					}

					values = append(values, value)

					req := struct {
						Feature struct {
							Custom_Fields map[string][]string `json:"custom_fields"`
						} `json:"feature"`
					}{}
					req.Feature.Custom_Fields = map[string][]string{}
					req.Feature.Custom_Fields[key] = values
					fmt.Printf("add Value: %#v\n", values)
					buf, _ := json.MarshalIndent(req, "", "  ")
					body = string(buf)
				}
				break
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::NoteField") {
				if cfd.API_Type == "note" {
					body = `{"feature":{"custom_fields":{"%s":"%s"}}}`
					body = fmt.Sprintf(body, key, value)
				} else {
					return fmt.Errorf("Unsupported cfd: %s", cfd.API_Type)
				}
				break
			}
		}
	}

	if body != "" {
		res, err := feature.Aha("PUT",
			feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
		if err != nil {
			return fmt.Errorf("Error setting feature(%s) field: %q to %q. %s",
				feature.Reference_Num, name, value, res.StatusCode, err.Error())
		}
		f := struct{ Feature Feature }{}
		err = json.Unmarshal([]byte(res.Body), &f)
		if err != nil {
			return err
		}

		f.Feature.AhaClient = feature.AhaClient
		f.Feature.Product = feature.Product
		*feature = f.Feature
	} else {
		return fmt.Errorf("Couldn't find name %q", name)
	}

	return nil
}

func (feature *Feature) RemoveCustomFieldValue(name, value string) error {
	// fmt.Printf("Feature.removefield %q.%q - %q\n", feature.Reference_Num, name, value)
	body := ""

	for _, sd := range feature.Product.Screen_Definitions {
		if sd.Screenable_Type != "Feature" {
			continue
		}

		for _, cfd := range sd.Custom_Field_Definitions {
			// Allow Name to be the real name or the key
			if cfd.Name != name && cfd.Key != name {
				continue
			}
			// Use the 'key' from this point on
			key := cfd.Key

			if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::UrlField") {
				if cfd.API_Type == "url" {
					body = `{"feature":{"custom_fields":{"` + key + `":""}}}`
				} else {
					return fmt.Errorf("Unsupported cfd: %s", cfd.API_Type)
				}
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::LinkMany") {
				if cfd.API_Type == "array" {
					// Find 'option' that has the key
					ID := ""
					for _, opt := range cfd.Options {
						if opt.Label == value {
							ID = opt.ID
							break
						}
					}
					if ID == "" {
						return fmt.Errorf("5- Can't find %s/%q as a valid option\n", name, value)
					}

					values := []string{}
					found := false

					// Get existing values
					for _, col := range feature.Custom_Object_Links {
						if col.Key == key {
							for _, rec := range col.Record_IDs {
								if rec == ID {
									found = true
									// Already there, so just exit
								} else {
									values = append(values, rec)
								}
							}
							break
						}
					}
					// Not in there so just return
					if !found {
						fmt.Printf("  Not there - 1\n")
						return nil
					}

					// Weird, but to erase all pass in an array with an
					// empty string
					if len(values) == 0 {
						values = []string{""}
					}

					req := struct {
						Feature struct {
							Custom_Object_Links map[string][]string `json:"custom_object_links"`
						} `json:"feature"`
					}{}
					req.Feature.Custom_Object_Links = map[string][]string{}
					req.Feature.Custom_Object_Links[key] = values
					buf, _ := json.MarshalIndent(req, "", "  ")
					body = string(buf)
				} else {
					return fmt.Errorf("Unsupported cfd-link: %s", cfd.API_Type)
				}
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::SelectConstant") {
				if cfd.API_Type == "string" {
					req := struct {
						Feature struct {
							Custom_Fields map[string]string `json:"custom_fields"`
						} `json:"feature"`
					}{}
					req.Feature.Custom_Fields = map[string]string{}
					req.Feature.Custom_Fields[key] = ""
					buf, _ := json.MarshalIndent(req, "", "  ")
					body = string(buf)
				}
				break
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::SelectMultipleConstant") {
				if cfd.API_Type == "array" {
					values := []string{}
					found := false

					// Get existing values
					for _, cf := range feature.Custom_Fields {
						if cf.Name == name {
							if cf.Value != nil {
								vals, ok := cf.Value.([]interface{})
								if !ok {
									fmt.Printf("cf: %#v\n", cf)
									return fmt.Errorf("Can't convert '%#v') to []interface{}string", cf.Value)
								}
								for _, val := range vals {
									v, ok := val.(string)
									if !ok {
										fmt.Printf("v: %#v\n", v)
										return fmt.Errorf("Can't convert '%#v') to string", v)
									}
									if v == value {
										// Already there, so just exit
										found = true
									} else {
										values = append(values, v)
									}
								}
							}
							break
						}
					}

					// Not in there so just return
					if !found {
						fmt.Printf("feature: %s\n", SprintfJSON(feature))
						fmt.Printf("  Not there - 2\n")
						return nil
					}

					// Weird, but to erase all pass in an array with an
					// empty string
					if len(values) == 0 {
						values = nil
					}

					req := struct {
						Feature struct {
							Custom_Fields map[string][]string `json:"custom_fields"`
						} `json:"feature"`
					}{}
					req.Feature.Custom_Fields = map[string][]string{}
					req.Feature.Custom_Fields[key] = values
					fmt.Printf("Value: %#v\n", values)
					buf, _ := json.MarshalIndent(req, "", "  ")
					body = string(buf)
				}
				break
			} else if strings.HasPrefix(cfd.Type, "CustomFieldDefinitions::NoteField") {
				if cfd.API_Type == "note" {
					body = `{"feature":{"custom_fields":{"` + key + `":"-"}}}`
					/*
						req := struct {
							Feature struct {
								Custom_Fields map[string]string `json:"custom_fields"`
							} `json:"feature"`
						}{}
						req.Feature.Custom_Fields = map[string]string{}
						req.Feature.Custom_Fields[key] = " "
						buf, _ := json.MarshalIndent(req, "", "  ")
						body = string(buf)
					*/
				}
				break
			}
			break
		}
	}

	if body != "" {
		// fmt.Printf("body: %s\n", body)
		res, err := feature.Aha("PUT",
			feature.AhaClient.URL+"/api/v1/features/"+feature.Reference_Num, body)
		if err != nil {
			return fmt.Errorf("Error setting feature(%s) field: %q to %q. %s",
				feature.Reference_Num, name, value, res.StatusCode, err.Error())
		}
		f := struct{ Feature Feature }{}
		err = json.Unmarshal([]byte(res.Body), &f)
		if err != nil {
			return err
		}

		f.Feature.AhaClient = feature.AhaClient
		f.Feature.Product = feature.Product
		*feature = f.Feature
	} else {
		return fmt.Errorf("Couldn't find name %q", name)
	}

	return nil
}

// Global funcs

func (ac *AhaClient) GetProducts() ([]*Product, error) {
	items, err := ac.GetAll(ac.URL+"/api/v1/products?fields=*", []*Product{})
	if err != nil {
		return nil, err
	}
	products := items.([]*Product)
	for _, p := range products {
		p.AhaClient = ac
	}
	return products, err
}

func (ac *AhaClient) GetProduct(id string) (*Product, error) {
	res, err := ac.Aha("GET", ac.URL+"/api/v1/products/"+id, "")
	if err != nil {
		return nil, err
	}

	p := struct{ Product Product }{}
	err = json.Unmarshal([]byte(res.Body), &p)
	if err != nil {
		return nil, err
	}
	p.Product.AhaClient = ac

	return &p.Product, nil
}

func (ac *AhaClient) DeleteFeature(id string) (bool, error) {
	res, err := ac.Aha("DELETE", ac.URL+"/api/v1/features/"+id, "")
	if err == nil {
		return true, nil
	}

	if res.StatusCode == 404 {
		return true, nil
	}

	return false, fmt.Errorf("Error deleting feature %q: %s", id, err)
}
