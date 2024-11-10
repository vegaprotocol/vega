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

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

var (
	vault1    = "b2f72afdf7678c22395527b874628c2cc26d1b62871eadbf3f54448c91e0517f"
	vault2    = "b57011103a2078c61a071850b101e58920f604ad2ba8698e0c655f15de2e45a0"
	vault3    = "e85395a01af16280bcdf6e8cfdc09bc7bfba5ef3ff24a6405c5d111933db9e16"
	party1    = "cd789e52ff3b59133cf7fa863b3c49bf3e166206292e389c12b9cea21a58895c"
	party2    = "c2eccddbde2b623e11458fdbaabd311709b126a46b2a107462708ba6d3749cdd"
	party3    = "b9361f9a5afe9485bc769f6339e68a839dafd3f86f834040e5fc3d0a62f17e1f"
	asset1    = "2f02f7c7b9477d209b0fff14e82cc124f5c6285f4f0c2ec3fdbfd3d8f6e85cff"
	asset2    = "dc464435256d3e40d84229faf1ab1f1ddd0dfb683ed0dd4a903c40147c512538"
	asset3    = "6b3b6ab23f93bf1e225f8d7f2cd84befce9fc28766feb44c2f355c512a8c2a35"
	request11 = "09a5fb5e76fd08353bfdf4dfec6ea7771da8ecea8116c26de2fca7d3d38074e3"
	request12 = "07e7cf9046dba213da5862384479b55813cd539aff8611b597adc14207fffae8"
	request21 = "a5ae20ec5d9e098a42654041c884725208d8bb8b673ff0b919b2a76efffe1ade"
	request22 = "e2664f8056d73ee0d3dcae5cb8b84dc24667bc6cb9852d6cb37ce00f493d1f35"
	request31 = "7a2ccdefff0f1f5facfb2c4e6bc6ce37baacfab951b01c1e06cd6300beff848b"
	request32 = "9a28aceb31eaf7d30c3eb52b5f47a1f0835572539191da9f7b646b87220e78b9"
)

func setupVaultRedemptionsTest(t *testing.T) *sqlstore.VaultRedemptions {
	t.Helper()
	plbs := sqlstore.NewVaultRedemptions(connectionSource)

	return plbs
}

func TestVaultRedemptionsInsert(t *testing.T) {
	vr := setupVaultRedemptionsTest(t)

	const (
		vault1 = "70432aa1dc6bc20a9b404d30f23e6a8def11a1692609dcef0ad8dc558d9df7db"
		party1 = "2e7a16d9ef690f0d2beed115fba13ba2aaa16c8f971910ad88c72b9db010c7d4"
	)

	var (
		asset1   = crypto.RandomHash()
		request1 = crypto.RandomHash()
	)

	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)

	t.Run("can insert successfully", func(t *testing.T) {
		w := entities.RedemptionRequest{
			RequestID:       entities.RedemptionRequestID(request1),
			VaultID:         entities.VaultID(vault1),
			PartyID:         entities.PartyID(party1),
			Asset:           entities.AssetID(asset1),
			RequestedAmount: num.DecimalFromFloat(100),
			RemainingAmount: num.DecimalFromFloat(5),
			EligibilityDate: now.Add(-24 * time.Hour),
			LastUpdated:     now,
			Status:          entities.RedeemStatusLate,
			VegaTime:        now,
		}

		require.NoError(t, vr.Add(ctx, &w))

		rr, err := vr.GetByRequestID(ctx, request1)
		require.NoError(t, err)
		require.Equal(t, party1, rr.PartyID.String())
		require.Equal(t, vault1, rr.VaultID.String())
		require.Equal(t, asset1, rr.Asset.String())
		require.Equal(t, "100", rr.RequestedAmount.String())
		require.Equal(t, "5", rr.RemainingAmount.String())
		require.Equal(t, now.UnixNano(), rr.LastUpdated.UnixNano())
		require.Equal(t, now.Add(-24*time.Hour).UnixNano(), rr.EligibilityDate.UnixNano())
		require.Equal(t, entities.RedeemStatusLate, rr.Status)
	})

	now = now.Add(24 * time.Hour).Truncate(time.Millisecond)

	t.Run("can replace exisisting values", func(t *testing.T) {
		w := entities.RedemptionRequest{
			RequestID:       entities.RedemptionRequestID(request1),
			VaultID:         entities.VaultID(vault1),
			PartyID:         entities.PartyID(party1),
			Asset:           entities.AssetID(asset1),
			RequestedAmount: num.DecimalFromFloat(100),
			RemainingAmount: num.DecimalFromFloat(1),
			EligibilityDate: now.Add(-24 * time.Hour),
			LastUpdated:     now.Add(2 * time.Hour),
			Status:          entities.RedeemStatusCompleted,
			VegaTime:        now,
		}
		require.NoError(t, vr.Add(ctx, &w))
		rr, err := vr.GetByRequestID(ctx, request1)
		require.NoError(t, err)
		require.NoError(t, err)
		require.Equal(t, party1, rr.PartyID.String())
		require.Equal(t, vault1, rr.VaultID.String())
		require.Equal(t, asset1, rr.Asset.String())
		require.Equal(t, "100", rr.RequestedAmount.String())
		require.Equal(t, "1", rr.RemainingAmount.String())
		require.Equal(t, now.Add(2*time.Hour).UnixNano(), rr.LastUpdated.UnixNano())
		require.Equal(t, now.Add(-24*time.Hour).UnixNano(), rr.EligibilityDate.UnixNano())
		require.Equal(t, entities.RedeemStatusCompleted, rr.Status)
	})
}

