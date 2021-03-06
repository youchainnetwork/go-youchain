// Copyright 2020 YOUCHAIN FOUNDATION LTD.
// This file is part of the go-youchain library.
//
// The go-youchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-youchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-youchain library. If not, see <http://www.gnu.org/licenses/>.

package params

// YouVersion is an integer that identifies a version of the consensus protocol.
type YouVersion uint64

const (
	// YouV1 is a baseline version of the YOUChain consensus protocol.
	YouV1 = YouVersion(1)
	// YouV2 adds the parameter `AllowedFutureBlockTime`
	YouV2 = YouVersion(2)
	// YouV3 adjust `SubsidyThreshold` and `UpgradeThreshold`, and tune txs-broadcast strategy
	YouV3 = YouVersion(3)
	// YouV4 fixes a bug on gas used for a failed staking-transaction.
	YouV4 = YouVersion(4)
)

// !!! ********************* !!!
// !!! *** Please update YouCurrentVersion when adding new protocol versions *** !!!
// !!! ********************* !!!

// YouCurrentVersion is the latest version and should be used
// when a specific version is not provided.
const YouCurrentVersion = YouV4
