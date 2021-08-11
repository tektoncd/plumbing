// Copyright © 2021 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

const url = "https://raw.githubusercontent.com/tektoncd/hub/main/config.yaml"

// Checks if the added category annotation is
// from predefined category list
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// Loads the Env variable
func LoadEnv() string {
	viper.SetConfigFile(".env")

	// Find and read the config file
	err := viper.ReadInConfig()
	if err != nil {
		return url
	}

	value, ok := viper.Get("CONFIG_FILE_URL").(string)
	if !ok {
		log.Fatalf("Invalid type assertion")
	}
	return value
}

// Gets the list of predefined category list
func GetCategories() ([]string, error) {
	// viper package read .env
	envVal := LoadEnv()

	resp, err := http.Get(envVal)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(resp.Status)
	}

	categoryData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Viper unmarshals data from config file into Data Object
	var data Data
	viper.SetConfigType("yaml")

	if err := viper.ReadConfig(bytes.NewBuffer(categoryData)); err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %v", err)
	}
	if err := viper.Unmarshal(&data); err != nil {

		return nil, fmt.Errorf("failed to unmarshal config data: %v", err)
	}

	categoriesList := []string{}
	for i := range data.Categories {
		categoriesList = append(categoriesList, data.Categories[i].Name)
	}

	return categoriesList, nil
}
