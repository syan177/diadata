// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
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

package types

import (
	"github.com/diadata-org/diadata/pkg/dia/helpers/substrate-helper/gsrpc/scale"
)

// MigrationCompute is an enum describing how a migration was computed.
type MigrationCompute struct {
	IsSigned bool
	IsAuto   bool
}

func (m *MigrationCompute) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsSigned = true
	case 1:
		m.IsAuto = true
	}

	return nil
}

func (m MigrationCompute) Encode(encoder scale.Encoder) error {
	switch {
	case m.IsSigned:
		return encoder.PushByte(0)
	case m.IsAuto:
		return encoder.PushByte(1)
	}

	return nil
}