func setupRedemptionRequests(t *testing.T, ctx context.Context, vr *sqlstore.VaultRedemptions, now time.Time) (entities.RedemptionRequest, entities.RedemptionRequest, entities.RedemptionRequest, entities.RedemptionRequest, entities.RedemptionRequest, entities.RedemptionRequest) {
	t.Helper()
	r11 := entities.RedemptionRequest{
		RequestID:       entities.RedemptionRequestID(request11),
		VaultID:         entities.VaultID(vault1),
		PartyID:         entities.PartyID(party1),
		Asset:           entities.AssetID(asset1),
		RequestedAmount: num.DecimalFromFloat(100),
		RemainingAmount: num.DecimalFromFloat(100),
		EligibilityDate: now.Add(-24 * time.Hour),
		LastUpdated:     now,
		Status:          entities.RedeemStatusPending,
		VegaTime:        now,
	}

	r12 := entities.RedemptionRequest{
		RequestID:       entities.RedemptionRequestID(request12),
		VaultID:         entities.VaultID(vault1),
		PartyID:         entities.PartyID(party2),
		Asset:           entities.AssetID(asset1),
		RequestedAmount: num.DecimalFromFloat(100),
		RemainingAmount: num.DecimalFromFloat(100),
		EligibilityDate: now.Add(-24 * time.Hour),
		LastUpdated:     now,
		Status:          entities.RedeemStatusCompleted,
		VegaTime:        now,
	}

	r21 := entities.RedemptionRequest{
		RequestID:       entities.RedemptionRequestID(request21),
		VaultID:         entities.VaultID(vault2),
		PartyID:         entities.PartyID(party2),
		Asset:           entities.AssetID(asset2),
		RequestedAmount: num.DecimalFromFloat(300),
		RemainingAmount: num.DecimalFromFloat(300),
		EligibilityDate: now.Add(-24 * time.Hour),
		LastUpdated:     now,
		Status:          entities.RedeemStatusCompleted,
		VegaTime:        now,
	}

	r22 := entities.RedemptionRequest{
		RequestID:       entities.RedemptionRequestID(request22),
		VaultID:         entities.VaultID(vault2),
		PartyID:         entities.PartyID(party3),
		Asset:           entities.AssetID(asset2),
		RequestedAmount: num.DecimalFromFloat(400),
		RemainingAmount: num.DecimalFromFloat(400),
		EligibilityDate: now.Add(-24 * time.Hour),
		LastUpdated:     now,
		Status:          entities.RedeemStatusLate,
		VegaTime:        now,
	}

	r31 := entities.RedemptionRequest{
		RequestID:       entities.RedemptionRequestID(request31),
		VaultID:         entities.VaultID(vault3),
		PartyID:         entities.PartyID(party3),
		Asset:           entities.AssetID(asset3),
		RequestedAmount: num.DecimalFromFloat(500),
		RemainingAmount: num.DecimalFromFloat(500),
		EligibilityDate: now.Add(-24 * time.Hour),
		LastUpdated:     now,
		Status:          entities.RedeemStatusLate,
		VegaTime:        now,
	}

	r32 := entities.RedemptionRequest{
		RequestID:       entities.RedemptionRequestID(request32),
		VaultID:         entities.VaultID(vault3),
		PartyID:         entities.PartyID(party1),
		Asset:           entities.AssetID(asset3),
		RequestedAmount: num.DecimalFromFloat(600),
		RemainingAmount: num.DecimalFromFloat(600),
		EligibilityDate: now.Add(-24 * time.Hour),
		LastUpdated:     now,
		Status:          entities.RedeemStatusPending,
		VegaTime:        now,
	}

	require.NoError(t, vr.Add(ctx, &r11))
	require.NoError(t, vr.Add(ctx, &r12))
	require.NoError(t, vr.Add(ctx, &r21))
	require.NoError(t, vr.Add(ctx, &r22))
	require.NoError(t, vr.Add(ctx, &r31))
	require.NoError(t, vr.Add(ctx, &r32))
	return r11, r12, r21, r22, r31, r32
}

