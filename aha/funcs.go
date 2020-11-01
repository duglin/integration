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

var AhaToken = ""
var AhaURL = ""
var AhaSecret = "" // used to verify events are from Aha

type AhaResponse struct {
	StatusCode int
	Body       string
	PageInfo   Pagination
}

func Aha(method string, url string, body string) (*AhaResponse, error) {
	defer fmt.Printf("\n")
	fmt.Printf("%s %s", method, url)
	ahaResponse := AhaResponse{}

	buf := []byte{}
	if body != "" {
		buf = []byte(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+AhaToken)
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
	fmt.Printf(" - %d", res.StatusCode)

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

func GetAll(daURL string, daItem interface{}) (interface{}, error) {
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
		if res, err = Aha("GET", daURL, ""); err != nil {
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

func (product *Product) GetFeatures() ([]*Feature, error) {
	fmt.Printf("getting features\n")
	items, err := GetAll(AhaURL+"/api/v1/products/"+product.ID+"/features?fields=*",
		[]*Feature{})
	fmt.Printf("done\n")
	if err != nil {
		return nil, err
	}

	features := items.([]*Feature)

	for _, f := range features {
		f.Product = product
	}

	return features, err
}

func (product *Product) GetFeatureByID(id string) (*Feature, error) {
	fmt.Printf("getting features: %s\n", id)

	res, err := Aha("GET", AhaURL+"/api/v1/features/"+id, "")
	if err != nil {
		return nil, err
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return nil, err
	}

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

	data := fmt.Sprintf(`{"feature":{"name":"%s",`+
		`"description":"%s",`+
		`"workflow_kind":"new",`+
		`"workflow_status":{"name":"%s"}}}`,
		title, desc, "Under consideration")

	fmt.Printf("Data: %s\n", data)

	res, err := Aha("POST", AhaURL+"/api/v1/releases/"+rel.Reference_Num+
		"/features", data)
	if err != nil {
		return nil, fmt.Errorf("Error creating Aha feature: %s", err)
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return nil, err
	}

	f.Feature.Product = product

	return &f.Feature, nil
}

func (product *Product) GetReleases() ([]*Release, error) {
	items, err := GetAll(AhaURL+"/api/v1/products/"+product.ID+"/releases?fields=*",
		[]*Release{})
	if err != nil {
		return nil, err
	}

	rels := items.([]*Release)

	for _, r := range rels {
		r.Product = product
	}

	return rels, err
}

func (product *Product) GetReleaseByID(id string) (*Release, error) {
	fmt.Printf("getting release: %s\n", id)

	res, err := Aha("GET", AhaURL+"/api/v1/releases/"+id, "")
	if err != nil {
		return nil, err
	}

	r := struct{ Release Release }{}
	err = json.Unmarshal([]byte(res.Body), &r)
	if err != nil {
		return nil, err
	}

	r.Release.Product = product

	return &r.Release, err
}

func (product *Product) GetReleaseByName(name string) (*Release, error) {
	fmt.Printf("getting release: %s\n", name)

	rels, err := product.GetReleases()
	if err != nil {
		return nil, err
	}

	for _, r := range rels {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, nil
}

func (product *Product) CreateRelease(name string, date string) error {
	data := Release{
		// Product_ID:   product.Reference_Num,
		Name:         name,
		Release_Date: date,
	}

	fmt.Printf("Data: %#v\n", data)

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = Aha("POST", AhaURL+"/api/v1/products/"+product.ID+"/releases",
		string(body))

	return err
}

/*
func (product *Product) GetEpic(id string) (*Epic, error) {
	fmt.Printf("getting epic: %s\n", id)

	res, err := Aha("GET", AhaURL+"/api/v1/epics/"+id, "")
	if err != nil {
		return nil, err
	}

	e := struct{ Epic Epic }{}
	err = json.Unmarshal([]byte(res.Body), &e)
	if err != nil {
		return nil, err
	}

	e.Epic.Product = product

	return &e.Epic, err
}
*/

func (product *Product) GetCustomObjectRecord(id string) (*Custom_Object_Record, error) {
	fmt.Printf("getting custom objects\n")

	// "{\"custom_object_record\":{\"id\":\"6880577663870072105\",\"product_id\":\"6424448796653305601\",\"key\":\"customer_2\",\"created_at\":\"2020-10-06T18:35:26.188Z\",\"updated_at\":\"2020-10-07T19:30:08.034Z\",\"custom_fields\":[{\"key\":\"customer_2_name\",\"name\":\"Name\",\"value\":\"Gartner - B8\",\"type\":\"string\"},{\"key\":\"customer_2_contact\",\"name\":\"Primary customer contact\",\"value\":\"Brett Walters\",\"type\":\"string\"},{\"key\":\"customer_2_phone\",\"name\":\"Phone number\",\"value\":\"\",\"type\":\"string\"},{\"key\":\"customer_2_email

	Aha("GET", AhaURL+"/api/v1/products/"+product.ID+"/custom_objects/customer_2/records", "")
	Aha("GET", AhaURL+"/api/v1/products/"+product.ID+"/custom_objects/public_cloud_customer_from_list/records", "")
	Aha("GET", AhaURL+"/api/v1/products/"+product.ID+"/custom_objects/public_cloud_customer_from_list", "")
	Aha("GET", AhaURL+"/api/v1/custom_object_records/public_cloud_customer_from_list", "")
	res, _ := Aha("GET", AhaURL+"/api/v1/custom_object_records/6858965262405902740", "")
	if res.Body != "" {
		fmt.Printf("%s\n", res.Body)
	}

	res, err := Aha("GET", AhaURL+"/api/v1/custom_object_records/"+id, "")
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n\n%s\n\n", res.Body)
	record := struct{ Custom_Object_Record *Custom_Object_Record }{}
	err = json.Unmarshal([]byte(res.Body), &record)
	if err != nil {
		return nil, err
	}

	return record.Custom_Object_Record, err
}

func (feature *Feature) Refresh() error {
	f, err := feature.Product.GetFeatureByID(feature.ID)
	if err != nil {
		return err
	}

	f.Product = feature.Product
	*feature = *f
	return nil
}

func (feature *Feature) Delete() (bool, error) {
	res, err := Aha("DELETE", AhaURL+"/api/v1/features/"+feature.Reference_Num, "")
	if err == nil {
		return true, nil
	}

	if res.StatusCode == 404 {
		return true, nil
	}

	return false, fmt.Errorf("Error deleting feature %q: %s", feature.Reference_Num, err)
}

func (feature *Feature) SetReleaseByID(id string) error {
	body := fmt.Sprintf(`{"feature":{"release":"%s"}}`, id)
	_, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
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
	body := `{"feature":{"custom_fields":{"ghe_url":"%s"}}}`
	body = fmt.Sprintf(body, url)

	res, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error setting Aha feature(%s) GitURL: %s",
			feature.Reference_Num, url)
	}
	fmt.Printf("res: %s\n", res.Body)

	return err
}

func (feature *Feature) SetName(name string) error {
	body := fmt.Sprintf(`{"feature":{"name":"%s"}}`, name)

	_, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error updating Aha feature(%s) title: %s",
			feature.Reference_Num, name)
	}

	return err
}

func (feature *Feature) SetStatus(status string) error {
	body := fmt.Sprintf(`{"feature":{"workflow_status":{"name":"%s"}}}`, status)

	_, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		err = fmt.Errorf("Error updating Aha feature(%s) status: %s",
			feature.Reference_Num, status)
	}

	return err
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

	res, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		return fmt.Errorf("Error adding tag %q: %s", tag, err)
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return err
	}

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

	res, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
	if err != nil {
		return fmt.Errorf("Error removing tag %q: %s", tag, err)
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return err
	}

	f.Feature.Product = feature.Product
	*feature = f.Feature
	return nil
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
		fmt.Printf("Got feature list\n")

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
						return fmt.Errorf("Can't find %q as a valid option", value)
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
						return fmt.Errorf("Can't find %q as a valid option", value)
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
			}
		}
	}

	if body != "" {
		res, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
		if err != nil {
			return fmt.Errorf("Error setting feature(%s) field: %q to %q. %s",
				feature.Reference_Num, name, value, res.StatusCode, err.Error())
		}
		f := struct{ Feature Feature }{}
		err = json.Unmarshal([]byte(res.Body), &f)
		if err != nil {
			return err
		}

		f.Feature.Product = feature.Product
		*feature = f.Feature
	} else {
		return fmt.Errorf("Couldn't find name %q", name)
	}

	return nil
}

