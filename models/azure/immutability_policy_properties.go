// Copyright (C) 2022 Specter Ops, Inc.
//
// This file is part of AzureHound.
//
// AzureHound is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// AzureHound is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package azure

import "github.com/bloodhoundad/azurehound/enums"

type ImmutabilityPolicyProperties struct {
	AllowProtectedAppendWrites            bool                          `json:"allowProtectedAppendWrites,omitempty"`
	AllowProtectedAppendWritesAll         bool                          `json:"allowProtectedAppendWritesAll,omitempty"`
	ImmutabilityPeriodSinceCreationInDays int                           `json:"immutabilityPeriodSinceCreationInDays,omitempty"`
	State                                 enums.ImmutabilityPolicyState `json:"updateHistory,omitempty"`
}