func TestListVaultRedemptionsFilterByVaults(t *testing.T) {
	vr := setupVaultRedemptionsTest(t)
	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)
	r11, r12, r21, r22, r31, r32 := setupRedemptionRequests(t, ctx, vr, now)
	// expect to get only redemptions from vault1
	rr, _, err := vr.ListRedemptionRequestsWithCursor(ctx, []string{vault1}, []string{}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r11, rr[1])

	// expect to get only redemptions from vault2
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{vault2}, []string{}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r21, rr[0])
	requireEqual(t, r22, rr[1])

	// expect to get only redemptions from vault3
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{vault3}, []string{}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r31, rr[0])
	requireEqual(t, r32, rr[1])

	// expect to get only redemptions from vault1, vault2
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{vault1, vault2}, []string{}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 4, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r11, rr[1])
	requireEqual(t, r21, rr[2])
	requireEqual(t, r22, rr[3])
}

func TestListVaultRedemptionsFilterByParty(t *testing.T) {
	vr := setupVaultRedemptionsTest(t)
	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)
	r11, r12, r21, r22, r31, r32 := setupRedemptionRequests(t, ctx, vr, now)

	// expect to get only redemptions from party1 in requests 11 and 32
	rr, _, err := vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{party1}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r11, rr[0])
	requireEqual(t, r32, rr[1])

	// expect to get only redemptions from party2 in requests 12 and 21
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{party2}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r21, rr[1])

	// expect to get only redemptions from vault3 in requests 22 and 31
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{party3}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r31, rr[0])
	requireEqual(t, r22, rr[1])

	// expect to get only redemptions from party1, party2 i.e.
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{party1, party2}, []string{}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 4, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r11, rr[1])
	requireEqual(t, r32, rr[2])
	requireEqual(t, r21, rr[3])
}

func TestListVaultRedemptionsFilterByAssets(t *testing.T) {
	vr := setupVaultRedemptionsTest(t)
	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)
	r11, r12, r21, r22, r31, r32 := setupRedemptionRequests(t, ctx, vr, now)

	// expect to get only redemptions from asset1
	rr, _, err := vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{asset1}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r11, rr[1])

	// expect to get only redemptions from asset2
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{asset2}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r21, rr[0])
	requireEqual(t, r22, rr[1])

	// expect to get only redemptions from asset3
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{asset3}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r31, rr[0])
	requireEqual(t, r32, rr[1])

	// expect to get only redemptions from asset1, asset2
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{asset1, asset2}, []vega.RedeemStatus{}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 4, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r11, rr[1])
	requireEqual(t, r21, rr[2])
	requireEqual(t, r22, rr[3])
}

