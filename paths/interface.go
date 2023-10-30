// Copyright (C) 2023  Gobalsky Labs Limited
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

package paths

// nolint: interfacebloat
type Paths interface {
	CreateCachePathFor(CachePath) (string, error)
	CreateCacheDirFor(CachePath) (string, error)
	CreateConfigPathFor(ConfigPath) (string, error)
	CreateConfigDirFor(ConfigPath) (string, error)
	CreateDataPathFor(DataPath) (string, error)
	CreateDataDirFor(DataPath) (string, error)
	CreateStatePathFor(StatePath) (string, error)
	CreateStateDirFor(StatePath) (string, error)
	CachePathFor(CachePath) string
	ConfigPathFor(ConfigPath) string
	DataPathFor(DataPath) string
	StatePathFor(StatePath) string
}

// New instantiates the specific implementation of the Paths interface based on
// the value of the customHome. If a customHome is specified the custom
// implementation CustomPaths is returned, the standard DefaultPaths otherwise.
func New(customHome string) Paths {
	if len(customHome) != 0 {
		return &CustomPaths{
			CustomHome: customHome,
		}
	}

	return &DefaultPaths{}
}
