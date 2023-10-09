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

package ratelimit

type Config struct {
	// How many requests
	Requests int `description:" " long:"requests"`

	// In the last `PerNBlocks` blocks
	PerNBlocks int `description:" " long:"per-n-blocks"`
}

// NewDefaultConfig allows 500 requests in the last 10 blocks.
func NewDefaultConfig() Config {
	return Config{
		Requests:   500,
		PerNBlocks: 10,
	}
}