func TestListVaultRedemptionsFilterByStatuses(t *testing.T) {
	vr := setupVaultRedemptionsTest(t)
	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)
	r11, r12, r21, r22, r31, r32 := setupRedemptionRequests(t, ctx, vr, now)

	// expect to get only redemptions with status pending - that would be r11, r32
	rr, _, err := vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{}, []vega.RedeemStatus{vega.RedeemStatus_REDEEM_STATUS_PENDING}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r11, rr[0])
	requireEqual(t, r32, rr[1])

	// expect to get only redemptions with status late - that would be r22, r31
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{}, []vega.RedeemStatus{vega.RedeemStatus_REDEEM_STATUS_LATE}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r31, rr[0])
	requireEqual(t, r22, rr[1])

	// expect to get only redemptions with status comleted - that would be r12, r21
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{}, []vega.RedeemStatus{vega.RedeemStatus_REDEEM_STATUS_COMPLETED}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r21, rr[1])

	// expect to get only redemptions from status completed and status late - i.e. r12, r21, r22, r31
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{}, []string{}, []string{}, []vega.RedeemStatus{vega.RedeemStatus_REDEEM_STATUS_COMPLETED, vega.RedeemStatus_REDEEM_STATUS_LATE}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 4, len(rr))
	requireEqual(t, r12, rr[0])
	requireEqual(t, r31, rr[1])
	requireEqual(t, r21, rr[2])
	requireEqual(t, r22, rr[3])
}

func TestFilterCombination(t *testing.T) {
	vr := setupVaultRedemptionsTest(t)
	ctx := tempTransaction(t)
	now := time.Now().Truncate(time.Millisecond)
	r11, _, _, r22, r31, _ := setupRedemptionRequests(t, ctx, vr, now)

	// expect to get only redemptions frmo vault1 and vault2 for parties party1, party3 with statuses late or pending
	// that would be r11 and r22
	rr, _, err := vr.ListRedemptionRequestsWithCursor(ctx, []string{vault1, vault2}, []string{party1, party3}, []string{}, []vega.RedeemStatus{vega.RedeemStatus_REDEEM_STATUS_LATE, vega.RedeemStatus_REDEEM_STATUS_PENDING}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 2, len(rr))
	requireEqual(t, r11, rr[0])
	requireEqual(t, r22, rr[1])

	// only vault3, for party1 and party3 where status is late or completed, that is only r31
	rr, _, err = vr.ListRedemptionRequestsWithCursor(ctx, []string{vault3}, []string{party1, party3}, []string{}, []vega.RedeemStatus{vega.RedeemStatus_REDEEM_STATUS_LATE, vega.RedeemStatus_REDEEM_STATUS_COMPLETED}, entities.DefaultCursorPagination(false))
	require.NoError(t, err)
	require.Equal(t, 1, len(rr))
	requireEqual(t, r31, rr[0])
}

func requireEqual(t *testing.T, expected, actual entities.RedemptionRequest) {
	t.Helper()
	require.Equal(t, expected.RequestID, actual.RequestID)
	require.Equal(t, expected.VaultID, actual.VaultID)
	require.Equal(t, expected.PartyID, actual.PartyID)
	require.Equal(t, expected.Asset, actual.Asset)
	require.Equal(t, expected.RequestedAmount.String(), actual.RequestedAmount.String())
	require.Equal(t, expected.RemainingAmount.String(), actual.RemainingAmount.String())
	require.Equal(t, expected.EligibilityDate.String(), actual.EligibilityDate.String())
	require.Equal(t, expected.LastUpdated.String(), actual.LastUpdated.String())
	require.Equal(t, expected.Status, actual.Status)
}
