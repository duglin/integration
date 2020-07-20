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
	// fmt.Printf("%s %s\n", method, url)
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
	return items.([]*Feature), err
}

func (feature *Feature) Refresh() error {
	res, err := Aha("GET", AhaURL+"/api/v1/features/"+feature.ID, "")
	if err != nil {
		return err
	}

	f := struct{ Feature Feature }{}
	err = json.Unmarshal([]byte(res.Body), &f)
	if err != nil {
		return err
	}

	*feature = f.Feature
	return nil
}

func (feature *Feature) GetGitURL() (string, error) {
	for _, i := range feature.Integration_Fields {
		if i.Service_Name != "github_enterprise" || i.Name != "url" {
			continue
		}
		return i.Value, nil
	}
	return "", nil
}

// Global funcs

func GetProducts() ([]*Product, error) {
	items, err := GetAll(AhaURL+"/api/v1/products?fields=*", []*Product{})
	if err != nil {
		return nil, err
	}
	return items.([]*Product), err
}

func GetProduct(name string) (*Product, error) {
	products, err := GetProducts()
	if err != nil {
		return nil, err
	}
	for _, p := range products {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, nil
}
