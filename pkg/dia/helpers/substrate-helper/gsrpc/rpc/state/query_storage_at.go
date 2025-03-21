// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2022 Snowfork
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

package state

import (
	"github.com/diadata-org/diadata/pkg/dia/helpers/substrate-helper/gsrpc/client"
	"github.com/diadata-org/diadata/pkg/dia/helpers/substrate-helper/gsrpc/types"
)

// QueryStorageAt performs a low-level storage query
func (s *state) QueryStorageAt(keys []types.StorageKey, block types.Hash) ([]types.StorageChangeSet, error) {
	return s.queryStorageAt(keys, &block)
}

// QueryStorageAtLatest performs a low-level storage query
func (s *state) QueryStorageAtLatest(keys []types.StorageKey) ([]types.StorageChangeSet, error) {
	return s.queryStorageAt(keys, nil)
}

func (s *state) queryStorageAt(keys []types.StorageKey, block *types.Hash) ([]types.StorageChangeSet, error) {
	hexKeys := make([]string, len(keys))
	for i, key := range keys {
		hexKeys[i] = key.Hex()
	}

	var res []types.StorageChangeSet
	err := client.CallWithBlockHash(s.client, &res, "state_queryStorageAt", block, hexKeys)
	if err != nil {
		return nil, err
	}

	return res, nil
}
