/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package telemetry

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/devtron-labs/devtron/util"
	"github.com/patrickmn/go-cache"
	"github.com/posthog/posthog-go"
	"go.uber.org/zap"
)

type PosthogClient struct {
	Client posthog.Client
	cache  *cache.Cache
}

var (
	PosthogApiKey        string = ""
	PosthogEndpoint      string = "https://app.posthog.com"
	SummaryCronExpr      string = "0 0 * * *" // Run once a day, midnight
	HeartbeatCronExpr    string = "0 0/6 * * *"
	CacheExpiry          int    = 1440
	PosthogEncodedApiKey string = ""
	IsOptOut             bool   = false
)

const (
	TelemetryApiKeyEndpoint   string = "aHR0cHM6Ly90ZWxlbWV0cnkuZGV2dHJvbi5haS9kZXZ0cm9uL3RlbGVtZXRyeS9wb3N0aG9nSW5mbw=="
	TelemetryOptOutApiBaseUrl string = "aHR0cHM6Ly90ZWxlbWV0cnkuZGV2dHJvbi5haS9kZXZ0cm9uL3RlbGVtZXRyeS9vcHQtb3V0"
	ResponseApiKey            string = "PosthogApiKey"
	ResponseUrlKey            string = "PosthogEndpoint"
)

func NewPosthogClient(logger *zap.SugaredLogger) (*PosthogClient, error) {
	if PosthogApiKey == "" {
		encodedApiKey, apiKey, posthogUrl, err := getPosthogApiKey(TelemetryApiKeyEndpoint, logger)
		if err != nil {
			logger.Errorw("exception caught while getting api key", "err", err)
		} else {
			PosthogApiKey = apiKey
			PosthogEncodedApiKey = encodedApiKey
			PosthogEndpoint = posthogUrl
		}
	}

	client, err := posthog.NewWithConfig(PosthogApiKey, posthog.Config{Endpoint: PosthogEndpoint})
	//defer client.Close()
	if err != nil {
		logger.Errorw("exception caught while creating posthog client", "err", err)
	}
	d := time.Duration(CacheExpiry)
	c := cache.New(d*time.Minute, d*time.Minute)
	pgClient := &PosthogClient{
		Client: client,
		cache:  c,
	}
	return pgClient, nil
}

func getPosthogApiKey(encodedPosthogApiKeyUrl string, logger *zap.SugaredLogger) (string, string, string, error) {
	decodedPosthogApiKeyUrl, err := base64.StdEncoding.DecodeString(encodedPosthogApiKeyUrl)
	if err != nil {
		logger.Errorw("error fetching posthog api key, decode error", "err", err)
		return "", "", "", err
	}
	apiKeyUrl := string(decodedPosthogApiKeyUrl)
	response, err := util.HttpRequest(apiKeyUrl)
	if err != nil {
		logger.Errorw("error fetching posthog api key, http call", "err", err)
		return "", "", "", err
	}
	posthogInfo := response["result"]
	posthogInfoByte, err := json.Marshal(posthogInfo)
	if err != nil {
		logger.Errorw("error in fetched posthog info, http call", "err", err)
		return "", "", "", err
	}
	var datamap map[string]string
	if err := json.Unmarshal(posthogInfoByte, &datamap); err != nil {
		logger.Errorw("error while unmarshal data", "err", err)
		return "", "", "", err
	}
	encodedApiKey := datamap[ResponseApiKey]
	posthogUrl := datamap[ResponseUrlKey]
	apiKey, err := base64.StdEncoding.DecodeString(encodedApiKey)
	if err != nil {
		return "", "", "", err
	}
	return encodedApiKey, string(apiKey), posthogUrl, err
}
