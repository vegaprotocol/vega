// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package types

import eventpb "code.vegaprotocol.io/vega/protos/vega/events/v1"

type LossType = eventpb.LossSocialization_Type

const (
	// LossTypeUnspecified is the default value.
	LossTypeUnspecified LossType = eventpb.LossSocialization_TYPE_UNSPECIFIED
	// LossTypeSettlement indicates loss socialisation occurred during MTM or final settlement.
	LossTypeSettlement LossType = eventpb.LossSocialization_TYPE_SETTLEMENT
	// LossTypeFunding indicates loss socialisation occurred during funding payments.
	LossTypeFunding LossType = eventpb.LossSocialization_TYPE_FUNDING_PAYMENT
)
