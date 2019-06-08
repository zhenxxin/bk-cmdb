/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package remote

import (
	"context"
	"net/http"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/util"
	"configcenter/src/storage/dal"
)

// Wrapper Interface for automatic processing of encapsulated transactions
// f parameter http.header, the handler must be accepted and processed. Subsequent passthrough to call subfunctions and APIs
func (c *Mongo) Wrapper(ctx context.Context, opt dal.TxnWrapperOption, f func(header http.Header) error) error {

	rid := util.GetHTTPCCRequestID(opt.Header)
	txn, err := c.Start(ctx)
	if err != nil {
		blog.Errorf("wrapper stranscation start error. err:%s, rid:%s", err.Error(), rid)
		return opt.CCErr.Errorf(common.CCErrCommStartTranscationFailed, err.Error())
	}
	header := txn.TxnInfo().IntoHeader(opt.Header)
	newCtx := util.GetDBContext(context.Background(), header)
	err = f(header)
	if err != nil {
		// Abort error. mongodb session can rollback
		txnErr := txn.Abort(newCtx)
		if txnErr != nil {
			blog.Errorf("wrapper stranscation start error. err:%s, txnErr:%s, rid:%s", err.Error(), txnErr.Error(), rid)
		}
		return err
	}

	err = txn.Commit(newCtx)
	if err != nil {
		blog.Errorf("wrapper stranscation commit error. err:%s, rid:%s", err.Error(), rid)
		return opt.CCErr.Errorf(common.CCErrCommCommitTranscationFailed, err.Error())
	}
	return nil

}
