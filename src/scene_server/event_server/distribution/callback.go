/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package distribution

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/redis.v5"

	"configcenter/src/common/blog"
	"configcenter/src/common/http/httpclient"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/event_server/types"
)

func (dh *DistHandler) SendCallback(receiver *metadata.Subscription, event string) (err error) {
	increaseTotal(dh.cache, receiver.SubscriptionID)

	body := bytes.NewBufferString(event)
	req, err := http.NewRequest("POST", receiver.CallbackURL, body)
	if err != nil {
		increaseFailure(dh.cache, receiver.SubscriptionID)
		return fmt.Errorf("event distribute fail, build request error: %v, data=[%s]", err, event)
	}
	var duration time.Duration
	if receiver.TimeOutSeconds == 0 {
		duration = timeout
	} else {
		duration = receiver.GetTimeout()
	}
	resp, err := httpCli.DoWithTimeout(duration, req)
	if err != nil {
		increaseFailure(dh.cache, receiver.SubscriptionID)
		return fmt.Errorf("event distribute fail, send request error: %v, data=[%s]", err, event)
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		increaseFailure(dh.cache, receiver.SubscriptionID)
		return fmt.Errorf("event distribute fail, read response error: %v, data=[%s]", err, event)
	}
	if receiver.ConfirmMode == metadata.ConfirmModeHTTPStatus {
		if strconv.Itoa(resp.StatusCode) != receiver.ConfirmPattern {
			increaseFailure(dh.cache, receiver.SubscriptionID)
			return fmt.Errorf("event distribute fail, received response %s, data=[%s]", respData, event)
		}
	} else if receiver.ConfirmMode == metadata.ConfirmModeRegular {
		pattern, err := regexp.Compile(receiver.ConfirmPattern)
		if err != nil {
			return fmt.Errorf("event distribute fail, build regexp error: %v", err)
		}
		if !pattern.Match(respData) {
			increaseFailure(dh.cache, receiver.SubscriptionID)
			return fmt.Errorf("event distribute fail, received response %s, data=[%s]", respData, event)
		}
		return nil
	}

	return
}

var httpCli = httpclient.NewHttpClient()

func increaseTotal(cache *redis.Client, subscriptionID int64) error {
	return increase(cache, subscriptionID, "total")
}

func increaseFailure(cache *redis.Client, subscriptionID int64) error {
	return increase(cache, subscriptionID, "failue")
}

func increase(cache *redis.Client, subscriptionID int64, key string) error {
	err := cache.HIncrBy(types.EventCacheDistCallBackCountPrefix+strconv.FormatInt(subscriptionID, 10), key, 1).Err()
	if err != nil {
		blog.V(3).Infof("increaseFailure %s", err.Error())
	}
	return err
}