func (feature *Feature) RemoveCustomFieldValue(name, value string) error {
	fmt.Printf("Feature.removefield %q.%q - %q\n", feature.Reference_Num, name, value)
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
						return fmt.Errorf("Can't find %q as a valid option", value)
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
						fmt.Printf("  Not there\n")
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
			}
			break
		}
	}

	if body != "" {
		// fmt.Printf("body: %s\n", body)
		res, err := Aha("PUT", AhaURL+"/api/v1/features/"+feature.Reference_Num, body)
		if err != nil {
			return fmt.Errorf("Error setting feature(%s) field: %q to %q. %s",
				feature.Reference_Num, name, value, res.StatusCode, err.Error())
		}
		f := struct{ Feature Feature }{}
		err = json.Unmarshal([]byte(res.Body), &f)
		if err != nil {
			return err
		}

		f.Feature.Product = feature.Product
		*feature = f.Feature
	} else {
		return fmt.Errorf("Couldn't find name %q", name)
	}

	return nil
}

// Global funcs

func GetProducts() ([]*Product, error) {
	items, err := GetAll(AhaURL+"/api/v1/products?fields=*", []*Product{})
	if err != nil {
		return nil, err
	}
	return items.([]*Product), err
}

func GetProduct(id string) (*Product, error) {
	res, err := Aha("GET", AhaURL+"/api/v1/products/"+id, "")
	if err != nil {
		return nil, err
	}

	p := struct{ Product Product }{}
	err = json.Unmarshal([]byte(res.Body), &p)
	if err != nil {
		return nil, err
	}

	return &p.Product, nil
}

func DeleteFeature(id string) (bool, error) {
	res, err := Aha("DELETE", AhaURL+"/api/v1/features/"+id, "")
	if err == nil {
		return true, nil
	}

	if res.StatusCode == 404 {
		return true, nil
	}

	return false, fmt.Errorf("Error deleting feature %q: %s", id, err)
}
